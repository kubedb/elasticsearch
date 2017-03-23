package controller

import (
	"fmt"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

func (w *Controller) ensureElastic() {
	resourceName := tapi.ResourceNameElastic + "." + tapi.V1beta1SchemeGroupVersion.Group

	if _, err := w.Client.Extensions().ThirdPartyResources().Get(resourceName); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Fatalln(err)
		}
	} else {
		return
	}

	thirdPartyResource := &extensions.ThirdPartyResource{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "ThirdPartyResource",
		},
		ObjectMeta: kapi.ObjectMeta{
			Name: resourceName,
		},
		Versions: []extensions.APIVersion{
			{
				Name: tapi.V1beta1SchemeGroupVersion.Version,
			},
		},
	}

	if _, err := w.Client.Extensions().ThirdPartyResources().Create(thirdPartyResource); err != nil {
		log.Fatalln(err)
	}
}

func (w *Controller) checkGoverningServiceAccount(name, namespace string) (bool, error) {
	serviceAccount, err := w.Client.Core().ServiceAccounts(namespace).Get(name)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	if serviceAccount == nil {
		return false, nil
	}

	return true, nil
}

func (w *Controller) checkService(namespace, serviceName string) (bool, error) {
	service, err := w.Client.Core().Services(namespace).Get(serviceName)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	if service == nil {
		return false, nil
	}

	if service.Spec.Selector[LabelDatabaseName] != serviceName {
		return false, fmt.Errorf(`Intended service "%v" already exists`, serviceName)
	}

	return true, nil
}

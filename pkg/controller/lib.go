package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapps "k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/intstr"
)

func (w *Controller) ensureElastic() {
	resourceName := tapi.ResourceNameElastic + "." + tapi.V1beta1SchemeGroupVersion.Group

	if _, err := w.Client.Extensions().ThirdPartyResources().Get(resourceName); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Fatal(err)
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
		log.Fatal(err)
	}
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
		return false, errors.New(fmt.Sprintf(`Intended service "%v" already exists`, serviceName))
	}

	return true, nil
}

func (w *Controller) createService(namespace, serviceName string) error {
	// Check if service name exists
	found, err := w.checkService(namespace, serviceName)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	label := map[string]string{
		LabelDatabaseName: serviceName,
	}
	service := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:   serviceName,
			Labels: label,
		},
		Spec: kapi.ServiceSpec{
			Ports: []kapi.ServicePort{
				{
					Name:       "api",
					Port:       9200,
					TargetPort: intstr.FromString("api"),
				},
				{
					Name:       "tcp",
					Port:       9300,
					TargetPort: intstr.FromString("tcp"),
				},
			},
			Selector: label,
		},
	}

	if _, err := w.Client.Core().Services(namespace).Create(service); err != nil {
		return err
	}

	return nil
}

func (w *Controller) deleteService(namespace, serviceName string) error {
	service, err := w.Client.Core().Services(namespace).Get(serviceName)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}
	if service == nil {
		return nil
	}

	if service.Spec.Selector[LabelDatabaseName] != serviceName {
		return nil
	}

	return w.Client.Core().Services(namespace).Delete(serviceName, nil)
}

func (w *Controller) checkGoverningServiceAccount(namespace, name string) (bool, error) {
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

func (w *Controller) createGoverningServiceAccount(namespace, name string) error {
	found, err := w.checkGoverningServiceAccount(namespace, name)
	if err != nil {
		return err

	}
	if found {
		return nil
	}

	serviceAccount := &kapi.ServiceAccount{
		ObjectMeta: kapi.ObjectMeta{
			Name: name,
		},
	}

	if _, err = w.Client.Core().ServiceAccounts(namespace).Create(serviceAccount); err != nil {
		return err

	}
	return nil
}

func (c *Controller) deleteStatefulSet(statefulSet *kapps.StatefulSet) error {
	// Update StatefulSet
	statefulSet.Spec.Replicas = 0
	if _, err := c.Client.Apps().StatefulSets(statefulSet.Namespace).Update(statefulSet); err != nil {
		return err
	}

	labelSelector := labels.SelectorFromSet(statefulSet.Spec.Selector.MatchLabels)

	check := 1
	for {
		time.Sleep(time.Second * 30)
		podList, err := c.Client.Core().Pods(kapi.NamespaceAll).List(kapi.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}
		if len(podList.Items) == 0 {
			break
		}

		if check == 5 {
			return errors.New("Fail to delete StatefulSet Pods")
		}
		check++
	}

	// Delete StatefulSet
	return c.Client.Apps().StatefulSets(statefulSet.Namespace).Delete(statefulSet.Name, nil)
}

package controller

import (
	"errors"
	"fmt"

	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
)

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

	if service.Spec.Selector[serviceSelector] != serviceName {
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
		serviceSelector: serviceName,
	}
	service := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:   serviceName,
			Labels: label,
		},
		Spec: kapi.ServiceSpec{
			Ports: []kapi.ServicePort{
				{
					Name: "api",
					Port: 9200,
				},
				{
					Name: "tcp",
					Port: 9300,
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

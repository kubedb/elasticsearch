package controller

import (
	"errors"
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/labels"
)

func (w *Controller) deleteService(name, namespace string) error {
	service, err := w.Client.Core().Services(namespace).Get(name)
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

	if service.Spec.Selector[LabelDatabaseName] != name {
		return nil
	}

	return w.Client.Core().Services(namespace).Delete(name, nil)
}

func (c *Controller) deleteStatefulSet(name, namespace string) error {
	statefulSet, err := c.Client.Apps().StatefulSets(namespace).Get(name)
	if err != nil {
		return err
	}

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

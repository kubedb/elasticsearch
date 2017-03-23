package controller

import (
	kapps "k8s.io/kubernetes/pkg/apis/apps"
)

func (w *Controller) updateStatefulSet(statefulSet *kapps.StatefulSet) error {
	_, err := w.Client.Apps().StatefulSets(statefulSet.Namespace).Update(statefulSet)
	return err
}

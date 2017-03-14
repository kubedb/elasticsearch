package controller

import (
	"fmt"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
)

func (w *Controller) update(updatedElastic *tapi.Elastic) {
	newReplicas := updatedElastic.Spec.Replicas

	statefulSetName := fmt.Sprintf("%v-%v", DatabaseNamePrefix, updatedElastic.Name)
	statefulSet, err := w.Client.Apps().StatefulSets(updatedElastic.Namespace).Get(statefulSetName)
	if err != nil {
		log.Errorln(err)
		return
	}

	statefulSet.Spec.Replicas = newReplicas
	if _, err := w.Client.Apps().StatefulSets(statefulSet.Namespace).Update(statefulSet); err != nil {
		log.Errorln(err)
		return
	}
}

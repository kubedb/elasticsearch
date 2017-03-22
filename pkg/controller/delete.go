package controller

import (
	"fmt"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	"gopkg.in/robfig/cron.v2"
)

func (w *Controller) delete(elastic *tapi.Elastic) {
	statefulSetName := fmt.Sprintf("%v-%v", DatabaseNamePrefix, elastic.Name)
	statefulSet, err := w.Client.Apps().StatefulSets(elastic.Namespace).Get(statefulSetName)
	if err != nil {
		log.Error(err)
		return
	}
	// Delete StatefulSet
	if err := w.deleteStatefulSet(statefulSet); err != nil {
		log.Error(err)
		return
	}
	// Delete Service
	if err := w.deleteService(elastic.Namespace, elastic.Name); err != nil {
		log.Error(err)
		return
	}

	// Remove previous cron job if exist
	if id, exists := w.cronEntryIDs.Pop(elastic.Name); exists {
		w.cron.Remove(id.(cron.EntryID))
	}
}

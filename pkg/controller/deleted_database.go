package controller

import (
	"fmt"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/labels"
)

type Deleter struct {
	*amc.Controller
}

func NewDeleter(c *amc.Controller) amc.Deleter {
	return &Deleter{c}
}

func (d *Deleter) Exists(deletedDb *tapi.DeletedDatabase) (bool, error) {

	if _, err := d.ExtClient.Elastics(deletedDb.Namespace).Get(deletedDb.Name); err != nil {
		if !k8serr.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

func (d *Deleter) DeleteDatabase(deletedDb *tapi.DeletedDatabase) error {
	// Delete Service
	if err := d.deleteService(deletedDb.Namespace, deletedDb.Name); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
	}

	statefulSetName := fmt.Sprintf("%v-%v", amc.DatabaseNamePrefix, deletedDb.Name)
	if err := d.deleteStatefulSet(statefulSetName, deletedDb.Namespace); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
	}
	return nil
}

func (d *Deleter) DestroyDatabase(deletedDb *tapi.DeletedDatabase) error {

	labelMap := map[string]string{
		amc.LabelDatabaseName: deletedDb.Name,
		amc.LabelDatabaseType: DatabaseElasticsearch,
	}

	labelSelector := labels.SelectorFromSet(labelMap)

	if err := d.DeletePersistentVolumeClaims(deletedDb.Namespace, labelSelector); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
	}
	return nil
}

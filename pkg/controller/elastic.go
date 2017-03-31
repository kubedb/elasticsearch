package controller

import (
	"fmt"
	"reflect"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
)

type dbController struct {
	*Controller
}

func (c *dbController) create(elastic *tapi.Elastic) {
	if err := c.validateElastic(elastic); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
		return
	}

	// create Governing Service
	governingService := GoverningElasticsearch
	if elastic.Spec.ServiceAccountName != "" {
		governingService = elastic.Spec.ServiceAccountName
	}

	if err := c.CreateGoverningServiceAccount(governingService, elastic.Namespace); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
		return
	}
	elastic.Spec.ServiceAccountName = governingService

	// create database Service
	if err := c.createService(elastic.Name, elastic.Namespace); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
		return
	}

	// Create statefulSet for Elastic database
	statefulSet, err := c.createStatefulSet(elastic)
	if err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
		return
	}

	// Check StatefulSet Pod status
	if err := c.CheckStatefulSets(statefulSet, durationCheckStatefulSet); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
		return
	}

	// Setup Schedule backup
	if elastic.Spec.BackupSchedule != nil {
		err := c.cronController.ScheduleBackup(elastic, elastic.ObjectMeta, elastic.Spec.BackupSchedule)
		if err != nil {
			/*
				TODO: Event
			*/
			log.Errorln(err)
		}
	}
}

func (c *dbController) delete(elastic *tapi.Elastic) {
	// Delete Service
	if err := c.deleteService(elastic.Namespace, elastic.Name); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
	}

	statefulSetName := fmt.Sprintf("%v-%v", DatabaseNamePrefix, elastic.Name)
	if err := c.deleteStatefulSet(statefulSetName, elastic.Namespace); err != nil {
		/*
			TODO: Event
		*/
		log.Errorln(err)
	}

	c.cronController.StopScheduleBackup(elastic.ObjectMeta)
}

func (c *dbController) update(oldElastic, updatedElastic *tapi.Elastic) {
	if (updatedElastic.Spec.Replicas != oldElastic.Spec.Replicas) && oldElastic.Spec.Replicas >= 0 {
		statefulSetName := fmt.Sprintf("%v-%v", DatabaseNamePrefix, updatedElastic.Name)
		statefulSet, err := c.Client.Apps().StatefulSets(updatedElastic.Namespace).Get(statefulSetName)
		if err != nil {
			/*
				TODO: Event
			*/
			log.Errorln(err)
			return
		}
		statefulSet.Spec.Replicas = oldElastic.Spec.Replicas
		if _, err := c.Client.Apps().StatefulSets(statefulSet.Namespace).Update(statefulSet); err != nil {
			/*
				TODO: Event
			*/
			log.Errorln(err)
			return
		}
	}

	if !reflect.DeepEqual(updatedElastic.Spec.BackupSchedule, oldElastic.Spec.BackupSchedule) {
		backupScheduleSpec := updatedElastic.Spec.BackupSchedule
		if backupScheduleSpec != nil {
			if err := c.ValidateBackupSchedule(backupScheduleSpec); err != nil {
				/*
					TODO: Event
				*/
				return
			}

			if err := c.CheckBucketAccess(
				backupScheduleSpec.BucketName, backupScheduleSpec.StorageSecret,
				updatedElastic.Namespace); err != nil {
				/*
					TODO: Event
				*/
				return
			}

			if err := c.cronController.ScheduleBackup(
				oldElastic, oldElastic.ObjectMeta, oldElastic.Spec.BackupSchedule); err != nil {
				/*
					TODO: Event
				*/
			}
		} else {
			c.cronController.StopScheduleBackup(oldElastic.ObjectMeta)
		}
	}
}

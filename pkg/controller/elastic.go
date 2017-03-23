package controller

import (
	"fmt"
	"reflect"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	"gopkg.in/robfig/cron.v2"
)

type dbController struct {
	*Controller
}

func (e *dbController) create(elastic *tapi.Elastic) {
	if err := e.validateElastic(elastic); err != nil {
		log.Errorln(err)
		return
	}

	// create Governing Service
	governingService := GoverningElasticsearch
	if elastic.Spec.ServiceAccountName != "" {
		governingService = elastic.Spec.ServiceAccountName
	}
	if err := e.createGoverningServiceAccount(governingService, elastic.Namespace); err != nil {
		log.Errorln(err)
		return
	}
	elastic.Spec.ServiceAccountName = governingService

	// create database Service
	if err := e.createService(elastic.Name, elastic.Namespace); err != nil {
		log.Errorln(err)
		return
	}

	// Create statefulSet for Elastic database
	statefulSet, err := e.createStatefulSet(elastic)
	if err != nil {
		log.Errorln(err)
		return
	}

	// Check StatefulSet Pod status
	if err := e.CheckStatefulSets(statefulSet, durationCheckStatefulSet); err != nil {
		log.Errorln(err)
		return
	}

	// Setup Schedule backup
	if elastic.Spec.BackupSchedule != nil {
		if err := e.scheduleBackup(elastic); err != nil {
			log.Errorln(err)
			return
		}
	}
}

func (e *dbController) delete(elastic *tapi.Elastic) {
	// Delete Service
	if err := e.deleteService(elastic.Namespace, elastic.Name); err != nil {
		log.Errorln(err)
	}

	statefulSetName := fmt.Sprintf("%v-%v", DatabaseNamePrefix, elastic.Name)
	if err := e.deleteStatefulSet(statefulSetName, elastic.Namespace); err != nil {
		log.Errorln(err)
	}

	// Remove previous cron job if exist
	if id, exists := e.cronEntryIDs.Pop(elastic.Name); exists {
		e.cron.Remove(id.(cron.EntryID))
	}
}

func (e *dbController) update(oldElastic, updatedElastic *tapi.Elastic) {
	if (updatedElastic.Spec.Replicas != oldElastic.Spec.Replicas) && oldElastic.Spec.Replicas >= 0 {
		statefulSetName := fmt.Sprintf("%v-%v", DatabaseNamePrefix, updatedElastic.Name)
		statefulSet, err := e.Client.Apps().StatefulSets(updatedElastic.Namespace).Get(statefulSetName)
		if err != nil {
			log.Errorln(err)
			return
		}
		statefulSet.Spec.Replicas = oldElastic.Spec.Replicas
		if err := e.updateStatefulSet(statefulSet); err != nil {
			log.Errorln(err)
			return
		}
	}

	if !reflect.DeepEqual(updatedElastic.Spec.BackupSchedule, oldElastic.Spec.BackupSchedule) {
		if updatedElastic.Spec.BackupSchedule != nil {

			// CronExpression can't be empty
			backupSchedule := updatedElastic.Spec.BackupSchedule
			if backupSchedule.CronExpression == "" {
				log.Errorln("Invalid cron expression")
				return
			}

			// Validate backup spec
			if err := e.validateBackupSpec(backupSchedule.SnapshotSpec, updatedElastic.Namespace); err != nil {
				log.Errorln(err)
				return
			}

			if err := e.scheduleBackup(updatedElastic); err != nil {
				log.Errorln(err)
				return
			}
		} else {
			// Remove previous cron job if exist
			if id, exists := e.cronEntryIDs.Pop(updatedElastic.Name); exists {
				e.cron.Remove(id.(cron.EntryID))
			}
		}
	}
}

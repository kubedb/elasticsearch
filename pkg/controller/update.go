package controller

import (
	"fmt"
	"reflect"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	"gopkg.in/robfig/cron.v2"
)

func (w *Controller) update(oldElastic, updatedElastic *tapi.Elastic) {
	if updatedElastic.Spec.Replicas != oldElastic.Spec.Replicas {
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

	if !reflect.DeepEqual(updatedElastic.Spec.BackupSchedule, oldElastic.Spec.BackupSchedule) {
		if updatedElastic.Spec.BackupSchedule != nil {

			// CronExpression can't be empty
			backupSchedule := updatedElastic.Spec.BackupSchedule
			if backupSchedule.CronExpression == "" {
				log.Errorln("Invalid cron expression")
				return
			}

			// Validate backup spec
			if err := w.validateBackupSpec(backupSchedule.SnapshotSpec, updatedElastic.Namespace); err != nil {
				log.Errorln(err)
				return
			}

			if err := w.ScheduleBackup(updatedElastic); err != nil {
				log.Errorln(err)
				return
			}
		} else {
			// Remove previous cron job if exist
			if id, exists := w.cronEntryIDs.Pop(updatedElastic.Name); exists {
				w.cron.Remove(id.(cron.EntryID))
			}
		}
	}
}

package controller

import (
	"fmt"
	"reflect"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	"github.com/k8sdb/apimachinery/pkg/eventer"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
)

type elasticController struct {
	*Controller
}

func (c *elasticController) create(elastic *tapi.Elastic) {
	if err := c.validateElastic(elastic); err != nil {
		c.eventRecorder.PushEvent(kapi.EventTypeWarning, eventer.EventReasonInvalid, err.Error(), elastic)
		log.Errorln(err)
		return
	}
	// Event for successful validation
	c.eventRecorder.PushEvent(
		kapi.EventTypeNormal, eventer.EventReasonSuccessfulValidate, "Successfully validate Elastic", elastic,
	)

	// Check if DeletedDatabase exists or not
	deletedDb, err := c.ExtClient.DeletedDatabases(elastic.Namespace).Get(elastic.Name)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			message := fmt.Sprintf(`Fail to get DeletedDatabase: "%v". Reason: %v`, elastic.Name, err)
			c.eventRecorder.PushEvent(kapi.EventTypeWarning, eventer.EventReasonFailedToGet, message, elastic)
			log.Errorln(err)
			return
		}
	} else {
		/*
			If exists, check label for database type in label.
			* If   - DeletedDatabase type is "elasticsearch", continue create process
			* Else - Do not create Further kubernetes objects.
			         These may overlaps/conflicts database data
			Destroy previous database and delete DeletedDatabase object
			Or use different name for your new Elastic database
		*/
		if deletedDb.Labels[LabelDatabaseType] != DatabaseElasticsearch {
			message := fmt.Sprintf(`Invalid Elastic: "%v". Exists irrelevent DeletedDatabase: "%v"`,
				elastic.Name, deletedDb.Name)
			// Set status to Failed
			elastic.Status.DatabaseStatus = tapi.StatusDatabaseFailed
			elastic.Status.Reason = message
			if _, err := c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
				message := fmt.Sprintf(`Fail to update Elastic: "%v". Reason: %v`, elastic.Name, err)
				c.eventRecorder.PushEvent(
					kapi.EventTypeWarning, eventer.EventReasonFailedToUpdate, message, elastic,
				)
				log.Errorln(err)
			}
			c.eventRecorder.PushEvent(
				kapi.EventTypeWarning, eventer.EventReasonFailedToCreate, message, elastic,
			)
			log.Infoln(message)
			return
		}
	}

	// Event for notification that kubernetes objects are creating
	c.eventRecorder.PushEvent(
		kapi.EventTypeNormal, eventer.EventReasonCreating, "Creating Kubernetes objects", elastic,
	)

	// create Governing Service
	governingService := GoverningElasticsearch
	if elastic.Spec.ServiceAccountName != "" {
		governingService = elastic.Spec.ServiceAccountName
	}

	if err := c.CreateGoverningServiceAccount(governingService, elastic.Namespace); err != nil {
		message := fmt.Sprintf(`Failed to create ServiceAccount: "%v". Reason: %v`, governingService, err)
		c.eventRecorder.PushEvent(kapi.EventTypeWarning, eventer.EventReasonFailedToCreate, message, elastic)
		log.Errorln(err)
		return
	}
	elastic.Spec.ServiceAccountName = governingService

	// create database Service
	if err := c.createService(elastic.Name, elastic.Namespace); err != nil {
		message := fmt.Sprintf(`Failed to create Service. Reason: %v`, err)
		c.eventRecorder.PushEvent(kapi.EventTypeWarning, eventer.EventReasonFailedToCreate, message, elastic)
		log.Errorln(err)
		return
	}

	// Create statefulSet for Elastic database
	statefulSet, err := c.createStatefulSet(elastic)
	if err != nil {
		message := fmt.Sprintf(`Failed to create StatefulSet. Reason: %v`, err)
		c.eventRecorder.PushEvent(kapi.EventTypeWarning, eventer.EventReasonFailedToCreate, message, elastic)
		log.Errorln(err)
		return
	}

	// Check StatefulSet Pod status
	if elastic.Spec.Replicas > 0 {
		if err := c.CheckStatefulSets(statefulSet, durationCheckStatefulSet); err != nil {
			message := fmt.Sprintf(`Failed to create StatefulSet. Reason: %v`, err)
			c.eventRecorder.PushEvent(
				kapi.EventTypeWarning, eventer.EventReasonFailedToStart, message, elastic,
			)
			log.Errorln(err)
			return
		} else {
			c.eventRecorder.PushEvent(
				kapi.EventTypeNormal, eventer.EventReasonSuccessfulCreate, "Successfully created Elastic",
				elastic,
			)
		}
	}

	if deletedDb != nil {
		// Delete DeletedDatabase instance
		if err := c.ExtClient.DeletedDatabases(deletedDb.Namespace).Delete(deletedDb.Name); err != nil {
			message := fmt.Sprintf(`Failed to delete DeletedDatabase: "%v". Reason: %v`, deletedDb.Name, err)
			c.eventRecorder.PushEvent(
				kapi.EventTypeWarning, eventer.EventReasonFailedToDelete, message, elastic,
			)
			log.Errorln(err)
		}
		message := fmt.Sprintf(`Successfully deleted DeletedDatabase: "%v"`, deletedDb.Name)
		c.eventRecorder.PushEvent(
			kapi.EventTypeNormal, eventer.EventReasonSuccessfulDelete, message, elastic,
		)
	}

	elastic.Status.DatabaseStatus = tapi.StatusDatabaseRunning
	if c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
		message := fmt.Sprintf(`Fail to update Elastic: "%v". Reason: %v`, elastic.Name, err)
		c.eventRecorder.PushEvent(
			kapi.EventTypeWarning, eventer.EventReasonFailedToUpdate, message, elastic,
		)
		log.Errorln(err)
	}

	// Setup Schedule backup
	if elastic.Spec.BackupSchedule != nil {
		err := c.cronController.ScheduleBackup(elastic, elastic.ObjectMeta, elastic.Spec.BackupSchedule)
		if err != nil {
			message := fmt.Sprintf(`Failed to schedule snapshot. Reason: %v`, err)
			c.eventRecorder.PushEvent(
				kapi.EventTypeWarning, eventer.EventReasonFailedToSchedule, message, elastic,
			)
			log.Errorln(err)
		}
	}
}

func (c *elasticController) delete(elastic *tapi.Elastic) {

	c.eventRecorder.PushEvent(
		kapi.EventTypeNormal, eventer.EventReasonDeleting, "Deleting Elastic", elastic,
	)

	if elastic.Spec.DoNotDelete {
		message := fmt.Sprintf(`Elastic "%v" is locked.`, elastic.Name)
		c.eventRecorder.PushEvent(
			kapi.EventTypeWarning, eventer.EventReasonFailedToDelete, message, elastic,
		)

		if err := c.reCreateElastic(elastic); err != nil {
			message := fmt.Sprintf(`Failed to recreate Elastic: "%v". Reason: %v`, elastic, err)
			c.eventRecorder.PushEvent(
				kapi.EventTypeWarning, eventer.EventReasonFailedToCreate, message, elastic,
			)
			log.Errorln(err)
			return
		}
		return
	}

	if _, err := c.createDeletedDatabase(elastic); err != nil {
		message := fmt.Sprintf(`Failed to create DeletedDatabase: "%v". Reason: %v`, elastic.Name, err)
		c.eventRecorder.PushEvent(
			kapi.EventTypeWarning, eventer.EventReasonFailedToCreate, message, elastic,
		)
		log.Errorln(err)
		return
	}
	message := fmt.Sprintf(`Successfully created DeletedDatabase: "%v"`, elastic.Name)
	c.eventRecorder.PushEvent(
		kapi.EventTypeNormal, eventer.EventReasonSuccessfulCreate, message, elastic,
	)

	c.cronController.StopScheduleBackup(elastic.ObjectMeta)
}

func (c *elasticController) update(oldElastic, updatedElastic *tapi.Elastic) {
	if (updatedElastic.Spec.Replicas != oldElastic.Spec.Replicas) && oldElastic.Spec.Replicas >= 0 {
		statefulSetName := fmt.Sprintf("%v-%v", DatabaseNamePrefix, updatedElastic.Name)
		statefulSet, err := c.Client.Apps().StatefulSets(updatedElastic.Namespace).Get(statefulSetName)
		if err != nil {
			message := fmt.Sprintf(`Failed to get StatefulSet: "%v". Reason: %v`, statefulSetName, err)
			c.eventRecorder.PushEvent(
				kapi.EventTypeNormal, eventer.EventReasonFailedToGet, message, updatedElastic,
			)
			log.Errorln(err)
			return
		}
		statefulSet.Spec.Replicas = oldElastic.Spec.Replicas
		if _, err := c.Client.Apps().StatefulSets(statefulSet.Namespace).Update(statefulSet); err != nil {
			message := fmt.Sprintf(`Failed to update StatefulSet: "%v". Reason: %v`, statefulSetName, err)
			c.eventRecorder.PushEvent(
				kapi.EventTypeNormal, eventer.EventReasonFailedToUpdate, message, updatedElastic,
			)
			log.Errorln(err)
			return
		}
	}

	if !reflect.DeepEqual(updatedElastic.Spec.BackupSchedule, oldElastic.Spec.BackupSchedule) {
		backupScheduleSpec := updatedElastic.Spec.BackupSchedule
		if backupScheduleSpec != nil {
			if err := c.ValidateBackupSchedule(backupScheduleSpec); err != nil {
				c.eventRecorder.PushEvent(
					kapi.EventTypeNormal, eventer.EventReasonInvalid, err, updatedElastic,
				)
				log.Errorln(err)
				return
			}

			if err := c.CheckBucketAccess(
				backupScheduleSpec.BucketName, backupScheduleSpec.StorageSecret,
				updatedElastic.Namespace); err != nil {
				c.eventRecorder.PushEvent(
					kapi.EventTypeNormal, eventer.EventReasonInvalid, err, updatedElastic,
				)
				log.Errorln(err)
				return
			}

			if err := c.cronController.ScheduleBackup(
				oldElastic, oldElastic.ObjectMeta, oldElastic.Spec.BackupSchedule); err != nil {
				message := fmt.Sprintf(`Failed to schedule snapshot. Reason: %v`, err)
				c.eventRecorder.PushEvent(
					kapi.EventTypeWarning, eventer.EventReasonFailedToSchedule, message, updatedElastic,
				)
				log.Errorln(err)
			}
		} else {
			c.cronController.StopScheduleBackup(oldElastic.ObjectMeta)
		}
	}
}

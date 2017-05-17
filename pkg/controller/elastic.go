package controller

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/apimachinery/pkg/eventer"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

func (c *Controller) create(elastic *tapi.Elastic) error {
	t := unversioned.Now()
	elastic.Status.CreationTime = &t
	elastic.Status.Phase = tapi.DatabasePhaseCreating
	var err error
	if _, err = c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToUpdate,
			`Fail to update Elastic: "%v". Reason: %v`,
			elastic.Name,
			err,
		)
		return err
	}

	if err := c.validateElastic(elastic); err != nil {
		c.eventRecorder.Event(elastic, kapi.EventTypeWarning, eventer.EventReasonInvalid, err.Error())

		if elastic, err = c.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name); err != nil {
			return err
		}

		elastic.Status.Phase = tapi.DatabasePhaseFailed
		elastic.Status.Reason = err.Error()
		if _, err := c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToUpdate,
				`Fail to update Elastic: "%v". Reason: %v`,
				elastic.Name,
				err,
			)
			log.Errorln(err)
		}
		return err
	}
	// Event for successful validation
	c.eventRecorder.Event(
		elastic,
		kapi.EventTypeNormal,
		eventer.EventReasonSuccessfulValidate,
		"Successfully validate Elastic",
	)

	// Check if DeletedDatabase exists or not
	recovering := false
	deletedDb, err := c.ExtClient.DeletedDatabases(elastic.Namespace).Get(elastic.Name)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToGet,
				`Fail to get DeletedDatabase: "%v". Reason: %v`,
				elastic.Name,
				err,
			)
			return err
		}
	} else {
		var message string

		if deletedDb.Labels[amc.LabelDatabaseKind] != tapi.ResourceKindElastic {
			message = fmt.Sprintf(`Invalid Elastic: "%v". Exists irrelevant DeletedDatabase: "%v"`,
				elastic.Name, deletedDb.Name)
		} else {
			if deletedDb.Status.Phase == tapi.DeletedDatabasePhaseRecovering {
				recovering = true
			} else {
				message = fmt.Sprintf(`Recover from DeletedDatabase: "%v"`, deletedDb.Name)
			}
		}
		if !recovering {
			if elastic, err = c.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name); err != nil {
				return err
			}

			// Set status to Failed
			elastic.Status.Phase = tapi.DatabasePhaseFailed
			elastic.Status.Reason = message
			if _, err := c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
				c.eventRecorder.Eventf(
					elastic,
					kapi.EventTypeWarning,
					eventer.EventReasonFailedToUpdate,
					`Fail to update Elastic: "%v". Reason: %v`,
					elastic.Name,
					err,
				)
				log.Errorln(err)
			}
			c.eventRecorder.Event(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				message,
			)
			return errors.New(message)
		}
	}

	// Event for notification that kubernetes objects are creating
	c.eventRecorder.Event(
		elastic,
		kapi.EventTypeNormal,
		eventer.EventReasonCreating,
		"Creating Kubernetes objects",
	)

	// create Governing Service
	governingService := c.governingService
	if err := c.CreateGoverningService(governingService, elastic.Namespace); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			`Failed to create ServiceAccount: "%v". Reason: %v`,
			governingService,
			err,
		)
		return err
	}

	// create database Service
	if err := c.createService(elastic.Name, elastic.Namespace); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			"Failed to create Service. Reason: %v",
			err,
		)
		return err
	}

	// Create statefulSet for Elastic database
	statefulSet, err := c.createStatefulSet(elastic)
	if err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			"Failed to create StatefulSet. Reason: %v",
			err,
		)
		return err
	}

	// Check StatefulSet Pod status
	if elastic.Spec.Replicas > 0 {
		if err := c.CheckStatefulSetPodStatus(statefulSet, durationCheckStatefulSet); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToStart,
				"Failed to create StatefulSet. Reason: %v",
				err,
			)
			return err
		} else {
			c.eventRecorder.Event(
				elastic,
				kapi.EventTypeNormal,
				eventer.EventReasonSuccessfulCreate,
				"Successfully created Elastic",
			)
		}
	}

	if elastic.Spec.Init != nil && elastic.Spec.Init.SnapshotSource != nil {
		if elastic, err = c.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name); err != nil {
			return err
		}

		elastic.Status.Phase = tapi.DatabasePhaseInitializing
		if _, err = c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToUpdate,
				`Fail to update Elastic: "%v". Reason: %v`,
				elastic.Name,
				err,
			)
			return err
		}

		if err := c.initialize(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToInitialize,
				"Failed to initialize. Reason: %v",
				err,
			)
		}
	}

	if recovering {
		// Delete DeletedDatabase instance
		if err := c.ExtClient.DeletedDatabases(deletedDb.Namespace).Delete(deletedDb.Name); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToDelete,
				`Failed to delete DeletedDatabase: "%v". Reason: %v`,
				deletedDb.Name,
				err,
			)
			log.Errorln(err)
		}
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeNormal,
			eventer.EventReasonSuccessfulDelete,
			`Successfully deleted DeletedDatabase: "%v"`,
			deletedDb.Name,
		)
	}

	if elastic, err = c.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name); err != nil {
		return err
	}

	elastic.Status.Phase = tapi.DatabasePhaseRunning
	if _, err = c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToUpdate,
			`Fail to update Elastic: "%v". Reason: %v`,
			elastic.Name,
			err,
		)
		log.Errorln(err)
	}

	// Setup Schedule backup
	if elastic.Spec.BackupSchedule != nil {
		err := c.cronController.ScheduleBackup(elastic, elastic.ObjectMeta, elastic.Spec.BackupSchedule)
		if err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToSchedule,
				"Failed to schedule snapshot. Reason: %v",
				err,
			)
			log.Errorln(err)
		}
	}
	return nil
}

const (
	durationCheckRestoreJob = time.Minute * 30
)

func (c *Controller) initialize(elastic *tapi.Elastic) error {
	snapshotSource := elastic.Spec.Init.SnapshotSource
	// Event for notification that kubernetes objects are creating
	c.eventRecorder.Eventf(
		elastic,
		kapi.EventTypeNormal,
		eventer.EventReasonInitializing,
		`Initializing from DatabaseSnapshot: "%v"`,
		snapshotSource.Name,
	)

	namespace := snapshotSource.Namespace
	if namespace == "" {
		namespace = elastic.Namespace
	}
	dbSnapshot, err := c.ExtClient.DatabaseSnapshots(namespace).Get(snapshotSource.Name)
	if err != nil {
		return err
	}

	job, err := c.createRestoreJob(elastic, dbSnapshot)
	if err != nil {
		return err
	}

	jobSuccess := c.CheckDatabaseRestoreJob(job, elastic, c.eventRecorder, durationCheckRestoreJob)
	if jobSuccess {
		c.eventRecorder.Event(
			elastic,
			kapi.EventTypeNormal,
			eventer.EventReasonSuccessfulInitialize,
			"Successfully completed initialization",
		)
	} else {
		c.eventRecorder.Event(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToInitialize,
			"Failed to complete initialization",
		)
	}
	return nil
}

func (c *Controller) delete(elastic *tapi.Elastic) error {

	c.eventRecorder.Event(elastic, kapi.EventTypeNormal, eventer.EventReasonDeleting, "Deleting Elastic")

	if elastic.Spec.DoNotDelete {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToDelete,
			`Elastic "%v" is locked.`,
			elastic.Name,
		)

		if err := c.reCreateElastic(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				`Failed to recreate Elastic: "%v". Reason: %v`,
				elastic.Name,
				err,
			)
			return err
		}
		return nil
	}

	if _, err := c.createDeletedDatabase(elastic); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			`Failed to create DeletedDatabase: "%v". Reason: %v`,
			elastic.Name,
			err,
		)
		return err
	}
	c.eventRecorder.Eventf(
		elastic,
		kapi.EventTypeNormal,
		eventer.EventReasonSuccessfulCreate,
		`Successfully created DeletedDatabase: "%v"`,
		elastic.Name,
	)

	c.cronController.StopBackupScheduling(elastic.ObjectMeta)
	return nil
}

func (c *Controller) update(oldElastic, updatedElastic *tapi.Elastic) error {
	if (updatedElastic.Spec.Replicas != oldElastic.Spec.Replicas) && oldElastic.Spec.Replicas >= 0 {
		statefulSetName := getStatefulSetName(updatedElastic.Name)
		statefulSet, err := c.Client.Apps().StatefulSets(updatedElastic.Namespace).Get(statefulSetName)
		if err != nil {
			c.eventRecorder.Eventf(
				updatedElastic,
				kapi.EventTypeNormal,
				eventer.EventReasonFailedToGet,
				`Failed to get StatefulSet: "%v". Reason: %v`,
				statefulSetName,
				err,
			)
			return err
		}
		statefulSet.Spec.Replicas = oldElastic.Spec.Replicas
		if _, err := c.Client.Apps().StatefulSets(statefulSet.Namespace).Update(statefulSet); err != nil {
			c.eventRecorder.Eventf(
				updatedElastic,
				kapi.EventTypeNormal,
				eventer.EventReasonFailedToUpdate,
				`Failed to update StatefulSet: "%v". Reason: %v`,
				statefulSetName,
				err,
			)
			return err
		}
	}

	if !reflect.DeepEqual(updatedElastic.Spec.BackupSchedule, oldElastic.Spec.BackupSchedule) {
		backupScheduleSpec := updatedElastic.Spec.BackupSchedule
		if backupScheduleSpec != nil {
			if err := c.ValidateBackupSchedule(backupScheduleSpec); err != nil {
				c.eventRecorder.Event(
					updatedElastic,
					kapi.EventTypeNormal,
					eventer.EventReasonInvalid,
					err.Error(),
				)
				return err
			}

			if err := c.CheckBucketAccess(
				backupScheduleSpec.SnapshotSpec, updatedElastic.Namespace); err != nil {
				c.eventRecorder.Event(
					updatedElastic,
					kapi.EventTypeNormal,
					eventer.EventReasonInvalid,
					err.Error(),
				)
				return err
			}

			if err := c.cronController.ScheduleBackup(
				oldElastic, oldElastic.ObjectMeta, oldElastic.Spec.BackupSchedule); err != nil {
				c.eventRecorder.Eventf(
					updatedElastic,
					kapi.EventTypeWarning,
					eventer.EventReasonFailedToSchedule,
					`Failed to schedule snapshot. Reason: %v`,
					err,
				)
				log.Errorln(err)
			}
		} else {
			c.cronController.StopBackupScheduling(oldElastic.ObjectMeta)
		}
	}
	return nil
}

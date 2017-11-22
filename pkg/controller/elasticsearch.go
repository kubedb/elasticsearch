package controller

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/appscode/go/log"
	api "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	kutildb "github.com/k8sdb/apimachinery/client/typed/kubedb/v1alpha1/util"
	"github.com/k8sdb/apimachinery/pkg/eventer"
	"github.com/k8sdb/apimachinery/pkg/storage"
	"github.com/k8sdb/elasticsearch/pkg/validator"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) create(elasticsearch *api.Elasticsearch) error {
	_, err := kutildb.TryPatchElasticsearch(c.ExtClient, elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
		t := metav1.Now()
		in.Status.CreationTime = &t
		in.Status.Phase = api.DatabasePhaseCreating
		return in
	})
	if err != nil {
		c.recorder.Eventf(elasticsearch.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
		return err
	}

	if err := validator.ValidateElasticsearch(c.Client, elasticsearch); err != nil {
		c.recorder.Event(elasticsearch.ObjectReference(), core.EventTypeWarning, eventer.EventReasonInvalid, err.Error())
		return err
	}

	// Event for successful validation
	c.recorder.Event(
		elasticsearch.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulValidate,
		"Successfully validate Elasticsearch",
	)
	// Check DormantDatabase
	// return True (as matched) only if Elasticsearch matched with DormantDatabase
	// If matched, It will be resumed
	if matched, err := c.matchDormantDatabase(elasticsearch); err != nil || matched {
		return err
	}

	// Event for notification that kubernetes objects are creating
	c.recorder.Event(elasticsearch.ObjectReference(), core.EventTypeNormal, eventer.EventReasonCreating, "Creating Kubernetes objects")

	// create Governing Service
	governingService := c.opt.GoverningService
	if err := c.CreateGoverningService(governingService, elasticsearch.Namespace); err != nil {
		c.recorder.Eventf(
			elasticsearch.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			`Failed to create ServiceAccount: "%v". Reason: %v`,
			governingService,
			err,
		)
		return err
	}

	// ensure database Service
	if err := c.ensureService(elasticsearch); err != nil {
		return err
	}

	// ensure database StatefulSet
	if err := c.ensureElasticsearchNode(elasticsearch); err != nil {
		return err
	}

	es, err := c.ExtClient.Elasticsearchs(elasticsearch.Namespace).Get(elasticsearch.Name, metav1.GetOptions{})
	if err != nil {
		c.recorder.Eventf(elasticsearch.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToGet, err.Error())
		return err
	}
	elasticsearch = es
	kutildb.AssignTypeKind(elasticsearch)

	if elasticsearch.Spec.Init != nil && elasticsearch.Spec.Init.SnapshotSource != nil {
		_, err := kutildb.TryPatchElasticsearch(c.ExtClient, elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
			in.Status.Phase = api.DatabasePhaseInitializing
			return in
		})
		if err != nil {
			c.recorder.Eventf(elasticsearch.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
			return err
		}

		if err := c.initialize(elasticsearch); err != nil {
			c.recorder.Eventf(
				elasticsearch.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToInitialize,
				"Failed to initialize. Reason: %v",
				err,
			)
		}
	}

	c.recorder.Event(
		elasticsearch.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulCreate,
		"Successfully created Elasticsearch",
	)

	// Ensure Schedule backup
	c.ensureBackupScheduler(elasticsearch)

	if elasticsearch.Spec.Monitor != nil {
		if err := c.addMonitor(elasticsearch); err != nil {
			c.recorder.Eventf(
				elasticsearch.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToAddMonitor,
				"Failed to add monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.recorder.Event(
			elasticsearch.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulMonitorAdd,
			"Successfully added monitoring system.",
		)
	}
	return nil
}

func (c *Controller) matchDormantDatabase(elasticsearch *api.Elasticsearch) (bool, error) {
	// Check if DormantDatabase exists or not
	dormantDb, err := c.ExtClient.DormantDatabases(elasticsearch.Namespace).Get(elasticsearch.Name, metav1.GetOptions{})
	if err != nil {
		if !kerr.IsNotFound(err) {
			c.recorder.Eventf(
				elasticsearch.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToGet,
				`Fail to get DormantDatabase: "%v". Reason: %v`,
				elasticsearch.Name,
				err,
			)
			return false, err
		}
		return false, nil
	}

	var sendEvent = func(message string, args ...interface{}) (bool, error) {
		c.recorder.Eventf(
			elasticsearch.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			message,
			args,
		)
		return false, fmt.Errorf(message, args)
	}

	// Check DatabaseKind
	if dormantDb.Labels[api.LabelDatabaseKind] != api.ResourceKindElasticsearch {
		return sendEvent(fmt.Sprintf(`Invalid Elasticsearch: "%v". Exists DormantDatabase "%v" of different Kind`,
			elasticsearch.Name, dormantDb.Name))
	}

	// Check InitSpec
	initSpecAnnotationStr := dormantDb.Annotations[api.ElasticsearchInitSpec]
	if initSpecAnnotationStr != "" {
		var initSpecAnnotation *api.InitSpec
		if err := json.Unmarshal([]byte(initSpecAnnotationStr), &initSpecAnnotation); err != nil {
			return sendEvent(err.Error())
		}

		if elasticsearch.Spec.Init != nil {
			if !reflect.DeepEqual(initSpecAnnotation, elasticsearch.Spec.Init) {
				return sendEvent("InitSpec mismatches with DormantDatabase annotation")
			}
		}
	}

	// Check Origin Spec
	drmnOriginSpec := dormantDb.Spec.Origin.Spec.Elasticsearch
	originalSpec := elasticsearch.Spec
	originalSpec.Init = nil

	if !reflect.DeepEqual(drmnOriginSpec, &originalSpec) {
		return sendEvent("Elasticsearch spec mismatches with OriginSpec in DormantDatabases")
	}

	//TODO: Use Annotation Key
	elasticsearch.Annotations = map[string]string{
		"kubedb.com/ignore": "",
	}

	if err := c.ExtClient.Elasticsearchs(elasticsearch.Namespace).Delete(elasticsearch.Name, &metav1.DeleteOptions{}); err != nil {
		return sendEvent(`failed to resume Elasticsearch "%v" from DormantDatabase "%v". Error: %v`, elasticsearch.Name, elasticsearch.Name, err)
	}

	_, err = kutildb.TryPatchDormantDatabase(c.ExtClient, elasticsearch.ObjectMeta, func(in *api.DormantDatabase) *api.DormantDatabase {
		in.Spec.Resume = true
		return in
	})
	if err != nil {
		c.recorder.Eventf(elasticsearch.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
		return sendEvent(err.Error())
	}

	return true, nil
}

func (c *Controller) ensureElasticsearchNode(elasticsearch *api.Elasticsearch) error {
	topology := elasticsearch.Spec.Topology
	if topology != nil {
		clientName := elasticsearch.OffshootName()
		if topology.Client.Prefix != "" {
			clientName = fmt.Sprintf("%v-%v", topology.Client.Prefix, clientName)
		}
		if err := c.ensureStatefulSet(elasticsearch, clientName, false, false, true, true); err != nil {
			return err
		}

		masterName := elasticsearch.OffshootName()
		if topology.Master.Prefix != "" {
			masterName = fmt.Sprintf("%v-%v", topology.Master.Prefix, masterName)
		}
		if err := c.ensureStatefulSet(elasticsearch, masterName, true, false, false, true); err != nil {
			return err
		}

		dataName := elasticsearch.OffshootName()
		if topology.Data.Prefix != "" {
			dataName = fmt.Sprintf("%v-%v", topology.Data.Prefix, dataName)
		}
		if err := c.ensureStatefulSet(elasticsearch, dataName, false, true, false, true); err != nil {
			return err
		}
	} else {
		if err := c.ensureStatefulSet(elasticsearch, elasticsearch.OffshootName(), true, true, true, false); err != nil {
			return err
		}
	}

	// Need some time to build elasticsearch cluster. Nodes will communicate with each other
	// TODO: find better way
	time.Sleep(time.Minute)

	_, err := kutildb.TryPatchElasticsearch(c.ExtClient, elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
		in.Status.Phase = api.DatabasePhaseRunning
		return in
	})
	if err != nil {
		c.recorder.Eventf(elasticsearch.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
		return err
	}

	return nil
}

func (c *Controller) ensureBackupScheduler(elastic *api.Elasticsearch) {
	// Setup Schedule backup
	if elastic.Spec.BackupSchedule != nil {
		err := c.cronController.ScheduleBackup(elastic, elastic.ObjectMeta, elastic.Spec.BackupSchedule)
		if err != nil {
			c.recorder.Eventf(
				elastic.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToSchedule,
				"Failed to schedule snapshot. Reason: %v",
				err,
			)
			log.Errorln(err)
		}
	} else {
		c.cronController.StopBackupScheduling(elastic.ObjectMeta)
	}
}

const (
	durationCheckRestoreJob = time.Minute * 30
)

func (c *Controller) initialize(elastic *api.Elasticsearch) error {
	snapshotSource := elastic.Spec.Init.SnapshotSource
	// Event for notification that kubernetes objects are creating
	c.recorder.Eventf(
		elastic.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonInitializing,
		`Initializing from Snapshot: "%v"`,
		snapshotSource.Name,
	)

	namespace := snapshotSource.Namespace
	if namespace == "" {
		namespace = elastic.Namespace
	}
	snapshot, err := c.ExtClient.Snapshots(namespace).Get(snapshotSource.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	secret, err := storage.NewOSMSecret(c.Client, snapshot)
	if err != nil {
		return err
	}
	_, err = c.Client.CoreV1().Secrets(secret.Namespace).Create(secret)
	if err != nil && !kerr.IsAlreadyExists(err) {
		return err
	}

	job, err := c.createRestoreJob(elastic, snapshot)
	if err != nil {
		return err
	}

	jobSuccess := c.CheckDatabaseRestoreJob(snapshot, job, elastic, c.recorder, durationCheckRestoreJob)
	if jobSuccess {
		c.recorder.Event(
			elastic.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulInitialize,
			"Successfully completed initialization",
		)
	} else {
		c.recorder.Event(
			elastic.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToInitialize,
			"Failed to complete initialization",
		)
	}
	return nil
}

func (c *Controller) pause(elasticsearch *api.Elasticsearch) error {
	if elasticsearch.Annotations != nil {
		if val, found := elasticsearch.Annotations["kubedb.com/ignore"]; found {
			//TODO: Add Event Reason "Ignored"
			c.recorder.Event(elasticsearch.ObjectReference(), core.EventTypeNormal, "Ignored", val)
			return nil
		}
	}

	c.recorder.Event(elasticsearch.ObjectReference(), core.EventTypeNormal, eventer.EventReasonPausing, "Pausing Elasticsearch")

	if elasticsearch.Spec.DoNotPause {
		c.recorder.Eventf(
			elasticsearch.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToPause,
			`Elasticsearch "%v" is locked.`,
			elasticsearch.Name,
		)

		if err := c.reCreateElastic(elasticsearch); err != nil {
			c.recorder.Eventf(
				elasticsearch.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				`Failed to recreate Elasticsearch: "%v". Reason: %v`,
				elasticsearch.Name,
				err,
			)
			return err
		}
		return nil
	}

	if _, err := c.createDormantDatabase(elasticsearch); err != nil {
		c.recorder.Eventf(
			elasticsearch.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			`Failed to create DormantDatabase: "%v". Reason: %v`,
			elasticsearch.Name,
			err,
		)
		return err
	}
	c.recorder.Eventf(
		elasticsearch.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulCreate,
		`Successfully created DormantDatabase: "%v"`,
		elasticsearch.Name,
	)

	c.cronController.StopBackupScheduling(elasticsearch.ObjectMeta)

	if elasticsearch.Spec.Monitor != nil {
		if err := c.deleteMonitor(elasticsearch); err != nil {
			c.recorder.Eventf(
				elasticsearch.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToDeleteMonitor,
				"Failed to delete monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.recorder.Event(
			elasticsearch.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulMonitorDelete,
			"Successfully deleted monitoring system.",
		)
	}
	return nil
}

func (c *Controller) update(oldElasticsearch, updatedElasticsearch *api.Elasticsearch) error {

	if err := validator.ValidateElasticsearch(c.Client, updatedElasticsearch); err != nil {
		c.recorder.Event(updatedElasticsearch, core.EventTypeWarning, eventer.EventReasonInvalid, err.Error())
		return err
	}
	// Event for successful validation
	c.recorder.Event(
		updatedElasticsearch.ObjectReference(),
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulValidate,
		"Successfully validate Elasticsearch",
	)

	if err := c.ensureService(updatedElasticsearch); err != nil {
		return err
	}

	if !reflect.DeepEqual(oldElasticsearch.Spec.Topology, updatedElasticsearch.Spec.Topology) ||
		oldElasticsearch.Spec.Replicas != updatedElasticsearch.Spec.Replicas {
		if err := c.ensureElasticsearchNode(updatedElasticsearch); err != nil {
			return err
		}
	}

	if !reflect.DeepEqual(updatedElasticsearch.Spec.BackupSchedule, oldElasticsearch.Spec.BackupSchedule) {
		c.ensureBackupScheduler(updatedElasticsearch)
	}

	if !reflect.DeepEqual(oldElasticsearch.Spec.Monitor, updatedElasticsearch.Spec.Monitor) {
		if err := c.updateMonitor(oldElasticsearch, updatedElasticsearch); err != nil {
			c.recorder.Eventf(
				updatedElasticsearch.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToUpdateMonitor,
				"Failed to update monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.recorder.Event(
			updatedElasticsearch.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulMonitorUpdate,
			"Successfully updated monitoring system.",
		)
	}

	return nil
}

func (c *Controller) reCreateElastic(elasticsearch *api.Elasticsearch) error {
	es := &api.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:        elasticsearch.Name,
			Namespace:   elasticsearch.Namespace,
			Labels:      elasticsearch.Labels,
			Annotations: elasticsearch.Annotations,
		},
		Spec:   elasticsearch.Spec,
		Status: elasticsearch.Status,
	}

	if _, err := c.ExtClient.Elasticsearchs(es.Namespace).Create(es); err != nil {
		return err
	}

	return nil
}

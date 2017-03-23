package controller

import (
	"errors"
	"fmt"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

func (w *Controller) validateElastic(elastic *tapi.Elastic) error {
	if elastic.Spec.Version == "" {
		return fmt.Errorf(`Object 'Version' is missing in '%v'`, elastic.Spec)
	}

	storage := elastic.Spec.Storage
	if storage != nil {
		if storage.Class == "" {
			return fmt.Errorf(`Object 'Class' is missing in '%v'`, *storage)
		}

		if _, err := w.Client.Storage().StorageClasses().Get(storage.Class); err != nil {
			if k8serr.IsNotFound(err) {
				return fmt.Errorf(`Spec.Storage.Class "%v" not found`, storage.Class)
			}
			return err
		}

		if len(storage.AccessModes) == 0 {
			storage.AccessModes = []kapi.PersistentVolumeAccessMode{
				kapi.ReadWriteOnce,
			}
			log.Infof(`Using "%v" as AccessModes in "%v"`, kapi.ReadWriteOnce, *storage)
		}

		if val, found := storage.Resources.Requests[kapi.ResourceStorage]; found {
			if val.Value() <= 0 {
				return errors.New("Invalid ResourceStorage request")
			}
		} else {
			return errors.New("Missing ResourceStorage request")
		}
	}

	if elastic.Spec.BackupSchedule != nil {
		// CronExpression can't be empty
		backupSchedule := elastic.Spec.BackupSchedule
		if backupSchedule.CronExpression == "" {
			return errors.New("Invalid cron expression")
		}

		// Validate backup spec
		if err := w.validateBackupSpec(backupSchedule.SnapshotSpec, elastic.Namespace); err != nil {
			return err
		}
	}

	return nil
}

func (w *Controller) validateBackupSpec(backup tapi.SnapshotSpec, namespace string) error {
	// BucketName can't be empty
	bucketName := backup.BucketName
	if bucketName == "" {
		return fmt.Errorf(`Object 'BucketName' is missing in '%v'`, backup)
	}

	// Need to provide Storage credential secret
	storageSecret := backup.StorageSecret
	if storageSecret == nil {
		return fmt.Errorf(`Object 'StorageSecret' is missing in '%v'`, backup)
	}

	// Credential SecretName  can't be empty
	storageSecretName := storageSecret.SecretName
	if storageSecretName == "" {
		return fmt.Errorf(`Object 'SecretName' is missing in '%v'`, *backup.StorageSecret)
	}

	// Check bucket access with provided storage credential
	if err := w.CheckBucketAccess(backup.BucketName, storageSecretName, namespace); err != nil {
		return fmt.Errorf(`Fail to access bucket "%v" using Secret "%v.%v". Error: %v`,
			backup.BucketName, storageSecretName, namespace, err)
	}

	return nil
}

func (w *Controller) validateDatabaseSnapshot(snapshot *tapi.DatabaseSnapshot) error {
	// Database name can't empty
	databaseName := snapshot.Spec.DatabaseName
	if databaseName == "" {
		return fmt.Errorf(`Object 'DatabaseName' is missing in '%v'`, snapshot.Spec)
	}

	labelMap := map[string]string{
		LabelDatabaseType:       DatabaseElasticsearch,
		LabelDatabaseName:       snapshot.Spec.DatabaseName,
		amc.LabelSnapshotActive: string(tapi.SnapshotRunning),
	}

	snapshotList, err := w.ExtClient.DatabaseSnapshot(snapshot.Namespace).List(kapi.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
	})
	if err != nil {
		return err
	}

	if len(snapshotList.Items) > 0 {
		unversionedNow := unversioned.Now()
		snapshot.Status.StartTime = &unversionedNow
		snapshot.Status.CompletionTime = &unversionedNow
		snapshot.Status.Status = tapi.SnapshotFailed
		snapshot.Status.Reason = "One DatabaseSnapshot is already Running"
		if _, err := w.ExtClient.DatabaseSnapshot(snapshot.Namespace).Update(snapshot); err != nil {
			return err
		}
		return errors.New("One DatabaseSnapshot is already Running")
	}

	if err := w.validateBackupSpec(snapshot.Spec.SnapshotSpec, snapshot.Namespace); err != nil {
		return err
	}

	return nil
}

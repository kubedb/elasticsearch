package controller

import (
	"fmt"

	tapi "github.com/k8sdb/apimachinery/api"
)

func (c *Controller) validateElastic(elastic *tapi.Elastic) error {
	if elastic.Spec.Version == "" {
		return fmt.Errorf(`Object 'Version' is missing in '%v'`, elastic.Spec)
	}

	storage := elastic.Spec.Storage
	if storage != nil {
		var err error
		if storage, err = c.ValidateStorageSpec(storage); err != nil {
			return err
		}
	}

	backupScheduleSpec := elastic.Spec.BackupSchedule
	if elastic.Spec.BackupSchedule != nil {
		if err := c.ValidateBackupSchedule(backupScheduleSpec); err != nil {
			return err
		}

		if err := c.CheckBucketAccess(
			backupScheduleSpec.BucketName, backupScheduleSpec.StorageSecret,
			elastic.Namespace); err != nil {
			return err
		}
	}
	return nil
}

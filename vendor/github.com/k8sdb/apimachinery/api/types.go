package api

import "k8s.io/kubernetes/pkg/api"

// StorageSpec defines storage provisioning
type StorageSpec struct {
	// Name of the StorageClass to use when requesting storage provisioning.
	Class string `json:"class"`
	// Persistent Volume Claim
	api.PersistentVolumeClaimSpec `json:",inline,omitempty"`
}

type InitialScriptSpec struct {
	ScriptPath       string `json:"scriptPath,omitempty"`
	api.VolumeSource `json:",inline,omitempty"`
}

type BackupScheduleSpec struct {
	CronExpression string `json:"cronExpression,omitempty"`
	BackupSpec     `json:",inline,omitempty"`
}

type BackupSpec struct {
	// Cloud credential secret
	CredSecret *api.SecretVolumeSource `json:"credSecret,omitempty"`
	// Database authentication secret
	// +optional
	AuthSecret *api.SecretVolumeSource `json:"authSecret,omitempty"`
	// Cloud bucket name
	BucketName string `json:"bucketName,omitempty"`
	// Database snapshot id
	// +optional
	SnapshotID string `json:"snapshotID,omitempty"`
}

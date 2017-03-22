package api

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

const (
	ResourceKindDatabaseSnapshot = "DatabaseSnapshot"
	ResourceNameDatabaseSnapshot = "database-snapshot"
	ResourceTypeDatabaseSnapshot = "databasesnapshots"
)

type DatabaseSnapshot struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 DatabaseSnapshotSpec   `json:"spec,omitempty"`
	Status               DatabaseSnapshotStatus `json:"status,omitempty"`
}

type DatabaseSnapshotSpec struct {
	// Database name
	DatabaseName string `json:"databaseName,omitempty"`
	// Snapshot Spec
	SnapshotSpec `json:",inline,omitempty"`
}

type SnapshoStatus string

const (
	// used for DatabaseSnapshots that are currently running
	SnapshotRunning SnapshoStatus = "Running"
	// used for DatabaseSnapshots that are Succeeded
	SnapshotSuccessed SnapshoStatus = "Succeeded"
	// used for PersistentVolumes that are Failed
	SnapshotFailed SnapshoStatus = "Failed"
)

type DatabaseSnapshotStatus struct {
	StartTime      *unversioned.Time `json:"startTime,omitempty"`
	CompletionTime *unversioned.Time `json:"completionTime,omitempty"`
	Status         SnapshoStatus     `json:"status,omitempty"`
	Reason         string            `json:"reason,omitempty"`
}

type DatabaseSnapshotList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	// Items is a list of DatabaseSnapshot TPR objects
	Items []DatabaseSnapshot `json:"items,omitempty"`
}

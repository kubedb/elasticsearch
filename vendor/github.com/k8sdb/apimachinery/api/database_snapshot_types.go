package api

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
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
	// Backup Spec
	Backup BackupSpec `json:"backup,omitempty"`
}

type DatabaseSnapshotStatus struct {
	Message string    `json:"message,omitempty"`
	Created time.Time `json:"created,omitempty"`
	Success time.Time `json:"success,omitempty"`
}

type DatabaseSnapshotList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	// Items is a list of DatabaseSnapshot TPR objects
	Items []DatabaseSnapshot `json:"items,omitempty"`
}

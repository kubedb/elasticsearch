package api

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type DeletedDatabase struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 DeletedDatabaseSpec   `json:"spec,omitempty"`
	Status               DeletedDatabaseStatus `json:"status,omitempty"`
}

type DeletedDatabaseSpec struct {
	// Database name
	DatabaseName string `json:"databaseName,omitempty"`
	// Database authentication secret
	// +optional
	AuthSecret *api.SecretVolumeSource `json:"authSecret,omitempty"`
	// If true, invoke destroy operation
	// +optional
	Destroy bool `json:"destroy,omitempty"`
}

type DeletedDatabaseStatus struct {
	Message string    `json:"message,omitempty"`
	Created time.Time `json:"created,omitempty"`
	Success time.Time `json:"success,omitempty"`
}

type DeletedDatabaseList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	// Items is a list of DeletedDatabase TPR objects
	Items []DeletedDatabase `json:"items,omitempty"`
}

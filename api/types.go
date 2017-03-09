package kube

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

// Elastic defines a Elasticsearch database.
type Elastic struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 ElasticSpec    `json:"spec,omitempty"`
	Status               *ElasticStatus `json:"status,omitempty"`
}

type ElasticSpec struct {
	// Version of Elasticsearch to be deployed.
	Version string `json:"version,omitempty"`
	// Number of instances to deploy for a Elasticsearch database.
	Replicas int32 `json:"replicas,omitempty"`
	// Storage spec to specify how storage shall be used.
	Storage *StorageSpec `json:"storage,omitempty"`
	// ServiceAccountName is the name of the ServiceAccount to use to run the
	// Prometheus Pods.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// StorageSpec defines storage provisioning
type StorageSpec struct {
	// Name of the StorageClass to use when requesting storage provisioning.
	Class string `json:"class"`
	// Persistent Volume Claim
	api.PersistentVolumeClaimSpec `json:",inline,omitempty"`
}

type ElasticStatus struct {
	// Total number of non-terminated pods targeted by this Elastic TPR
	Replicas int32 `json:"replicas"`
	// Total number of available pods targeted by this Elastic TPR.
	AvailableReplicas int32 `json:"availableReplicas"`
}

type ElasticList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	// Items is a list of Elastic TPR objects
	Items []*Elastic `json:"items,omitempty"`
}

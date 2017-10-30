package validator

import (
	"fmt"

	tapi "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/k8sdb/apimachinery/pkg/docker"
	amv "github.com/k8sdb/apimachinery/pkg/validator"
	"k8s.io/client-go/kubernetes"
)

func ValidateElasticsearch(client kubernetes.Interface, elasticsearch *tapi.Elasticsearch) error {
	if elasticsearch.Spec.Version == "" {
		return fmt.Errorf(`Object 'Version' is missing in '%v'`, elasticsearch.Spec)
	}

	if err := docker.CheckDockerImageVersion(docker.ImageElasticsearch, string(elasticsearch.Spec.Version)); err != nil {
		return fmt.Errorf(`Image %v:%v not found`, docker.ImageElasticsearch, elasticsearch.Spec.Version)
	}

	if elasticsearch.Spec.Storage != nil {
		var err error
		if err = amv.ValidateStorage(client, elasticsearch.Spec.Storage); err != nil {
			return err
		}
	}

	backupScheduleSpec := elasticsearch.Spec.BackupSchedule
	if backupScheduleSpec != nil {
		if err := amv.ValidateBackupSchedule(client, backupScheduleSpec, elasticsearch.Namespace); err != nil {
			return err
		}
	}

	monitorSpec := elasticsearch.Spec.Monitor
	if monitorSpec != nil {
		if err := amv.ValidateMonitorSpec(monitorSpec); err != nil {
			return err
		}

	}
	return nil
}

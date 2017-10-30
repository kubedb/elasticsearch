package framework

import (
	"errors"
	"time"

	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyTPR() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			// Check Elasticsearch TPR
			if _, err := f.extClient.Elasticsearchs(core.NamespaceAll).List(metav1.ListOptions{}); err != nil {
				return errors.New("thirdpartyresources are not ready")
			}

			// Check Snapshots TPR
			if _, err := f.extClient.Snapshots(core.NamespaceAll).List(metav1.ListOptions{}); err != nil {
				return errors.New("thirdpartyresources are not ready")
			}

			// Check DormantDatabases TPR
			if _, err := f.extClient.DormantDatabases(core.NamespaceAll).List(metav1.ListOptions{}); err != nil {
				return errors.New("thirdpartyresources are not ready")
			}

			return nil
		},
		time.Minute*2,
		time.Second*10,
	)
}

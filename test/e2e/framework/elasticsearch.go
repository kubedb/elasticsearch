package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/encoding/json/types"
	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Invocation) Elasticsearch() *tapi.Elasticsearch {
	return &tapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("elasticsearch"),
			Namespace: f.namespace,
			Labels: map[string]string{
				"app": f.app,
			},
		},
		Spec: tapi.ElasticsearchSpec{
			Version:  types.StrYo("2.3.1"),
			Replicas: 1,
		},
	}
}

func (f *Framework) CreateElasticsearch(obj *tapi.Elasticsearch) error {
	_, err := f.extClient.Elasticsearches(obj.Namespace).Create(obj)
	return err
}

func (f *Framework) GetElasticsearch(meta metav1.ObjectMeta) (*tapi.Elasticsearch, error) {
	return f.extClient.Elasticsearches(meta.Namespace).Get(meta.Name)
}

func (f *Framework) UpdateElasticsearch(meta metav1.ObjectMeta, transformer func(tapi.Elasticsearch) tapi.Elasticsearch) (*tapi.Elasticsearch, error) {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.extClient.Elasticsearches(meta.Namespace).Get(meta.Name)
		if err != nil {
			return nil, err
		}

		modified := transformer(*cur)
		updated, err := f.extClient.Elasticsearches(cur.Namespace).Update(&modified)
		if err == nil {
			return updated, nil
		}

		log.Errorf("Attempt %d failed to update Elasticsearch %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}

	return nil, fmt.Errorf("Failed to update Elasticsearch %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) DeleteElasticsearch(meta metav1.ObjectMeta) error {
	return f.extClient.Elasticsearches(meta.Namespace).Delete(meta.Name)
}

func (f *Framework) EventuallyElasticsearch(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.extClient.Elasticsearches(meta.Namespace).Get(meta.Name)
			if err != nil {
				if kerr.IsNotFound(err) {
					return false
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			}
			return true
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Framework) EventuallyElasticsearchRunning(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			elasticsearch, err := f.extClient.Elasticsearches(meta.Namespace).Get(meta.Name)
			Expect(err).NotTo(HaveOccurred())
			return elasticsearch.Status.Phase == tapi.DatabasePhaseRunning
		},
		time.Minute*5,
		time.Second*5,
	)
}

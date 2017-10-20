package framework

import (
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/encoding/json/types"
	tapi "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	kutildb "github.com/k8sdb/apimachinery/client/typed/kubedb/v1alpha1/util"
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
	_, err := f.extClient.Elasticsearchs(obj.Namespace).Create(obj)
	return err
}

func (f *Framework) GetElasticsearch(meta metav1.ObjectMeta) (*tapi.Elasticsearch, error) {
	return f.extClient.Elasticsearchs(meta.Namespace).Get(meta.Name)
}

func (f *Framework) TryPatchElasticsearch(meta metav1.ObjectMeta, transform func(*tapi.Elasticsearch) *tapi.Elasticsearch) (*tapi.Elasticsearch, error) {
	return kutildb.TryPatchElasticsearch(f.extClient, meta, transform)
}

func (f *Framework) DeleteElasticsearch(meta metav1.ObjectMeta) error {
	return f.extClient.Elasticsearchs(meta.Namespace).Delete(meta.Name)
}

func (f *Framework) EventuallyElasticsearch(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.extClient.Elasticsearchs(meta.Namespace).Get(meta.Name)
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
			elasticsearch, err := f.extClient.Elasticsearchs(meta.Namespace).Get(meta.Name)
			Expect(err).NotTo(HaveOccurred())
			return elasticsearch.Status.Phase == tapi.DatabasePhaseRunning
		},
		time.Minute*5,
		time.Second*5,
	)
}

package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	jtypes "github.com/appscode/go/encoding/json/types"
	"github.com/appscode/go/types"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	kutildb "github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	. "github.com/onsi/gomega"
	"gopkg.in/olivere/elastic.v5"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *Invocation) CombinedElasticsearch() *api.Elasticsearch {
	return &api.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("elasticsearch"),
			Namespace: i.namespace,
			Labels: map[string]string{
				"app": i.app,
			},
		},
		Spec: api.ElasticsearchSpec{
			Version:   jtypes.StrYo("5.6.4"),
			Replicas:  types.Int32P(1),
			EnableSSL: true,
		},
	}
}

func (i *Invocation) DedicatedElasticsearch() *api.Elasticsearch {
	return &api.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("elasticsearch"),
			Namespace: i.namespace,
			Labels: map[string]string{
				"app": i.app,
			},
		},
		Spec: api.ElasticsearchSpec{
			Version: jtypes.StrYo("5.6.4"),
			Topology: &api.ElasticsearchClusterTopology{
				Master: api.ElasticsearchNode{
					Replicas: types.Int32P(2),
					Prefix:   "master",
				},
				Data: api.ElasticsearchNode{
					Replicas: types.Int32P(2),
					Prefix:   "data",
				},
				Client: api.ElasticsearchNode{
					Replicas: types.Int32P(2),
					Prefix:   "client",
				},
			},
			EnableSSL: true,
		},
	}
}

func (f *Framework) CreateElasticsearch(obj *api.Elasticsearch) error {
	_, err := f.extClient.Elasticsearches(obj.Namespace).Create(obj)
	return err
}

func (f *Framework) GetElasticsearch(meta metav1.ObjectMeta) (*api.Elasticsearch, error) {
	return f.extClient.Elasticsearches(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
}

func (f *Framework) TryPatchElasticsearch(meta metav1.ObjectMeta, transform func(*api.Elasticsearch) *api.Elasticsearch) (*api.Elasticsearch, error) {
	elasticsearch, err := f.extClient.Elasticsearches(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	elasticsearch, _, err = kutildb.PatchElasticsearch(f.extClient, elasticsearch, transform)
	return elasticsearch, err
}

func (f *Framework) DeleteElasticsearch(meta metav1.ObjectMeta) error {
	return f.extClient.Elasticsearches(meta.Namespace).Delete(meta.Name, nil)
}

func (f *Framework) EventuallyElasticsearch(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.extClient.Elasticsearches(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
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
			elasticsearch, err := f.extClient.Elasticsearches(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return elasticsearch.Status.Phase == api.DatabasePhaseRunning
		},
		time.Minute*15,
		time.Second*10,
	)
}

func (f *Framework) EventuallyElasticsearchClientReady(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			client, err := f.GetElasticClient(meta)
			if err != nil {
				return false
			}
			client.Stop()
			return true
		},
		time.Minute*15,
		time.Second*5,
	)
}

func (f *Framework) EventuallyElasticsearchIndicesCount(client *elastic.Client) GomegaAsyncAssertion {
	return Eventually(
		func() int {
			count, err := f.CountIndex(client)
			if err != nil {
				return -1
			}
			return count
		},
		time.Minute*10,
		time.Second*5,
	)
}

func (f *Framework) CleanElasticsearch() {
	elasticsearchList, err := f.extClient.Elasticsearches(f.namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, e := range elasticsearchList.Items {
		if _, _, err := kutildb.PatchElasticsearch(f.extClient, &e, func(in *api.Elasticsearch) *api.Elasticsearch {
			in.ObjectMeta.Finalizers = nil
			in.Spec.DoNotPause = false
			return in
		}); err != nil {
			fmt.Printf("error Patching Elasticsearch. error: %v", err)
		}
	}
	if err := f.extClient.Elasticsearches(f.namespace).DeleteCollection(deleteInBackground(), metav1.ListOptions{}); err != nil {
		fmt.Printf("error in deletion of Elasticsearch. Error: %v", err)
	}
}

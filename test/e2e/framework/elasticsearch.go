/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"fmt"
	"strconv"

	"kubedb.dev/apimachinery/apis/kubedb"
	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha2/util"
	"kubedb.dev/elasticsearch/pkg/util/es"

	. "github.com/onsi/gomega"
	"gomodules.xyz/pointer"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	meta_util "kmodules.xyz/client-go/meta"
)

var (
	JobPvcStorageSize = "200Mi"
	DBPvcStorageSize  = "100Mi"
)

const (
	kindEviction = "Eviction"
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
			Version:   DBCatalogName,
			Replicas:  pointer.Int32P(1),
			EnableSSL: true,
			Storage: &core.PersistentVolumeClaimSpec{
				Resources: core.ResourceRequirements{
					Requests: core.ResourceList{
						core.ResourceStorage: resource.MustParse(DBPvcStorageSize),
					},
				},
				StorageClassName: pointer.StringP(i.StorageClass),
			},
			TerminationPolicy: api.TerminationPolicyHalt,
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
			Version: DBCatalogName,
			Topology: &api.ElasticsearchClusterTopology{
				Master: api.ElasticsearchNode{
					Replicas: pointer.Int32P(2),
					Prefix:   "master",
					Storage: &core.PersistentVolumeClaimSpec{
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceStorage: resource.MustParse(DBPvcStorageSize),
							},
						},
						StorageClassName: pointer.StringP(i.StorageClass),
					},
				},
				Data: api.ElasticsearchNode{
					Replicas: pointer.Int32P(2),
					Prefix:   "data",
					Storage: &core.PersistentVolumeClaimSpec{
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceStorage: resource.MustParse(DBPvcStorageSize),
							},
						},
						StorageClassName: pointer.StringP(i.StorageClass),
					},
				},
				Ingest: api.ElasticsearchNode{
					Replicas: pointer.Int32P(2),
					Prefix:   "ingest",
					Storage: &core.PersistentVolumeClaimSpec{
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceStorage: resource.MustParse(DBPvcStorageSize),
							},
						},
						StorageClassName: pointer.StringP(i.StorageClass),
					},
				},
			},
			EnableSSL:         true,
			TerminationPolicy: api.TerminationPolicyHalt,
		},
	}
}

func (f *Framework) CreateElasticsearch(obj *api.Elasticsearch) error {
	_, err := f.dbClient.KubedbV1alpha2().Elasticsearches(obj.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
	return err
}

func (f *Framework) GetElasticsearch(meta metav1.ObjectMeta) (*api.Elasticsearch, error) {
	return f.dbClient.KubedbV1alpha2().Elasticsearches(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
}

func (f *Framework) PatchElasticsearch(meta metav1.ObjectMeta, transform func(*api.Elasticsearch) *api.Elasticsearch) (*api.Elasticsearch, error) {
	elasticsearch, err := f.dbClient.KubedbV1alpha2().Elasticsearches(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	elasticsearch, _, err = util.PatchElasticsearch(context.TODO(), f.dbClient.KubedbV1alpha2(), elasticsearch, transform, metav1.PatchOptions{})
	return elasticsearch, err
}

func (f *Framework) DeleteElasticsearch(meta metav1.ObjectMeta) error {
	return f.dbClient.KubedbV1alpha2().Elasticsearches(meta.Namespace).Delete(context.TODO(), meta.Name, meta_util.DeleteInBackground())
}

func (f *Framework) EventuallyServices(es *api.Elasticsearch) error {
	sel, err := metav1.LabelSelectorAsSelector(metav1.SetAsLabelSelector(es.OffshootSelectors()))
	if err != nil {
		return err
	}

	return wait.PollImmediate(WaitLoopInterval, WaitLoopTimeout, func() (bool, error) {
		svcList, err := f.kubeClient.CoreV1().Services(es.Namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: sel.String(),
		})
		if err != nil {
			return false, nil
		}
		return len(svcList.Items) == 0, nil
	})
}

func (f *Framework) EventuallyElasticsearch(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.dbClient.KubedbV1alpha2().Elasticsearches(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return false
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			}
			return true
		},
		WaitLoopTimeout,
		WaitLoopInterval,
	)
}

func (f *Framework) EventuallyElasticsearchPhase(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() api.DatabasePhase {
			db, err := f.dbClient.KubedbV1alpha2().Elasticsearches(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return db.Status.Phase
		},
		WaitLoopTimeout,
		WaitLoopInterval,
	)
}

func (f *Framework) EventuallyElasticsearchReady(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			elasticsearch, err := f.dbClient.KubedbV1alpha2().Elasticsearches(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return elasticsearch.Status.Phase == api.DatabasePhaseReady
		},
		WaitLoopTimeout,
		WaitLoopInterval,
	)
}

func (f *Framework) EventuallyElasticsearchClientReady(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			db, err := f.GetElasticsearch(meta)
			if err != nil {
				return false
			}
			client, err := f.GetElasticClient(meta)
			if err != nil {
				return false
			}
			defer client.Stop()
			defer f.Tunnel.Close()

			url := fmt.Sprintf("%v://127.0.0.1:%d", db.GetConnectionScheme(), f.Tunnel.Local)
			if _, err := client.Ping(url); err != nil {
				return false
			}
			if db.Spec.Topology != nil {
				// cluster health status will be green only for dedicated elasicsearch
				if err := client.WaitForGreenStatus("10s"); err != nil {
					return false
				}
			} else if err := client.WaitForYellowStatus("10s"); err != nil {
				return false
			}

			return true
		},
		WaitLoopTimeout,
		WaitLoopInterval,
	)
}

func (f *Framework) ElasticsearchIndicesCount(client es.ESClient) (int, error) {
	var count int
	var err error
	err = wait.PollImmediate(WaitLoopInterval, WaitLoopTimeout, func() (bool, error) {
		count, err = client.CountIndex()
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	return count, err
}

func (f *Framework) EventuallyElasticsearchIndicesCount(client es.ESClient) GomegaAsyncAssertion {
	return Eventually(
		func() int {
			count, err := client.CountIndex()
			if err != nil {
				return -1
			}
			return count
		},
		WaitLoopTimeout,
		WaitLoopInterval,
	)
}

func (f *Framework) CleanElasticsearch() {
	elasticsearchList, err := f.dbClient.KubedbV1alpha2().Elasticsearches(f.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, e := range elasticsearchList.Items {
		if _, _, err := util.PatchElasticsearch(context.TODO(), f.dbClient.KubedbV1alpha2(), &e, func(in *api.Elasticsearch) *api.Elasticsearch {
			in.ObjectMeta.Finalizers = nil
			in.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
			return in
		}, metav1.PatchOptions{}); err != nil {
			fmt.Printf("error Patching Elasticsearch. error: %v", err)
		}
	}
	if err := f.dbClient.KubedbV1alpha2().Elasticsearches(f.namespace).DeleteCollection(context.TODO(), meta_util.DeleteInForeground(), metav1.ListOptions{}); err != nil {
		fmt.Printf("error in deletion of Elasticsearch. Error: %v", err)
	}
}

func (f *Framework) EvictPodsFromStatefulSet(meta metav1.ObjectMeta) error {
	var err error
	labelSelector := labels.Set{
		meta_util.ManagedByLabelKey: kubedb.GroupName,
		api.LabelDatabaseKind:       api.ResourceKindElasticsearch,
		api.LabelDatabaseName:       meta.GetName(),
	}
	// get sts in the namespace
	stsList, err := f.kubeClient.AppsV1().StatefulSets(meta.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector.String()})
	if err != nil {
		return err
	}
	for _, sts := range stsList.Items {
		// if PDB is not found, send error
		var pdb *policy.PodDisruptionBudget
		pdb, err = f.kubeClient.PolicyV1beta1().PodDisruptionBudgets(sts.Namespace).Get(context.TODO(), sts.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		eviction := &policy.Eviction{
			TypeMeta: metav1.TypeMeta{
				APIVersion: policy.SchemeGroupVersion.String(),
				Kind:       kindEviction,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      sts.Name,
				Namespace: sts.Namespace,
			},
			DeleteOptions: &metav1.DeleteOptions{},
		}

		if pdb.Spec.MaxUnavailable == nil {
			return fmt.Errorf("found pdb %s spec.maxUnavailable nil", pdb.Name)
		}

		// try to evict as many pod as allowed in pdb. No err should occur
		maxUnavailable := pdb.Spec.MaxUnavailable.IntValue()
		for i := 0; i < maxUnavailable; i++ {
			eviction.Name = sts.Name + "-" + strconv.Itoa(i)

			err := f.kubeClient.PolicyV1beta1().Evictions(eviction.Namespace).Evict(context.TODO(), eviction)
			if err != nil {
				return err
			}
		}

		// try to evict one extra pod. TooManyRequests err should occur
		eviction.Name = sts.Name + "-" + strconv.Itoa(maxUnavailable)
		err = f.kubeClient.PolicyV1beta1().Evictions(eviction.Namespace).Evict(context.TODO(), eviction)
		if kerr.IsTooManyRequests(err) {
			err = nil
		} else if err != nil {
			return err
		} else {
			return fmt.Errorf("expected pod %s/%s to be not evicted due to pdb %s", sts.Namespace, eviction.Name, pdb.Name)
		}
	}
	return err
}

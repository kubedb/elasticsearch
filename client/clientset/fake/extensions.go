package fake

import (
	"github.com/k8sdb/elasticsearch/client/clientset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeExtensionClient struct {
	*testing.Fake
}

func NewFakeExtensionClient(objects ...runtime.Object) *FakeExtensionClient {
	o := testing.NewObjectTracker(api.Scheme, api.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Group == "k8sdb.com" {
			if err := o.Add(obj); err != nil {
				panic(err)
			}
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o, registered.RESTMapper()))

	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &FakeExtensionClient{&fakePtr}
}

func (m *FakeExtensionClient) Elasticsearch(ns string) client.ElasticsearchInterface {
	return &FakeElasticsearch{m.Fake, ns}
}
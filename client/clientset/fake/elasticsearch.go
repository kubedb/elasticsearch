package fake

import (
	aci "github.com/k8sdb/elasticsearch/api"
	"k8s.io/kubernetes/pkg/api"
	schema "k8s.io/kubernetes/pkg/api/unversioned"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeElasticsearch struct {
	Fake *testing.Fake
	ns   string
}

var elasticsearchResource = schema.GroupVersionResource{Group: "k8sdb.com", Version: "v1beta1", Resource: "elasticsearchs"}

// Get returns the Elasticsearch by name.
func (mock *FakeElasticsearch) Get(name string) (*aci.Elasticsearch, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(elasticsearchResource, mock.ns, name), &aci.Elasticsearch{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Elasticsearch), err
}

// List returns the a of Elasticsearchs.
func (mock *FakeElasticsearch) List(opts api.ListOptions) (*aci.ElasticsearchList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(elasticsearchResource, mock.ns, opts), &aci.Elasticsearch{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &aci.ElasticsearchList{}
	for _, item := range obj.(*aci.ElasticsearchList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Elasticsearch.
func (mock *FakeElasticsearch) Create(svc *aci.Elasticsearch) (*aci.Elasticsearch, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(elasticsearchResource, mock.ns, svc), &aci.Elasticsearch{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Elasticsearch), err
}

// Update updates a Elasticsearch.
func (mock *FakeElasticsearch) Update(svc *aci.Elasticsearch) (*aci.Elasticsearch, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(elasticsearchResource, mock.ns, svc), &aci.Elasticsearch{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Elasticsearch), err
}

// Delete deletes a Elasticsearch by name.
func (mock *FakeElasticsearch) Delete(name string, _ *api.DeleteOptions) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(elasticsearchResource, mock.ns, name), &aci.Elasticsearch{})

	return err
}

func (mock *FakeElasticsearch) UpdateStatus(srv *aci.Elasticsearch) (*aci.Elasticsearch, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(elasticsearchResource, "status", mock.ns, srv), &aci.Elasticsearch{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Elasticsearch), err
}

func (mock *FakeElasticsearch) Watch(opts api.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(elasticsearchResource, mock.ns, opts))
}

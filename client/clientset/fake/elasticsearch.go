package fake

import (
	aci "github.com/k8sdb/elasticsearch/api"
	"k8s.io/kubernetes/pkg/api"
	schema "k8s.io/kubernetes/pkg/api/unversioned"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeElastic struct {
	Fake *testing.Fake
	ns   string
}

var elasticResource = schema.GroupVersionResource{Group: "k8sdb.com", Version: "v1beta1", Resource: "elastics"}

// Get returns the Elastic by name.
func (mock *FakeElastic) Get(name string) (*aci.Elastic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(elasticResource, mock.ns, name), &aci.Elastic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Elastic), err
}

// List returns a list of Elastics.
func (mock *FakeElastic) List(opts api.ListOptions) (*aci.ElasticList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(elasticResource, mock.ns, opts), &aci.Elastic{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &aci.ElasticList{}
	for _, item := range obj.(*aci.ElasticList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Elastic.
func (mock *FakeElastic) Create(svc *aci.Elastic) (*aci.Elastic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(elasticResource, mock.ns, svc), &aci.Elastic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Elastic), err
}

// Update updates a Elastic.
func (mock *FakeElastic) Update(svc *aci.Elastic) (*aci.Elastic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(elasticResource, mock.ns, svc), &aci.Elastic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Elastic), err
}

// Delete deletes a Elastic by name.
func (mock *FakeElastic) Delete(name string) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(elasticResource, mock.ns, name), &aci.Elastic{})

	return err
}

func (mock *FakeElastic) UpdateStatus(srv *aci.Elastic) (*aci.Elastic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(elasticResource, "status", mock.ns, srv), &aci.Elastic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Elastic), err
}

func (mock *FakeElastic) Watch(opts api.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(elasticResource, mock.ns, opts))
}

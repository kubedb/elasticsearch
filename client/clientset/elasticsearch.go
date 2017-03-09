package client

import (
	aci "github.com/k8sdb/elasticsearch/api"
	"k8s.io/kubernetes/pkg/api"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/watch"
)

type ElasticsearchNamespacer interface {
	Elasticsearch(namespace string) ElasticsearchInterface
}

type ElasticsearchInterface interface {
	List(opts api.ListOptions) (*aci.ElasticsearchList, error)
	Get(name string) (*aci.Elasticsearch, error)
	Create(elasticsearch *aci.Elasticsearch) (*aci.Elasticsearch, error)
	Update(elasticsearch *aci.Elasticsearch) (*aci.Elasticsearch, error)
	Delete(name string, options *api.DeleteOptions) error
	Watch(opts api.ListOptions) (watch.Interface, error)
	UpdateStatus(elasticsearch *aci.Elasticsearch) (*aci.Elasticsearch, error)
}

type ElasticsearchImpl struct {
	r  rest.Interface
	ns string
}

func newElasticsearch(c *AppsCodeExtensionsClient, namespace string) *ElasticsearchImpl {
	return &ElasticsearchImpl{c.restClient, namespace}
}

func (c *ElasticsearchImpl) List(opts api.ListOptions) (result *aci.ElasticsearchList, err error) {
	result = &aci.ElasticsearchList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource("elasticsearchs").
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *ElasticsearchImpl) Get(name string) (result *aci.Elasticsearch, err error) {
	result = &aci.Elasticsearch{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource("elasticsearchs").
		Name(name).
		Do().
		Into(result)
	return
}

func (c *ElasticsearchImpl) Create(elasticsearch *aci.Elasticsearch) (result *aci.Elasticsearch, err error) {
	result = &aci.Elasticsearch{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource("elasticsearchs").
		Body(elasticsearch).
		Do().
		Into(result)
	return
}

func (c *ElasticsearchImpl) Update(elasticsearch *aci.Elasticsearch) (result *aci.Elasticsearch, err error) {
	result = &aci.Elasticsearch{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource("elasticsearchs").
		Name(elasticsearch.Name).
		Body(elasticsearch).
		Do().
		Into(result)
	return
}

func (c *ElasticsearchImpl) Delete(name string, options *api.DeleteOptions) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource("elasticsearchs").
		Name(name).
		Body(options).
		Do().
		Error()
}

func (c *ElasticsearchImpl) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource("elasticsearchs").
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *ElasticsearchImpl) UpdateStatus(elasticsearch *aci.Elasticsearch) (result *aci.Elasticsearch, err error) {
	result = &aci.Elasticsearch{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource("elasticsearchs").
		Name(elasticsearch.Name).
		SubResource("status").
		Body(elasticsearch).
		Do().
		Into(result)
	return
}

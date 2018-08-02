package framework

import (
	"fmt"

	"github.com/appscode/kutil/tools/portforward"
	amc "github.com/kubedb/apimachinery/pkg/controller"
	"github.com/kubedb/elasticsearch/pkg/controller"
	"github.com/kubedb/elasticsearch/pkg/docker"
	"github.com/kubedb/elasticsearch/pkg/es-util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) GetElasticClient(meta metav1.ObjectMeta) (es_util.ESClient, error) {
	es, err := f.GetElasticsearch(meta)
	if err != nil {
		return nil, err
	}
	clientName := es.Name

	if es.Spec.Topology != nil {
		if es.Spec.Topology.Client.Prefix != "" {
			clientName = fmt.Sprintf("%v-%v", es.Spec.Topology.Client.Prefix, clientName)
		}
	}
	clientPodName := fmt.Sprintf("%v-0", clientName)
	tunnel := portforward.NewTunnel(
		f.kubeClient.CoreV1().RESTClient(),
		f.restConfig,
		es.Namespace,
		clientPodName,
		controller.ElasticsearchRestPort,
	)
	if err := tunnel.ForwardPort(); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://127.0.0.1:%d", tunnel.Local)
	c := controller.New(nil, f.kubeClient, nil, nil, nil, nil, docker.Docker{}, amc.Config{})
	return es_util.GetElasticClient(c.Client, es, url)
}

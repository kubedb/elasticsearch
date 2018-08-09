package framework

import (
	"fmt"

	"github.com/appscode/kutil/tools/portforward"
	amc "github.com/kubedb/apimachinery/pkg/controller"
	"github.com/kubedb/elasticsearch/pkg/controller"
	"github.com/kubedb/elasticsearch/pkg/util/es"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) GetElasticClient(meta metav1.ObjectMeta) (es.ESClient, error) {
	db, err := f.GetElasticsearch(meta)
	if err != nil {
		return nil, err
	}
	clientName := db.Name

	if db.Spec.Topology != nil {
		if db.Spec.Topology.Client.Prefix != "" {
			clientName = fmt.Sprintf("%v-%v", db.Spec.Topology.Client.Prefix, clientName)
		}
	}
	clientPodName := fmt.Sprintf("%v-0", clientName)
	tunnel := portforward.NewTunnel(
		f.kubeClient.CoreV1().RESTClient(),
		f.restConfig,
		db.Namespace,
		clientPodName,
		controller.ElasticsearchRestPort,
	)
	if err := tunnel.ForwardPort(); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://127.0.0.1:%d", tunnel.Local)
	c := controller.New(nil, f.kubeClient, nil, nil, nil, nil, amc.Config{})
	return es.GetElasticClient(c.Client, db, url)
}

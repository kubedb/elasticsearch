package framework

import (
	"context"
	"fmt"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/kutil/tools/portforward"
	"github.com/kubedb/elasticsearch/pkg/controller"
	"gopkg.in/olivere/elastic.v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) GetElasticClient(meta metav1.ObjectMeta) (*elastic.Client, error) {
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
		9200,
	)
	if err := tunnel.ForwardPort(); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://127.0.0.1:%d", tunnel.Local)
	c := controller.New(nil, f.kubeClient, nil, nil, nil, nil, controller.Options{})
	return c.GetElasticClient(es, url)
}

func (f *Framework) CreateIndex(client *elastic.Client, count int) error {
	for i := 0; i < count; i++ {
		_, err := client.CreateIndex(rand.Characters(5)).Do(context.Background())
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *Framework) CountIndex(client *elastic.Client) (int, error) {
	resp, err := client.ClusterStats().Do(context.Background())
	if err != nil {
		return 0, err
	}
	return resp.Indices.Count, nil
}

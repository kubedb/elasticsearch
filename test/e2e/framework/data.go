package framework

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/appscode/go/crypto/rand"
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
	fmt.Println(clientPodName)

	url, err := f.getProxyURL(es.Namespace, clientPodName, 9200)
	if err != nil {
		return nil, err
	}

	secret, err := f.kubeClient.Core().Secrets(es.Namespace).Get(es.Spec.DatabaseSecret.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return elastic.NewClient(
		elastic.SetHttpClient(&http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}),
		elastic.SetBasicAuth("admin", string(secret.Data["ADMIN_PASSWORD"])),
		elastic.SetURL(url),
		elastic.SetHealthcheck(true),
		elastic.SetSniff(false),
	)
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

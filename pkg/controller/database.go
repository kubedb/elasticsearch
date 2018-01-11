package controller

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/portforward"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"gopkg.in/olivere/elastic.v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (c *Controller) GetElasticClient(elasticsearch *api.Elasticsearch, url string) (*elastic.Client, error) {
	secret, err := c.Client.CoreV1().Secrets(elasticsearch.Namespace).Get(elasticsearch.Spec.DatabaseSecret.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var client *elastic.Client
	err = wait.PollImmediate(time.Second*30, time.Minute*5, func() (bool, error) {
		client, err = elastic.NewClient(
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
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.New("failed to connect Elasticsearch")
	}
	return client, nil
}

func (c *Controller) getAllIndices(elasticsearch *api.Elasticsearch) (string, error) {
	var url string
	if meta.PossiblyInCluster() {
		url = fmt.Sprintf("https://%s.%s:9200", elasticsearch.OffshootName(), elasticsearch.Namespace)
	} else {
		clientName := elasticsearch.OffshootName()
		if elasticsearch.Spec.Topology != nil {
			if elasticsearch.Spec.Topology.Client.Prefix != "" {
				clientName = fmt.Sprintf("%v-%v", elasticsearch.Spec.Topology.Client.Prefix, clientName)
			}
		}
		clientPodName := fmt.Sprintf("%v-0", clientName)
		tunnel := portforward.NewTunnel(
			c.Client.CoreV1().RESTClient(),
			c.config,
			elasticsearch.Namespace,
			clientPodName,
			9200,
		)
		if err := tunnel.ForwardPort(); err != nil {
			return "", err
		}
		url = fmt.Sprintf("https://127.0.0.1:%d", tunnel.Local)
	}

	client, err := c.GetElasticClient(elasticsearch, url)
	if err != nil {
		return "", err
	}
	resp, err := client.Aliases().Do(context.Background())
	if err != nil {
		return "", err
	}
	indices := make([]string, 0)

	for k := range resp.Indices {
		indices = append(indices, k)
	}

	return strings.Join(indices, ","), nil
}

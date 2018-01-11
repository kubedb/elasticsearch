package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"gopkg.in/olivere/elastic.v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func (c *Controller) GetElasticClient(elasticsearch *api.Elasticsearch, url string) (*elastic.Client, error) {
	secret, err := c.Client.CoreV1().Secrets(elasticsearch.Namespace).Get(elasticsearch.Spec.DatabaseSecret.SecretName, metav1.GetOptions{})
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

func (c *Controller) getAllIndices(elasticsearch *api.Elasticsearch) (string, error) {
	var url string
	_, err := restclient.InClusterConfig()
	if err == nil {
		url = fmt.Sprintf("https://%s:9200", elasticsearch.OffshootName())
	} else {
		clientName := elasticsearch.OffshootName()
		if elasticsearch.Spec.Topology != nil {
			if elasticsearch.Spec.Topology.Client.Prefix != "" {
				clientName = fmt.Sprintf("%v-%v", elasticsearch.Spec.Topology.Client.Prefix, clientName)
			}
		}
		clientPodName := fmt.Sprintf("%v-0", clientName)
		url, err = c.GetProxyURL(c.config, elasticsearch.Namespace, clientPodName, 9200)
		if err != nil {
			return "", err
		}
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

	for k, _ := range resp.Indices {
		indices = append(indices, k)
	}

	return strings.Join(indices, ","), nil
}

func (c *Controller) GetProxyURL(config *restclient.Config, namespace, podName string, port int) (string, error) {
	tunnel := newTunnel(c.Client, config, namespace, podName, port)
	if err := tunnel.forwardPort(); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://127.0.0.1:%d", tunnel.Local), nil
}

type tunnel struct {
	Local      int
	Remote     int
	Namespace  string
	PodName    string
	Out        io.Writer
	stopChan   chan struct{}
	readyChan  chan struct{}
	config     *restclient.Config
	kubeClient kubernetes.Interface
}

func newTunnel(client kubernetes.Interface, config *restclient.Config, namespace, podName string, remote int) *tunnel {
	return &tunnel{
		config:     config,
		kubeClient: client,
		Namespace:  namespace,
		PodName:    podName,
		Remote:     remote,
		stopChan:   make(chan struct{}, 1),
		readyChan:  make(chan struct{}, 1),
		Out:        ioutil.Discard,
	}
}

func (t *tunnel) forwardPort() error {
	u := t.kubeClient.Core().RESTClient().Post().
		Resource("pods").
		Namespace(t.Namespace).
		Name(t.PodName).
		SubResource("portforward").URL()

	transport, upgrader, err := spdy.RoundTripperFor(t.config)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", u)

	local, err := getAvailablePort()
	if err != nil {
		return fmt.Errorf("could not find an available port: %s", err)
	}
	t.Local = local

	ports := []string{fmt.Sprintf("%d:%d", t.Local, t.Remote)}

	pf, err := portforward.New(dialer, ports, t.stopChan, t.readyChan, t.Out, t.Out)
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		errChan <- pf.ForwardPorts()
	}()

	select {
	case err = <-errChan:
		return fmt.Errorf("forwarding ports: %v", err)
	case <-pf.Ready:
		return nil
	}
}

func getAvailablePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	_, p, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return 0, err
	}
	return port, err
}

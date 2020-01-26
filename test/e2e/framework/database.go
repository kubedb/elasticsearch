/*
Copyright The KubeDB Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package framework

import (
	"fmt"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
	amc "kubedb.dev/apimachinery/pkg/controller"
	"kubedb.dev/elasticsearch/pkg/controller"
	"kubedb.dev/elasticsearch/pkg/util/es"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kmodules.xyz/client-go/tools/portforward"
)

func (f *Framework) GetClientPodName(elasticsearch *api.Elasticsearch) string {
	clientName := elasticsearch.Name

	if elasticsearch.Spec.Topology != nil {
		if elasticsearch.Spec.Topology.Client.Prefix != "" {
			clientName = fmt.Sprintf("%v-%v", elasticsearch.Spec.Topology.Client.Prefix, clientName)
		}
	}
	return fmt.Sprintf("%v-0", clientName)
}

func (f *Framework) GetElasticClient(meta metav1.ObjectMeta) (es.ESClient, error) {
	db, err := f.GetElasticsearch(meta)
	if err != nil {
		return nil, err
	}
	clientPodName := f.GetClientPodName(db)
	f.Tunnel = portforward.NewTunnel(
		f.kubeClient.CoreV1().RESTClient(),
		f.restConfig,
		db.Namespace,
		clientPodName,
		api.ElasticsearchRestPort,
	)
	if err := f.Tunnel.ForwardPort(); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%v://127.0.0.1:%d", db.GetConnectionScheme(), f.Tunnel.Local)
	c := controller.New(
		nil,
		f.kubeClient,
		nil,
		f.dbClient,
		nil,
		nil,
		nil,
		nil,
		amc.Config{},
		f.topology,
		nil,
	)
	return es.GetElasticClient(c.Client, c.ExtClient, db, url)
}

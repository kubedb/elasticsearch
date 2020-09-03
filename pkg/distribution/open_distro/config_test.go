/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package open_distro

import (
	"context"
	"fmt"
	"testing"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestElasticsearch_getInternalUserConfig(t *testing.T) {
	t.SkipNow()

	type fields struct {
		kClient       kubernetes.Interface
		elasticsearch *api.Elasticsearch
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Check output",
			fields: fields{
				kClient: &fake.Clientset{},
				elasticsearch: &api.Elasticsearch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-jrjwe",
						Namespace: "test-23wefjds",
					},
					Spec: api.ElasticsearchSpec{
						DatabaseSecret: &corev1.SecretVolumeSource{
							SecretName: "db-secret",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &Elasticsearch{
				kClient:       tt.fields.kClient,
				elasticsearch: tt.fields.elasticsearch,
			}
			_, err := es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Create(context.TODO(), &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-secret",
					Namespace: es.elasticsearch.Namespace,
				},
				Data: map[string][]byte{},
			}, metav1.CreateOptions{})
			if err != nil {
				panic(err)
			}
			got, err := es.getInternalUserConfig()
			fmt.Println(got)
			if (err != nil) != tt.wantErr {
				t.Errorf("getInternalUserConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

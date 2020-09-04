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

package search_guard

import (
	"context"
	"testing"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestElasticsearch_getInternalUserConfig(t *testing.T) {

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
				kClient: fake.NewSimpleClientset(),
				elasticsearch: &api.Elasticsearch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-es",
						Namespace: "test",
					},
					Spec: api.ElasticsearchSpec{
						InternalUsers: map[string]api.ElasticsearchUserSpec{
							"user1": {
								Reserved:         true,
								Hidden:           false,
								BackendRoles:     []string{"role1", "role2"},
								SearchGuardRoles: []string{"sgRole1", "sgRole2"},
								Attributes: map[string]string{
									"attr1": "b",
									"attr2": "d",
								},
								Description: "test user1",
							},
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
					Name: "test-es-user1-cred",
				},
				Data: map[string][]byte{
					corev1.BasicAuthUsernameKey: []byte("user1"),
					corev1.BasicAuthPasswordKey: []byte("password"),
				},
				Type: corev1.SecretTypeBasicAuth,
			}, metav1.CreateOptions{})
			if err != nil {
				panic(err)
			}

			_, err = es.getInternalUserConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("getInternalUserConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

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

package elastic_stack

import (
	"context"
	"fmt"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

const (
	ConfigFileName             = "elasticsearch.yml"
	ConfigFileMountPath        = "/usr/share/elasticsearch/config"
	TempConfigFileMountPath    = "/elasticsearch/temp-config"
	DatabaseConfigSecretSuffix = "config"
)

var xpack_config = `
xpack.security.enabled: true

xpack.security.transport.ssl.enabled: true
xpack.security.transport.ssl.verification_mode: certificate
xpack.security.transport.ssl.keystore.path: /usr/share/elasticsearch/config/certs/node.jks
xpack.security.transport.ssl.keystore.password: ${KEY_PASS}
xpack.security.transport.ssl.truststore.path: /usr/share/elasticsearch/config/certs/root.jks
xpack.security.transport.ssl.truststore.password: ${KEY_PASS}

xpack.security.http.ssl.enabled: ${SSL_ENABLE}
xpack.security.http.ssl.keystore.path: /usr/share/elasticsearch/config/certs/client.jks
xpack.security.http.ssl.keystore.password: ${KEY_PASS}
xpack.security.http.ssl.truststore.path: /usr/share/elasticsearch/config/certs/root.jks
xpack.security.http.ssl.truststore.password: ${KEY_PASS}
`

func (es *Elasticsearch) EnsureDefaultConfig() error {
	if !es.elasticsearch.Spec.DisableSecurity {
		if err := es.findDefaultConfig(); err != nil {
			return err
		}

		secretMeta := metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", es.elasticsearch.OffshootName(), DatabaseConfigSecretSuffix),
			Namespace: es.elasticsearch.Namespace,
		}
		owner := metav1.NewControllerRef(es.elasticsearch, api.SchemeGroupVersion.WithKind(api.ResourceKindElasticsearch))

		if _, _, err := core_util.CreateOrPatchSecret(context.TODO(), es.kClient, secretMeta, func(in *corev1.Secret) *corev1.Secret {
			in.Labels = core_util.UpsertMap(in.Labels, es.elasticsearch.OffshootLabels())
			core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
			in.Data = map[string][]byte{
				ConfigFileName: []byte(xpack_config),
			}
			return in
		}, metav1.PatchOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (es *Elasticsearch) findDefaultConfig() error {
	sName := fmt.Sprintf("%v-%v", es.elasticsearch.OffshootName(), DatabaseConfigSecretSuffix)

	secret, err := es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Get(context.TODO(), sName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	if secret.Labels[api.LabelDatabaseKind] != api.ResourceKindElasticsearch &&
		secret.Labels[api.LabelDatabaseName] != es.elasticsearch.Name {
		return fmt.Errorf(`intended k8s secret: "%v/%v" already exists`, es.elasticsearch.Namespace, sName)
	}

	return nil
}
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

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
	"kubedb.dev/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"

	"github.com/appscode/go/crypto/rand"
	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AdminUser               = "admin"
	KibanaserverUser        = "kibanaserver"
	KeyAdminUserName        = "ADMIN_USERNAME"
	KeyAdminPassword        = "ADMIN_PASSWORD"
	KeyKibanaServerUserName = "KIBANASERVER_USERNAME"
	KeyKibanaServerPassword = "KIBANASERVER_PASSWORD"
)

func (es *Elasticsearch) EnsureDatabaseSecret() error {
	databaseSecretVolume := es.elasticsearch.Spec.DatabaseSecret
	if databaseSecretVolume == nil {
		var err error
		if databaseSecretVolume, err = es.createDatabaseSecret(); err != nil {
			return err
		}
		newES, _, err := util.PatchElasticsearch(context.TODO(), es.extClient.KubedbV1alpha1(), es.elasticsearch, func(in *api.Elasticsearch) *api.Elasticsearch {
			in.Spec.DatabaseSecret = databaseSecretVolume
			return in
		}, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		es.elasticsearch = newES
		return nil
	}
	return nil
}

func (es *Elasticsearch) createDatabaseSecret() (*corev1.SecretVolumeSource, error) {
	databaseSecret, err := es.findDatabaseSecret()
	if err != nil {
		return nil, err
	}
	if databaseSecret != nil {
		return &corev1.SecretVolumeSource{
			SecretName: databaseSecret.Name,
		}, nil
	}

	adminPassword := rand.Characters(8)
	kibanaPassword := rand.Characters(8)
	var data = map[string][]byte{
		KeyAdminUserName:        []byte(AdminUser),
		KeyAdminPassword:        []byte(adminPassword),
		KeyKibanaServerUserName: []byte(KibanaserverUser),
		KeyKibanaServerPassword: []byte(kibanaPassword),
	}

	name := fmt.Sprintf("%v-auth", es.elasticsearch.OffshootName())
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: es.elasticsearch.OffshootLabels(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	if _, err := es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		return nil, err
	}

	return &corev1.SecretVolumeSource{
		SecretName: secret.Name,
	}, nil
}

func (es *Elasticsearch) findDatabaseSecret() (*corev1.Secret, error) {
	name := fmt.Sprintf("%v-auth", es.elasticsearch.OffshootName())
	secret, err := es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	if secret.Labels[api.LabelDatabaseKind] != api.ResourceKindElasticsearch ||
		secret.Labels[api.LabelDatabaseName] != es.elasticsearch.Name {
		return nil, fmt.Errorf(`intended secret "%v/%v" already exists`, es.elasticsearch.Namespace, name)
	}

	return secret, nil
}

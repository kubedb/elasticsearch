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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
	"kubedb.dev/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	certlib "kubedb.dev/elasticsearch/pkg/lib/cert"
	"kubedb.dev/elasticsearch/pkg/lib/cert/pkcs8"

	"github.com/appscode/go/crypto/rand"
	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (es *Elasticsearch) EnsureCertSecrets() error {
	if es.elasticsearch.Spec.DisableSecurity {
		return nil
	}

	certSecretVolumeSource := es.elasticsearch.Spec.CertificateSecret
	if certSecretVolumeSource == nil {
		var err error
		if certSecretVolumeSource, err = es.createCertSecret(); err != nil {
			return err
		}
		newES, _, err := util.PatchElasticsearch(context.TODO(), es.extClient.KubedbV1alpha1(), es.elasticsearch, func(in *api.Elasticsearch) *api.Elasticsearch {
			in.Spec.CertificateSecret = certSecretVolumeSource
			return in
		}, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		es.elasticsearch = newES
	}
	return nil
}

func (es *Elasticsearch) createCertSecret() (*corev1.SecretVolumeSource, error) {
	certSecret, err := es.findCertSecret()
	if err != nil {
		return nil, err
	}

	if certSecret != nil {
		return &corev1.SecretVolumeSource{
			SecretName: certSecret.Name,
		}, nil
	}

	certPath := fmt.Sprintf("%v/%v", certlib.CertsDir, rand.Characters(3))
	if err := os.MkdirAll(certPath, os.ModePerm); err != nil {
		return nil, err
	}

	caKey, caCert, err := pkcs8.CreateCaCertificatePEM(certPath)
	if err != nil {
		return nil, err
	}
	rootCa, err := ioutil.ReadFile(filepath.Join(certPath, certlib.RootCert))
	if err != nil {
		return nil, err
	}
	rootKey, err := ioutil.ReadFile(filepath.Join(certPath, certlib.RootKey))
	if err != nil {
		return nil, err
	}
	data := map[string][]byte{
		certlib.RootCert: rootCa,
		certlib.RootKey:  rootKey,
	}

	err = pkcs8.CreateNodeCertificatePEM(certPath, es.elasticsearch, caKey, caCert)
	if err != nil {
		return nil, err
	}
	nodeCert, err := ioutil.ReadFile(filepath.Join(certPath, certlib.NodeCert))
	if err != nil {
		return nil, err
	}
	nodeKey, err := ioutil.ReadFile(filepath.Join(certPath, certlib.NodeKey))
	if err != nil {
		return nil, err
	}
	data[certlib.NodeKey] = nodeKey
	data[certlib.NodeCert] = nodeCert

	if err := pkcs8.CreateAdminCertificatePEM(certPath, es.elasticsearch, caKey, caCert); err != nil {
		return nil, err
	}
	adminCert, err := ioutil.ReadFile(filepath.Join(certPath, certlib.AdminCert))
	if err != nil {
		return nil, err
	}
	adminKey, err := ioutil.ReadFile(filepath.Join(certPath, certlib.AdminKey))
	if err != nil {
		return nil, err
	}
	data[certlib.AdminKey] = adminKey
	data[certlib.AdminCert] = adminCert

	if err := pkcs8.CreateClientCertificatePEM(certPath, es.elasticsearch, caKey, caCert); err != nil {
		return nil, err
	}
	clientCert, err := ioutil.ReadFile(filepath.Join(certPath, certlib.ClientCert))
	if err != nil {
		return nil, err
	}
	clientKey, err := ioutil.ReadFile(filepath.Join(certPath, certlib.ClientKey))
	if err != nil {
		return nil, err
	}
	data[certlib.ClientKey] = clientKey
	data[certlib.ClientCert] = clientCert

	name := fmt.Sprintf("%v-cert", es.elasticsearch.OffshootName())
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

	secretVolumeSource := &corev1.SecretVolumeSource{
		SecretName: secret.Name,
	}

	return secretVolumeSource, nil
}

func (es *Elasticsearch) findCertSecret() (*corev1.Secret, error) {
	name := fmt.Sprintf("%v-cert", es.elasticsearch.OffshootName())

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

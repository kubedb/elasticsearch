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
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
	certlib "kubedb.dev/elasticsearch/pkg/lib/cert"
	"kubedb.dev/elasticsearch/pkg/lib/cert/pkcs8"

	"github.com/appscode/go/crypto/rand"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

// EnsureCertSecrets creates certificates if they don't exist.
// If the "TLS.IssuerRef" is set, the operator won't create certificates.
func (es *Elasticsearch) EnsureCertSecrets() error {
	if es.elasticsearch.Spec.DisableSecurity {
		return nil
	}

	if es.elasticsearch.Spec.TLS == nil {
		return errors.New("tls configuration is missing")
	}

	// Certificates are managed by the enterprise operator.
	// Ignore sync/creation.
	if es.elasticsearch.Spec.TLS.IssuerRef != nil {
		return nil
	}

	certPath := fmt.Sprintf("%v/%v", certlib.CertsDir, rand.Characters(3))
	if err := os.MkdirAll(certPath, os.ModePerm); err != nil {
		return err
	}

	caKey, caCert, err := es.createRootCertSecret(certPath)
	if err != nil {
		return err
	}

	err = es.createNodeCertSecret(caKey, caCert, certPath)
	if err != nil {
		return err
	}

	if es.elasticsearch.Spec.EnableSSL {
		// When SSL is enabled, create certificates for HTTP layer
		err = es.createClientCertSecret(caKey, caCert, certPath)
		if err != nil {
			return err
		}

	}

	return nil
}

func (es *Elasticsearch) createRootCertSecret(cPath string) (*rsa.PrivateKey, *x509.Certificate, error) {
	rSecret, err := es.findSecret(es.elasticsearch.MustCertSecretName(api.ElasticsearchRootCert))
	if err != nil {
		return nil, nil, err
	}

	if rSecret == nil {
		// create certs here
		caKey, caCert, err := pkcs8.CreateCaCertificatePEM(cPath)
		if err != nil {
			return nil, nil, err
		}
		rootCa, err := ioutil.ReadFile(filepath.Join(cPath, certlib.RootCert))
		if err != nil {
			return nil, nil, err
		}
		rootKey, err := ioutil.ReadFile(filepath.Join(cPath, certlib.RootKey))
		if err != nil {
			return nil, nil, err
		}

		data := map[string][]byte{
			certlib.RootCert: rootCa,
			certlib.RootKey:  rootKey,
		}

		owner := metav1.NewControllerRef(es.elasticsearch, api.SchemeGroupVersion.WithKind(api.ResourceKindElasticsearch))

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   es.elasticsearch.MustCertSecretName(api.ElasticsearchRootCert),
				Labels: es.elasticsearch.OffshootLabels(),
			},
			Type: corev1.SecretTypeTLS,
			Data: data,
		}
		core_util.EnsureOwnerReference(&secret.ObjectMeta, owner)

		_, err = es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return nil, nil, err
		}

		return caKey, caCert, nil
	}

	data := rSecret.Data
	var caKey *rsa.PrivateKey
	var caCert []*x509.Certificate

	if value, ok := data[certlib.RootCert]; ok {
		caCert, err = cert.ParseCertsPEM(value)
		if err != nil || len(caCert) == 0 {
			return nil, nil, errors.Wrap(err, "failed to parse root-ca.pem")
		}
	} else {
		return nil, nil, errors.New("root-ca.pem is missing")
	}

	if value, ok := data[certlib.RootKey]; ok {
		key, err := cert.ParsePrivateKeyPEM(value)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse root-key.pem")
		}

		caKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, errors.New("failed to typecast the root-key.pem")
		}

	} else {
		return nil, nil, errors.New("root-key.pem is missing")
	}

	return caKey, caCert[0], nil
}

func (es *Elasticsearch) createNodeCertSecret(caKey *rsa.PrivateKey, caCert *x509.Certificate, cPath string) error {
	nSecret, err := es.findSecret(es.elasticsearch.MustCertSecretName(api.ElasticsearchTransportCert))
	if err != nil {
		return err
	}

	if nSecret == nil {
		// create certs here
		err := pkcs8.CreateNodeCertificatePEM(cPath, es.elasticsearch, caKey, caCert)
		if err != nil {
			return err
		}

		rootCa, err := ioutil.ReadFile(filepath.Join(cPath, certlib.RootCert))
		if err != nil {
			return err
		}

		nodeCert, err := ioutil.ReadFile(filepath.Join(cPath, certlib.NodeCert))
		if err != nil {
			return err
		}

		nodeKey, err := ioutil.ReadFile(filepath.Join(cPath, certlib.NodeKey))
		if err != nil {
			return err
		}

		data := map[string][]byte{
			certlib.RootCert: rootCa,
			certlib.NodeKey:  nodeKey,
			certlib.NodeCert: nodeCert,
		}

		owner := metav1.NewControllerRef(es.elasticsearch, api.SchemeGroupVersion.WithKind(api.ResourceKindElasticsearch))

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   es.elasticsearch.MustCertSecretName(api.ElasticsearchTransportCert),
				Labels: es.elasticsearch.OffshootLabels(),
			},
			Type: corev1.SecretTypeTLS,
			Data: data,
		}
		core_util.EnsureOwnerReference(&secret.ObjectMeta, owner)

		_, err = es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		return nil
	}

	// If the secret already exists,
	// check whether the keys exist too.
	if value, ok := nSecret.Data[certlib.RootCert]; !ok || len(value) == 0 {
		return errors.New("root-ca.pem is missing")
	}

	if value, ok := nSecret.Data[certlib.NodeKey]; !ok || len(value) == 0 {
		return errors.New("node-key.pem is missing")
	}

	if value, ok := nSecret.Data[certlib.NodeCert]; !ok || len(value) == 0 {
		return errors.New("node.pem is missing")
	}

	return nil
}

func (es *Elasticsearch) createClientCertSecret(caKey *rsa.PrivateKey, caCert *x509.Certificate, cPath string) error {
	cSecret, err := es.findSecret(es.elasticsearch.MustCertSecretName(api.ElasticsearchHTTPCert))
	if err != nil {
		return err
	}

	if cSecret == nil {
		// If issuerRef is set, the certificates are handled by the enterprise operator.
		// Not ready yet, wait for enterprise operator to make the certificate available.
		if es.elasticsearch.Spec.TLS.IssuerRef != nil {
			return nil
		}

		// create certs here
		if err := pkcs8.CreateClientCertificatePEM(cPath, es.elasticsearch, caKey, caCert); err != nil {
			return err
		}

		rootCa, err := ioutil.ReadFile(filepath.Join(cPath, certlib.RootCert))
		if err != nil {
			return err
		}

		clientCert, err := ioutil.ReadFile(filepath.Join(cPath, certlib.ClientCert))
		if err != nil {
			return err
		}

		clientKey, err := ioutil.ReadFile(filepath.Join(cPath, certlib.ClientKey))
		if err != nil {
			return err
		}

		data := map[string][]byte{
			certlib.RootCert:   rootCa,
			certlib.ClientKey:  clientKey,
			certlib.ClientCert: clientCert,
		}

		owner := metav1.NewControllerRef(es.elasticsearch, api.SchemeGroupVersion.WithKind(api.ResourceKindElasticsearch))

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   es.elasticsearch.MustCertSecretName(api.ElasticsearchHTTPCert),
				Labels: es.elasticsearch.OffshootLabels(),
			},
			Type: corev1.SecretTypeTLS,
			Data: data,
		}
		core_util.EnsureOwnerReference(&secret.ObjectMeta, owner)

		_, err = es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		return nil
	}

	// If the secret already exists,
	// check whether the keys exist too.
	if value, ok := cSecret.Data[certlib.RootCert]; !ok || len(value) == 0 {
		return errors.New("root-ca.pem is missing")
	}

	if value, ok := cSecret.Data[certlib.ClientKey]; !ok || len(value) == 0 {
		return errors.New("client-key.pem is missing")
	}

	if value, ok := cSecret.Data[certlib.ClientCert]; !ok || len(value) == 0 {
		return errors.New("client.pem is missing")
	}

	return nil
}

func (es *Elasticsearch) findSecret(name string) (*corev1.Secret, error) {

	secret, err := es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return secret, nil
}

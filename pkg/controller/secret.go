package controller

import (
	"fmt"
	"os"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/io"
	api "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	"golang.org/x/crypto/bcrypt"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
)

func (c *Controller) findCertSecret(elasticsearch *api.Elasticsearch) (*core.Secret, error) {
	name := fmt.Sprintf("%v-cert", elasticsearch.OffshootName())

	secret, err := c.Client.CoreV1().Secrets(elasticsearch.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	if secret.Labels[api.LabelDatabaseKind] != api.ResourceKindElasticsearch ||
		secret.Labels[api.LabelDatabaseName] != elasticsearch.Name {
		return nil, fmt.Errorf(`Intended secret "%v" already exists`, name)
	}

	return secret, nil
}

func (c *Controller) createCertSecret(elasticsearch *api.Elasticsearch) (*core.SecretVolumeSource, error) {

	certSecret, err := c.findCertSecret(elasticsearch)
	if err != nil {
		return nil, err
	}
	if certSecret != nil {
		return &core.SecretVolumeSource{
			SecretName: certSecret.Name,
		}, nil
	}

	certPath := fmt.Sprintf("%v/%v", certsDir, rand.Characters(3))
	os.Mkdir(certPath, os.ModePerm)

	caKey, caCert, err := createCaCertificate(certPath)
	if err != nil {
		return nil, err
	}
	err = createNodeCertificate(certPath, elasticsearch, caKey, caCert)
	if err != nil {
		return nil, err
	}
	err = createAdminCertificate(certPath, caKey, caCert)
	if err != nil {
		return nil, err
	}
	err = createClientCertificate(certPath, caKey, caCert)
	if err != nil {
		return nil, err
	}

	truststore, err := io.ReadFile(fmt.Sprintf("%s/truststore.jks", certPath))
	if err != nil {
		return nil, err
	}
	keyStore, err := io.ReadFile(fmt.Sprintf("%s/keystore.jks", certPath))
	if err != nil {
		return nil, err
	}
	sgadmin, err := io.ReadFile(fmt.Sprintf("%s/sgadmin.jks", certPath))
	if err != nil {
		return nil, err
	}

	clientKey, err := io.ReadFile(fmt.Sprintf("%s/client-key.pem", certPath))
	if err != nil {
		return nil, err
	}
	clientCert, err := io.ReadFile(fmt.Sprintf("%s/client.pem", certPath))
	if err != nil {
		return nil, err
	}

	data := map[string][]byte{
		"ca.pem":         cert.EncodeCertPEM(caCert),
		"truststore.jks": []byte(truststore),
		"keystore.jks":   []byte(keyStore),
		"sgadmin.jks":    []byte(sgadmin),
		"client-key.pem": []byte(clientKey),
		"client.pem":     []byte(clientCert),
	}

	name := fmt.Sprintf("%v-cert", elasticsearch.OffshootName())
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				api.LabelDatabaseKind: api.ResourceKindElasticsearch,
				api.LabelDatabaseName: elasticsearch.Name,
			},
		},
		Type: core.SecretTypeOpaque,
		Data: data,
	}
	if _, err := c.Client.CoreV1().Secrets(elasticsearch.Namespace).Create(secret); err != nil {
		return nil, err
	}

	return &core.SecretVolumeSource{
		SecretName: secret.Name,
	}, nil
}

func (c *Controller) findAuthSecret(elasticsearch *api.Elasticsearch) (*core.Secret, error) {
	name := fmt.Sprintf("%v-auth", elasticsearch.OffshootName())

	secret, err := c.Client.CoreV1().Secrets(elasticsearch.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	if secret.Labels[api.LabelDatabaseKind] != api.ResourceKindElasticsearch ||
		secret.Labels[api.LabelDatabaseName] != elasticsearch.Name {
		return nil, fmt.Errorf(`Intended secret "%v" already exists`, name)
	}

	return secret, nil
}

var action_group = `
UNLIMITED:
  - "*"

READ:
  - "indices:data/read*"
  - "indices:admin/mappings/fields/get*"

CLUSTER_COMPOSITE_OPS_RO:
  - "indices:data/read/mget"
  - "indices:data/read/msearch"
  - "indices:data/read/mtv"
  - "indices:data/read/coordinate-msearch*"
  - "indices:admin/aliases/exists*"
  - "indices:admin/aliases/get*"

CLUSTER_KUBEDB_SNAPSHOT:
  - "indices:data/read/scroll*"

INDICES_KUBEDB_SNAPSHOT:
  - "indices:admin/get"
`

var config = `
searchguard:
  dynamic:
    authc:
      basic_internal_auth_domain:
        enabled: true
        order: 4
        http_authenticator:
          type: basic
          challenge: true
        authentication_backend:
          type: intern
`

var internal_user = `
admin:
  hash: %s

readall:
  hash: %s
`

var roles = `
sg_all_access:
  cluster:
    - UNLIMITED
  indices:
    '*':
      '*':
        - UNLIMITED
  tenants:
    adm_tenant: RW
    test_tenant_ro: RW

sg_readall:
  cluster:
    - CLUSTER_COMPOSITE_OPS_RO
    - CLUSTER_KUBEDB_SNAPSHOT
  indices:
    '*':
      '*':
        - READ
        - INDICES_KUBEDB_SNAPSHOT
`

var roles_mapping = `
sg_all_access:
  users:
    - admin

sg_readall:
  users:
    - readall
`

func (c *Controller) createAuthSecret(elasticsearch *api.Elasticsearch) (*core.SecretVolumeSource, error) {
	authSecret, err := c.findAuthSecret(elasticsearch)
	if err != nil {
		return nil, err
	}
	if authSecret != nil {
		return &core.SecretVolumeSource{
			SecretName: authSecret.Name,
		}, nil
	}

	adminPassword := rand.Characters(8)
	hashedAdminPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	readallPassword := rand.Characters(8)
	hashedReadallPassword, err := bcrypt.GenerateFromPassword([]byte(readallPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	data := map[string][]byte{
		"ADMIN_PASSWORD":        []byte(adminPassword),
		"READALL_PASSWORD":      []byte(readallPassword),
		"sg_action_groups.yml":  []byte(action_group),
		"sg_config.yml":         []byte(config),
		"sg_internal_users.yml": []byte(fmt.Sprintf(internal_user, hashedAdminPassword, hashedReadallPassword)),
		"sg_roles.yml":          []byte(roles),
		"sg_roles_mapping.yml":  []byte(roles_mapping),
	}

	name := fmt.Sprintf("%v-auth", elasticsearch.OffshootName())
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				api.LabelDatabaseKind: api.ResourceKindElasticsearch,
				api.LabelDatabaseName: elasticsearch.Name,
			},
		},
		Type: core.SecretTypeOpaque,
		Data: data,
	}
	if _, err := c.Client.CoreV1().Secrets(elasticsearch.Namespace).Create(secret); err != nil {
		return nil, err
	}

	return &core.SecretVolumeSource{
		SecretName: secret.Name,
	}, nil

}

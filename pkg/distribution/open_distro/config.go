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
	certlib "kubedb.dev/elasticsearch/pkg/lib/cert"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

const (
	ConfigFileName              = "elasticsearch.yml"
	ConfigFileMountPath         = "/usr/share/elasticsearch/config"
	TempConfigFileMountPath     = "/elasticsearch/temp-config"
	DatabaseConfigMapSuffix     = "config"
	SecurityConfigFileMountPath = "/usr/share/elasticsearch/plugins/opendistro_security/securityconfig"
	InternalUserFileName        = "internal_users.yml"
)

var adminDNTemplate = `
opendistro_security.authcz.admin_dn:
- "%s"
`

var nodesDNTemplate = `
opendistro_security.nodes_dn:
- "%s"
`

var internalUserConfigFile = `
admin:
  hash: "%s"

kibanaserver:
  hash: "%s"

kibanaro:
  hash: "%s"

logstash:
  hash: "%s"

readall:
  hash: "%s"

snapshotrestore:
  hash: "%s"
`

var opendistro_security_enabled = `######## Start OpenDistro for Elasticsearch Security Configuration ########
opendistro_security.ssl.transport.pemcert_filepath: certs/node.pem
opendistro_security.ssl.transport.pemkey_filepath: certs/node-key.pem
opendistro_security.ssl.transport.pemtrustedcas_filepath: certs/root-ca.pem
opendistro_security.ssl.transport.enforce_hostname_verification: false

opendistro_security.ssl.http.enabled: ${SSL_ENABLE}
opendistro_security.ssl.http.pemcert_filepath: certs/client.pem
opendistro_security.ssl.http.pemkey_filepath: certs/client-key.pem
opendistro_security.ssl.http.pemtrustedcas_filepath: certs/root-ca.pem
opendistro_security.allow_default_init_securityindex: true

# opendistro_security.authcz.admin_dn:
%s

# opendistro_security.nodes_dn:
%s

opendistro_security.audit.type: internal_elasticsearch
opendistro_security.enable_snapshot_restore_privilege: true
opendistro_security.check_snapshot_restore_write_privileges: true
opendistro_security.restapi.roles_enabled: ["all_access", "security_rest_api_access"]
cluster.routing.allocation.disk.threshold_enabled: false
node.max_local_storage_nodes: 3
######## End OpenDistro for Elasticsearch Security Configuration ########
`

var opendistro_security_disabled = `######## Start OpenDistro for Elasticsearch Security Configuration ########
opendistro_security.disabled: true

cluster.routing.allocation.disk.threshold_enabled: false
node.max_local_storage_nodes: 3
######## End OpenDistro for Elasticsearch Security Configuration ########
`

func (es *Elasticsearch) EnsureDefaultConfig() error {
	if err := es.findDefaultConfig(); err != nil {
		return err
	}

	secretMeta := metav1.ObjectMeta{
		Name:      fmt.Sprintf("%v-%v", es.elasticsearch.OffshootName(), DatabaseConfigMapSuffix),
		Namespace: es.elasticsearch.Namespace,
	}

	// set owner reference for the secret.
	// let, elasticsearch object be the owner.
	owner := metav1.NewControllerRef(es.elasticsearch, api.SchemeGroupVersion.WithKind(api.ResourceKindElasticsearch))

	// password for default users: admin, kibanaserver
	inUserConfig, err := es.getInternalUserConfig()
	if err != nil {
		return errors.Wrap(err, "failed to generate default internal user config")
	}

	data := make(map[string][]byte)

	if !es.elasticsearch.Spec.DisableSecurity {
		if es.elasticsearch.Spec.CertificateSecret == nil {
			return errors.New("certificateSecret is empty")
		}
		certSecret, err := es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Get(context.TODO(), es.elasticsearch.Spec.CertificateSecret.SecretName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get certificateSecret:%s/%s", es.elasticsearch.Namespace, es.elasticsearch.Spec.CertificateSecret.SecretName))
		}

		// TODO: handle case for different node certs for different nodes
		nodesDN := ""
		if value, ok := certSecret.Data[certlib.NodeCert]; ok {
			subj, err := certlib.ExtractSubjectFromCertificate(value)
			if err != nil {
				return err
			}
			nodesDN = fmt.Sprintf(nodesDNTemplate, subj.String())
		}

		adminDN := ""
		if value, ok := certSecret.Data[certlib.AdminCert]; ok {
			subj, err := certlib.ExtractSubjectFromCertificate(value)
			if err != nil {
				return err
			}
			adminDN = fmt.Sprintf(adminDNTemplate, subj.String())
		}

		data[ConfigFileName] = []byte(fmt.Sprintf(opendistro_security_enabled, adminDN, nodesDN))
		data[InternalUserFileName] = []byte(inUserConfig)
	} else {
		data[ConfigFileName] = []byte(opendistro_security_disabled)
	}
	_, _, err = core_util.CreateOrPatchSecret(context.TODO(), es.kClient, secretMeta, func(in *corev1.Secret) *corev1.Secret {
		in.Labels = core_util.UpsertMap(in.Labels, es.elasticsearch.OffshootLabels())
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
		in.Data = data
		return in
	}, metav1.PatchOptions{})

	return err
}

func (es *Elasticsearch) findDefaultConfig() error {
	cmName := fmt.Sprintf("%v-%v", es.elasticsearch.OffshootName(), DatabaseConfigMapSuffix)

	configMap, err := es.kClient.CoreV1().ConfigMaps(es.elasticsearch.Namespace).Get(context.TODO(), cmName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	if configMap.Labels[api.LabelDatabaseKind] != api.ResourceKindElasticsearch &&
		configMap.Labels[api.LabelDatabaseName] != es.elasticsearch.Name {
		return fmt.Errorf(`intended configMap "%v/%v" already exists`, es.elasticsearch.Namespace, cmName)
	}

	return nil
}

func (es *Elasticsearch) getInternalUserConfig() (string, error) {
	dbSecret := es.elasticsearch.Spec.DatabaseSecret
	if dbSecret == nil {
		return "", errors.New("database secret is empty")
	}

	secret, err := es.kClient.CoreV1().Secrets(es.elasticsearch.GetNamespace()).Get(context.TODO(), dbSecret.SecretName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get database secret")
	}

	adminPH, err := generatePasswordHash("admin")
	if err != nil {
		return "", err
	}
	if value, ok := secret.Data[KeyAdminPassword]; ok {
		adminPH, err = generatePasswordHash(string(value))
		if err != nil {
			return "", err
		}
	}

	kibanaserverPH, err := generatePasswordHash("kibanaserver")
	if err != nil {
		return "", err
	}
	if value, ok := secret.Data[KeyKibanaServerPassword]; ok {
		kibanaserverPH, err = generatePasswordHash(string(value))
		if err != nil {
			return "", err
		}
	}

	kibanaroPH, err := generatePasswordHash("kibanaro")
	if err != nil {
		return "", nil
	}

	logstashPH, err := generatePasswordHash("logstash")
	if err != nil {
		return "", nil
	}

	readallPH, err := generatePasswordHash("readall")
	if err != nil {
		return "", nil
	}

	snapshotrestorePH, err := generatePasswordHash("snapshotrestore")
	if err != nil {
		return "", nil
	}

	return fmt.Sprintf(internalUserConfigFile,
		adminPH,
		kibanaserverPH,
		kibanaroPH,
		logstashPH,
		readallPH,
		snapshotrestorePH), nil
}

func generatePasswordHash(password string) (string, error) {
	pHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(pHash), nil
}

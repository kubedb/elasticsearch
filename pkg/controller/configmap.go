package controller

import (
	"fmt"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"kubedb.dev/apimachinery/apis/catalog/v1alpha1"
	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
)

const (
	ConfigFileName          = "elasticsearch.yml"
	ConfigFileMountPathSG   = "/elasticsearch/config"
	ConfigFileMountPath     = "/usr/share/elasticsearch/config"
	TempConfigFileMountPath = "/elasticsearch/temp-config"
	DatabaseConfigMapSuffix = `config`
)

var xpack_config = `
xpack.security.enabled: true

xpack.security.transport.ssl.enabled: true
xpack.security.transport.ssl.verification_mode: certificate
xpack.security.transport.ssl.keystore.path: /usr/share/elasticsearch/config/certs/node.jks
xpack.security.transport.ssl.keystore.password: ${KEY_PASS}
xpack.security.transport.ssl.truststore.path: /usr/share/elasticsearch/config/certs/root.jks
xpack.security.transport.ssl.truststore.password: ${KEY_PASS}

xpack.security.http.ssl.keystore.path: /usr/share/elasticsearch/config/certs/client.jks
xpack.security.http.ssl.keystore.password: ${KEY_PASS}
xpack.security.http.ssl.truststore.path: /usr/share/elasticsearch/config/certs/root.jks
xpack.security.http.ssl.truststore.password: ${KEY_PASS}
`

func (c *Controller) ensureDatabaseConfigForXPack(elasticsearch *api.Elasticsearch) error {
	esVersion, err := c.esVersionLister.Get(string(elasticsearch.Spec.Version))
	if err != nil {
		return err
	}
	if esVersion.Spec.AuthPlugin != v1alpha1.ElasticsearchAuthPluginXpack {
		return nil
	}
	if !elasticsearch.Spec.DisableSecurity {
		if err := c.findDatabaseConfig(elasticsearch); err != nil {
			return err
		}

		cmMeta := metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", elasticsearch.OffshootName(), DatabaseConfigMapSuffix),
			Namespace: elasticsearch.Namespace,
		}
		ref, err := reference.GetReference(clientsetscheme.Scheme, elasticsearch)
		if err != nil {
			return err
		}

		if _, _, err := core_util.CreateOrPatchConfigMap(c.Client, cmMeta, func(in *core.ConfigMap) *core.ConfigMap {
			in.Labels = core_util.UpsertMap(in.Labels, elasticsearch.OffshootLabels())
			core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
			in.Data = map[string]string{
				ConfigFileName: xpack_config,
			}
			return in
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) findDatabaseConfig(elasticsearch *api.Elasticsearch) error {
	cmName := fmt.Sprintf("%v-%v", elasticsearch.OffshootName(), DatabaseConfigMapSuffix)

	configMap, err := c.Client.CoreV1().ConfigMaps(elasticsearch.Namespace).Get(cmName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	if configMap.Labels[api.LabelDatabaseKind] != api.ResourceKindElasticsearch &&
		configMap.Labels[api.LabelDatabaseName] != elasticsearch.Name {
		return fmt.Errorf(`intended configMap "%v/%v" already exists`, elasticsearch.Namespace, cmName)
	}

	return nil
}

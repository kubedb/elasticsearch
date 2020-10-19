/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	v1alpha12 "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/elasticsearch/pkg/util/es"

	string_util "github.com/appscode/go/strings"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"sigs.k8s.io/yaml"
)

var (
	WaitLoopTimeout  = 10 * time.Minute
	WaitLoopInterval = 5 * time.Second
)

func (f *Invocation) getDataPath() string {
	return "/usr/share/elasticsearch/data"
}

func (f *Invocation) GetCommonConfig() string {
	dataPath := f.getDataPath()

	commonSetting := es.Setting{
		Path: &es.PathSetting{
			Logs: filepath.Join(dataPath, "/elasticsearch/common-logdir"),
		},
	}
	data, err := yaml.Marshal(commonSetting)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

func (f *Invocation) GetMasterConfig() string {
	dataPath := f.getDataPath()

	masterSetting := es.Setting{
		Path: &es.PathSetting{
			Data: []string{filepath.Join(dataPath, "/elasticsearch/master-datadir")},
		},
	}
	data, err := yaml.Marshal(masterSetting)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

func (f *Invocation) GetClientConfig() string {
	dataPath := f.getDataPath()
	clientSetting := es.Setting{
		Path: &es.PathSetting{
			Data: []string{filepath.Join(dataPath, "/elasticsearch/client-datadir")},
		},
	}
	data, err := yaml.Marshal(clientSetting)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

func (f *Invocation) GetDataConfig() string {
	dataPath := f.getDataPath()
	dataSetting := es.Setting{
		Path: &es.PathSetting{
			Data: []string{filepath.Join(dataPath, "/elasticsearch/data-datadir")},
		},
	}
	data, err := yaml.Marshal(dataSetting)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

func (f *Invocation) GetCustomConfig() *core.Secret {
	return &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		StringData: map[string]string{},
	}
}

func (f *Invocation) IsUsingProvidedConfig(elasticsearch *v1alpha12.Elasticsearch, nodeInfo []es.NodeInfo) bool {
	for _, node := range nodeInfo {
		if string_util.Contains(node.Roles, "master") || strings.HasSuffix(node.Name, "master") {
			masterConfig := &es.Setting{}
			err := yaml.Unmarshal([]byte(f.GetMasterConfig()), masterConfig)
			Expect(err).NotTo(HaveOccurred())

			if !string_util.EqualSlice(node.Settings.Path.Data, masterConfig.Path.Data) {
				return false
			}
		}
		if (string_util.Contains(node.Roles, "ingest") &&
			!string_util.Contains(node.Roles, "master")) ||
			strings.HasSuffix(node.Name, "ingest") { // master config has higher precedence

			ingestConfig := &es.Setting{}
			err := yaml.Unmarshal([]byte(f.GetClientConfig()), ingestConfig)
			Expect(err).NotTo(HaveOccurred())

			if !string_util.EqualSlice(node.Settings.Path.Data, ingestConfig.Path.Data) {
				return false
			}
		}
		if (string_util.Contains(node.Roles, "data") &&
			!(string_util.Contains(node.Roles, "master") || string_util.Contains(node.Roles, "ingest"))) ||
			strings.HasSuffix(node.Name, "data") { //master and ingest config has higher precedence
			dataConfig := &es.Setting{}
			err := yaml.Unmarshal([]byte(f.GetDataConfig()), dataConfig)
			Expect(err).NotTo(HaveOccurred())
			if !string_util.EqualSlice(node.Settings.Path.Data, dataConfig.Path.Data) {
				return false
			}
		}

		// check for common config
		commonConfig := &es.Setting{}
		err := yaml.Unmarshal([]byte(f.GetCommonConfig()), commonConfig)
		Expect(err).NotTo(HaveOccurred())
		if node.Settings.Path.Logs != commonConfig.Path.Logs {
			return false
		}
	}
	return true
}

func (f *Invocation) CreateConfigMap(obj *core.ConfigMap) error {
	_, err := f.kubeClient.CoreV1().ConfigMaps(obj.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
	return err
}

func (f *Invocation) DeleteConfigMap(meta metav1.ObjectMeta) error {
	err := f.kubeClient.CoreV1().ConfigMaps(meta.Namespace).Delete(context.TODO(), meta.Name, meta_util.DeleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

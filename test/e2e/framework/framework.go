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

package framework

import (
	"context"

	cs "kubedb.dev/apimachinery/client/clientset/versioned"

	"github.com/appscode/go/crypto/rand"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/portforward"
	appcat_cs "kmodules.xyz/custom-resources/client/clientset/versioned/typed/appcatalog/v1alpha1"
	scs "stash.appscode.dev/apimachinery/client/clientset/versioned"
)

var (
	DockerRegistry = "kubedbci"
	DBCatalogName  = "7.3.2"
)

type Framework struct {
	restConfig       *rest.Config
	kubeClient       kubernetes.Interface
	apiExtKubeClient crd_cs.ApiextensionsV1beta1Interface
	dbClient         cs.Interface
	kaClient         ka.Interface
	appCatalogClient appcat_cs.AppcatalogV1alpha1Interface
	Tunnel           *portforward.Tunnel
	stashClient      scs.Interface
	namespace        string
	name             string
	StorageClass     string
	topology         *core_util.Topology
}

func New(
	restConfig *rest.Config,
	kubeClient kubernetes.Interface,
	apiExtKubeClient crd_cs.ApiextensionsV1beta1Interface,
	dbClient cs.Interface,
	kaClient ka.Interface,
	appCatalogClient appcat_cs.AppcatalogV1alpha1Interface,
	stashClient scs.Interface,
	storageClass string,
) (*Framework, error) {
	topology, err := core_util.DetectTopology(context.TODO(), kubeClient)
	if err != nil {
		return nil, err
	}
	return &Framework{
		restConfig:       restConfig,
		kubeClient:       kubeClient,
		apiExtKubeClient: apiExtKubeClient,
		dbClient:         dbClient,
		kaClient:         kaClient,
		appCatalogClient: appCatalogClient,
		stashClient:      stashClient,
		name:             "elasticsearch-operator",
		namespace:        rand.WithUniqSuffix("elasticsearch"),
		StorageClass:     storageClass,
		topology:         topology,
	}, nil
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework: f,
		app:       rand.WithUniqSuffix("elasticsearch-e2e"),
	}
}

func (fi *Invocation) ExtClient() cs.Interface {
	return fi.dbClient
}

func (fi *Invocation) KubeClient() kubernetes.Interface {
	return fi.kubeClient
}

func (fi *Invocation) RestConfig() *rest.Config {
	return fi.restConfig
}

type Invocation struct {
	*Framework
	app string
}

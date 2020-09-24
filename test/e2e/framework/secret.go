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
	"fmt"
	"os"
	"time"

	"kubedb.dev/apimachinery/apis/catalog/v1alpha1"
	"kubedb.dev/apimachinery/apis/kubedb"
	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/log"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/constants/aws"
	"kmodules.xyz/constants/azure"
	"kmodules.xyz/constants/google"
	"kmodules.xyz/constants/openstack"
	"stash.appscode.dev/apimachinery/pkg/restic"
)

func (i *Invocation) SecretForLocalBackend() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-local"),
			Namespace: i.namespace,
		},
		Data: map[string][]byte{},
	}
}

func (i *Invocation) SecretForS3Backend() *corev1.Secret {
	if os.Getenv(aws.AWS_ACCESS_KEY_ID) == "" ||
		os.Getenv(aws.AWS_SECRET_ACCESS_KEY) == "" {
		return &corev1.Secret{}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-s3"),
			Namespace: i.namespace,
		},
		Data: map[string][]byte{
			aws.AWS_ACCESS_KEY_ID:     []byte(os.Getenv(aws.AWS_ACCESS_KEY_ID)),
			aws.AWS_SECRET_ACCESS_KEY: []byte(os.Getenv(aws.AWS_SECRET_ACCESS_KEY)),
		},
	}
}

func (i *Invocation) SecretForGCSBackend() *corev1.Secret {
	jsonKey := google.ServiceAccountFromEnv()
	if jsonKey == "" || os.Getenv(google.GOOGLE_PROJECT_ID) == "" {
		return &corev1.Secret{}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-gcs"),
			Namespace: i.namespace,
		},
		Data: map[string][]byte{
			google.GOOGLE_PROJECT_ID:               []byte(os.Getenv(google.GOOGLE_PROJECT_ID)),
			google.GOOGLE_SERVICE_ACCOUNT_JSON_KEY: []byte(jsonKey),
		},
	}
}

func (i *Invocation) SecretForAzureBackend() *corev1.Secret {
	if os.Getenv(azure.AZURE_ACCOUNT_NAME) == "" ||
		os.Getenv(azure.AZURE_ACCOUNT_KEY) == "" {
		return &corev1.Secret{}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-azure"),
			Namespace: i.namespace,
		},
		Data: map[string][]byte{
			azure.AZURE_ACCOUNT_NAME: []byte(os.Getenv(azure.AZURE_ACCOUNT_NAME)),
			azure.AZURE_ACCOUNT_KEY:  []byte(os.Getenv(azure.AZURE_ACCOUNT_KEY)),
		},
	}
}

func (i *Invocation) SecretForSwiftBackend() *corev1.Secret {
	if os.Getenv(openstack.OS_AUTH_URL) == "" ||
		(os.Getenv(openstack.OS_TENANT_ID) == "" && os.Getenv(openstack.OS_TENANT_NAME) == "") ||
		os.Getenv(openstack.OS_USERNAME) == "" ||
		os.Getenv(openstack.OS_PASSWORD) == "" {
		return &corev1.Secret{}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-swift"),
			Namespace: i.namespace,
		},
		Data: map[string][]byte{
			openstack.OS_AUTH_URL:    []byte(os.Getenv(openstack.OS_AUTH_URL)),
			openstack.OS_TENANT_ID:   []byte(os.Getenv(openstack.OS_TENANT_ID)),
			openstack.OS_TENANT_NAME: []byte(os.Getenv(openstack.OS_TENANT_NAME)),
			openstack.OS_USERNAME:    []byte(os.Getenv(openstack.OS_USERNAME)),
			openstack.OS_PASSWORD:    []byte(os.Getenv(openstack.OS_PASSWORD)),
			openstack.OS_REGION_NAME: []byte(os.Getenv(openstack.OS_REGION_NAME)),
		},
	}
}

func (i *Invocation) PatchSecretForRestic(secret *corev1.Secret) *corev1.Secret {
	if secret == nil {
		return secret
	}

	secret.StringData = v1.UpsertMap(secret.StringData, map[string]string{
		restic.RESTIC_PASSWORD: "RESTIC_PASSWORD",
	})

	return secret
}

// TODO: Add more methods for Swift, Backblaze B2, Rest server backend.
func (f *Framework) CreateSecret(obj *corev1.Secret) error {
	_, err := f.kubeClient.CoreV1().Secrets(obj.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
	return err
}

func (f *Framework) UpdateSecret(meta metav1.ObjectMeta, transformer func(corev1.Secret) corev1.Secret) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.kubeClient.CoreV1().Secrets(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.kubeClient.CoreV1().Secrets(cur.Namespace).Update(context.TODO(), &modified, metav1.UpdateOptions{})
			if err == nil {
				return nil
			}
		}
		log.Errorf("Attempt %d failed to update Secret %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return fmt.Errorf("failed to update Secret %s@%s after %d attempts", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) DeleteSecret(meta metav1.ObjectMeta) error {
	err := f.kubeClient.CoreV1().Secrets(meta.Namespace).Delete(context.TODO(), meta.Name, meta_util.DeleteInForeground())
	if !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyDBSecretCount(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	labelMap := map[string]string{
		api.LabelDatabaseKind: api.ResourceKindElasticsearch,
		api.LabelDatabaseName: meta.Name,
	}
	labelSelector := labels.SelectorFromSet(labelMap)

	return Eventually(
		func() int {
			secretList, err := f.kubeClient.CoreV1().Secrets(meta.Namespace).List(
				context.TODO(),
				metav1.ListOptions{
					LabelSelector: labelSelector.String(),
				},
			)
			Expect(err).NotTo(HaveOccurred())

			return len(secretList.Items)
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Framework) CheckSecret(secret *corev1.Secret) error {
	_, err := f.kubeClient.CoreV1().Secrets(f.namespace).Get(context.TODO(), secret.Name, metav1.GetOptions{})
	return err
}

func (i *Invocation) SecretForDatabaseAuthentication(es *api.Elasticsearch, mangedByKubeDB bool) *corev1.Secret {
	esVersion, err := i.dbClient.CatalogV1alpha1().ElasticsearchVersions().Get(context.TODO(), string(es.Spec.Version), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	//mangedByKubeDB mimics a secret created and manged by kubedb and not user.
	// It should get deleted during wipeout
	adminPassword := rand.Characters(8)

	var dbObjectMeta = metav1.ObjectMeta{
		Name:      fmt.Sprintf("kubedb-%v-%v", es.Name, CustomSecretSuffix),
		Namespace: es.Namespace,
	}
	if mangedByKubeDB {
		dbObjectMeta.Labels = map[string]string{
			meta_util.ManagedByLabelKey: kubedb.GroupName,
		}
	}

	var data map[string][]byte

	if esVersion.Spec.AuthPlugin == v1alpha1.ElasticsearchAuthPluginSearchGuard || esVersion.Spec.AuthPlugin == v1alpha1.ElasticsearchAuthPluginOpenDistro {
		data = map[string][]byte{
			corev1.BasicAuthUsernameKey: []byte(api.ElasticsearchInternalUserAdmin),
			corev1.BasicAuthPasswordKey: []byte(adminPassword),
		}
	} else if esVersion.Spec.AuthPlugin == v1alpha1.ElasticsearchAuthPluginXpack {
		data = map[string][]byte{
			corev1.BasicAuthUsernameKey: []byte(api.ElasticsearchInternalUserElastic),
			corev1.BasicAuthPasswordKey: []byte(adminPassword),
		}
	}

	return &corev1.Secret{
		ObjectMeta: dbObjectMeta,
		Data:       data,
	}
}

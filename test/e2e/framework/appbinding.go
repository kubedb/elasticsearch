package framework

import (
	"fmt"
	"time"

	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyAppBinding(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.appCatalogClient.AppBindings(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return false
				}
				Expect(err).NotTo(HaveOccurred())
			}
			return true
		},
		time.Minute*5,
		time.Second*5,
	)
}

func (f *Framework) CheckAppBindingSpec(meta metav1.ObjectMeta) error {
	elasticsearch, err := f.GetElasticsearch(meta)
	Expect(err).NotTo(HaveOccurred())

	appBinding, err := f.appCatalogClient.AppBindings(elasticsearch.Namespace).Get(elasticsearch.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if appBinding.Spec.ClientConfig.Service == nil ||
		appBinding.Spec.ClientConfig.Service.Name != elasticsearch.ServiceName() ||
		appBinding.Spec.ClientConfig.Service.Port != api.ElasticsearchRestPort {
		return fmt.Errorf("appbinding %v/%v contains invalid data", appBinding.Namespace, appBinding.Name)
	}
	if appBinding.Spec.Secret == nil ||
		appBinding.Spec.Secret.Name != elasticsearch.Spec.DatabaseSecret.SecretName {
		return fmt.Errorf("appbinding %v/%v contains invalid data", appBinding.Namespace, appBinding.Name)
	}
	return nil
}

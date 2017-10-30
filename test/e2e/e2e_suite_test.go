package e2e_test

import (
	"flag"
	"log"
	"path/filepath"
	"testing"
	"time"

	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	api "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	cs "github.com/k8sdb/apimachinery/client/typed/kubedb/v1alpha1"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/elasticsearch/pkg/controller"
	"github.com/k8sdb/elasticsearch/test/e2e/framework"
	"github.com/mitchellh/go-homedir"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var storageClass string

func init() {
	flag.StringVar(&storageClass, "storageclass", "", "Kubernetes StorageClass name")
}

const (
	TIMEOUT = 20 * time.Minute
)

var (
	ctrl *controller.Controller
	root *framework.Framework
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TIMEOUT)

	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "e2e Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {

	userHome, err := homedir.Dir()
	Expect(err).NotTo(HaveOccurred())

	// Kubernetes config
	kubeconfigPath := filepath.Join(userHome, ".kube/config")
	By("Using kubeconfig from " + kubeconfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	Expect(err).NotTo(HaveOccurred())
	// Clients
	kubeClient := kubernetes.NewForConfigOrDie(config)
	restClient := kubeClient.RESTClient()
	apiExtKubeClient := crd_cs.NewForConfigOrDie(config)
	extClient := cs.NewForConfigOrDie(config)
	promClient, err := pcm.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}
	// Framework
	root = framework.New(config, restClient, kubeClient, extClient, storageClass)

	By("Using namespace " + root.Namespace())

	// Create namespace
	err = root.CreateNamespace()
	Expect(err).NotTo(HaveOccurred())

	cronController := amc.NewCronController(kubeClient, extClient)
	// Start Cron
	cronController.StartCron()

	opt := controller.Options{
		ElasticDumpTag:    "5.6.3",
		OperatorNamespace: root.Namespace(),
		GoverningService:  api.DatabaseNamePrefix,
	}

	// Controller
	ctrl = controller.New(kubeClient, apiExtKubeClient, extClient, promClient, cronController, opt)
	ctrl.Run()
	root.EventuallyTPR().Should(Succeed())
})

var _ = AfterSuite(func() {
	err := root.DeleteNamespace()
	Expect(err).NotTo(HaveOccurred())
	By("Deleted namespace")
})

package cmd

import (
	"fmt"

	"github.com/appscode/go/version"
	"github.com/appscode/log"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	tcs "github.com/k8sdb/apimachinery/client/clientset"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/elasticsearch/pkg/controller"
	"github.com/spf13/cobra"
	"io/ioutil"
	cgcmd "k8s.io/client-go/tools/clientcmd"
	kapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/runtime"
	"os"
	"strings"
)

const (
	// Default tag
	canary = "canary"
)

func NewCmdRun() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
	)

	opt := &controller.Option{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Elasticsearch in Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get kubernetes config: %s", err)
			}

			// Check elasticdump docker image tag
			if err := amc.CheckDockerImageVersion(controller.ImageElasticDump, opt.ElasticDumpTag); err != nil {
				log.Fatalf(`Image %v:%v not found.`, controller.ImageElasticDump, opt.ElasticDumpTag)
			}

			client := clientset.NewForConfigOrDie(config)
			extClient := tcs.NewExtensionsForConfigOrDie(config)

			cgConfig, err := cgcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get kubernetes config: %s", err)
			}

			promClient, err := pcm.NewForConfig(cgConfig)
			if err != nil {
				log.Fatalln(err)
			}

			w := controller.New(client, extClient, promClient, opt)
			defer runtime.HandleCrash()
			fmt.Println("Starting operator...")
			w.RunAndHold()
		},
	}

	operatorVersion := version.Version.Version
	if operatorVersion == "" {
		operatorVersion = canary
	}

	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.OperatorTag, "operator", operatorVersion, "Tag of elasticsearch opearator")
	cmd.Flags().StringVar(&opt.ElasticDumpTag, "elasticdump", canary, "Tag of elasticdump")
	cmd.Flags().StringVar(&opt.GoverningService, "governing-service", "kubedb", "Governing service for database statefulset")
	cmd.Flags().StringVar(&opt.ExporterNamespace, "exporter-ns", namespace(), "Namespace for monitoring exporter")

	return cmd
}

func namespace() string {
	if ns := os.Getenv("OPERATOR_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return kapi.NamespaceDefault
}

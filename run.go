package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/appscode/go/runtime"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/log"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	tapi "github.com/k8sdb/apimachinery/api"
	tcs "github.com/k8sdb/apimachinery/client/clientset"
	"github.com/k8sdb/apimachinery/pkg/analytics"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/apimachinery/pkg/docker"
	"github.com/k8sdb/apimachinery/pkg/migrator"
	"github.com/k8sdb/elasticsearch/pkg/controller"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdRun() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
	)

	opt := controller.Options{
		ElasticDumpTag:    "canary",
		DiscoveryTag:      stringz.Val(Version, "canary"),
		OperatorNamespace: namespace(),
		ExporterTag:       "0.6.0",
		GoverningService:  "kubedb",
		Address:           ":8080",
		EnableAnalytics:   true,
		EnableRbac:        false,
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Elasticsearch in Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get kubernetes config: %s", err)
			}

			// Check elasticdump docker image tag
			if err := docker.CheckDockerImageVersion(docker.ImageElasticdump, opt.ElasticDumpTag); err != nil {
				log.Fatalf(`Image %v:%v not found.`, docker.ImageElasticdump, opt.ElasticDumpTag)
			}

			client := clientset.NewForConfigOrDie(config)
			apiExtKubeClient := apiextensionsclient.NewForConfigOrDie(config)
			extClient := tcs.NewForConfigOrDie(config)
			promClient, err := pcm.NewForConfig(config)
			if err != nil {
				log.Fatalln(err)
			}

			cronController := amc.NewCronController(client, extClient)
			// Start Cron
			cronController.StartCron()
			// Stop Cron
			defer cronController.StopCron()

			tprMigrator := migrator.NewMigrator(client, apiExtKubeClient, extClient)
			err = tprMigrator.RunMigration(
				&tapi.Elasticsearch{},
				&tapi.Snapshot{},
				&tapi.DormantDatabase{},
			)
			if err != nil {
				log.Fatalln(err)
			}

			w := controller.New(client, apiExtKubeClient, extClient, promClient, cronController, opt)
			defer runtime.HandleCrash()
			fmt.Println("Starting operator...")
			analytics.SendEvent(docker.ImageElasticOperator, "started", Version)
			w.RunAndHold()
		},
	}

	// operator flags
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.GoverningService, "governing-service", opt.GoverningService, "Governing service for database statefulset")
	cmd.Flags().StringVar(&opt.ExporterTag, "exporter-tag", opt.ExporterTag, "Tag of kubedb/operator used as exporter")
	cmd.Flags().StringVar(&opt.Address, "address", opt.Address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().BoolVar(&opt.EnableRbac, "rbac", opt.EnableRbac, "Enable RBAC for database workloads")
	// elasticdump flags
	cmd.Flags().StringVar(&opt.ElasticDumpTag, "elasticdump.tag", opt.ElasticDumpTag, "Tag of elasticdump")

	// Analytics flags
	cmd.Flags().BoolVar(&opt.EnableAnalytics, "analytics", opt.EnableAnalytics, "Send analytical event to Google Analytics")

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
	return apiv1.NamespaceDefault
}

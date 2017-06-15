package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/go/version"
	"github.com/appscode/log"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	tcs "github.com/k8sdb/apimachinery/client/clientset"
	"github.com/k8sdb/apimachinery/pkg/analytics"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/apimachinery/pkg/docker"
	"github.com/k8sdb/elasticsearch/pkg/controller"
	"github.com/spf13/cobra"
"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
apiv1 "k8s.io/client-go/pkg/api/v1"
clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
"k8s.io/kubernetes/pkg/util/runtime"
)

func NewCmdRun() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
	)

	opt := controller.Options{
		ElasticDumpTag:    "canary",
		OperatorTag:       stringz.Val(version.Version.Version, "canary"),
		OperatorNamespace: namespace(),
		GoverningService:  "kubedb",
		Address:           ":8080",
		EnableAnalytics:   true,
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
			extClient := tcs.NewForConfigOrDie(config)

			cgConfig, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get kubernetes config: %s", err)
			}

			promClient, err := pcm.NewForConfig(cgConfig)
			if err != nil {
				log.Fatalln(err)
			}

			cronController := amc.NewCronController(client, extClient)
			// Start Cron
			cronController.StartCron()
			// Stop Cron
			defer cronController.StopCron()

			w := controller.New(client, extClient, promClient, cronController, opt)
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
	cmd.Flags().StringVar(&opt.Address, "address", opt.Address, "Address to listen on for web interface and telemetry.")

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

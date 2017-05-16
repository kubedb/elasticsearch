package cmd

import (
	"fmt"
	"time"

	"github.com/appscode/go/version"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/elasticsearch/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/runtime"
)

const (
	// Default tag
	canary = "canary"
)

func NewCmdRun() *cobra.Command {
	var (
		masterURL        string
		kubeconfigPath   string
		operatorTag      string
		elasticDumpTag   string
		governingService string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Elasticsearch in Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				fmt.Printf("Could not get kubernetes config: %s", err)
				time.Sleep(30 * time.Minute)
				panic(err)
			}
			defer runtime.HandleCrash()

			// Check elasticdump docker image tag
			if err := amc.CheckDockerImageVersion(controller.ImageElasticDump, elasticDumpTag); err != nil {
				panic(fmt.Sprintf(`Image %v:%v not found.`, controller.ImageElasticDump, elasticDumpTag))
			}

			w := controller.New(config, operatorTag, elasticDumpTag, governingService)
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
	cmd.Flags().StringVar(&operatorTag, "operator", operatorVersion, "Tag of elasticsearch opearator")
	cmd.Flags().StringVar(&elasticDumpTag, "elasticdump", canary, "Tag of elasticdump")
	cmd.Flags().StringVar(&governingService, "governing-service", "k8sdb", "Governing service for database statefulset")

	return cmd
}

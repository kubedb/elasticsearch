package cmd

import (
	"fmt"
	"time"

	"github.com/appscode/go/flags"
	_ "github.com/k8sdb/elasticsearch/api/install"
	"github.com/k8sdb/elasticsearch/pkg/discover"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/runtime"
)

func NewCmdDiscover() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		namespace      string
		service        string
	)

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover Elasticsearch Endpoints",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				fmt.Printf("Could not get kubernetes config: %s", err)
				time.Sleep(30 * time.Minute)
				panic(err)
			}
			defer runtime.HandleCrash()

			flags.EnsureRequiredFlags(cmd, "service")
			discover.DiscoverEndpoints(config, service, namespace)

		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes service namespace for Elasticsearch database. Default: default")
	cmd.Flags().StringVar(&service, "service", "", "Kubernetes service for Elasticsearch database")
	return cmd
}

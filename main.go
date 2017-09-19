package main

import (
	"flag"
	"log"
	"os"

	"github.com/appscode/go/version"
	logs "github.com/appscode/log/golog"
	_ "github.com/k8sdb/apimachinery/client/scheme"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func main() {
	defer logs.FlushLogs()
	var rootCmd = &cobra.Command{
		Use: "es-operator",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})
	logs.InitLogs()

	rootCmd.AddCommand(version.NewCmdVersion())
	rootCmd.AddCommand(NewCmdRun())
	rootCmd.AddCommand(NewCmdDiscover())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

package main

import (
	"log"

	_ "github.com/kubedb/apimachinery/client/clientset/versioned/scheme"
	"github.com/kubedb/elasticsearch/pkg/cmds"
	"kmodules.xyz/client-go/logs"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	if err := cmds.NewRootCmd(Version).Execute(); err != nil {
		log.Fatal(err)
	}
}

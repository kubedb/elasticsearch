package main

import (
	"log"

	"kmodules.xyz/client-go/logs"
	_ "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	"kubedb.dev/elasticsearch/pkg/cmds"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	if err := cmds.NewRootCmd(Version).Execute(); err != nil {
		log.Fatal(err)
	}
}

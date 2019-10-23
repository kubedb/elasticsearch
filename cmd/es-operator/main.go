package main

import (
	"log"

	_ "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	"kubedb.dev/elasticsearch/pkg/cmds"

	"kmodules.xyz/client-go/logs"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	if err := cmds.NewRootCmd(Version).Execute(); err != nil {
		log.Fatal(err)
	}
}

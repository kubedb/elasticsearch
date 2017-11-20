package main

import (
	"log"

	logs "github.com/appscode/go/log/golog"
	_ "github.com/k8sdb/apimachinery/client/scheme"
	"github.com/k8sdb/elasticsearch/pkg/cmds"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	if err := cmds.NewRootCmd(Version).Execute(); err != nil {
		log.Fatal(err)
	}
}

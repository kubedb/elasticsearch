#!/usr/bin/env bash

pushd $GOPATH/src/github.com/k8sdb/elasticsearch/hack/gendocs
go run main.go
popd

#!/usr/bin/env bash

pushd $GOPATH/src/kubedb.dev/elasticsearch/hack/gendocs
go run main.go
popd

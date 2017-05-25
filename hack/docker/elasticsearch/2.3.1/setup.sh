#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/k8sdb/elasticsearch

source "$REPO_ROOT/hack/libbuild/common/kubedb_image.sh"

IMG=elasticsearch
TAG=2.3.1

clean() {
    pushd $REPO_ROOT/hack/docker/elasticsearch/2.3.1
    rm -rf lib
    popd
}

build_docker() {
    pushd $REPO_ROOT/hack/docker/elasticsearch/2.3.1
    cp -r ../lib .
	local cmd="docker build -t kubedb/$IMG:$TAG ."
	echo $cmd; $cmd
    rm -r lib
	popd
}

build() {
    build_docker
}

binary_repo $@

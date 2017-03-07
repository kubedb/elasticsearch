#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/k8sdb/elasticsearch

source "$REPO_ROOT/hack/libbuild/common/lib.sh"
source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

IMG=elasticsearch
TAG=2.3.1-v2

DIST=$GOPATH/src/github.com/k8sdb/elasticsearch/dist
mkdir -p $DIST
if [ -f "$DIST/.tag" ]; then
	export $(cat $DIST/.tag | xargs)
fi

clean() {
    pushd $REPO_ROOT/hack/docker/elasticsearch/2.3.1
    rm -f elasticsearch_discovery
    rm -rf lib
    popd
}

build_binary() {
    pushd $REPO_ROOT
    ./hack/builddeps.sh
    ./hack/make.py build elasticsearch_discovery
    detect_tag $DIST/.tag
    popd
}

build_docker() {
    pushd $REPO_ROOT/hack/docker/elasticsearch/2.3.1
    cp -r ../lib .
    cp $DIST/elasticsearch_discovery/elasticsearch_discovery-linux-amd64 elasticsearch_discovery
    chmod 755 elasticsearch_discovery
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd
	rm elasticsearch_discovery
    rm -r lib
	popd
}

build() {
    build_binary
    build_docker
}

binary_repo $@


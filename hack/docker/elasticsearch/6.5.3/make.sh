#!/bin/bash
set -xeou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/kubedb/elasticsearch"
source "$REPO_ROOT/hack/libbuild/common/kubedb_image.sh"

DOCKER_REGISTRY=${DOCKER_REGISTRY:-kubedb}
IMG=elasticsearch
DB_VERSION=6.5.3
TAG="$DB_VERSION"
YQ_VER=${YQ_VER:-2.1.1}

build() {
  pushd "$REPO_ROOT/hack/docker/elasticsearch/$DB_VERSION"

  # config merger script
  chmod +x ./config-merger.sh

  # download yq
  wget https://github.com/mikefarah/yq/releases/download/$YQ_VER/yq_linux_amd64
  chmod +x yq_linux_amd64
  mv yq_linux_amd64 yq

  local cmd="docker build --pull -t $DOCKER_REGISTRY/$IMG:$TAG ."
  echo $cmd; $cmd

  rm yq
  popd
}

binary_repo $@

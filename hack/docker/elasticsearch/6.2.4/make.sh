#!/bin/bash
set -xeou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/kubedb/elasticsearch"
source "$REPO_ROOT/hack/libbuild/common/kubedb_image.sh"

DOCKER_REGISTRY=${DOCKER_REGISTRY:-kubedb}
IMG=elasticsearch
TAG=6.2.4
YQ_VER=${YQ_VER:-2.1.1}

build() {
  pushd "$REPO_ROOT/hack/docker/elasticsearch/$TAG"

  # config merger script
  chmod +x ./config-merger.sh

  # download yq
  wget https://github.com/mikefarah/yq/releases/download/$YQ_VER/yq_linux_amd64
  chmod +x yq_linux_amd64
  mv yq_linux_amd64 yq

  local cmd="docker build -t $DOCKER_REGISTRY/$IMG:$TAG ."
  echo $cmd
  $cmd

  rm yq
  popd
}

binary_repo $@

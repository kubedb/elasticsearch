#!/bin/bash

# Copyright The KubeDB Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -xeou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/kubedb.dev/elasticsearch"
source "$REPO_ROOT/hack/libbuild/common/kubedb_image.sh"

DOCKER_REGISTRY=${DOCKER_REGISTRY:-kubedb}
IMG=elasticsearch
SUFFIX=v1
DB_VERSION=6.2.4
TAG="$DB_VERSION-$SUFFIX"
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

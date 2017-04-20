#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/k8sdb/elasticsearch"

source "$REPO_ROOT/hack/libbuild/common/lib.sh"
source "$REPO_ROOT/hack/libbuild/common/k8sdb_image.sh"

IMG=elasticdump
TAG=2.4.2

pushd "$REPO_ROOT/hack/docker/elasticdump"

binary_repo $@

popd

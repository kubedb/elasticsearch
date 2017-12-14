#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/kubedb/elasticsearch"

source "$REPO_ROOT/hack/libbuild/common/kubedb_image.sh"

IMG=elasticdump
TAG=3.3.1

pushd "$REPO_ROOT/hack/docker/elasticdump/3.3.1"

binary_repo $@

popd

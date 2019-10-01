#!/bin/bash
set -xeou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/kubedb.dev/elasticsearch"
source "$REPO_ROOT/hack/libbuild/common/kubedb_image.sh"

DOCKER_REGISTRY=${DOCKER_REGISTRY:-kubedb}
IMG=elasticsearch
DB_VERSION=6.8.0
TAG="$DB_VERSION"

docker pull "docker.elastic.co/elasticsearch/elasticsearch:$DB_VERSION"

docker tag "docker.elastic.co/elasticsearch/elasticsearch:$DB_VERSION" "$DOCKER_REGISTRY/$IMG:$TAG"
docker push "$DOCKER_REGISTRY/$IMG:$TAG"

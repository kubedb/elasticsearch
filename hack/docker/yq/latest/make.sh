#!/bin/bash
set -xeou pipefail

DOCKER_REGISTRY=${DOCKER_REGISTRY:-kubedb}
IMG=yq
TAG="latest"

# ref: https://hub.docker.com/r/mikefarah/yq
docker pull "mikefarah/$IMG:$TAG"

docker tag "mikefarah/$IMG:$TAG" "$DOCKER_REGISTRY/$IMG:$TAG"
docker push "$DOCKER_REGISTRY/$IMG:$TAG"

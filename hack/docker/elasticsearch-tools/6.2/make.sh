#!/bin/bash
set -xeou pipefail

IMG=elasticsearch-tools
TAG=6.2
PATCH=6.2.4

docker pull "$DOCKER_REGISTRY/$IMG:$PATCH"

docker tag "$DOCKER_REGISTRY/$IMG:$PATCH" "$DOCKER_REGISTRY/$IMG:$TAG"
docker push "$DOCKER_REGISTRY/$IMG:$TAG"

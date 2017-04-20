#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
SRC=$GOPATH/src
BIN=$GOPATH/bin
ROOT=$GOPATH
REPO_ROOT=$GOPATH/src/github.com/k8sdb/elasticsearch

source "$REPO_ROOT/hack/libbuild/common/lib.sh"
source "$REPO_ROOT/hack/libbuild/common/k8sdb_image.sh"

APPSCODE_ENV=${APPSCODE_ENV:-dev}
IMG=k8s-es

DIST=$GOPATH/src/github.com/k8sdb/elasticsearch/dist
mkdir -p $DIST
if [ -f "$DIST/.tag" ]; then
	export $(cat $DIST/.tag | xargs)
fi

clean() {
    pushd $REPO_ROOT/hack/docker/k8s-es
    rm -f k8s-es Dockerfile
    popd
}

build_binary() {
    pushd $REPO_ROOT
    ./hack/builddeps.sh
    ./hack/make.py build k8s-es
    detect_tag $DIST/.tag
    popd
}

build_docker() {
    pushd $REPO_ROOT/hack/docker/k8s-es
    cp $DIST/k8s-es/k8s-es-linux-amd64 k8s-es
    chmod 755 k8s-es

    cat >Dockerfile <<EOL
FROM alpine

COPY k8s-es /k8s-es

USER nobody:nobody
ENTRYPOINT ["/k8s-es"]
EOL
    local cmd="docker build -t k8sdb/$IMG:$TAG ."
    echo $cmd; $cmd

    rm k8s-es Dockerfile
    popd
}

build() {
    build_binary
    build_docker
}

docker_push() {
    if [ "$APPSCODE_ENV" = "prod" ]; then
        echo "Nothing to do in prod env. Are you trying to 'release' binaries to prod?"
        exit 0
    fi
    hub_canary k8sdb
}

docker_release() {
    if [ "$APPSCODE_ENV" != "prod" ]; then
        echo "'release' only works in PROD env."
        exit 1
    fi
    if [ "$TAG_STRATEGY" != "git_tag" ]; then
        echo "'apply_tag' to release binaries and/or docker images."
        exit 1
    fi
    hub_up k8sdb
}

source_repo $@

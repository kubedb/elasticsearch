#!/bin/bash
set -xeou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/kubedb/elasticsearch

# $REPO_ROOT/hack/docker/elasticsearch/5.6.4/make.sh build
# $REPO_ROOT/hack/docker/elasticsearch/5.6.4/make.sh push

# $REPO_ROOT/hack/docker/elasticsearch/5.6/make.sh

# $REPO_ROOT/hack/docker/elasticsearch/6.2.4/make.sh build
# $REPO_ROOT/hack/docker/elasticsearch/6.2.4/make.sh push

# $REPO_ROOT/hack/docker/elasticsearch/6.2/make.sh

$REPO_ROOT/hack/docker/elasticsearch-tools/5.6.4/make.sh build
$REPO_ROOT/hack/docker/elasticsearch-tools/5.6.4/make.sh push

$REPO_ROOT/hack/docker/elasticsearch-tools/5.6/make.sh

$REPO_ROOT/hack/docker/elasticsearch-tools/6.2.4/make.sh build
$REPO_ROOT/hack/docker/elasticsearch-tools/6.2.4/make.sh push

$REPO_ROOT/hack/docker/elasticsearch-tools/6.2/make.sh

# $REPO_ROOT/hack/docker/es-operator/make.sh build
# $REPO_ROOT/hack/docker/es-operator/make.sh push


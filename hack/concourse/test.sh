#!/bin/bash

set -eoux pipefail

set +x
DOCKER_USER=${DOCKER_USER:-}
DOCKER_PASS=${DOCKER_PASS:-}

# start docker and log-in to docker-hub
entrypoint.sh
docker login --username=$DOCKER_USER --password=$DOCKER_PASS
set -x
docker run hello-world

# install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl &> /dev/null
chmod +x ./kubectl
mv ./kubectl /bin/kubectl

# install onessl
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-amd64 \
  && chmod +x onessl \
  && mv onessl /usr/local/bin/

# install pharmer
mkdir -p $GOPATH/src/github.com/pharmer
pushd $GOPATH/src/github.com/pharmer
git clone https://github.com/pharmer/pharmer
cd pharmer
./hack/builddeps.sh
./hack/make.py
popd

function cleanup {
    set +e

    # Workload Descriptions if the test fails
    cowsay "describe deployment"
    kubectl describe deploy -n kube-system -l app=kubedb
    cowsay "describe replicaset"
    kubectl describe replicasets -n kube-system -l app=kubedb
    cowsay "describe pods"
    kubectl describe pods -n kube-system -l app=kubedb
    cowsay "describe nodes"
    kubectl get nodes
    kubectl describe nodes

    # delete operator
    pushd $GOPATH/src/github.com/kubedb/elasticsearch
    ./hack/deploy/setup.sh --uninstall --purge
    popd

    # delete cluster on exit
    pharmer get cluster
    pharmer delete cluster $NAME
    pharmer get cluster
    sleep 300
    pharmer apply $NAME
    pharmer get cluster

    # delete docker image on exit
    curl -LO https://raw.githubusercontent.com/appscodelabs/libbuild/master/docker.py
    chmod +x docker.py
    CUSTOM_OPERATOR_TAG=${CUSTOM_OPERATOR_TAG:-}
    ./docker.py del_tag kubedbci es-operator $CUSTOM_OPERATOR_TAG
}
trap cleanup EXIT

# name of the cluster
# nameing is based on repo+commit_hash
pushd elasticsearch
NAME=elasticsearch-$(git rev-parse --short HEAD)
popd

#copy elasticsearch to $GOPATH
mkdir -p $GOPATH/src/github.com/kubedb
cp -r elasticsearch $GOPATH/src/github.com/kubedb
pushd $GOPATH/src/github.com/kubedb/elasticsearch

./hack/builddeps.sh
export APPSCODE_ENV=dev
export DOCKER_REGISTRY=kubedbci
./hack/docker/es-operator/make.sh build
./hack/docker/es-operator/make.sh push
popd

#create cluster using pharmer
pharmer create credential --from-file=creds/gke.json --provider=GoogleCloud cred
pharmer create cluster $NAME --provider=gke --zone=us-central1-f --nodes=n1-standard-2=1 --credential-uid=cred --v=10 --kubernetes-version=1.10.4-gke.2
pharmer apply $NAME

#wait for cluster to be ready
sleep 300
pharmer use cluster $NAME
kubectl get nodes

# create config/.env file that have all necessary creds
cp creds/gcs.json /gcs.json
cp creds/.env $GOPATH/src/github.com/kubedb/elasticsearch/hack/config/.env

pushd $GOPATH/src/github.com/kubedb/elasticsearch

# run tests
./hack/builddeps.sh
export APPSCODE_ENV=dev
export DOCKER_REGISTRY=kubedbci
source ./hack/deploy/setup.sh --docker-registry=kubedbci

./hack/make.py test e2e --v=1 --storageclass=standard --selfhosted-operator=true

popd

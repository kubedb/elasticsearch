#!/bin/bash

set -x -e

# start docker and log-in to docker-hub
entrypoint.sh
docker login --username=$DOCKER_USER --password=$DOCKER_PASS
docker run hello-world

# install python pip
apt-get update > /dev/null
apt-get install -y python python-pip > /dev/null

# install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl &> /dev/null
chmod +x ./kubectl
mv ./kubectl /bin/kubectl

# install onessl
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-amd64 \
  && chmod +x onessl \
  && mv onessl /usr/local/bin/

# install pharmer
go get -u github.com/pharmer/pharmer

function cleanup {
    # delete cluster on exit
    pharmer get cluster || true
    pharmer delete cluster $NAME || true
    pharmer get cluster || true
    sleep 120 || true
    pharmer apply $NAME || true
    pharmer get cluster || true

    # delete docker image on exit
    curl -LO https://raw.githubusercontent.com/appscodelabs/libbuild/master/docker.py || true
    chmod +x docker.py || true
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
pharmer create credential --from-file=creds/gcs/gke.json --provider=GoogleCloud cred
pharmer create cluster $NAME --provider=gke gk10 --zone=us-central1-f --nodes=n1-standard-1=1 --credential-uid=cred --kubernetes-version=1.9.7-gke.0
pharmer apply $NAME
pharmer use cluster $NAME
#wait for cluster to be ready
sleep 300
kubectl get nodes

#create storageclass
cat > sc.yaml <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
parameters:
  zone: nyc1
provisioner: external/pharmer
EOF

#create storage-class
kubectl create -f sc.yaml
sleep 120
kubectl get storageclass

export CRED_DIR=$(pwd)/creds/gcs/gcs.json

pushd $GOPATH/src/github.com/kubedb/elasticsearch

# create config/.env file that have all necessary creds
cat > hack/config/.env <<EOF
AWS_ACCESS_KEY_ID=$AWS_KEY_ID
AWS_SECRET_ACCESS_KEY=$AWS_SECRET

GOOGLE_PROJECT_ID=$GCE_PROJECT_ID
GOOGLE_APPLICATION_CREDENTIALS=$CRED_DIR

AZURE_ACCOUNT_NAME=$AZURE_ACCOUNT_NAME
AZURE_ACCOUNT_KEY=$AZURE_ACCOUNT_KEY

OS_AUTH_URL=$OS_AUTH_URL
OS_TENANT_ID=$OS_TENANT_ID
OS_TENANT_NAME=$OS_TENANT_NAME
OS_USERNAME=$OS_USERNAME
OS_PASSWORD=$OS_PASSWORD
OS_REGION_NAME=$OS_REGION_NAME

S3_BUCKET_NAME=$S3_BUCKET_NAME
GCS_BUCKET_NAME=$GCS_BUCKET_NAME
AZURE_CONTAINER_NAME=$AZURE_CONTAINER_NAME
SWIFT_CONTAINER_NAME=$SWIFT_CONTAINER_NAME
EOF


# run tests
./hack/builddeps.sh
export APPSCODE_ENV=dev
export DOCKER_REGISTRY=kubedbci
./hack/docker/es-operator/make.sh build
./hack/docker/es-operator/make.sh push
source ./hack/deploy/setup.sh --docker-registry=kubedbci
./hack/make.py test e2e --v=1 --storageclass=standard --selfhosted-operator=true

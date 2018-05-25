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
pushd /tmp
curl -LO https://cdn.appscode.com/binaries/pharmer/0.1.0-rc.4/pharmer-linux-amd64
chmod +x pharmer-linux-amd64
mv pharmer-linux-amd64 /bin/pharmer
popd

function cleanup {
    # Workload Descriptions if the test fails
    if [ $? -ne 0 ]; then
        echo ""
        kubectl describe deploy -n kube-system -l app=kubedb || true
        echo ""
        echo ""
        kubectl describe replicasets -n kube-system -l app=kubedb || true
        echo ""
        echo ""
        kubectl describe pods -n kube-system -l app=kubedb || true
    fi

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
    ./docker.py del_tag kubedbci es-operator $CUSTOM_OPERATOR_TAG || true
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
pharmer create cluster $NAME --provider=gke --zone=us-central1-f --nodes=n1-standard-2=1 --credential-uid=cred --v=10 --kubernetes-version=1.10.2-gke.1
pharmer apply $NAME

# gcloud-sdk
pushd /tmp
curl -LO https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-202.0.0-linux-x86_64.tar.gz
tar --extract --file google-cloud-sdk-202.0.0-linux-x86_64.tar.gz
CLOUDSDK_CORE_DISABLE_PROMPTS=1 ./google-cloud-sdk/install.sh
source /tmp/google-cloud-sdk/path.bash.inc
popd
gcloud auth activate-service-account --key-file creds/gcs/gke.json
gcloud container clusters get-credentials $NAME --zone us-central1-f --project k8s-qa
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=k8s-qa@k8s-qa.iam.gserviceaccount.com

#wait for cluster to be ready
sleep 300
kubectl get nodes

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
source ./hack/deploy/setup.sh --docker-registry=kubedbci
./hack/make.py test e2e --v=1 --storageclass=standard --selfhosted-operator=true

kubectl describe pods -n kube-system -l app=kubedb || true


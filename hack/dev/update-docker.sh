#!/bin/bash

# Copyright AppsCode Inc. and Contributors
#
# Licensed under the AppsCode Community License 1.0.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.




set -eou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=${GOPATH}/src/kubedb.dev/elasticsearch

export TOOLS_UPDATE=1
export EXPORTER_UPDATE=1
export KIBANA_UPDATE=1
export YQ_UPDATE=1

show_help() {
  echo "update-docker.sh [options]"
  echo " "
  echo "options:"
  echo "-h, --help                       show brief help"
  echo "    --db-only                    update only database images"
  echo "    --tools-only                 update only database-tools images"
  echo "    --exporter-only              update only database-exporter images"
  echo "    --kibana-only                update only kibana images"
  echo "    --yq-only                update only kibana images"
}

while test $# -gt 0; do
  case "$1" in
    -h | --help)
      show_help
      exit 0
      ;;
    --tools-only)
      export TOOLS_UPDATE=1
      export EXPORTER_UPDATE=0
      export KIBANA_UPDATE=0
      export YQ_UPDATE=0
      shift
      ;;
    --exporter-only)
      export TOOLS_UPDATE=0
      export EXPORTER_UPDATE=1
      export KIBANA_UPDATE=0
      export YQ_UPDATE=0
      shift
      ;;
    --kibana-only)
      export TOOLS_UPDATE=0
      export EXPORTER_UPDATE=0
      export KIBANA_UPDATE=1
      export YQ_UPDATE=0
      shift
      ;;
    --yq-only)
      export TOOLS_UPDATE=0
      export EXPORTER_UPDATE=0
      export KIBANA_UPDATE=0
      export YQ_UPDATE=1
      shift
      ;;
    *)
      show_help
      exit 1
      ;;
  esac
done

dbversions=(
#  5.6.4
#  5.6
#  6.2.4
#  6.2
#  6.3.0
#  6.3
#  6.4.0
#  6.4
#  6.5.3
#  6.5
#  6.8.0-sg
#  7.2.0-sg
  6.8.0
  6.8
  7.2.0
  7.2
  7.3.2
  7.3
)

exporters=(
  1.0.2
)

kibanaimages=(
#  6.3.0
#  6.5.3
  6.8.0
  7.2.0
  7.3.2
)

yqimages=(
  2.4.0
  latest
)

echo ""
env | sort | grep -e DOCKER_REGISTRY -e APPSCODE_ENV || true
echo ""

if [ "$TOOLS_UPDATE" -eq 1 ]; then
  cowsay -f tux "Processing database-tools images" || echo "Processing database-tools images"
  for db in "${dbversions[@]}"; do
    ${REPO_ROOT}/hack/docker/elasticsearch-tools/${db}/make.sh build
    ${REPO_ROOT}/hack/docker/elasticsearch-tools/${db}/make.sh push
  done
fi

if [ "$EXPORTER_UPDATE" -eq 1 ]; then
  cowsay -f tux "Processing database-exporter images" || echo "Processing database-exporter images"
  for exporter in "${exporters[@]}"; do
    ${REPO_ROOT}/hack/docker/elasticsearch_exporter/${exporter}/make.sh
  done
fi

if [ "$KIBANA_UPDATE" -eq 1 ]; then
  cowsay -f tux "Processing Kibana images" || echo "Processing Kibana images"
  for kibana in "${kibanaimages[@]}"; do
    ${REPO_ROOT}/hack/docker/kibana/${kibana}/make.sh build
    ${REPO_ROOT}/hack/docker/kibana/${kibana}/make.sh push
  done
fi

if [ "$YQ_UPDATE" -eq 1 ]; then
  cowsay -f tux "Processing YQ images" || echo "Processing YQ images"
  for yq in "${yqimages[@]}"; do
    ${REPO_ROOT}/hack/docker/yq/${yq}/make.sh
  done
fi

#!/bin/bash

# Copyright The KubeDB Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -x
set -o errexit
set -o pipefail

sync

# if custom config file exist then process them
CUSTOM_CONFIG_DIR="/elasticsearch/custom-config"

if [ -d "$CUSTOM_CONFIG_DIR" ]; then

  configs=($(find $CUSTOM_CONFIG_DIR -maxdepth 1 -name "*.yaml"))
  configs+=($(find $CUSTOM_CONFIG_DIR -maxdepth 1 -name "*.yml"))
  if [ ${#configs[@]} -gt 0 ]; then
    config-merger.sh
  fi
fi

echo "Starting runit..."
exec /sbin/runsvdir -P /etc/service

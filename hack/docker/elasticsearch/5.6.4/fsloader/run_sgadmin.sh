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


searchguard="/elasticsearch/plugins/search-guard-5"
certs="/elasticsearch/config/certs"

sync

SERVER='http://localhost:9200'

if [ "$SSL_ENABLE" == true ]; then
  SERVER='https://localhost:9200'
fi

until curl -s "$SERVER" --insecure; do
  sleep 0.1
done

"$searchguard"/tools/sgadmin.sh \
  -ks "$certs"/sgadmin.jks \
  -kspass "$KEY_PASS" \
  -ts "$certs"/root.jks \
  -tspass "$KEY_PASS" \
  -cd "$searchguard"/sgconfig -icl -nhnv

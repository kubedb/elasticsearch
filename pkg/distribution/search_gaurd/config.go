/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package search_gaurd

const (
	ConfigFileName          = "elasticsearch.yml"
	ConfigFileMountPath     = "/usr/share/elasticsearch/config"
	TempConfigFileMountPath = "/elasticsearch/temp-config"
	DatabaseConfigMapSuffix = "config"
)

var action_group = `
UNLIMITED:
  - "*"

READ:
  - "indices:data/read*"
  - "indices:admin/mappings/fields/get*"

CLUSTER_COMPOSITE_OPS_RO:
  - "indices:data/read/mget"
  - "indices:data/read/msearch"
  - "indices:data/read/mtv"
  - "indices:data/read/coordinate-msearch*"
  - "indices:admin/aliases/exists*"
  - "indices:admin/aliases/get*"

CLUSTER_KUBEDB_SNAPSHOT:
  - "indices:data/read/scroll*"
  - "cluster:monitor/main"

INDICES_KUBEDB_SNAPSHOT:
  - "indices:admin/get"
  - "indices:monitor/settings/get"
  - "indices:admin/mappings/get"
`

var action_group_es7 = `
_sg_meta:
  type: "actiongroups"
  config_version: 2

UNLIMITED:
  allowed_actions:
    - "*"

READ:
  allowed_actions:
    - "indices:data/read*"
    - "indices:admin/mappings/fields/get*"

CLUSTER_COMPOSITE_OPS_RO:
  allowed_actions:
    - "indices:data/read/mget"
    - "indices:data/read/msearch"
    - "indices:data/read/mtv"
    - "indices:data/read/coordinate-msearch*"
    - "indices:admin/aliases/exists*"
    - "indices:admin/aliases/get*"

CLUSTER_KUBEDB_SNAPSHOT:
  allowed_actions:
    - "indices:data/read/scroll*"
    - "cluster:monitor/main"

INDICES_KUBEDB_SNAPSHOT:
  allowed_actions:
    - "indices:admin/get"
    - "indices:monitor/settings/get"
    - "indices:admin/mappings/get"
`

var config = `
searchguard:
  dynamic:
    authc:
      basic_internal_auth_domain:
        enabled: true
        order: 4
        http_authenticator:
          type: basic
          challenge: true
        authentication_backend:
          type: internal
`

var config_es7 = `
_sg_meta:
  type: "config"
  config_version: 2
sg_config:
  dynamic:
    authc:
      basic_internal_auth_domain:
        http_enabled: true
        transport_enabled: true
        order: 4
        http_authenticator:
          type: basic
          challenge: true
        authentication_backend:
          type: internal
`

var internal_user = `
admin:
  hash: %s

readall:
  hash: %s
`

var internal_user_es7 = `
_sg_meta:
  type: "internalusers"
  config_version: 2

admin:
  hash: %s

readall:
  hash: %s
`

var roles = `
sg_all_access:
  cluster:
    - UNLIMITED
  indices:
    '*':
      '*':
        - UNLIMITED
  tenants:
    adm_tenant: RW
    test_tenant_ro: RW

sg_readall:
  cluster:
    - CLUSTER_COMPOSITE_OPS_RO
    - CLUSTER_KUBEDB_SNAPSHOT
  indices:
    '*':
      '*':
        - READ
        - INDICES_KUBEDB_SNAPSHOT
`

var roles_es7 = `
_sg_meta:
  type: "roles"
  config_version: 2
sg_all_access:
  cluster_permissions:
  - UNLIMITED
  index_permissions:
  - index_patterns:
    - "*"
    allowed_actions:
    - "UNLIMITED"
  tenant_permissions:
  - tenant_patterns:
    - adm_tenant
    - test_tenant_ro
    allowed_actions:
    - SGS_KIBANA_ALL_WRITE
sg_readall:
  cluster_permissions:
  - "CLUSTER_COMPOSITE_OPS_RO"
  - "CLUSTER_KUBEDB_SNAPSHOT"
  index_permissions:
  - index_patterns:
    - "*"
    allowed_actions:
    - "READ"
    - "INDICES_KUBEDB_SNAPSHOT"
  tenant_permissions: []
`

var roles_mapping = `
sg_all_access:
  users:
    - admin

sg_readall:
  users:
    - readall
`

var roles_mapping_es7 = `
_sg_meta:
  type: "rolesmapping"
  config_version: 2

sg_all_access:
  users:
    - admin

sg_readall:
  users:
    - readall
`

var tenants = `
_sg_meta:
  type: "tenants"
  config_version: 2
test_tenant_ro:
  reserved: false
  hidden: false
  description: "test_tenant_ro. Migrated from v6"
  static: false
adm_tenant:
  reserved: false
  hidden: false
  description: "adm_tenant. Migrated from v6"
  static: false
`

func (es *Elasticsearch) EnsureDefaultConfig() error {
	return nil
}

/*
Copyright The KubeDB Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package framework

import (
	"github.com/appscode/go/crypto/rand"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	podsecuritypolicies = "podsecuritypolicies"
	rbacApiGroup        = "rbac.authorization.k8s.io"
	GET                 = "get"
	LIST                = "list"
	PATCH               = "patch"
	CREATE              = "create"
	UPDATE              = "update"
	USE                 = "use"
	POLICY              = "policy"
	Role                = "Role"
	ServiceAccount      = "ServiceAccount"
)

var (
	CustomSecretSuffix = "custom-secret"
	CustomUsername     = "username1234567890"
	CustomPassword     = "password0987654321"
	AdminUser          = "admin"
	KeyAdminUserName   = "ADMIN_USERNAME"
	KeyAdminPassword   = "ADMIN_PASSWORD"
	ReadAllUser        = "readall"
	KeyReadAllUserName = "READALL_USERNAME"
	KeyReadAllPassword = "READALL_PASSWORD"
	ExporterSecretPath = "/var/run/secrets/kubedb.com/"
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

var internal_user = `
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

var roles_mapping = `
sg_all_access:
  users:
    - admin

sg_readall:
  users:
    - readall
`

func (i *Invocation) ServiceAccount() *core.ServiceAccount {
	return &core.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-es"),
			Namespace: i.namespace,
		},
	}
}

func (i *Invocation) RoleForElasticsearch(meta metav1.ObjectMeta) *rbac.Role {
	return &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-es"),
			Namespace: i.namespace,
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{
					POLICY,
				},
				ResourceNames: []string{
					meta.Name,
				},
				Resources: []string{
					podsecuritypolicies,
				},
				Verbs: []string{
					USE,
				},
			},
		},
	}
}

func (i *Invocation) RoleForSnapshot(meta metav1.ObjectMeta) *rbac.Role {
	return &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-es"),
			Namespace: i.namespace,
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{
					POLICY,
				},
				ResourceNames: []string{
					meta.Name,
				},
				Resources: []string{
					podsecuritypolicies,
				},
				Verbs: []string{
					USE,
				},
			},
		},
	}
}

func (i *Invocation) RoleBinding(saName string, roleName string) *rbac.RoleBinding {
	return &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(i.app + "-es"),
			Namespace: i.namespace,
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbacApiGroup,
			Kind:     Role,
			Name:     roleName,
		},
		Subjects: []rbac.Subject{
			{
				Kind:      ServiceAccount,
				Namespace: i.namespace,
				Name:      saName,
			},
		},
	}
}

func (f *Framework) CreateServiceAccount(obj *core.ServiceAccount) error {
	_, err := f.kubeClient.CoreV1().ServiceAccounts(obj.Namespace).Create(obj)
	return err
}

func (f *Framework) CreateRole(obj *rbac.Role) error {
	_, err := f.kubeClient.RbacV1().Roles(obj.Namespace).Create(obj)
	return err
}

func (f *Framework) CreateRoleBinding(obj *rbac.RoleBinding) error {
	_, err := f.kubeClient.RbacV1().RoleBindings(obj.Namespace).Create(obj)
	return err
}

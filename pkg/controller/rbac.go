package controller

import (
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	core "k8s.io/api/core/v1"
	policy_v1beta1 "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	policy_util "kmodules.xyz/client-go/policy/v1beta1"
	rbac_util "kmodules.xyz/client-go/rbac/v1beta1"
)

func (c *Controller) ensureRole(elasticsearch *api.Elasticsearch, name string) error {
	ref, rerr := reference.GetReference(clientsetscheme.Scheme, elasticsearch)
	if rerr != nil {
		return rerr
	}

	// Create new Roles
	_, _, err := rbac_util.CreateOrPatchRole(
		c.Client,
		metav1.ObjectMeta{
			Name:      name,
			Namespace: elasticsearch.Namespace,
		},
		func(in *rbac.Role) *rbac.Role {
			core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
			in.Rules = []rbac.PolicyRule{
				{
					APIGroups:     []string{policy_v1beta1.GroupName},
					Resources:     []string{"podsecuritypolicies"},
					Verbs:         []string{"use"},
					ResourceNames: []string{name},
				},
			}
			return in
		},
	)
	return err
}

func (c *Controller) createRoleBinding(elasticsearch *api.Elasticsearch, name string) error {
	ref, rerr := reference.GetReference(clientsetscheme.Scheme, elasticsearch)
	if rerr != nil {
		return rerr
	}
	// Ensure new RoleBindings
	_, _, err := rbac_util.CreateOrPatchRoleBinding(
		c.Client,
		metav1.ObjectMeta{
			Name:      name,
			Namespace: elasticsearch.Namespace,
		},
		func(in *rbac.RoleBinding) *rbac.RoleBinding {
			core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
			in.RoleRef = rbac.RoleRef{
				APIGroup: rbac.GroupName,
				Kind:     "Role",
				Name:     name,
			}
			in.Subjects = []rbac.Subject{
				{
					Kind:      rbac.ServiceAccountKind,
					Name:      name,
					Namespace: elasticsearch.Namespace,
				},
			}
			return in
		},
	)
	return err
}

func (c *Controller) ensurePSP(elasticsearch *api.Elasticsearch) error {
	ref, rerr := reference.GetReference(clientsetscheme.Scheme, elasticsearch)
	if rerr != nil {
		return rerr
	}

	// Ensure Pod Security policy for Elasticsearch resources
	escalate := true
	_, _, err := policy_util.CreateOrPatchPodSecurityPolicy(c.Client,
		metav1.ObjectMeta{
			Name: elasticsearch.OffshootName(),
		},
		func(in *policy_v1beta1.PodSecurityPolicy) *policy_v1beta1.PodSecurityPolicy {
			//TODO: possible function EnsureOwnerReference(&psp.ObjectMeta, ref) in kmodules/client-go for non namespaced resources.
			in.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: ref.APIVersion,
					Kind:       ref.Kind,
					Name:       ref.Name,
					UID:        ref.UID,
				},
			}
			in.Spec = policy_v1beta1.PodSecurityPolicySpec{
				Privileged:               true,
				AllowPrivilegeEscalation: &escalate,
				Volumes: []policy_v1beta1.FSType{
					policy_v1beta1.All,
				},
				HostIPC:     false,
				HostNetwork: false,
				HostPID:     false,
				RunAsUser: policy_v1beta1.RunAsUserStrategyOptions{
					Rule: policy_v1beta1.RunAsUserStrategyRunAsAny,
				},
				SELinux: policy_v1beta1.SELinuxStrategyOptions{
					Rule: policy_v1beta1.SELinuxStrategyRunAsAny,
				},
				FSGroup: policy_v1beta1.FSGroupStrategyOptions{
					Rule: policy_v1beta1.FSGroupStrategyRunAsAny,
				},
				SupplementalGroups: policy_v1beta1.SupplementalGroupsStrategyOptions{
					Rule: policy_v1beta1.SupplementalGroupsStrategyRunAsAny,
				},
				AllowedCapabilities: []core.Capability{
					"IPC_LOCK",
					"SYS_RESOURCE",
				},
			}
			return in
		},
	)
	return err
}

func (c *Controller) ensureSnapshotPSP(elasticsearch *api.Elasticsearch) error {
	ref, rerr := reference.GetReference(clientsetscheme.Scheme, elasticsearch)
	if rerr != nil {
		return rerr
	}

	// Ensure Pod Security policy for Elasticsearch DB Snapshot
	noEscalation := false
	_, _, err := policy_util.CreateOrPatchPodSecurityPolicy(c.Client,
		metav1.ObjectMeta{
			Name: elasticsearch.SnapshotSAName(),
		},
		func(in *policy_v1beta1.PodSecurityPolicy) *policy_v1beta1.PodSecurityPolicy {
			//TODO: possible function EnsureOwnerReference(&psp.ObjectMeta, ref) in kutil for non namespaced resources.
			in.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: ref.APIVersion,
					Kind:       ref.Kind,
					Name:       ref.Name,
					UID:        ref.UID,
				},
			}
			in.Spec = policy_v1beta1.PodSecurityPolicySpec{
				Privileged:               false,
				AllowPrivilegeEscalation: &noEscalation,
				Volumes: []policy_v1beta1.FSType{
					policy_v1beta1.All,
				},
				HostIPC:     false,
				HostNetwork: false,
				HostPID:     false,
				RunAsUser: policy_v1beta1.RunAsUserStrategyOptions{
					Rule: policy_v1beta1.RunAsUserStrategyRunAsAny,
				},
				SELinux: policy_v1beta1.SELinuxStrategyOptions{
					Rule: policy_v1beta1.SELinuxStrategyRunAsAny,
				},
				FSGroup: policy_v1beta1.FSGroupStrategyOptions{
					Rule: policy_v1beta1.FSGroupStrategyRunAsAny,
				},
				SupplementalGroups: policy_v1beta1.SupplementalGroupsStrategyOptions{
					Rule: policy_v1beta1.SupplementalGroupsStrategyRunAsAny,
				},
			}
			return in
		},
	)
	return err
}

func (c *Controller) createServiceAccount(elasticsearch *api.Elasticsearch, saName string) error {
	ref, rerr := reference.GetReference(clientsetscheme.Scheme, elasticsearch)
	if rerr != nil {
		return rerr
	}
	// Create new ServiceAccount
	_, _, err := core_util.CreateOrPatchServiceAccount(
		c.Client,
		metav1.ObjectMeta{
			Name:      saName,
			Namespace: elasticsearch.Namespace,
		},
		func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
			return in
		},
	)
	return err
}

func (c *Controller) ensureRBACStuff(elasticsearch *api.Elasticsearch) error {
	//Create PSP
	if err := c.ensurePSP(elasticsearch); err != nil {
		return err
	}

	// Create New Role
	if err := c.ensureRole(elasticsearch, elasticsearch.OffshootName()); err != nil {
		return err
	}

	// Create New RoleBinding
	if err := c.createRoleBinding(elasticsearch, elasticsearch.OffshootName()); err != nil {
		return err
	}
	// Create New ServiceAccount
	if err := c.createServiceAccount(elasticsearch, elasticsearch.OffshootName()); err != nil {
		if !kerr.IsAlreadyExists(err) {
			return err
		}
	}

	//Create PSP for Snapshot
	if err := c.ensureSnapshotPSP(elasticsearch); err != nil {
		return err
	}

	// Create New Role for Snapshot
	if err := c.ensureRole(elasticsearch, elasticsearch.SnapshotSAName()); err != nil {
		return err
	}

	// Create New RoleBinding for Snapshot
	if err := c.createRoleBinding(elasticsearch, elasticsearch.SnapshotSAName()); err != nil {
		return err
	}

	// Create New Snapshot ServiceAccount
	if err := c.createServiceAccount(elasticsearch, elasticsearch.SnapshotSAName()); err != nil {
		if !kerr.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

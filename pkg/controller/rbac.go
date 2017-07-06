package controller

import (
	tapi "github.com/k8sdb/apimachinery/api"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
)

func (c *Controller) deleteRole(name, namespace string) error {
	// Delete existing Roles
	if err := c.Client.RbacV1beta1().Roles(namespace).Delete(name, nil); err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *Controller) createRole(elastic *tapi.Elastic) error {
	// Create new Roles
	role := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      elastic.Name,
			Namespace: elastic.Namespace,
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups:     []string{tapi.GroupName},
				Resources:     []string{tapi.ResourceTypeElastic},
				ResourceNames: []string{elastic.Name},
				Verbs:         []string{"get"},
			},
		},
	}
	if _, err := c.Client.RbacV1beta1().Roles(role.Namespace).Create(role); err != nil {
		return err
	}

	return nil
}

func (c *Controller) deleteServiceAccount(name, namespace string) error {
	// Delete existing ServiceAccount
	if err := c.Client.CoreV1().ServiceAccounts(namespace).Delete(name, nil); err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *Controller) createServiceAccount(elastic *tapi.Elastic) error {
	// Create new ServiceAccount
	sa := &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      elastic.Name,
			Namespace: elastic.Namespace,
		},
	}
	if _, err := c.Client.CoreV1().ServiceAccounts(sa.Namespace).Create(sa); err != nil {
		return err
	}

	return nil
}

func (c *Controller) deleteRoleBinding(name, namespace string) error {
	// Delete existing RoleBindings
	if err := c.Client.RbacV1beta1().RoleBindings(namespace).Delete(name, nil); err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *Controller) createRoleBinding(elastic *tapi.Elastic) error {
	// Create new RoleBindings
	roleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      elastic.Name,
			Namespace: elastic.Namespace,
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     elastic.Name,
		},
		Subjects: []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      elastic.Name,
				Namespace: elastic.Namespace,
			},
		},
	}
	if _, err := c.Client.RbacV1beta1().RoleBindings(roleBinding.Namespace).Create(roleBinding); err != nil {
		return err
	}

	return nil
}

func (c *Controller) createRBACStuff(elastic *tapi.Elastic) error {
	// Delete Existing Role
	if err := c.deleteRole(elastic.Name, elastic.Namespace); err != nil {
		return err
	}
	// Create New Role
	if err := c.createRole(elastic); err != nil {
		return err
	}

	// Create New ServiceAccount
	if err := c.createServiceAccount(elastic); err != nil {
		if !kerr.IsAlreadyExists(err) {
			return err
		}
	}

	// Create New RoleBinding
	if err := c.createRoleBinding(elastic); err != nil {
		if !kerr.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func (c *Controller) deleteRBACStuff(name, namespace string) error {
	// Delete Existing Role
	if err := c.deleteRole(name, namespace); err != nil {
		return err
	}

	// Delete ServiceAccount
	if err := c.deleteServiceAccount(name, namespace); err != nil {
		return err
	}

	// Delete New RoleBinding
	if err := c.deleteRoleBinding(name, namespace); err != nil {
		return err
	}

	return nil
}

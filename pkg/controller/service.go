package controller

import (
	"fmt"

	tapi "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/k8sdb/apimachinery/pkg/eventer"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	NodeRoleMaster = "node.role.master"
	NodeRoleClient = "node.role.client"
	NodeRoleData   = "node.role.data"
)

func (c *Controller) ensureService(elasticsearch *tapi.Elasticsearch) error {
	name := elasticsearch.OffshootName()
	// Check if service name exists
	found, err := c.findService(elasticsearch, name)
	if err != nil {
		return err
	}
	if !found {
		// create database Service
		if err := c.createService(elasticsearch); err != nil {
			c.recorder.Eventf(
				elasticsearch.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				"Failed to create Service. Reason: %v",
				err,
			)
			return err
		}
	}

	discoveryService := fmt.Sprintf("%s-discovery", name)
	found, err = c.findService(elasticsearch, discoveryService)
	if err != nil {
		return err
	}
	if !found {
		// create database Discovery Service
		if err := c.createDiscoveryService(elasticsearch); err != nil {
			c.recorder.Eventf(
				elasticsearch.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				"Failed to create Discovery Service. Reason: %v",
				err,
			)
			return err
		}
	}
	return nil
}

func (c *Controller) findService(elasticsearch *tapi.Elasticsearch, name string) (bool, error) {
	elasticsearchName := elasticsearch.OffshootName()

	service, err := c.Client.CoreV1().Services(elasticsearch.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	if service.Spec.Selector[tapi.LabelDatabaseName] != elasticsearchName {
		return false, fmt.Errorf(`intended service "%v" already exists`, name)
	}

	return true, nil
}

func (c *Controller) createService(elastic *tapi.Elasticsearch) error {
	svc := &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   elastic.Name,
			Labels: elastic.OffshootLabels(),
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "http",
					Port:       9200,
					TargetPort: intstr.FromString("http"),
				},
			},
			Selector: elastic.OffshootLabels(),
		},
	}
	svc.Spec.Selector[NodeRoleClient] = "set"

	if elastic.Spec.Monitor != nil &&
		elastic.Spec.Monitor.Agent == tapi.AgentCoreosPrometheus &&
		elastic.Spec.Monitor.Prometheus != nil {
		svc.Spec.Ports = append(svc.Spec.Ports, core.ServicePort{
			Name:       tapi.PrometheusExporterPortName,
			Port:       tapi.PrometheusExporterPortNumber,
			TargetPort: intstr.FromString(tapi.PrometheusExporterPortName),
		})
	}

	if _, err := c.Client.CoreV1().Services(elastic.Namespace).Create(svc); err != nil {
		return err
	}

	return nil
}

func (c *Controller) createDiscoveryService(elasticsearch *tapi.Elasticsearch) error {
	// TODO
	serviceName := fmt.Sprintf("%v-discovery", elasticsearch.Name)
	svc := &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName,
			Labels: elasticsearch.OffshootLabels(),
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "transport",
					Port:       9300,
					TargetPort: intstr.FromString("transport"),
				},
			},
			Selector: elasticsearch.OffshootLabels(),
		},
	}
	svc.Spec.Selector[NodeRoleMaster] = "set"

	if _, err := c.Client.CoreV1().Services(elasticsearch.Namespace).Create(svc); err != nil {
		return err
	}

	return nil
}

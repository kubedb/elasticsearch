package controller

import (
	"fmt"

	tapi "github.com/k8sdb/apimachinery/api"
	"github.com/k8sdb/apimachinery/pkg/monitor"
)

const ImageExporter = "kubedb/exporter"

func (c *Controller) newMonitorController(elastic *tapi.Elastic) (monitor.Monitor, error) {
	monitorSpec := elastic.Spec.Monitor

	if monitorSpec == nil {
		return nil, fmt.Errorf("MonitorSpec not found in %v", elastic.Spec)
	}

	if monitorSpec.Prometheus != nil {
		image := fmt.Sprintf("%v:%v", ImageExporter, c.option.ExporterTag)
		return monitor.NewPrometheusController(c.Client, c.promClient, c.option.ExporterNamespace, image), nil
	}

	return nil, fmt.Errorf("Monitoring controller not found for %v", monitorSpec)
}

func (c *Controller) addMonitor(elastic *tapi.Elastic) error {
	monitor, err := c.newMonitorController(elastic)
	if err != nil {
		return err
	}
	return monitor.AddMonitor(elastic.ObjectMeta, elastic.Spec.Monitor)
}

func (c *Controller) deleteMonitor(elastic *tapi.Elastic) error {
	m, err := c.newMonitorController(elastic)
	if err != nil {
		return err
	}
	return m.DeleteMonitor(elastic.ObjectMeta, elastic.Spec.Monitor)
}

func (c *Controller) updateMonitor(oldElastic, updatedElastic *tapi.Elastic) error {
	var err error
	var monitor monitor.Monitor
	if updatedElastic.Spec.Monitor == nil {
		monitor, err = c.newMonitorController(oldElastic)
	} else {
		monitor, err = c.newMonitorController(updatedElastic)
	}
	if err != nil {
		return err
	}
	return monitor.UpdateMonitor(updatedElastic.ObjectMeta, oldElastic.Spec.Monitor, updatedElastic.Spec.Monitor)
}

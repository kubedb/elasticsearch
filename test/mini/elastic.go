package mini

import (
	"fmt"
	"time"

	"errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/elasticsearch/pkg/controller"
	kapi "k8s.io/kubernetes/pkg/api"
)

const durationCheckElastic = time.Minute * 30

func NewElastic() *tapi.Elastic {
	elastic := &tapi.Elastic{
		ObjectMeta: kapi.ObjectMeta{
			Name:      rand.WithUniqSuffix("e2e-elastic"),
		},
		Spec: tapi.ElasticSpec{
			Version: "canary",
			Replicas: 1,
		},
	}
	return elastic
}

func ReCreateElastic(c *controller.Controller, elastic *tapi.Elastic) (*tapi.Elastic, error) {
	_elastic := &tapi.Elastic{
		ObjectMeta: kapi.ObjectMeta{
			Name:        elastic.Name,
			Namespace:   elastic.Namespace,
			Labels:      elastic.Labels,
			Annotations: elastic.Annotations,
		},
		Spec:   elastic.Spec,
		Status: elastic.Status,
	}

	return c.ExtClient.Elastics(_elastic.Namespace).Create(_elastic)
}

func CheckElasticStatus(c *controller.Controller, elastic *tapi.Elastic) (bool, error) {
	elasticReady := false
	then := time.Now()
	now := time.Now()
	for now.Sub(then) < durationCheckElastic {
		_elastic, err := c.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name)
		if err != nil {
			return false, err
		}

		log.Debugf("Pod Phase: %v", _elastic.Status.Phase)

		if _elastic.Status.Phase == tapi.DatabasePhaseRunning {
			elasticReady = true
			break
		}
		time.Sleep(time.Minute)
		now = time.Now()

	}

	if !elasticReady {
		return false, nil
	}

	return true, nil
}

func CheckElasticWorkload(c *controller.Controller, elastic *tapi.Elastic) error {
	if _, err := c.Client.Core().Services(elastic.Namespace).Get(elastic.Name); err != nil {
		return err
	}

	// SatatefulSet for Elastic database
	statefulSetName := fmt.Sprintf("%v-%v", amc.DatabaseNamePrefix, elastic.Name)
	if _, err := c.Client.Apps().StatefulSets(elastic.Namespace).Get(statefulSetName); err != nil {
		return err
	}

	podName := fmt.Sprintf("%v-%v", statefulSetName, 0)
	pod, err := c.Client.Core().Pods(elastic.Namespace).Get(podName)
	if err != nil {
		return err
	}

	// If job is success
	if pod.Status.Phase != kapi.PodRunning {
		return errors.New("Pod is not running")
	}

	return nil
}

func DeleteElastic(c *controller.Controller, elastic *tapi.Elastic) error {
	return c.ExtClient.Elastics(elastic.Namespace).Delete(elastic.Name)
}

func UpdateElastic(c *controller.Controller, elastic *tapi.Elastic) (*tapi.Elastic, error) {
	return c.ExtClient.Elastics(elastic.Namespace).Update(elastic)
}

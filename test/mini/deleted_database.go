package mini

import (
	"time"
	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	"github.com/k8sdb/elasticsearch/pkg/controller"
kerr "k8s.io/apimachinery/pkg/api/errors"
)

const durationCheckDormantDatabase = time.Minute * 30

func CheckDormantDatabasePhase(c *controller.Controller, elastic *tapi.Elastic, phase tapi.DormantDatabasePhase) (bool, error) {
	doneChecking := false
	then := time.Now()
	now := time.Now()

	for now.Sub(then) < durationCheckDormantDatabase {
		dormantDb, err := c.ExtClient.DormantDatabases(elastic.Namespace).Get(elastic.Name)
		if err != nil {
			if kerr.IsNotFound(err) {
				time.Sleep(time.Second * 10)
				now = time.Now()
				continue
			} else {
				return false, err
			}
		}

		log.Debugf("DormantDatabase Phase: %v", dormantDb.Status.Phase)

		if dormantDb.Status.Phase == phase {
			doneChecking = true
			break
		}

		time.Sleep(time.Second * 10)
		now = time.Now()

	}

	if !doneChecking {
		return false, nil
	}

	return true, nil
}

func WipeOutDormantDatabase(c *controller.Controller, elastic *tapi.Elastic) error {
	dormantDb, err := c.ExtClient.DormantDatabases(elastic.Namespace).Get(elastic.Name)
	if err != nil {
		return err
	}

	dormantDb.Spec.WipeOut = true

	_, err = c.ExtClient.DormantDatabases(dormantDb.Namespace).Update(dormantDb)
	return err
}

func DeleteDormantDatabase(c *controller.Controller, elastic *tapi.Elastic) error {
	return c.ExtClient.DormantDatabases(elastic.Namespace).Delete(elastic.Name)
}

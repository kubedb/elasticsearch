package controller

import (
	"fmt"
	"time"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amcs "github.com/k8sdb/apimachinery/client/clientset"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"gopkg.in/robfig/cron.v2"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
)

type backup struct {
	extClient amcs.ExtensionInterface
	elastic   *tapi.Elastic
}

func getSnapshotID(extClient amcs.ExtensionInterface, elasticName, elasticNamespace string) (string, error) {
	current := time.Now().UTC()
	snapshotName := fmt.Sprintf("%v-%d%02d%02d-%02d%02d%02d", elasticName,
		current.Year(), current.Month(), current.Day(),
		current.Hour(), current.Minute(), current.Second())

	return snapshotName, nil
}

func (b *backup) createDatabaseSnapshot() {
	labelMap := map[string]string{
		LabelDatabaseType:       DatabaseElasticsearch,
		LabelDatabaseName:       b.elastic.Name,
		amc.LabelSnapshotActive: string(tapi.SnapshotRunning),
	}

	snapshotList, err := b.extClient.DatabaseSnapshot(b.elastic.Namespace).List(kapi.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
	})
	if err != nil {
		log.Errorln(err)
		return
	}

	if len(snapshotList.Items) > 0 {
		log.Debugln("Skipping scheduled Backup. One is still active.")
		return
	}

	// Set label. Elastic controller will detect this using label selector
	labelMap = map[string]string{
		LabelDatabaseType: DatabaseElasticsearch,
		LabelDatabaseName: b.elastic.Name,
	}

	snapshotName, err := getSnapshotID(b.extClient, b.elastic.Name, b.elastic.Namespace)
	if err != nil {
		log.Errorln(err)
		return
	}
	snapshot := &tapi.DatabaseSnapshot{
		ObjectMeta: kapi.ObjectMeta{
			Name:      snapshotName,
			Namespace: b.elastic.Namespace,
			Labels:    labelMap,
		},
		Spec: tapi.DatabaseSnapshotSpec{
			DatabaseName: b.elastic.Name,
			SnapshotSpec: b.elastic.Spec.BackupSchedule.SnapshotSpec,
		},
	}

	if _, err := b.extClient.DatabaseSnapshot(snapshot.Namespace).Create(snapshot); err != nil {
		log.Errorln(err)
	}
}

// Backup schedule process with internal cron job.
func (w *Controller) ScheduleBackup(elastic *tapi.Elastic) error {
	// Remove previous cron job if exist
	if id, exists := w.cronEntryIDs.Pop(elastic.Name); exists {
		w.cron.Remove(id.(cron.EntryID))
	}

	b := &backup{
		extClient: w.Controller.ExtClient,
		elastic:   elastic,
	}

	// Set cron job
	entryID, err := w.cron.AddFunc(elastic.Spec.BackupSchedule.CronExpression, b.createDatabaseSnapshot)
	if err != nil {
		return err
	}

	// Add job entryID
	w.cronEntryIDs.Set(elastic.Name, entryID)
	return nil
}

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

func (b *backup) createDatabaseSnapshotObject() {
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

	now := time.Now().UTC()
	snapshotName := fmt.Sprintf("%v-%v", b.elastic.Name, now.Format("20060102-150405"))

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
func (w *Controller) scheduleBackup(elastic *tapi.Elastic) error {
	// Remove previous cron job if exist
	if id, exists := w.cronEntryIDs.Pop(elastic.Name); exists {
		w.cron.Remove(id.(cron.EntryID))
	}

	b := &backup{
		extClient: w.Controller.ExtClient,
		elastic:   elastic,
	}

	// Set cron job
	entryID, err := w.cron.AddFunc(elastic.Spec.BackupSchedule.CronExpression, b.createDatabaseSnapshotObject)
	if err != nil {
		return err
	}

	// Add job entryID
	w.cronEntryIDs.Set(elastic.Name, entryID)
	return nil
}

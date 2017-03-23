package controller

import (
	"time"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
)

type snapshotController struct {
	*Controller
}

const (
	imageElasticDump         = "appscode/elasticdump"
	LabelJobType             = "job.k8sdb.com/type"
	SnapshotProcess_Backup   = "backup"
	snapshotType_DumpBackup  = "dump-backup"
	storageSecretMountPath   = "/var/credentials/"
	tagElasticDump           = "2.4.2-v2"
	durationCheckSnapshotJob = time.Minute * 30
)

func (e *snapshotController) create(snapshot *tapi.DatabaseSnapshot) {
	// Validate DatabaseSnapshot TPR object
	if err := e.validateDatabaseSnapshot(snapshot); err != nil {
		log.Errorln(err)
		return
	}

	databaseName := snapshot.Spec.DatabaseName

	// Elastic TPR object must exist
	var elastic *tapi.Elastic
	var err error
	if elastic, err = e.ExtClient.Elastic(snapshot.Namespace).Get(databaseName); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Errorln(err)
			return
		} else {
			log.Errorf(`thirdpartyresource Elastic "%v" not found`, databaseName)
			return
		}
	}

	job, err := e.createBackupJob(snapshot, elastic)
	if err != nil {
		log.Errorln(err)
		return
	}

	snapshot.Labels[LabelDatabaseName] = snapshot.Spec.DatabaseName
	// Check Job for Backup process
	go e.Controller.CheckDatabaseSnapshotJob(snapshot, job.Name, durationCheckSnapshotJob)
}

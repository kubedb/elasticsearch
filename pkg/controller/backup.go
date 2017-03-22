package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/labels"
)

const (
	imageElasticDump         = "appscode/elasticdump"
	LabelJobType             = "job.k8sdb.com/type"
	SnapshotProcess_Backup   = "backup"
	snapshotType_DumpBackup  = "dump-backup"
	storageSecretMountPath   = "/var/credentials/"
	tagElasticDump           = "2.4.2-v2"
	durationCheckSnapshotJob = time.Minute * 30
)

func (w *Controller) backup(snapshot *tapi.DatabaseSnapshot) {
	// Validate DatabaseSnapshot TPR object
	if !w.validateDatabaseSnapshot(snapshot) {
		return
	}

	databaseName := snapshot.Spec.DatabaseName
	// Elastic TPR object must exist
	var elastic *tapi.Elastic
	var err error
	if elastic, err = w.ExtClient.Elastic(snapshot.Namespace).Get(databaseName); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Errorln(err)
			return
		} else {
			log.Errorf(`thirdpartyresource Elastic "%v" not found`, databaseName)
			return
		}
	}

	// Generate job name for backup
	// TODO: Use more accurate Job name
	jobName := rand.WithUniqSuffix(SnapshotProcess_Backup + "-" + databaseName)

	jobLabel := map[string]string{
		LabelDatabaseName: databaseName,
		LabelJobType:      SnapshotProcess_Backup,
	}

	backupSpec := snapshot.Spec.SnapshotSpec

	// Get PersistentVolume object for Backup Util pod.
	persistentVolume, err := w.GetVolumeForSnapshot(elastic.Spec.Storage, jobName, snapshot.Namespace)
	if err != nil {
		log.Errorln(err)
		return
	}

	// Folder name inside Cloud bucket where backup will be uploaded
	folderName := DatabaseElasticsearch + "-" + databaseName

	job := &batch.Job{
		ObjectMeta: kapi.ObjectMeta{
			Name:   jobName,
			Labels: jobLabel,
		},
		Spec: batch.JobSpec{
			Template: kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels: jobLabel,
				},
				Spec: kapi.PodSpec{
					Containers: []kapi.Container{
						{
							Name:  SnapshotProcess_Backup,
							Image: imageElasticDump + ":" + tagElasticDump,
							Args: []string{
								fmt.Sprintf(`--process=%s`, SnapshotProcess_Backup),
								fmt.Sprintf(`--host=%s`, databaseName),
								fmt.Sprintf(`--bucket=%s`, backupSpec.BucketName),
								fmt.Sprintf(`--folder=%s`, folderName),
								fmt.Sprintf(`--snapshot=%s`, snapshot.Name),
							},
							VolumeMounts: []kapi.VolumeMount{
								{
									Name:      "cloud",
									MountPath: storageSecretMountPath,
								},
								{
									Name:      persistentVolume.Name,
									MountPath: "/var/" + snapshotType_DumpBackup + "/",
								},
							},
						},
					},
					Volumes: []kapi.Volume{
						{
							Name: "cloud",
							VolumeSource: kapi.VolumeSource{
								Secret: backupSpec.StorageSecret,
							},
						},
						{
							Name:         persistentVolume.Name,
							VolumeSource: persistentVolume.VolumeSource,
						},
					},
					RestartPolicy: kapi.RestartPolicyNever,
				},
			},
		},
	}

	if _, err := w.Client.Batch().Jobs(snapshot.Namespace).Create(job); err != nil {
		log.Errorln(err)
		return
	}

	snapshot.Labels[LabelDatabaseName] = snapshot.Spec.DatabaseName
	// Check Job for Backup process
	go w.Controller.CheckDatabaseSnapshotJob(snapshot, job.Name, durationCheckSnapshotJob)
}

func (w *Controller) validateDatabaseSnapshot(snapshot *tapi.DatabaseSnapshot) bool {
	// Database name can't empty
	databaseName := snapshot.Spec.DatabaseName
	if databaseName == "" {
		log.Errorf(`Object 'DatabaseName' is missing in '%v'`, snapshot.Spec)
		return false
	}

	labelMap := map[string]string{
		LabelDatabaseType:       DatabaseElasticsearch,
		LabelDatabaseName:       snapshot.Spec.DatabaseName,
		amc.LabelSnapshotActive: string(tapi.SnapshotRunning),
	}

	snapshotList, err := w.ExtClient.DatabaseSnapshot(snapshot.Namespace).List(kapi.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
	})
	if err != nil {
		log.Errorln(err)
		return false
	}

	if len(snapshotList.Items) > 0 {
		unversionedNow := unversioned.Now()
		snapshot.Status.StartTime = &unversionedNow
		snapshot.Status.CompletionTime = &unversionedNow
		snapshot.Status.Status = tapi.SnapshotFailed
		snapshot.Status.Reason = "One DatabaseSnapshot is already Running"
		if _, err := w.ExtClient.DatabaseSnapshot(snapshot.Namespace).Update(snapshot); err != nil {
			log.Errorln(err)
		}
		return false
	}

	if err := w.validateBackupSpec(snapshot.Spec.SnapshotSpec, snapshot.Namespace); err != nil {
		log.Errorln(err)
		return false
	}

	return true
}

func (w *Controller) validateBackupSpec(backup tapi.SnapshotSpec, namespace string) error {
	// BucketName can't be empty
	bucketName := backup.BucketName
	if bucketName == "" {
		return errors.New(
			fmt.Sprintf(`Object 'BucketName' is missing in '%v'`, backup),
		)
	}

	// Need to provide Storage credential secret
	storageSecret := backup.StorageSecret
	if storageSecret == nil {
		return errors.New(fmt.Sprintf(`Object 'StorageSecret' is missing in '%v'`, backup))
	}

	// Credential SecretName  can't be empty
	storageSecretName := storageSecret.SecretName
	if storageSecretName == "" {
		return errors.New(
			fmt.Sprintf(`Object 'SecretName' is missing in '%v'`, *backup.StorageSecret),
		)
	}

	// Check bucket access with provided storage credential
	if err := w.CheckBucketAccess(backup.BucketName, storageSecretName, namespace); err != nil {
		return errors.New(
			fmt.Sprintf(`Fail to access bucket "%v" using Secret "%v.%v". Error: %v`,
				backup.BucketName, storageSecretName, namespace, err))
	}

	return nil
}

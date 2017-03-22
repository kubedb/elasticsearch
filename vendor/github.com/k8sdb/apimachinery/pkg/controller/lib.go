package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/appscode/log"
	"github.com/graymeta/stow"
	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/s3"
	tapi "github.com/k8sdb/apimachinery/api"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapps "k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/labels"
)

func (w *Controller) EnsureDatabaseSnapshot() {
	resourceName := "database-snapshot" + "." + tapi.V1beta1SchemeGroupVersion.Group

	if _, err := w.Client.Extensions().ThirdPartyResources().Get(resourceName); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Fatalln(err)
		}
	} else {
		return
	}

	thirdPartyResource := &extensions.ThirdPartyResource{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "ThirdPartyResource",
		},
		ObjectMeta: kapi.ObjectMeta{
			Name: resourceName,
		},
		Versions: []extensions.APIVersion{
			{
				Name: tapi.V1beta1SchemeGroupVersion.Version,
			},
		},
	}

	if _, err := w.Client.Extensions().ThirdPartyResources().Create(thirdPartyResource); err != nil {
		log.Fatalln(err)
	}
}

func (w *Controller) EnsureDeletedDatabase() {
	resourceName := "deleted-database" + "." + tapi.V1beta1SchemeGroupVersion.Group

	if _, err := w.Client.Extensions().ThirdPartyResources().Get(resourceName); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Fatalln(err)
		}
	} else {
		return
	}

	thirdPartyResource := &extensions.ThirdPartyResource{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "ThirdPartyResource",
		},
		ObjectMeta: kapi.ObjectMeta{
			Name: resourceName,
		},
		Versions: []extensions.APIVersion{
			{
				Name: tapi.V1beta1SchemeGroupVersion.Version,
			},
		},
	}

	if _, err := w.Client.Extensions().ThirdPartyResources().Create(thirdPartyResource); err != nil {
		log.Fatalln(err)
	}
}

const (
	LabelSnapshotActive = "elastic.k8sdb.com/status"
)

func (w *Controller) CheckDatabaseSnapshotJob(snapshot *tapi.DatabaseSnapshot, jobName string, checkTime float64) {

	unversionedNow := unversioned.Now()
	snapshot.Status.StartTime = &unversionedNow
	snapshot.Status.Status = tapi.SnapshotRunning

	snapshot.Labels[LabelSnapshotActive] = string(tapi.SnapshotRunning)
	var err error
	if snapshot, err = w.ExtClient.DatabaseSnapshot(snapshot.Namespace).Update(snapshot); err != nil {
		log.Errorln(err)
	}

	var jobSuccess bool = false
	var job *batch.Job

	then := time.Now()
	now := time.Now()
	for now.Sub(then).Minutes() < checkTime {
		log.Debugln("Checking for Job ", jobName)
		job, err = w.Client.Batch().Jobs(snapshot.Namespace).Get(jobName)
		if err != nil {
			break
		}
		log.Debugf("Pods Statuses:	%d Running / %d Succeeded / %d Failed",
			job.Status.Active, job.Status.Succeeded, job.Status.Failed)
		// If job is success
		if job.Status.Succeeded > 0 {
			jobSuccess = true
			break
		} else if job.Status.Failed > 0 {
			break
		}

		time.Sleep(time.Minute)
		now = time.Now()
	}

	podList, err := w.Client.Core().Pods(job.Namespace).List(
		kapi.ListOptions{
			LabelSelector: labels.SelectorFromSet(job.Spec.Selector.MatchLabels),
		},
	)
	if err != nil {
		log.Errorln(err)
		return
	}

	for _, pod := range podList.Items {
		if err := w.Client.Core().Pods(pod.Namespace).Delete(pod.Name, nil); err != nil {
			log.Errorln(err)
		}
	}

	for _, volume := range job.Spec.Template.Spec.Volumes {
		claim := volume.PersistentVolumeClaim
		if claim != nil {
			err := w.Client.Core().PersistentVolumeClaims(job.Namespace).Delete(claim.ClaimName, nil)
			if err != nil {
				log.Errorln(err)
			}
		}
	}

	if err := w.Client.Batch().Jobs(job.Namespace).Delete(job.Name, nil); err != nil {
		log.Errorln(err)
	}

	if snapshot, err = w.ExtClient.DatabaseSnapshot(snapshot.Namespace).Get(snapshot.Name); err != nil {
		log.Errorln(err)
		return
	}

	unversionedNow = unversioned.Now()
	snapshot.Status.CompletionTime = &unversionedNow
	if jobSuccess {
		snapshot.Status.Status = tapi.SnapshotSuccessed
	} else {
		snapshot.Status.Status = tapi.SnapshotFailed
	}

	delete(snapshot.Labels, LabelSnapshotActive)

	if _, err := w.ExtClient.DatabaseSnapshot(snapshot.Namespace).Update(snapshot); err != nil {
		log.Errorln(err)
	}
}

func (w *Controller) CheckStatefulSets(statefulSet *kapps.StatefulSet, checkTime float64) error {
	podName := fmt.Sprintf("%v-%v", statefulSet.Name, 0)

	podReady := false
	then := time.Now()
	now := time.Now()
	for now.Sub(then).Minutes() < checkTime {
		pod, err := w.Client.Core().Pods(statefulSet.Namespace).Get(podName)
		if err != nil {
			if k8serr.IsNotFound(err) {
				time.Sleep(time.Second * 10)
				now = time.Now()
				continue
			} else {
				return err
			}
		}
		log.Debugf("Pod Phase: %v", pod.Status.Phase)

		// If job is success
		if pod.Status.Phase == kapi.PodRunning {
			podReady = true
			break
		}

		time.Sleep(time.Minute)
		now = time.Now()
	}
	if !podReady {
		return errors.New("Database fails to be Ready")
	}
	return nil
}

func (w *Controller) GetVolumeForSnapshot(storage *tapi.StorageSpec, jobName, namespace string) (*kapi.Volume, error) {
	volume := &kapi.Volume{
		Name: "util-volume",
	}
	if storage != nil {
		claim := &kapi.PersistentVolumeClaim{
			ObjectMeta: kapi.ObjectMeta{
				Name:      jobName,
				Namespace: namespace,
				Annotations: map[string]string{
					"volume.beta.kubernetes.io/storage-class": storage.Class,
				},
			},
			Spec: storage.PersistentVolumeClaimSpec,
		}

		if _, err := w.Client.Core().PersistentVolumeClaims(claim.Namespace).Create(claim); err != nil {
			return nil, err
		}

		volume.PersistentVolumeClaim = &kapi.PersistentVolumeClaimVolumeSource{
			ClaimName: claim.Name,
		}
	} else {
		volume.EmptyDir = &kapi.EmptyDirVolumeSource{}
	}
	return volume, nil
}

const (
	keyProvider = "provider"
	keyConfig   = "config"
)

func (w *Controller) CheckBucketAccess(bucketName, secretName, namespace string) error {
	secret, err := w.Client.Core().Secrets(namespace).Get(secretName)
	if err != nil {
		return err
	}

	provider := secret.Data[keyProvider]
	if provider == nil {
		return errors.New("Missing provider key")
	}
	configData := secret.Data[keyConfig]
	if configData == nil {
		return errors.New("Missing config key")
	}

	var config stow.ConfigMap
	if err := json.Unmarshal(configData, &config); err != nil {
		return errors.New("Fail to Unmarshal config data")
	}

	loc, err := stow.Dial(string(provider), config)
	if err != nil {
		return err
	}

	container, err := loc.Container(bucketName)
	if err != nil {
		return err
	}

	r := bytes.NewReader([]byte("CheckBucketAccess"))
	item, err := container.Put(".k8sdb", r, r.Size(), nil)
	if err != nil {
		return err
	}

	if err := container.RemoveItem(item.ID()); err != nil {
		return err
	}
	return nil
}

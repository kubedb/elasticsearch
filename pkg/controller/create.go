package controller

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	tapi "github.com/k8sdb/apimachinery/api"
	kapi "k8s.io/kubernetes/pkg/api"
	kapps "k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/util/intstr"
)

const (
	annotationDatabaseVersion  = "elastic.k8sdb.com/version"
	DatabaseElasticsearch      = "elasticsearch"
	DatabaseNamePrefix         = "k8sdb"
	GoverningElasticsearch     = "governing-elasticsearch"
	imageElasticsearch         = "appscode/elasticsearch"
	imageOperatorElasticsearch = "appscode/k8ses"
	LabelDatabaseType          = "k8sdb.com/type"
	LabelDatabaseName          = "elastic.k8sdb.com/name"
	tagOperatorElasticsearch   = "0.1"
	// Duration in Minute
	// Check whether pod under StatefulSet is running or not
	// Continue checking for this duration until failure
	durationCheckStatefulSet = time.Minute * 30
)

func (w *Controller) createGoverningServiceAccount(name, namespace string) error {
	found, err := w.checkGoverningServiceAccount(name, namespace)
	if err != nil {
		return err

	}
	if found {
		return nil
	}

	serviceAccount := &kapi.ServiceAccount{
		ObjectMeta: kapi.ObjectMeta{
			Name: name,
		},
	}

	_, err = w.Client.Core().ServiceAccounts(namespace).Create(serviceAccount)
	return err
}

func (w *Controller) createService(name, namespace string) error {
	// Check if service name exists
	found, err := w.checkService(namespace, name)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	label := map[string]string{
		LabelDatabaseName: name,
	}
	service := &kapi.Service{
		ObjectMeta: kapi.ObjectMeta{
			Name:   name,
			Labels: label,
		},
		Spec: kapi.ServiceSpec{
			Ports: []kapi.ServicePort{
				{
					Name:       "api",
					Port:       9200,
					TargetPort: intstr.FromString("api"),
				},
				{
					Name:       "tcp",
					Port:       9300,
					TargetPort: intstr.FromString("tcp"),
				},
			},
			Selector: label,
		},
	}

	if _, err := w.Client.Core().Services(namespace).Create(service); err != nil {
		return err
	}

	return nil
}

func (w *Controller) createStatefulSet(elastic *tapi.Elastic) (*kapps.StatefulSet, error) {
	// Set labels
	if elastic.Labels == nil {
		elastic.Labels = make(map[string]string)
	}
	elastic.Labels[LabelDatabaseType] = DatabaseElasticsearch
	// Set Annotations
	if elastic.Annotations == nil {
		elastic.Annotations = make(map[string]string)
	}
	elastic.Annotations[annotationDatabaseVersion] = elastic.Spec.Version

	podLabels := make(map[string]string)
	for key, val := range elastic.Labels {
		podLabels[key] = val
	}
	podLabels[LabelDatabaseName] = elastic.Name

	dockerImage := fmt.Sprintf("%v:%v", imageElasticsearch, elastic.Spec.Version)
	initContainerImage := fmt.Sprintf("%v:%v", imageOperatorElasticsearch, tagOperatorElasticsearch)

	// SatatefulSet for Elastic database
	statefulSetName := fmt.Sprintf("%v-%v", DatabaseNamePrefix, elastic.Name)
	statefulSet := &kapps.StatefulSet{
		ObjectMeta: kapi.ObjectMeta{
			Name:        statefulSetName,
			Namespace:   elastic.Namespace,
			Labels:      elastic.Labels,
			Annotations: elastic.Annotations,
		},
		Spec: kapps.StatefulSetSpec{
			Replicas:    elastic.Spec.Replicas,
			ServiceName: elastic.Spec.ServiceAccountName,
			Template: kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels:      podLabels,
					Annotations: elastic.Annotations,
				},
				Spec: kapi.PodSpec{
					Containers: []kapi.Container{
						{
							Name:            DatabaseElasticsearch,
							Image:           dockerImage,
							ImagePullPolicy: kapi.PullIfNotPresent,
							Ports: []kapi.ContainerPort{
								{
									Name:          "api",
									ContainerPort: 9200,
								},
								{
									Name:          "tcp",
									ContainerPort: 9300,
								},
							},
							VolumeMounts: []kapi.VolumeMount{
								{
									Name:      "discovery",
									MountPath: "/tmp/discovery",
								},
								{
									Name:      "volume",
									MountPath: "/var/pv",
								},
							},
							Env: []kapi.EnvVar{
								{
									Name:  "CLUSTER_NAME",
									Value: elastic.Name,
								},
								{
									Name:  "KUBE_NAMESPACE",
									Value: elastic.Namespace,
								},
							},
						},
					},
					InitContainers: []kapi.Container{
						{
							Name:            "discover",
							Image:           initContainerImage,
							ImagePullPolicy: kapi.PullIfNotPresent,
							Args: []string{
								"discover",
								fmt.Sprintf("--service=%v", elastic.Name),
								fmt.Sprintf("--namespace=%v", elastic.Namespace),
							},
							Env: []kapi.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &kapi.EnvVarSource{
										FieldRef: &kapi.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.name",
										},
									},
								},
							},
							VolumeMounts: []kapi.VolumeMount{
								{
									Name:      "discovery",
									MountPath: "/tmp/discovery",
								},
							},
						},
					},
					NodeSelector: elastic.Spec.NodeSelector,
					Volumes: []kapi.Volume{
						{
							Name: "discovery",
							VolumeSource: kapi.VolumeSource{
								EmptyDir: &kapi.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// Add Data volume for StatefulSet
	addDataVolume(statefulSet, elastic.Spec.Storage)

	if _, err := w.Client.Apps().StatefulSets(statefulSet.Namespace).Create(statefulSet); err != nil {
		return nil, err
	}

	return statefulSet, nil
}

func addDataVolume(statefulSet *kapps.StatefulSet, storage *tapi.StorageSpec) {
	if storage != nil {
		// volume claim templates
		// Dynamically attach volume
		storageClassName := storage.Class
		statefulSet.Spec.VolumeClaimTemplates = []kapi.PersistentVolumeClaim{
			{
				ObjectMeta: kapi.ObjectMeta{
					Name: "volume",
					Annotations: map[string]string{
						"volume.beta.kubernetes.io/storage-class": storageClassName,
					},
				},
				Spec: storage.PersistentVolumeClaimSpec,
			},
		}
	} else {
		// Attach Empty directory
		statefulSet.Spec.Template.Spec.Volumes = append(
			statefulSet.Spec.Template.Spec.Volumes,
			kapi.Volume{
				Name: "volume",
				VolumeSource: kapi.VolumeSource{
					EmptyDir: &kapi.EmptyDirVolumeSource{},
				},
			},
		)
	}
}

func (w *Controller) createBackupJob(snapshot *tapi.DatabaseSnapshot, elastic *tapi.Elastic) (*batch.Job, error) {

	databaseName := snapshot.Spec.DatabaseName

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
		return nil, err
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
		return nil, err
	}

	return job, nil
}

package controller

import (
	"fmt"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/elasticsearch/api"
	kapi "k8s.io/kubernetes/pkg/api"
	kapps "k8s.io/kubernetes/pkg/apis/apps"
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
)

func (w *Controller) create(elastic *tapi.Elastic) {
	if !w.validateElastic(elastic) {
		return
	}

	governingService := GoverningElasticsearch
	if elastic.Spec.ServiceAccountName != "" {
		governingService = elastic.Spec.ServiceAccountName
	}
	if err := w.createGoverningServiceAccount(elastic.Namespace, governingService); err != nil {
		log.Errorln(err)
		return
	}

	if err := w.createService(elastic.Namespace, elastic.Name); err != nil {
		log.Errorln(err)
		return
	}

	if elastic.Labels == nil {
		elastic.Labels = make(map[string]string)
	}
	elastic.Labels[LabelDatabaseType] = DatabaseElasticsearch

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
			ServiceName: governingService,
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

	// Add PersistentVolumeClaim for StatefulSet
	w.addPersistentVolumeClaim(statefulSet, elastic.Spec.Storage)

	if _, err := w.Client.Apps().StatefulSets(statefulSet.Namespace).Create(statefulSet); err != nil {
		log.Errorln(err)
		return
	}
}

func (w *Controller) validateElastic(elastic *tapi.Elastic) bool {
	if elastic.Spec.Version == "" {
		log.Errorln(fmt.Sprintf(`Object 'Version' is missing in '%v'`, elastic.Spec))
		return false
	}

	storage := elastic.Spec.Storage
	if storage != nil {
		if storage.Class == "" {
			log.Errorln(fmt.Sprintf(`Object 'Class' is missing in '%v'`, *storage))
			return false
		}
		storageClass, err := w.Client.Storage().StorageClasses().Get(storage.Class)
		if err != nil {
			log.Errorln(err)
			return false
		}
		if storageClass == nil {
			log.Errorln(fmt.Sprintf(`Spec.Storage.Class "%v" not found`, storage.Class))
			return false
		}
	}

	return true
}

func (w *Controller) addPersistentVolumeClaim(statefulSet *kapps.StatefulSet, storage *tapi.StorageSpec) {
	if storage != nil {
		// volume claim templates
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
	}
}

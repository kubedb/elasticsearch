package controller

import (
	"fmt"

	"encoding/json"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/elasticsearch/api"
	kapi "k8s.io/kubernetes/pkg/api"
	kapps "k8s.io/kubernetes/pkg/apis/apps"
)

const (
	databasePrefix       = "k8sdb"
	databaseType         = "elasticsearch"
	dockerImage          = "appscode/elasticsearch"
	governingServiceName = "governing-elasticsearch"
	operatorImageTag     = "0.1"
	serviceSelector      = "es.k8sdb.com/name"
)

func (w *Controller) create(elasticsearch *tapi.Elasticsearch) {
	if !w.validateElasticsearch(elasticsearch) {
		return
	}

	governingService := governingServiceName
	if elasticsearch.Spec.ServiceAccountName != "" {
		governingService = elasticsearch.Spec.ServiceAccountName
	}
	if err := w.createGoverningServiceAccount(elasticsearch.Namespace, governingService); err != nil {
		log.Errorln(err)
		return
	}

	if err := w.createService(elasticsearch.Namespace, elasticsearch.Name); err != nil {
		log.Errorln(err)
		return
	}

	if elasticsearch.Labels == nil {
		elasticsearch.Labels = make(map[string]string)
	}
	if elasticsearch.Annotations == nil {
		elasticsearch.Annotations = make(map[string]string)
	}

	elasticsearch.Labels[serviceSelector] = elasticsearch.Name
	elasticsearch.Annotations["k8sdb.com/type"] = databaseType

	dockerImage := fmt.Sprintf("%v:%v", dockerImage, elasticsearch.Spec.Version)

	statefulSet := &kapps.StatefulSet{
		ObjectMeta: kapi.ObjectMeta{
			Name:        fmt.Sprintf("%v-%v", databasePrefix, elasticsearch.Name),
			Namespace:   elasticsearch.Namespace,
			Labels:      elasticsearch.Labels,
			Annotations: elasticsearch.Annotations,
		},
		Spec: kapps.StatefulSetSpec{
			Replicas:    elasticsearch.Spec.Replicas,
			ServiceName: governingService,
			Template: kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{
					Labels:      elasticsearch.Labels,
					Annotations: elasticsearch.Annotations,
				},
				Spec: kapi.PodSpec{
					Containers: []kapi.Container{
						{
							Name:            databaseType,
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
						},
					},
					NodeSelector: elasticsearch.Spec.NodeSelector,
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

	initContainer := []kapi.Container{
		{
			Name:            "discover",
			Image:           "appscode/k8ses:" + operatorImageTag,
			ImagePullPolicy: "IfNotPresent",
			Args: []string{
				"discover",
				fmt.Sprintf("--service=%v", elasticsearch.Name),
				fmt.Sprintf("--namespace=%v", elasticsearch.Namespace),
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
	}

	initContainerDataByte, err := json.MarshalIndent(initContainer, "", "	")
	if err != nil {
		log.Errorln(err)
		return
	}
	statefulSet.Spec.Template.Annotations["pod.beta.kubernetes.io/init-containers"] = string(initContainerDataByte)

	// Add PersistentVolumeClaim for StatefulSet
	w.addPersistentVolumeClaim(statefulSet, elasticsearch.Spec.Storage)

	if _, err := w.Client.Apps().StatefulSets(statefulSet.Namespace).Create(statefulSet); err != nil {
		log.Errorln(err)
		return
	}
}

func (w *Controller) validateElasticsearch(elasticsearch *tapi.Elasticsearch) bool {
	if elasticsearch.Spec.Version == "" {
		log.Errorln(fmt.Sprintf(`Object 'Version' is missing in '%v'`, elasticsearch.Spec))
		return false
	}

	storage := elasticsearch.Spec.Storage
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

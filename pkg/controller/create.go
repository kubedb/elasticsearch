package controller

import (
	"fmt"
	"time"

	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	kapps "k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/util/intstr"
)

const (
	annotationDatabaseVersion  = "elastic.k8sdb.com/version"
	DatabaseElasticsearch      = "elasticsearch"
	GoverningElasticsearch     = "governing-elasticsearch"
	imageElasticsearch         = "appscode/elasticsearch"
	imageOperatorElasticsearch = "appscode/k8ses"
	tagOperatorElasticsearch   = "0.1"
	// Duration in Minute
	// Check whether pod under StatefulSet is running or not
	// Continue checking for this duration until failure
	durationCheckStatefulSet = time.Minute * 30
)

func (c *elasticController) checkService(name, namespace string) (bool, error) {
	service, err := c.Client.Core().Services(namespace).Get(name)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	if service == nil {
		return false, nil
	}

	if service.Spec.Selector[amc.LabelDatabaseName] != name {
		return false, fmt.Errorf(`Intended service "%v" already exists`, name)
	}

	return true, nil
}

func (c *elasticController) createService(name, namespace string) error {
	// Check if service name exists
	found, err := c.checkService(name, namespace)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	label := map[string]string{
		amc.LabelDatabaseName: name,
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

	if _, err := c.Client.Core().Services(namespace).Create(service); err != nil {
		return err
	}

	return nil
}

func (c *elasticController) createStatefulSet(elastic *tapi.Elastic) (*kapps.StatefulSet, error) {
	// Set labels
	if elastic.Labels == nil {
		elastic.Labels = make(map[string]string)
	}
	elastic.Labels[amc.LabelDatabaseType] = DatabaseElasticsearch
	// Set Annotations
	if elastic.Annotations == nil {
		elastic.Annotations = make(map[string]string)
	}
	elastic.Annotations[annotationDatabaseVersion] = elastic.Spec.Version

	podLabels := make(map[string]string)
	for key, val := range elastic.Labels {
		podLabels[key] = val
	}
	podLabels[amc.LabelDatabaseName] = elastic.Name

	dockerImage := fmt.Sprintf("%v:%v", imageElasticsearch, elastic.Spec.Version)
	initContainerImage := fmt.Sprintf("%v:%v", imageOperatorElasticsearch, tagOperatorElasticsearch)

	// SatatefulSet for Elastic database
	statefulSetName := fmt.Sprintf("%v-%v", amc.DatabaseNamePrefix, elastic.Name)
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

	if _, err := c.Client.Apps().StatefulSets(statefulSet.Namespace).Create(statefulSet); err != nil {
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

func (w *elasticController) createDeletedDatabase(elastic *tapi.Elastic) (*tapi.DeletedDatabase, error) {
	deletedDb := &tapi.DeletedDatabase{
		ObjectMeta: kapi.ObjectMeta{
			Name:      elastic.Name,
			Namespace: elastic.Namespace,
			Labels: map[string]string{
				amc.LabelDatabaseType: DatabaseElasticsearch,
			},
		},
	}
	return w.ExtClient.DeletedDatabases(deletedDb.Namespace).Create(deletedDb)
}

func (w *elasticController) reCreateElastic(elastic *tapi.Elastic) error {
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

	if _, err := w.ExtClient.Elastics(_elastic.Namespace).Create(_elastic); err != nil {
		return err
	}

	return nil
}

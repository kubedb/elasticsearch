package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	kutilapps "github.com/appscode/kutil/apps/v1beta1"
	tapi "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	kutildb "github.com/k8sdb/apimachinery/client/typed/kubedb/v1alpha1/util"
	"github.com/k8sdb/apimachinery/pkg/docker"
	"github.com/k8sdb/apimachinery/pkg/eventer"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) ensureStatefulSet(elasticsearch *tapi.Elasticsearch, name string, isMasterNode, isDataNode, isClientNode, isDedicated bool) error {
	found, err := c.findStatefulSet(elasticsearch, name)
	if err != nil {
		return err
	}
	if found {
		_statefulset, err := c.Client.AppsV1beta1().StatefulSets(elasticsearch.Namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		kutilapps.TryPatchStatefulSet(c.Client, _statefulset.ObjectMeta, func(in *apps.StatefulSet) *apps.StatefulSet {
			var replicas int32
			topology := elasticsearch.Spec.Topology
			if isMasterNode {
				var totalMaster int32
				if isDedicated {
					totalMaster = topology.Master.Replicas
					replicas = totalMaster
				} else {
					totalMaster = elasticsearch.Spec.Replicas
				}
				if totalMaster == 0 {
					totalMaster = 1
				}
				for _, container := range in.Spec.Template.Spec.Containers {
					if container.Name == tapi.ResourceNameElasticsearch {
						for i, env := range container.Env {
							if env.Name == "NUMBER_OF_MASTERS" {
								container.Env[i].Value = fmt.Sprintf("%v", (totalMaster/2)+1)
								break
							}
						}
						break
					}
				}
				in.Spec.UpdateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
			} else if isDataNode {
				if isDedicated {
					replicas = topology.Data.Replicas
				}
			} else if isClientNode {
				if isDedicated {
					replicas = topology.Client.Replicas
				}
			}
			if replicas == 0 {
				replicas = 1
			}
			in.Spec.Replicas = types.Int32P(replicas)
			return in
		})
		return nil
	}

	// Create statefulSet for Elasticsearch database
	statefulSet, err := c.createStatefulSet(elasticsearch, name, isMasterNode, isDataNode, isClientNode, isDedicated)
	if err != nil {
		c.recorder.Eventf(
			elasticsearch.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			"Failed to create StatefulSet. Reason: %v",
			err,
		)
		return err
	}

	// Check StatefulSet Pod status
	if err := c.CheckStatefulSetPodStatus(statefulSet, durationCheckStatefulSet); err != nil {
		c.recorder.Eventf(
			elasticsearch.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToStart,
			"Failed to create StatefulSet. Reason: %v",
			err,
		)

		return err
	} else {
		c.recorder.Event(
			elasticsearch.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulCreate,
			"Successfully created StatefulSet",
		)
	}

	return nil
}

func (c *Controller) findStatefulSet(elasticsearch *tapi.Elasticsearch, name string) (bool, error) {
	elasticsearchName := elasticsearch.OffshootName()
	// SatatefulSet for Elasticsearch database
	statefulSet, err := c.Client.AppsV1beta1().StatefulSets(elasticsearch.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	if statefulSet.Labels[tapi.LabelDatabaseKind] != tapi.ResourceKindElasticsearch ||
		statefulSet.Labels[tapi.LabelDatabaseName] != elasticsearchName {
		return false, fmt.Errorf(`Intended statefulSet "%v" already exists`, name)
	}

	return true, nil
}

func (c *Controller) createStatefulSet(elasticsearch *tapi.Elasticsearch, name string, isMasterNode, isDataNode, isClientNode, isDedicated bool) (*apps.StatefulSet, error) {
	dockerImage := fmt.Sprintf("%v:%v", docker.ImageElasticsearch, elasticsearch.Spec.Version)

	containerPortList := []core.ContainerPort{
		{
			Name:          "transport",
			ContainerPort: 9300,
			Protocol:      core.ProtocolTCP,
		},
	}
	envList := []core.EnvVar{
		{
			Name:  "CLUSTER_NAME",
			Value: elasticsearch.Name,
		},
		{
			Name: "NODE_NAME",
			ValueFrom: &core.EnvVarSource{
				FieldRef: &core.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name:  "ES_JAVA_OPTS",
			Value: "-Xms512m -Xmx512m",
		},
		{
			Name:  "DISCOVERY_SERVICE",
			Value: fmt.Sprintf("%v-discovery", elasticsearch.OffshootName()),
		},
	}
	labels := elasticsearch.StatefulSetLabels()

	topology := elasticsearch.Spec.Topology
	var replicas int32

	if isMasterNode {
		var totalMaster int32
		if isDedicated {
			totalMaster = topology.Master.Replicas
			replicas = totalMaster
		} else {
			totalMaster = elasticsearch.Spec.Replicas
		}
		if totalMaster == 0 {
			totalMaster = 1
		}
		envList = append(envList, core.EnvVar{
			Name:  "NUMBER_OF_MASTERS",
			Value: fmt.Sprintf("%v", (totalMaster/2)+1),
		})
		labels[LabelNodeRoleMaster] = "set"
	}

	if isDataNode {
		labels[LabelNodeRoleData] = "set"
		if isDedicated {
			replicas = topology.Data.Replicas
		}
	}
	if isClientNode {
		containerPortList = append(containerPortList, core.ContainerPort{
			Name:          "http",
			ContainerPort: 9200,
			Protocol:      core.ProtocolTCP,
		})
		envList = append(envList, core.EnvVar{
			Name:  "MODE",
			Value: "client",
		})
		labels[LabelNodeRoleClient] = "set"
		if isDedicated {
			replicas = topology.Client.Replicas
		}
	}
	if replicas <= 0 {
		replicas = 1
	}

	envList = append(envList, []core.EnvVar{
		{
			Name:  "NODE_MASTER",
			Value: fmt.Sprintf("%v", isMasterNode),
		},
		{
			Name:  "NODE_DATA",
			Value: fmt.Sprintf("%v", isDataNode),
		},
		{
			Name:  "NODE_INGEST",
			Value: fmt.Sprintf("%v", isClientNode),
		},
		{
			Name:  "HTTP_ENABLE",
			Value: fmt.Sprintf("%v", isClientNode),
		},
		{
			Name:  "SSL_ENABLE",
			Value: fmt.Sprintf("%v", elasticsearch.Spec.EnableSSL),
		},
	}...)

	// SatatefulSet for Elasticsearch database
	statefulSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   elasticsearch.Namespace,
			Labels:      labels,
			Annotations: elasticsearch.StatefulSetAnnotations(),
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    types.Int32P(replicas),
			ServiceName: c.opt.GoverningService,
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: core.PodSpec{
					InitContainers: []core.Container{
						{
							Name:            "init-sysctl",
							Image:           "busybox",
							ImagePullPolicy: core.PullIfNotPresent,
							Command:         []string{"sysctl", "-w", "vm.max_map_count=262144"},
							SecurityContext: &core.SecurityContext{
								Privileged: types.BoolP(true),
							},
						},
					},
					Containers: []core.Container{
						{
							Name:            tapi.ResourceNameElasticsearch,
							Image:           dockerImage,
							ImagePullPolicy: core.PullAlways,
							Env:             envList,
							Ports:           containerPortList,
							SecurityContext: &core.SecurityContext{
								Privileged: types.BoolP(false),
								Capabilities: &core.Capabilities{
									Add: []core.Capability{"IPC_LOCK", "SYS_RESOURCE"},
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "certs",
									MountPath: "/elasticsearch/config/certs",
								},
							},
						},
					},
					NodeSelector:  elasticsearch.Spec.NodeSelector,
					Volumes:       []core.Volume{},
					Affinity:      elasticsearch.Spec.Affinity,
					SchedulerName: elasticsearch.Spec.SchedulerName,
					Tolerations:   elasticsearch.Spec.Tolerations,
				},
			},
		},
	}

	if elasticsearch.Spec.Monitor != nil &&
		elasticsearch.Spec.Monitor.Agent == tapi.AgentCoreosPrometheus &&
		elasticsearch.Spec.Monitor.Prometheus != nil {
		exporter := core.Container{
			Name: "exporter",
			Args: []string{
				"export",
				fmt.Sprintf("--address=:%d", tapi.PrometheusExporterPortNumber),
				"--v=3",
			},
			Image:           docker.ImageOperator + ":" + c.opt.ExporterTag,
			ImagePullPolicy: core.PullIfNotPresent,
			Ports: []core.ContainerPort{
				{
					Name:          tapi.PrometheusExporterPortName,
					Protocol:      core.ProtocolTCP,
					ContainerPort: int32(tapi.PrometheusExporterPortNumber),
				},
			},
		}
		statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers, exporter)
	}

	if elasticsearch.Spec.CertificateSecret == nil {
		certSecretVolumeSource, err := c.createCertSecret(elasticsearch)
		if err != nil {
			return nil, err
		}

		_elasticsearch, err := kutildb.TryPatchElasticsearch(c.ExtClient, elasticsearch.ObjectMeta, func(in *tapi.Elasticsearch) *tapi.Elasticsearch {
			in.Spec.CertificateSecret = certSecretVolumeSource
			return in
		})
		if err != nil {
			c.recorder.Eventf(elasticsearch.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
			return nil, err
		}
		elasticsearch = _elasticsearch
	}

	// Add CertSecretVolume for Certificates
	addCertSecretVolume(statefulSet, elasticsearch.Spec.CertificateSecret, isClientNode)

	if isClientNode {
		if elasticsearch.Spec.AuthSecret == nil {
			authSecretVolumeSource, err := c.createAuthSecret(elasticsearch)
			if err != nil {
				return nil, err
			}

			_elasticsearch, err := kutildb.TryPatchElasticsearch(c.ExtClient, elasticsearch.ObjectMeta, func(in *tapi.Elasticsearch) *tapi.Elasticsearch {
				in.Spec.AuthSecret = authSecretVolumeSource
				return in
			})
			if err != nil {
				c.recorder.Eventf(elasticsearch.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToUpdate, err.Error())
				return nil, err
			}
			elasticsearch = _elasticsearch
		}

		containers := statefulSet.Spec.Template.Spec.Containers
		for i, container := range containers {
			if container.Name == tapi.ResourceNameElasticsearch {
				containers[i].VolumeMounts = append(container.VolumeMounts, core.VolumeMount{
					Name:      "sgconfig",
					MountPath: "/elasticsearch/plugins/search-guard-5/sgconfig",
				})
				break
			}
		}

		// Add CertSecretVolume for Certificates
		addAuthSecretVolume(statefulSet, elasticsearch.Spec.AuthSecret)
	}

	// Add Data volume for StatefulSet
	addDataVolume(statefulSet, elasticsearch.Spec.Storage)

	if c.opt.EnableRbac {
		// Ensure ClusterRoles for database statefulsets
		if err := c.createRBACStuff(elasticsearch); err != nil {
			return nil, err
		}

		statefulSet.Spec.Template.Spec.ServiceAccountName = elasticsearch.Name
	}

	if _, err := c.Client.AppsV1beta1().StatefulSets(statefulSet.Namespace).Create(statefulSet); err != nil {
		return nil, err
	}

	return statefulSet, nil
}

func addDataVolume(statefulSet *apps.StatefulSet, pvcSpec *core.PersistentVolumeClaimSpec) {
	if pvcSpec != nil {
		if len(pvcSpec.AccessModes) == 0 {
			pvcSpec.AccessModes = []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			}
			log.Infof(`Using "%v" as AccessModes in "%v"`, core.ReadWriteOnce, *pvcSpec)
		}
		// volume claim templates
		// Dynamically attach volume
		statefulSet.Spec.VolumeClaimTemplates = []core.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
					Annotations: map[string]string{
						"volume.beta.kubernetes.io/storage-class": *pvcSpec.StorageClassName,
					},
				},
				Spec: *pvcSpec,
			},
		}
	} else {
		// Attach Empty directory
		statefulSet.Spec.Template.Spec.Volumes = append(
			statefulSet.Spec.Template.Spec.Volumes,
			core.Volume{
				Name: "data",
				VolumeSource: core.VolumeSource{
					EmptyDir: &core.EmptyDirVolumeSource{},
				},
			},
		)
	}
}

func addCertSecretVolume(statefulSet *apps.StatefulSet, secretVolumeSource *core.SecretVolumeSource, addSgAdminCert bool) error {
	svs := &core.SecretVolumeSource{
		SecretName: secretVolumeSource.SecretName,
		Items: []core.KeyToPath{
			{
				Key:  "truststore.jks",
				Path: "truststore.jks",
			},
			{
				Key:  "keystore.jks",
				Path: "keystore.jks",
			},
		},
	}

	if addSgAdminCert {
		svs.Items = append(svs.Items, core.KeyToPath{
			Key:  "sgadmin.jks",
			Path: "sgadmin.jks",
		})
	}

	statefulSet.Spec.Template.Spec.Volumes = append(statefulSet.Spec.Template.Spec.Volumes,
		core.Volume{
			Name: "certs",
			VolumeSource: core.VolumeSource{
				Secret: svs,
			},
		},
	)
	return nil
}

func addAuthSecretVolume(statefulSet *apps.StatefulSet, secretVolumeSource *core.SecretVolumeSource) error {
	statefulSet.Spec.Template.Spec.Volumes = append(statefulSet.Spec.Template.Spec.Volumes,
		core.Volume{
			Name: "sgconfig",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: secretVolumeSource.SecretName,
				},
			},
		},
	)
	return nil
}

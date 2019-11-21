/*
Copyright The KubeDB Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package e2e_test

import (
	"fmt"
	"os"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"
	"kubedb.dev/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	"kubedb.dev/elasticsearch/test/e2e/framework"
	"kubedb.dev/elasticsearch/test/e2e/matcher"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	core_util "kmodules.xyz/client-go/core/v1"
	exec_util "kmodules.xyz/client-go/tools/exec"
	store "kmodules.xyz/objectstore-api/api/v1"
	stashV1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	stashV1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

const (
	S3_BUCKET_NAME       = "S3_BUCKET_NAME"
	GCS_BUCKET_NAME      = "GCS_BUCKET_NAME"
	AZURE_CONTAINER_NAME = "AZURE_CONTAINER_NAME"
	SWIFT_CONTAINER_NAME = "SWIFT_CONTAINER_NAME"
)

var _ = Describe("Elasticsearch", func() {
	var (
		err                      error
		f                        *framework.Invocation
		elasticsearch            *api.Elasticsearch
		garbageElasticsearch     *api.ElasticsearchList
		snapshot                 *api.Snapshot
		snapshotPVC              *core.PersistentVolumeClaim
		secret                   *core.Secret
		skipMessage              string
		indicesCount             int
		skipSnapshotDataChecking bool
	)

	BeforeEach(func() {
		f = root.Invoke()
		elasticsearch = f.CombinedElasticsearch()
		garbageElasticsearch = new(api.ElasticsearchList)
		snapshot = f.Snapshot()
		indicesCount = 0
		secret = nil
		skipMessage = ""
		skipSnapshotDataChecking = true
	})

	var createAndWaitForRunning = func() {
		By("Create Elasticsearch: " + elasticsearch.Name)
		err = f.CreateElasticsearch(elasticsearch)
		Expect(err).NotTo(HaveOccurred())

		By("Wait for Running elasticsearch")
		f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

		By("Wait for AppBinding to create")
		f.EventuallyAppBinding(elasticsearch.ObjectMeta).Should(BeTrue())

		By("Check valid AppBinding Specs")
		err := f.CheckAppBindingSpec(elasticsearch.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	}

	var createAndInsertData = func() {

		createAndWaitForRunning()

		By("Check for Elastic client")
		f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

		elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		By("Creating new indices")
		err = elasticClient.CreateIndex(2)
		Expect(err).NotTo(HaveOccurred())
		indicesCount += 2

		By("Checking new indices")
		f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))

		elasticClient.Stop()
		f.Tunnel.Close()
	}

	var deleteTestResource = func() {
		if elasticsearch == nil {
			Skip("Skipping")
		}

		By("Check if elasticsearch " + elasticsearch.Name + " exists.")
		es, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
		if err != nil {
			if kerr.IsNotFound(err) {
				// Elasticsearch was not created. Hence, rest of cleanup is not necessary.
				return
			}
			Expect(err).NotTo(HaveOccurred())
		}

		By("Delete elasticsearch: " + elasticsearch.Name)
		err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
		if err != nil {
			if kerr.IsNotFound(err) {
				// Elasticsearch was not created. Hence, rest of cleanup is not necessary.
				return
			}
			Expect(err).NotTo(HaveOccurred())
		}

		if es.Spec.TerminationPolicy == api.TerminationPolicyPause {
			By("Wait for elasticsearch to be paused")
			f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

			By("Set DormantDatabase Spec.WipeOut to true")
			_, err = f.PatchDormantDatabase(elasticsearch.ObjectMeta, func(in *api.DormantDatabase) *api.DormantDatabase {
				in.Spec.WipeOut = true
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Delete Dormant Database")
			err = f.DeleteDormantDatabase(elasticsearch.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		}

		By("Wait for elasticsearch resources to be wipedOut")
		f.EventuallyWipedOut(elasticsearch.ObjectMeta).Should(Succeed())
	}

	AfterEach(func() {
		// Delete test resource
		deleteTestResource()

		for _, es := range garbageElasticsearch.Items {
			*elasticsearch = es
			// Delete test resource
			deleteTestResource()
		}

		if !skipSnapshotDataChecking {
			By("Check for snapshot data")
			f.EventuallySnapshotDataFound(snapshot).Should(BeFalse())
		}

		if secret != nil {
			err := f.DeleteSecret(secret.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		}

		if snapshotPVC != nil {
			err := f.DeletePersistentVolumeClaim(snapshotPVC.ObjectMeta)
			if err != nil && !kerr.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		}
	})

	// if secret is empty (no .env file) then skip
	JustBeforeEach(func() {
		if secret != nil && len(secret.Data) == 0 && (snapshot != nil && snapshot.Spec.Local == nil) {
			Skip("Missing repository credential")
		}
	})

	Describe("Test", func() {

		Context("General", func() {

			Context("-", func() {

				var shouldRunSuccessfully = func() {
					if skipMessage != "" {
						Skip(skipMessage)
					}

					// create elasticsearch and insert data
					createAndInsertData()

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for elasticsearch to be paused")
					f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

					// create elasticsearch object again to resume it
					By("Create Elasticsearch: " + elasticsearch.Name)
					err = f.CreateElasticsearch(elasticsearch)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for DormantDatabase to be deleted")
					f.EventuallyDormantDatabase(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Wait for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Check for Elastic client")
					f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

					elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					defer elasticClient.Stop()
					defer f.Tunnel.Close()

					By("Checking new indices")
					f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
				}

				Context("with Default Resource", func() {
					It("should run successfully", shouldRunSuccessfully)
				})

				Context("Custom Resource", func() {
					BeforeEach(func() {
						elasticsearch.Spec.PodTemplate.Spec.Resources = core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceMemory: resource.MustParse("512Mi"),
							},
						}
					})

					It("should run successfully", shouldRunSuccessfully)
				})

				Context("with authPlugin disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
						elasticsearch.Spec.DisableSecurity = true
					})

					It("should run successfully", shouldRunSuccessfully)
				})

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should run successfully", shouldRunSuccessfully)
				})

				Context("Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
					})

					Context("with Default Resource", func() {

						It("should run successfully", shouldRunSuccessfully)
					})

					Context("Custom Resource", func() {
						BeforeEach(func() {
							elasticsearch.Spec.Topology.Client.Resources = core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse("128Mi"),
								},
							}
							elasticsearch.Spec.Topology.Master.Resources = core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse("128Mi"),
								},
							}
							elasticsearch.Spec.Topology.Data.Resources = core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse("128Mi"),
								},
							}
						})

						It("should run successfully", shouldRunSuccessfully)
					})

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should take Snapshot successfully", shouldRunSuccessfully)
					})

				})
			})

			Context("with custom SA Name", func() {
				BeforeEach(func() {
					customSecret := f.SecretForDatabaseAuthentication(elasticsearch, false)
					elasticsearch.Spec.DatabaseSecret = &core.SecretVolumeSource{
						SecretName: customSecret.Name,
					}
					err := f.CreateSecret(customSecret)
					Expect(err).NotTo(HaveOccurred())
					elasticsearch.Spec.PodTemplate.Spec.ServiceAccountName = "my-custom-sa"
					elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyPause
				})

				It("should start and resume successfully", func() {
					//shouldTakeSnapshot()
					createAndWaitForRunning()
					if elasticsearch == nil {
						Skip("Skipping")
					}
					By("Check if Postgres " + elasticsearch.Name + " exists.")
					_, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
					if err != nil {
						if kerr.IsNotFound(err) {
							// Postgres was not created. Hence, rest of cleanup is not necessary.
							return
						}
						Expect(err).NotTo(HaveOccurred())
					}

					By("Delete elasticsearch: " + elasticsearch.Name)
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					if err != nil {
						if kerr.IsNotFound(err) {
							// Postgres was not created. Hence, rest of cleanup is not necessary.
							log.Infof("Skipping rest of cleanup. Reason: Postgres %s is not found.", elasticsearch.Name)
							return
						}
						Expect(err).NotTo(HaveOccurred())
					}

					By("Wait for elasticsearch to be paused")
					f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

					By("Resume DB")
					createAndWaitForRunning()
				})
			})

			Context("PDB", func() {

				It("should run eviction successfully", func() {
					// create elasticsearch
					By("Create DB")
					elasticsearch.Spec.Replicas = types.Int32P(3)
					elasticsearch.Spec.MaxUnavailable = &intstr.IntOrString{IntVal: 1}
					createAndWaitForRunning()
					//Evict a Elasticsearch pod
					By("Try to evict Pods")
					err := f.EvictPodsFromStatefulSet(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should run eviction on cluster successfully", func() {
					// create elasticsearch
					By("Create DB")
					elasticsearch = f.DedicatedElasticsearch()
					elasticsearch.Spec.Topology.Client.Replicas = types.Int32P(3)
					elasticsearch.Spec.Topology.Master.Replicas = types.Int32P(3)
					elasticsearch.Spec.Topology.Data.Replicas = types.Int32P(3)

					elasticsearch.Spec.Topology.Client.MaxUnavailable = &intstr.IntOrString{IntVal: 1}
					elasticsearch.Spec.Topology.Data.MaxUnavailable = &intstr.IntOrString{IntVal: 1}
					elasticsearch.Spec.Topology.Master.MaxUnavailable = &intstr.IntOrString{IntVal: 1}
					createAndWaitForRunning()
					//Evict a Elasticsearch pod
					By("Try to evict Pods")
					err := f.EvictPodsFromStatefulSet(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		Context("Snapshot", func() {
			BeforeEach(func() {
				skipSnapshotDataChecking = false
				snapshot.Spec.DatabaseName = elasticsearch.Name
			})

			var shouldTakeSnapshot = func() {
				// create elasticsearch and insert data
				createAndInsertData()

				By("Create Secret")
				err = f.CreateSecret(secret)
				Expect(err).NotTo(HaveOccurred())

				By("Create Snapshot")
				err = f.CreateSnapshot(snapshot)
				Expect(err).NotTo(HaveOccurred())

				By("Check for succeeded snapshot")
				f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseSucceeded))

				if !skipSnapshotDataChecking {
					By("Check for snapshot data")
					f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
				}
			}

			Context("For Custom Resources", func() {

				BeforeEach(func() {
					secret = f.SecretForGCSBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.GCS = &store.GCSSpec{
						Bucket: os.Getenv(GCS_BUCKET_NAME),
					}
				})

				Context("with custom SA", func() {
					var customSAForDB *core.ServiceAccount
					var customRoleForDB *rbac.Role
					var customRoleBindingForDB *rbac.RoleBinding
					var customSAForSnapshot *core.ServiceAccount
					var customRoleForSnapshot *rbac.Role
					var customRoleBindingForSnapshot *rbac.RoleBinding

					BeforeEach(func() {
						skipSnapshotDataChecking = false
						snapshot.Spec.StorageSecretName = secret.Name
						snapshot.Spec.DatabaseName = elasticsearch.Name
						//Get custom resources
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
						customSAForDB = f.ServiceAccount()
						elasticsearch.Spec.PodTemplate.Spec.ServiceAccountName = customSAForDB.Name
						customRoleForDB = f.RoleForElasticsearch(elasticsearch.ObjectMeta)
						customRoleBindingForDB = f.RoleBinding(customSAForDB.Name, customRoleForDB.Name)

						customSAForSnapshot = f.ServiceAccount()
						snapshot.Spec.PodTemplate.Spec.ServiceAccountName = customSAForSnapshot.Name
						customRoleForSnapshot = f.RoleForSnapshot(elasticsearch.ObjectMeta)
						customRoleBindingForSnapshot = f.RoleBinding(customSAForSnapshot.Name, customRoleForSnapshot.Name)

						//Create custom resources
						By("Create Database SA")
						err = f.CreateServiceAccount(customSAForDB)
						Expect(err).NotTo(HaveOccurred())
						By("Create Database Role")
						err = f.CreateRole(customRoleForDB)
						Expect(err).NotTo(HaveOccurred())
						By("Create Database RoleBinding")
						err = f.CreateRoleBinding(customRoleBindingForDB)
						Expect(err).NotTo(HaveOccurred())

						By("Create Snapshot SA")
						err = f.CreateServiceAccount(customSAForSnapshot)
						Expect(err).NotTo(HaveOccurred())
						By("Create Snapshot Role")
						err = f.CreateRole(customRoleForSnapshot)
						Expect(err).NotTo(HaveOccurred())
						By("Create Snapshot RoleBinding")
						err = f.CreateRoleBinding(customRoleBindingForSnapshot)
						Expect(err).NotTo(HaveOccurred())
					})

					It("should take snapshot to GCS successfully", func() {
						shouldTakeSnapshot()
					})

					It("should init from GCS snapshot successfully", func() {
						By("Start initializing From Snapshot")
						// create elasticsearch and insert data
						createAndInsertData()

						By("Create Secret")
						err := f.CreateSecret(secret)
						Expect(err).NotTo(HaveOccurred())

						By("Create Snapshot")
						err = f.CreateSnapshot(snapshot)
						Expect(err).NotTo(HaveOccurred())

						By("Check for succeeded snapshot")
						f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseSucceeded))

						if !skipSnapshotDataChecking {
							By("Check for snapshot data")
							f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
						}
						By("Getting old ES")
						oldElasticsearch, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						garbageElasticsearch.Items = append(garbageElasticsearch.Items, *oldElasticsearch)

						By("Create elasticsearch from snapshot")
						*elasticsearch = *f.CombinedElasticsearch()
						elasticsearch.Spec.Init = &api.InitSpec{
							SnapshotSource: &api.SnapshotSourceSpec{
								Namespace: snapshot.Namespace,
								Name:      snapshot.Name,
							},
						}

						By("Creating new role and bind with existing SA")
						By("Get new Role and RB")
						customRoleForReplayDB := f.RoleForElasticsearch(elasticsearch.ObjectMeta)
						customRoleBindingForReplayDB := f.RoleBinding(customSAForDB.Name, customRoleForReplayDB.Name)

						By("Create Database Role")
						err = f.CreateRole(customRoleForReplayDB)
						Expect(err).NotTo(HaveOccurred())
						By("Create Database RoleBinding")
						err = f.CreateRoleBinding(customRoleBindingForReplayDB)
						Expect(err).NotTo(HaveOccurred())

						By("Setting SA name in ES")
						elasticsearch.Spec.PodTemplate.Spec.ServiceAccountName = customSAForDB.Name

						// create and wait for running Elasticsearch
						createAndWaitForRunning()

						By("Check for Elastic client")
						f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

						elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())
						defer elasticClient.Stop()
						defer f.Tunnel.Close()

						By("Checking indices")
						f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
					})
				})

				Context("with custom Secret", func() {
					var kubedbSecret *core.Secret
					var customSecret *core.Secret

					BeforeEach(func() {
						customSecret = f.SecretForDatabaseAuthentication(elasticsearch, false)
						elasticsearch.Spec.DatabaseSecret = &core.SecretVolumeSource{
							SecretName: customSecret.Name,
						}
					})

					It("should keep secret after taking snapshot to GCS  successfully", func() {
						By("Create Database Secret")
						err := f.CreateSecret(customSecret)
						Expect(err).NotTo(HaveOccurred())

						By("Take snapshot")
						shouldTakeSnapshot()
						By("Delete test resources")
						deleteTestResource()
						By("Check if custom secret exists")
						err = f.CheckSecret(customSecret)
						Expect(err).NotTo(HaveOccurred())
					})

					It("should remove secret after taking snapshot to GCS successfully", func() {
						kubedbSecret = f.SecretForDatabaseAuthentication(elasticsearch, true)
						elasticsearch.Spec.DatabaseSecret = &core.SecretVolumeSource{
							SecretName: kubedbSecret.Name,
						}
						By("Create Database Secret")
						err = f.CreateSecret(kubedbSecret)
						Expect(err).NotTo(HaveOccurred())

						By("Take snapshot")
						shouldTakeSnapshot()
						By("Check if custom secret id deleted")
						deleteTestResource()
						err = f.CheckSecret(kubedbSecret)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("In Local", func() {
				BeforeEach(func() {
					skipSnapshotDataChecking = true
					secret = f.SecretForLocalBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.Local = &store.LocalSpec{
						MountPath: "/repo",
						VolumeSource: core.VolumeSource{
							EmptyDir: &core.EmptyDirVolumeSource{},
						},
					}
				})

				Context("With EmptyDir as Snapshot's backend", func() {

					It("should take Snapshot successfully", shouldTakeSnapshot)
				})

				Context("With PVC as Snapshot's backend", func() {

					BeforeEach(func() {
						snapshotPVC = f.GetPersistentVolumeClaim()
						By("Creating PVC for local backend snapshot")
						err := f.CreatePersistentVolumeClaim(snapshotPVC)
						Expect(err).NotTo(HaveOccurred())

						snapshot.Spec.Local = &store.LocalSpec{
							MountPath: "/repo",
							VolumeSource: core.VolumeSource{
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
									ClaimName: snapshotPVC.Name,
								},
							},
						}
					})

					It("should delete Snapshot successfully", func() {
						shouldTakeSnapshot()

						By("Deleting Snapshot")
						err := f.DeleteSnapshot(snapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Waiting Snapshot to be deleted")
						f.EventuallySnapshot(snapshot.ObjectMeta).Should(BeFalse())
					})
				})

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should take Snapshot successfully", shouldTakeSnapshot)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						snapshot.Spec.DatabaseName = elasticsearch.Name
					})

					It("should take Snapshot successfully", shouldTakeSnapshot)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should take Snapshot successfully", shouldTakeSnapshot)
					})
				})
			})

			Context("In S3", func() {
				BeforeEach(func() {
					secret = f.SecretForS3Backend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.S3 = &store.S3Spec{
						Bucket: os.Getenv(S3_BUCKET_NAME),
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should take Snapshot successfully", shouldTakeSnapshot)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						snapshot.Spec.DatabaseName = elasticsearch.Name
					})
					It("should take Snapshot successfully", shouldTakeSnapshot)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should take Snapshot successfully", shouldTakeSnapshot)
					})
				})

				Context("Delete One Snapshot keeping others", func() {

					It("Delete One Snapshot keeping others", func() {
						// create and wait for running elasticsearch
						shouldTakeSnapshot()

						oldSnapshot := snapshot.DeepCopy()

						// New snapshot that has old snapshot's name in prefix
						snapshot.Name += "-2"

						By(fmt.Sprintf("Create Snapshot %v", snapshot.Name))
						err = f.CreateSnapshot(snapshot)
						Expect(err).NotTo(HaveOccurred())

						By("Check for Succeeded snapshot")
						f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseSucceeded))

						if !skipSnapshotDataChecking {
							By("Check for snapshot data")
							f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
						}

						// delete old snapshot
						By(fmt.Sprintf("Delete old Snapshot %v", oldSnapshot.Name))
						err = f.DeleteSnapshot(oldSnapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Waiting for old Snapshot to be deleted")
						f.EventuallySnapshot(oldSnapshot.ObjectMeta).Should(BeFalse())
						if !skipSnapshotDataChecking {
							By(fmt.Sprintf("Check data for old snapshot %v", oldSnapshot.Name))
							f.EventuallySnapshotDataFound(oldSnapshot).Should(BeFalse())
						}

						// check remaining snapshot
						By(fmt.Sprintf("Checking another Snapshot %v still exists", snapshot.Name))
						_, err = f.GetSnapshot(snapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						if !skipSnapshotDataChecking {
							By(fmt.Sprintf("Check data for remaining snapshot %v", snapshot.Name))
							f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
						}
					})
				})
			})

			Context("In GCS", func() {
				BeforeEach(func() {
					secret = f.SecretForGCSBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.GCS = &store.GCSSpec{
						Bucket: os.Getenv(GCS_BUCKET_NAME),
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)

				Context("faulty snapshot", func() {
					BeforeEach(func() {
						skipSnapshotDataChecking = true
						snapshot.Spec.StorageSecretName = secret.Name
						snapshot.Spec.GCS = &store.GCSSpec{
							Bucket: "nonexisting",
						}
					})
					It("snapshot should fail", func() {
						// create and wait for running Elasticsearch
						createAndWaitForRunning()

						By("Create Secret")
						err := f.CreateSecret(secret)
						Expect(err).NotTo(HaveOccurred())

						By("Create Snapshot")
						err = f.CreateSnapshot(snapshot)
						Expect(err).NotTo(HaveOccurred())

						By("Check for failed snapshot")
						f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseFailed))
					})
				})

				Context("Delete One Snapshot keeping others", func() {

					It("Delete One Snapshot keeping others", func() {
						// create and wait for running elasticsearch
						shouldTakeSnapshot()

						oldSnapshot := snapshot.DeepCopy()

						// New snapshot that has old snapshot's name in prefix
						snapshot.Name += "-2"

						By(fmt.Sprintf("Create Snapshot %v", snapshot.Name))
						err = f.CreateSnapshot(snapshot)
						Expect(err).NotTo(HaveOccurred())

						By("Check for Succeeded snapshot")
						f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseSucceeded))

						if !skipSnapshotDataChecking {
							By("Check for snapshot data")
							f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
						}

						// delete old snapshot
						By(fmt.Sprintf("Delete old Snapshot %v", oldSnapshot.Name))
						err = f.DeleteSnapshot(oldSnapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Waiting for old Snapshot to be deleted")
						f.EventuallySnapshot(oldSnapshot.ObjectMeta).Should(BeFalse())
						if !skipSnapshotDataChecking {
							By(fmt.Sprintf("Check data for old snapshot %v", oldSnapshot.Name))
							f.EventuallySnapshotDataFound(oldSnapshot).Should(BeFalse())
						}

						// check remaining snapshot
						By(fmt.Sprintf("Checking another Snapshot %v still exists", snapshot.Name))
						_, err = f.GetSnapshot(snapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						if !skipSnapshotDataChecking {
							By(fmt.Sprintf("Check data for remaining snapshot %v", snapshot.Name))
							f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
						}
					})
				})
			})

			Context("In Azure", func() {
				BeforeEach(func() {
					secret = f.SecretForAzureBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.Azure = &store.AzureSpec{
						Container: os.Getenv(AZURE_CONTAINER_NAME),
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)

				Context("Delete One Snapshot keeping others", func() {

					It("Delete One Snapshot keeping others", func() {
						// create and wait for running elasticsearch
						shouldTakeSnapshot()

						oldSnapshot := snapshot.DeepCopy()

						// New snapshot that has old snapshot's name in prefix
						snapshot.Name += "-2"

						By(fmt.Sprintf("Create Snapshot %v", snapshot.Name))
						err = f.CreateSnapshot(snapshot)
						Expect(err).NotTo(HaveOccurred())

						By("Check for Succeeded snapshot")
						f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseSucceeded))

						if !skipSnapshotDataChecking {
							By("Check for snapshot data")
							f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
						}

						// delete old snapshot
						By(fmt.Sprintf("Delete old Snapshot %v", oldSnapshot.Name))
						err = f.DeleteSnapshot(oldSnapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Waiting for old Snapshot to be deleted")
						f.EventuallySnapshot(oldSnapshot.ObjectMeta).Should(BeFalse())
						if !skipSnapshotDataChecking {
							By(fmt.Sprintf("Check data for old snapshot %v", oldSnapshot.Name))
							f.EventuallySnapshotDataFound(oldSnapshot).Should(BeFalse())
						}

						// check remaining snapshot
						By(fmt.Sprintf("Checking another Snapshot %v still exists", snapshot.Name))
						_, err = f.GetSnapshot(snapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						if !skipSnapshotDataChecking {
							By(fmt.Sprintf("Check data for remaining snapshot %v", snapshot.Name))
							f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
						}
					})
				})
			})

			Context("In Swift", func() {
				BeforeEach(func() {
					secret = f.SecretForSwiftBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.Swift = &store.SwiftSpec{
						Container: os.Getenv(SWIFT_CONTAINER_NAME),
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)
			})

			Context("Invalid Database Secret", func() {

				BeforeEach(func() {
					skipSnapshotDataChecking = true
					secret = f.SecretForLocalBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.Local = &store.LocalSpec{
						MountPath: "/repo",
						VolumeSource: core.VolumeSource{
							EmptyDir: &core.EmptyDirVolumeSource{},
						},
					}
				})

				Context("In Local Backend", func() {

					It("should fail to take Snapshot", func() {

						By("Checking Snapshot succeed with valid database secret")
						shouldTakeSnapshot()

						By("Deleting succeeded snapshot")
						err := f.DeleteSnapshot(snapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Patching Database Secret to invalid one")
						es, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						dbSecret, err := f.KubeClient().CoreV1().Secrets(es.Namespace).Get(es.Spec.DatabaseSecret.SecretName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						_, _, err = core_util.PatchSecret(f.KubeClient(), dbSecret, func(in *core.Secret) *core.Secret {
							in.StringData = map[string]string{
								"ADMIN_PASSWORD": "invalid",
							}
							return in
						})
						Expect(err).NotTo(HaveOccurred())

						By("Create Snapshot")
						err = f.CreateSnapshot(snapshot)
						Expect(err).NotTo(HaveOccurred())

						By("Check for failed snapshot")
						f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseFailed))

						By("Check for failure reason: Unauthorized")
						snap, err := f.GetSnapshot(snapshot.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						Expect(snap.Status.Reason).Should(ContainSubstring("Unauthorized"))
					})
				})

			})

			Context("Snapshot PodVolume Template - In S3", func() {

				BeforeEach(func() {
					secret = f.SecretForS3Backend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.S3 = &store.S3Spec{
						Bucket: os.Getenv(S3_BUCKET_NAME),
					}
				})

				var shouldHandleJobVolumeSuccessfully = func() {
					// create elasticsearch and insert data
					createAndInsertData()

					By("Get Elasticsearch")
					es, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					elasticsearch.Spec = es.Spec

					By("Create Secret")
					err = f.CreateSecret(secret)
					Expect(err).NotTo(HaveOccurred())

					// determine pvcSpec and storageType for job
					// start
					pvcSpec := snapshot.Spec.PodVolumeClaimSpec
					if pvcSpec == nil {
						if elasticsearch.Spec.Topology != nil {
							pvcSpec = elasticsearch.Spec.Topology.Data.Storage
						} else {
							pvcSpec = elasticsearch.Spec.Storage
						}
					}
					st := snapshot.Spec.StorageType
					if st == nil {
						st = &elasticsearch.Spec.StorageType
					}
					Expect(st).NotTo(BeNil())
					// end

					By("Create Snapshot")
					err = f.CreateSnapshot(snapshot)
					if *st == api.StorageTypeDurable && pvcSpec == nil {
						By("Create Snapshot should have failed")
						Expect(err).Should(HaveOccurred())
						return
					} else {
						Expect(err).NotTo(HaveOccurred())
					}

					By("Get Snapshot")
					snap, err := f.GetSnapshot(snapshot.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					snapshot.Spec = snap.Spec

					if *st == api.StorageTypeEphemeral {
						storageSize := "0"
						if pvcSpec != nil {
							if sz, found := pvcSpec.Resources.Requests[core.ResourceStorage]; found {
								storageSize = sz.String()
							}
						}
						By(fmt.Sprintf("Check for Job Empty volume size: %v", storageSize))
						f.EventuallyJobVolumeEmptyDirSize(snapshot.ObjectMeta).Should(Equal(storageSize))
					} else if *st == api.StorageTypeDurable {
						sz, found := pvcSpec.Resources.Requests[core.ResourceStorage]
						Expect(found).NotTo(BeFalse())

						By("Check for Job PVC Volume size from snapshot")
						f.EventuallyJobPVCSize(snapshot.ObjectMeta).Should(Equal(sz.String()))
					}

					By("Check for succeeded snapshot")
					f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseSucceeded))

					if !skipSnapshotDataChecking {
						By("Check for snapshot data")
						f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
					}
				}

				// db StorageType Scenarios
				// ==============> Start
				var dbStorageTypeScenarios = func() {
					Context("DBStorageType - Durable", func() {
						BeforeEach(func() {
							elasticsearch.Spec.StorageType = api.StorageTypeDurable
							storage := core.PersistentVolumeClaimSpec{
								Resources: core.ResourceRequirements{
									Requests: core.ResourceList{
										core.ResourceStorage: resource.MustParse(framework.DBPvcStorageSize),
									},
								},
								StorageClassName: types.StringP(root.StorageClass),
							}
							if elasticsearch.Spec.Topology != nil {
								elasticsearch.Spec.Topology.Client.Storage = &storage
								elasticsearch.Spec.Topology.Data.Storage = &storage
								elasticsearch.Spec.Topology.Master.Storage = &storage
							} else {
								elasticsearch.Spec.Storage = &storage
							}
						})

						It("should Handle Job Volume Successfully", shouldHandleJobVolumeSuccessfully)
					})

					Context("DBStorageType - Ephemeral", func() {
						BeforeEach(func() {
							elasticsearch.Spec.StorageType = api.StorageTypeEphemeral
							elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
						})

						Context("DBPvcSpec is nil", func() {
							BeforeEach(func() {
								elasticsearch.Spec.Storage = nil
								if elasticsearch.Spec.Topology != nil {
									elasticsearch.Spec.Topology.Client.Storage = nil
									elasticsearch.Spec.Topology.Data.Storage = nil
									elasticsearch.Spec.Topology.Master.Storage = nil
								} else {
									elasticsearch.Spec.Storage = nil
								}
							})

							It("should Handle Job Volume Successfully", shouldHandleJobVolumeSuccessfully)
						})

						Context("DBPvcSpec is given [not nil]", func() {
							BeforeEach(func() {
								storage := core.PersistentVolumeClaimSpec{
									Resources: core.ResourceRequirements{
										Requests: core.ResourceList{
											core.ResourceStorage: resource.MustParse(framework.DBPvcStorageSize),
										},
									},
									StorageClassName: types.StringP(root.StorageClass),
								}
								if elasticsearch.Spec.Topology != nil {
									elasticsearch.Spec.Topology.Client.Storage = &storage
									elasticsearch.Spec.Topology.Data.Storage = &storage
									elasticsearch.Spec.Topology.Master.Storage = &storage
								} else {
									elasticsearch.Spec.Storage = &storage
								}
							})

							It("should Handle Job Volume Successfully", shouldHandleJobVolumeSuccessfully)
						})
					})
				}
				// End <==============

				// Snapshot PVC Scenarios
				// ==============> Start
				var snapshotPvcScenarios = func() {
					Context("Snapshot PVC is given [not nil]", func() {
						BeforeEach(func() {
							snapshot.Spec.PodVolumeClaimSpec = &core.PersistentVolumeClaimSpec{
								Resources: core.ResourceRequirements{
									Requests: core.ResourceList{
										core.ResourceStorage: resource.MustParse(framework.JobPvcStorageSize),
									},
								},
								StorageClassName: types.StringP(root.StorageClass),
							}
						})

						dbStorageTypeScenarios()
					})

					Context("Snapshot PVC is nil", func() {
						BeforeEach(func() {
							snapshot.Spec.PodVolumeClaimSpec = nil
						})

						dbStorageTypeScenarios()
					})
				}
				// End <==============

				Context("Snapshot StorageType is nil", func() {
					BeforeEach(func() {
						snapshot.Spec.StorageType = nil
					})

					Context("-", func() {
						snapshotPvcScenarios()
					})

					Context("with Dedicated elasticsearch", func() {
						BeforeEach(func() {
							elasticsearch = f.DedicatedElasticsearch()
							snapshot.Spec.DatabaseName = elasticsearch.Name
						})
						snapshotPvcScenarios()
					})
				})

				Context("Snapshot StorageType is Ephemeral", func() {
					BeforeEach(func() {
						ephemeral := api.StorageTypeEphemeral
						snapshot.Spec.StorageType = &ephemeral
					})

					Context("-", func() {
						snapshotPvcScenarios()
					})

					Context("with Dedicated elasticsearch", func() {
						BeforeEach(func() {
							elasticsearch = f.DedicatedElasticsearch()
							snapshot.Spec.DatabaseName = elasticsearch.Name
						})
						snapshotPvcScenarios()
					})
				})

				Context("Snapshot StorageType is Durable", func() {
					BeforeEach(func() {
						durable := api.StorageTypeDurable
						snapshot.Spec.StorageType = &durable
					})

					Context("-", func() {
						snapshotPvcScenarios()
					})

					Context("with Dedicated elasticsearch", func() {
						BeforeEach(func() {
							elasticsearch = f.DedicatedElasticsearch()
							snapshot.Spec.DatabaseName = elasticsearch.Name
						})
						snapshotPvcScenarios()
					})
				})
			})
		})

		Context("Initialize", func() {

			BeforeEach(func() {
				skipSnapshotDataChecking = false
				secret = f.SecretForS3Backend()
				snapshot.Spec.StorageSecretName = secret.Name
				snapshot.Spec.S3 = &store.S3Spec{
					Bucket: os.Getenv(S3_BUCKET_NAME),
				}
				snapshot.Spec.DatabaseName = elasticsearch.Name
			})

			var shouldInitialize = func() {
				// create and wait for running Elasticsearch
				createAndInsertData()

				By("Create Secret")
				err := f.CreateSecret(secret)
				Expect(err).NotTo(HaveOccurred())

				By("Create Snapshot")
				err = f.CreateSnapshot(snapshot)
				Expect(err).NotTo(HaveOccurred())

				By("Check for succeeded snapshot")
				f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseSucceeded))

				if !skipSnapshotDataChecking {
					By("Check for snapshot data")
					f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
				}

				oldElasticsearch, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				garbageElasticsearch.Items = append(garbageElasticsearch.Items, *oldElasticsearch)

				By("Create elasticsearch from snapshot")
				*elasticsearch = *f.CombinedElasticsearch()
				elasticsearch.Spec.Init = &api.InitSpec{
					SnapshotSource: &api.SnapshotSourceSpec{
						Namespace: snapshot.Namespace,
						Name:      snapshot.Name,
					},
				}

				// create and wait for running Elasticsearch
				createAndWaitForRunning()

				By("Check for Elastic client")
				f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

				elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				defer elasticClient.Stop()
				defer f.Tunnel.Close()

				By("Checking indices")
				f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
			}

			Context("-", func() {
				It("should initialize database successfully", shouldInitialize)
			})

			Context("with local volume", func() {

				BeforeEach(func() {
					snapshotPVC = f.GetPersistentVolumeClaim()
					By("Creating PVC for local backend snapshot")
					err := f.CreatePersistentVolumeClaim(snapshotPVC)
					Expect(err).NotTo(HaveOccurred())

					skipSnapshotDataChecking = true
					secret = f.SecretForLocalBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.Backend = store.Backend{
						Local: &store.LocalSpec{
							MountPath: "/repo",
							VolumeSource: core.VolumeSource{
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
									ClaimName: snapshotPVC.Name,
								},
							},
						},
					}
				})

				It("should initialize database successfully", shouldInitialize)

			})

			Context("with authPlugin disabled", func() {
				BeforeEach(func() {
					elasticsearch.Spec.EnableSSL = false
					elasticsearch.Spec.DisableSecurity = true
				})

				It("should run successfully", shouldInitialize)
			})

			Context("with SSL disabled", func() {
				BeforeEach(func() {
					elasticsearch.Spec.EnableSSL = false
				})

				It("should initialize database successfully", shouldInitialize)
			})

			Context("with Dedicated elasticsearch", func() {
				BeforeEach(func() {
					elasticsearch = f.DedicatedElasticsearch()
					snapshot.Spec.DatabaseName = elasticsearch.Name
				})

				It("should initialize database successfully", shouldInitialize)

				Context("with authPlugin disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
						elasticsearch.Spec.DisableSecurity = true
					})

					It("should run successfully", shouldInitialize)
				})

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should initialize database successfully", shouldInitialize)
				})
			})

			// To run this test,
			// 1st: Deploy stash latest operator
			// https://github.com/stashed/elasticsearch
			Context("With Stash/Restic", func() {
				var bc *stashV1beta1.BackupConfiguration
				var rs *stashV1beta1.RestoreSession
				var repo *stashV1alpha1.Repository

				BeforeEach(func() {
					skipSnapshotDataChecking = true
					if !f.FoundStashCRDs() {
						Skip("Skipping tests for stash integration. reason: stash operator is not running.")
					}
				})

				AfterEach(func() {
					By("Deleting BackupConfiguration")
					err := f.DeleteBackupConfiguration(bc.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Deleting RestoreSession")
					err = f.DeleteRestoreSession(rs.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Deleting Repository")
					err = f.DeleteRepository(repo.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
				})

				var createAndWaitForInitializing = func() {
					By("Creating Elasticsearch: " + elasticsearch.Name)
					err = f.CreateElasticsearch(elasticsearch)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for Initializing elasticsearch")
					f.EventuallyElasticsearchPhase(elasticsearch.ObjectMeta).Should(Equal(api.DatabasePhaseInitializing))

					By("Wait for AppBinding to create")
					f.EventuallyAppBinding(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Check valid AppBinding Specs")
					err = f.CheckAppBindingSpec(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
				}

				var shouldInitializeFromStash = func() {

					// create elasticsearch and insert data
					createAndInsertData()

					By("Create Secret")
					err = f.CreateSecret(secret)
					Expect(err).NotTo(HaveOccurred())

					By("Create Stash-Repositories")
					err = f.CreateRepository(repo)
					Expect(err).NotTo(HaveOccurred())

					By("Create Stash-BackupConfiguration")
					err = f.CreateBackupConfiguration(bc)
					Expect(err).NotTo(HaveOccurred())

					By("Check for snapshot count in stash-repository")
					f.EventuallySnapshotInRepository(repo.ObjectMeta).Should(matcher.MoreThan(2))

					By("Pause BackupConfiguration scheduling")
					err = f.PauseBackupConfiguration(bc.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					oldElasticsearch, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					garbageElasticsearch.Items = append(garbageElasticsearch.Items, *oldElasticsearch)

					By("Create elasticsearch from stash")
					*elasticsearch = *f.CombinedElasticsearch()
					rs = f.RestoreSession(elasticsearch.ObjectMeta, oldElasticsearch.ObjectMeta)
					elasticsearch.Spec.DatabaseSecret = oldElasticsearch.Spec.DatabaseSecret
					elasticsearch.Spec.Init = &api.InitSpec{
						StashRestoreSession: &core.LocalObjectReference{
							Name: rs.Name,
						},
					}

					// create and wait for running Elasticsearch
					createAndWaitForInitializing()

					By("Create Stash-RestoreSession")
					err = f.CreateRestoreSession(rs)
					Expect(err).NotTo(HaveOccurred())

					// eventually backupsession succeeded
					By("Check for Succeeded restoreSession")
					f.EventuallyRestoreSessionPhase(rs.ObjectMeta).Should(Equal(stashV1beta1.RestoreSessionSucceeded))

					By("Wait for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Check for Elastic client")
					f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

					elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					defer elasticClient.Stop()
					defer f.Tunnel.Close()

					By("Checking indices")
					f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
				}

				Context("From GCS backend", func() {

					BeforeEach(func() {
						secret = f.SecretForGCSBackend()
						secret = f.PatchSecretForRestic(secret)
						bc = f.BackupConfiguration(elasticsearch.ObjectMeta)
						repo = f.Repository(elasticsearch.ObjectMeta, secret.Name)

						repo.Spec.Backend = store.Backend{
							GCS: &store.GCSSpec{
								Bucket: os.Getenv("GCS_BUCKET_NAME"),
								Prefix: fmt.Sprintf("stash/%v/%v", elasticsearch.Namespace, elasticsearch.Name),
							},
							StorageSecretName: secret.Name,
						}
					})

					It("should run successfully", shouldInitializeFromStash)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should take Snapshot successfully", shouldInitializeFromStash)
					})

					Context("with Dedicated elasticsearch", func() {
						BeforeEach(func() {
							elasticsearch = f.DedicatedElasticsearch()
							bc = f.BackupConfiguration(elasticsearch.ObjectMeta)
							repo = f.Repository(elasticsearch.ObjectMeta, secret.Name)

							repo.Spec.Backend = store.Backend{
								GCS: &store.GCSSpec{
									Bucket: os.Getenv("GCS_BUCKET_NAME"),
									Prefix: fmt.Sprintf("stash/%v/%v", elasticsearch.Namespace, elasticsearch.Name),
								},
								StorageSecretName: secret.Name,
							}
						})

						It("should take Snapshot successfully", shouldInitializeFromStash)

						Context("with SSL disabled", func() {
							BeforeEach(func() {
								elasticsearch.Spec.EnableSSL = false
							})

							It("should take Snapshot successfully", shouldInitializeFromStash)
						})
					})
				})

			})
		})

		Context("Resume", func() {
			var usedInitialized bool
			BeforeEach(func() {
				usedInitialized = false
			})

			var shouldResumeSuccessfully = func() {
				// create and wait for running Elasticsearch
				createAndWaitForRunning()

				By("Delete elasticsearch")
				err := f.DeleteElasticsearch(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Wait for elasticsearch to be paused")
				f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

				// create elasticsearch object again to resume it
				By("Create Elasticsearch: " + elasticsearch.Name)
				err = f.CreateElasticsearch(elasticsearch)
				Expect(err).NotTo(HaveOccurred())

				By("Wait for DormantDatabase to be deleted")
				f.EventuallyDormantDatabase(elasticsearch.ObjectMeta).Should(BeFalse())

				By("Wait for Running elasticsearch")
				f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

				es, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				*elasticsearch = *es
				if usedInitialized {
					_, ok := elasticsearch.Annotations[api.AnnotationInitialized]
					Expect(ok).Should(BeTrue())
				}
			}

			Context("-", func() {
				It("should resume DormantDatabase successfully", shouldResumeSuccessfully)
			})

			Context("with SSL disabled", func() {
				BeforeEach(func() {
					elasticsearch.Spec.EnableSSL = false
				})

				It("should initialize database successfully", shouldResumeSuccessfully)
			})

			Context("with Dedicated elasticsearch", func() {
				BeforeEach(func() {
					elasticsearch = f.DedicatedElasticsearch()
					snapshot.Spec.DatabaseName = elasticsearch.Name
				})

				It("should initialize database successfully", shouldResumeSuccessfully)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should initialize database successfully", shouldResumeSuccessfully)
				})
			})
		})

		Context("SnapshotScheduler", func() {
			AfterEach(func() {
				err := f.DeleteSecret(secret.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
			})

			BeforeEach(func() {
				secret = f.SecretForLocalBackend()
				snapshot = nil
			})

			Context("With Startup", func() {
				BeforeEach(func() {
					elasticsearch.Spec.BackupSchedule = &api.BackupScheduleSpec{
						CronExpression: "@every 1m",
						Backend: store.Backend{
							StorageSecretName: secret.Name,
							Local: &store.LocalSpec{
								MountPath: "/repo",
								VolumeSource: core.VolumeSource{
									EmptyDir: &core.EmptyDirVolumeSource{},
								},
							},
						},
					}
				})

				var shouldStartupSchedular = func() {
					// create and wait for running Elasticsearch
					createAndWaitForRunning()

					By("Create Secret")
					err := f.CreateSecret(secret)
					Expect(err).NotTo(HaveOccurred())

					By("Count multiple Snapshot")
					f.EventuallySnapshotCount(elasticsearch.ObjectMeta).Should(matcher.MoreThan(3))
				}

				Context("-", func() {
					It("should run schedular successfully", shouldStartupSchedular)
				})

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should run schedular successfully", shouldStartupSchedular)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						elasticsearch.Spec.BackupSchedule = &api.BackupScheduleSpec{
							CronExpression: "@every 1m",
							Backend: store.Backend{
								StorageSecretName: secret.Name,
								Local: &store.LocalSpec{
									MountPath: "/repo",
									VolumeSource: core.VolumeSource{
										EmptyDir: &core.EmptyDirVolumeSource{},
									},
								},
							},
						}
					})
					It("should run schedular successfully", shouldStartupSchedular)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should run schedular successfully", shouldStartupSchedular)
					})
				})
			})

			Context("With Update", func() {
				var shouldScheduleWithUpdate = func() {
					// create and wait for running Elasticsearch
					createAndWaitForRunning()

					By("Create Secret")
					err := f.CreateSecret(secret)
					Expect(err).NotTo(HaveOccurred())

					By("Update elasticsearch")
					_, err = f.TryPatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.BackupSchedule = &api.BackupScheduleSpec{
							CronExpression: "@every 1m",
							Backend: store.Backend{
								StorageSecretName: secret.Name,
								Local: &store.LocalSpec{
									MountPath: "/repo",
									VolumeSource: core.VolumeSource{
										EmptyDir: &core.EmptyDirVolumeSource{},
									},
								},
							},
						}

						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Count multiple Snapshot")
					f.EventuallySnapshotCount(elasticsearch.ObjectMeta).Should(matcher.MoreThan(3))
				}
				Context("-", func() {
					It("should run schedular successfully", shouldScheduleWithUpdate)
				})
			})
		})

		Context("Termination Policy", func() {
			BeforeEach(func() {
				skipSnapshotDataChecking = false
				secret = f.SecretForS3Backend()
				snapshot.Spec.StorageSecretName = secret.Name
				snapshot.Spec.S3 = &store.S3Spec{
					Bucket: os.Getenv(S3_BUCKET_NAME),
				}
				snapshot.Spec.DatabaseName = elasticsearch.Name
			})

			AfterEach(func() {
				if snapshot != nil {
					By("Delete Existing snapshot")
					err := f.DeleteSnapshot(snapshot.ObjectMeta)
					if err != nil {
						if kerr.IsNotFound(err) {
							// Elasticsearch was not created. Hence, rest of cleanup is not necessary.
							return
						}
						Expect(err).NotTo(HaveOccurred())
					}
				}
			})

			var shouldRunWithSnapshot = func() {
				// create elasticsearch and insert data
				createAndInsertData()

				By("Create Secret")
				err := f.CreateSecret(secret)
				Expect(err).NotTo(HaveOccurred())

				By("Create Snapshot")
				err = f.CreateSnapshot(snapshot)
				Expect(err).NotTo(HaveOccurred())

				By("Check for succeeded snapshot")
				f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(api.SnapshotPhaseSucceeded))

				if !skipSnapshotDataChecking {
					By("Check for snapshot data")
					f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
				}
			}

			Context("with TerminationPolicyDoNotTerminate", func() {

				BeforeEach(func() {
					skipSnapshotDataChecking = true
					elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyDoNotTerminate
				})

				It("should work successfully", func() {
					// create and wait for running Elasticsearch
					createAndWaitForRunning()

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).Should(HaveOccurred())

					By("Elasticsearch is not paused. Check for elasticsearch")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Check for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Update elasticsearch to set spec.terminationPolicy = Pause")
					_, err := f.TryPatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.TerminationPolicy = api.TerminationPolicyPause
						return in
					})
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with TerminationPolicyPause (default)", func() {
				var shouldRunWithTerminationPause = func() {
					shouldRunWithSnapshot()

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					// DormantDatabase.Status= paused, means elasticsearch object is deleted
					By("Wait for elasticsearch to be paused")
					f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

					By("Check for intact snapshot")
					_, err := f.GetSnapshot(snapshot.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					if !skipSnapshotDataChecking {
						By("Check for snapshot data")
						f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
					}

					// create elasticsearch object again to resume it
					By("Create (pause) Elasticsearch: " + elasticsearch.Name)
					err = f.CreateElasticsearch(elasticsearch)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for DormantDatabase to be deleted")
					f.EventuallyDormantDatabase(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Wait for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Check for Elastic client")
					f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

					elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					defer elasticClient.Stop()
					defer f.Tunnel.Close()

					By("Checking new indices")
					f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
				}

				It("should create dormantdatabase successfully", shouldRunWithTerminationPause)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should create dormantdatabase successfully", shouldRunWithTerminationPause)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						snapshot.Spec.DatabaseName = elasticsearch.Name
					})
					It("should initialize database successfully", shouldRunWithTerminationPause)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should initialize database successfully", shouldRunWithTerminationPause)
					})
				})
			})

			Context("with TerminationPolicyDelete", func() {
				BeforeEach(func() {
					elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyDelete
				})

				var shouldRunWithTerminationDelete = func() {
					shouldRunWithSnapshot()

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("wait until elasticsearch is deleted")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Checking DormantDatabase is not created")
					f.EventuallyDormantDatabase(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Check for deleted PVCs")
					f.EventuallyPVCCount(elasticsearch.ObjectMeta).Should(Equal(0))

					By("Check for intact Secrets")
					f.EventuallyDBSecretCount(elasticsearch.ObjectMeta).ShouldNot(Equal(0))

					By("Check for intact snapshot")
					_, err := f.GetSnapshot(snapshot.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					if !skipSnapshotDataChecking {
						By("Check for intact snapshot data")
						f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
					}

					By("Delete snapshot")
					err = f.DeleteSnapshot(snapshot.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					if !skipSnapshotDataChecking {
						By("Check for deleted snapshot data")
						f.EventuallySnapshotDataFound(snapshot).Should(BeFalse())
					}
				}

				It("should run with TerminationPolicyDelete", shouldRunWithTerminationDelete)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})
					It("should run with TerminationPolicyDelete", shouldRunWithTerminationDelete)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyDelete
						snapshot.Spec.DatabaseName = elasticsearch.Name
					})
					It("should initialize database successfully", shouldRunWithTerminationDelete)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should initialize database successfully", shouldRunWithTerminationDelete)
					})
				})
			})

			Context("with TerminationPolicyWipeOut", func() {

				BeforeEach(func() {
					elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
				})

				var shouldRunWithTerminationWipeOut = func() {
					shouldRunWithSnapshot()

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("wait until elasticsearch is deleted")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Checking DormantDatabase is not created")
					f.EventuallyDormantDatabase(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Check for deleted PVCs")
					f.EventuallyPVCCount(elasticsearch.ObjectMeta).Should(Equal(0))

					By("Check for deleted Secrets")
					f.EventuallyDBSecretCount(elasticsearch.ObjectMeta).Should(Equal(0))

					By("Check for deleted Snapshots")
					f.EventuallySnapshotCount(snapshot.ObjectMeta).Should(Equal(0))

					if !skipSnapshotDataChecking {
						By("Check for deleted snapshot data")
						f.EventuallySnapshotDataFound(snapshot).Should(BeFalse())
					}
				}

				It("should run with TerminationPolicyWipeOut", shouldRunWithTerminationWipeOut)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})
					It("should run with TerminationPolicyDelete", shouldRunWithTerminationWipeOut)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						snapshot.Spec.DatabaseName = elasticsearch.Name
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
					})
					It("should initialize database successfully", shouldRunWithTerminationWipeOut)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should initialize database successfully", shouldRunWithTerminationWipeOut)
					})
				})
			})
		})

		Context("Environment Variables", func() {

			allowedEnvList := []core.EnvVar{
				{
					Name:  "CLUSTER_NAME",
					Value: "kubedb-es-e2e-cluster",
				},
				{
					Name:  "NUMBER_OF_MASTERS",
					Value: "1",
				},
				{
					Name:  "ES_JAVA_OPTS",
					Value: "-Xms256m -Xmx256m",
				},
				{
					Name:  "REPO_LOCATIONS",
					Value: "/backup",
				},
				{
					Name:  "MEMORY_LOCK",
					Value: "true",
				},
				{
					Name:  "HTTP_ENABLE",
					Value: "true",
				},
			}

			forbiddenEnvList := []core.EnvVar{
				{
					Name:  "NODE_NAME",
					Value: "kubedb-es-e2e-node",
				},
				{
					Name:  "NODE_MASTER",
					Value: "true",
				},
				{
					Name:  "NODE_DATA",
					Value: "true",
				},
			}

			var shouldRunSuccessfully = func() {
				if skipMessage != "" {
					Skip(skipMessage)
				}

				// create elasticsearch and insert data
				createAndInsertData()

				By("Delete elasticsearch")
				err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Wait for elasticsearch to be paused")
				f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

				// create elasticsearch object again to resume it
				By("Create Elasticsearch: " + elasticsearch.Name)
				err = f.CreateElasticsearch(elasticsearch)
				Expect(err).NotTo(HaveOccurred())

				By("Wait for DormantDatabase to be deleted")
				f.EventuallyDormantDatabase(elasticsearch.ObjectMeta).Should(BeFalse())

				By("Wait for Running elasticsearch")
				f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

				By("Check for Elastic client")
				f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

				elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				defer elasticClient.Stop()
				defer f.Tunnel.Close()

				By("Checking new indices")
				f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
			}

			Context("With allowed Envs", func() {

				var shouldRunWithAllowedEnvs = func() {
					elasticsearch.Spec.PodTemplate.Spec.Env = allowedEnvList
					shouldRunSuccessfully()

					podName := f.GetClientPodName(elasticsearch)

					By("Checking pod started with given envs")
					pod, err := f.KubeClient().CoreV1().Pods(elasticsearch.Namespace).Get(podName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					out, err := exec_util.ExecIntoPod(f.RestConfig(), pod, exec_util.Command("env"))
					Expect(err).NotTo(HaveOccurred())
					for _, env := range allowedEnvList {
						Expect(out).Should(ContainSubstring(env.Name + "=" + env.Value))
					}
				}

				Context("-", func() {
					It("should run successfully with given envs", shouldRunWithAllowedEnvs)
				})

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should run successfully with given envs", shouldRunWithAllowedEnvs)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						snapshot.Spec.DatabaseName = elasticsearch.Name
					})

					It("should run successfully with given envs", shouldRunWithAllowedEnvs)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should run successfully with given envs", shouldRunWithAllowedEnvs)
					})
				})
			})

			Context("With forbidden Envs", func() {

				It("should reject to create Elasticsearch CRD", func() {
					for _, env := range forbiddenEnvList {
						elasticsearch.Spec.PodTemplate.Spec.Env = []core.EnvVar{
							env,
						}

						By("Creating Elasticsearch with " + env.Name + " env var.")
						err := f.CreateElasticsearch(elasticsearch)
						Expect(err).To(HaveOccurred())
					}
				})
			})

			Context("Update Envs", func() {

				It("should not reject to update Envs", func() {
					elasticsearch.Spec.PodTemplate.Spec.Env = allowedEnvList

					shouldRunSuccessfully()

					By("Updating Envs")
					_, _, err := util.PatchElasticsearch(f.ExtClient().KubedbV1alpha1(), elasticsearch, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.PodTemplate.Spec.Env = []core.EnvVar{
							{
								Name:  "CLUSTER_NAME",
								Value: "kubedb-es-e2e-cluster-patched",
							},
						}
						return in
					})
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		Context("Custom Configuration", func() {

			var userConfig *core.ConfigMap

			var shouldRunWithCustomConfig = func() {
				userConfig.Data = map[string]string{
					"common-config.yaml": f.GetCommonConfig(elasticsearch),
					"master-config.yaml": f.GetMasterConfig(elasticsearch),
					"client-config.yaml": f.GetClientConfig(elasticsearch),
					"data-config.yaml":   f.GetDataConfig(elasticsearch),
				}

				By("Creating configMap: " + userConfig.Name)
				err := f.CreateConfigMap(userConfig)
				Expect(err).NotTo(HaveOccurred())

				elasticsearch.Spec.ConfigSource = &core.VolumeSource{
					ConfigMap: &core.ConfigMapVolumeSource{
						LocalObjectReference: core.LocalObjectReference{
							Name: userConfig.Name,
						},
					},
				}

				// create elasticsearch
				createAndWaitForRunning()

				By("Check for Elastic client")
				f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

				elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Reading Nodes information")
				settings, err := elasticClient.GetAllNodesInfo()
				Expect(err).NotTo(HaveOccurred())

				By("Checking nodes are using provided config")
				Expect(f.IsUsingProvidedConfig(elasticsearch, settings)).Should(BeTrue())

				elasticClient.Stop()
				f.Tunnel.Close()
			}

			Context("With Topology", func() {
				BeforeEach(func() {
					elasticsearch = f.DedicatedElasticsearch()
					userConfig = f.GetCustomConfig()
				})

				AfterEach(func() {
					By("Deleting configMap: " + userConfig.Name)
					err := f.DeleteConfigMap(userConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should use config provided in config files", shouldRunWithCustomConfig)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should run successfully with given envs", shouldRunWithCustomConfig)
				})
			})

			Context("Without Topology", func() {
				BeforeEach(func() {
					userConfig = f.GetCustomConfig()
				})

				AfterEach(func() {
					By("Deleting configMap: " + userConfig.Name)
					err := f.DeleteConfigMap(userConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should use config provided in config files", shouldRunWithCustomConfig)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should run successfully with given envs", shouldRunWithCustomConfig)
				})
			})
		})

		Context("StorageType ", func() {

			var shouldRunSuccessfully = func() {

				if skipMessage != "" {
					Skip(skipMessage)
				}

				// create elasticsearch and insert data
				createAndInsertData()

			}

			Context("Ephemeral", func() {

				Context("Combined Elasticsearch", func() {

					BeforeEach(func() {
						elasticsearch.Spec.StorageType = api.StorageTypeEphemeral
						elasticsearch.Spec.Storage = nil
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
					})

					It("should run successfully", shouldRunSuccessfully)
				})

				Context("Dedicated Elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						elasticsearch.Spec.StorageType = api.StorageTypeEphemeral
						elasticsearch.Spec.Topology.Master.Storage = nil
						elasticsearch.Spec.Topology.Client.Storage = nil
						elasticsearch.Spec.Topology.Data.Storage = nil
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
					})

					It("should run successfully", shouldRunSuccessfully)
				})

				Context("With TerminationPolicyPause", func() {

					BeforeEach(func() {
						elasticsearch.Spec.StorageType = api.StorageTypeEphemeral
						elasticsearch.Spec.Storage = nil
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyPause
					})

					It("should reject to create Elasticsearch object", func() {

						By("Creating Elasticsearch: " + elasticsearch.Name)
						err := f.CreateElasticsearch(elasticsearch)
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})
	})
})

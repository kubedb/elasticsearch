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
	"context"
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
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	exec_util "kmodules.xyz/client-go/tools/exec"
	store "kmodules.xyz/objectstore-api/api/v1"
	stashV1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	stashV1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
)

var _ = Describe("Elasticsearch", func() {
	var (
		err                  error
		f                    *framework.Invocation
		elasticsearch        *api.Elasticsearch
		garbageElasticsearch *api.ElasticsearchList
		secret               *core.Secret
		skipMessage          string
		indicesCount         int
	)

	BeforeEach(func() {
		f = root.Invoke()
		elasticsearch = f.CombinedElasticsearch()
		garbageElasticsearch = new(api.ElasticsearchList)
		indicesCount = 0
		secret = nil
		skipMessage = ""
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

		By("Check for Elastic client")
		f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())
	}

	var createAndInsertData = func() {

		createAndWaitForRunning()

		elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		defer elasticClient.Stop()
		defer f.Tunnel.Close()

		By("Creating new indices")
		err = elasticClient.CreateIndex(2)
		Expect(err).NotTo(HaveOccurred())
		indicesCount += 2

		By("Checking new indices")
		f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
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

		By("Update elasticsearch to set spec.terminationPolicy = WipeOut")
		_, err = f.PatchElasticsearch(es.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
			in.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
			return in
		})
		Expect(err).NotTo(HaveOccurred())

		By("Delete elasticsearch: " + elasticsearch.Name)
		err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
		if err != nil {
			if kerr.IsNotFound(err) {
				// Elasticsearch was not created. Hence, rest of cleanup is not necessary.
				return
			}
			Expect(err).NotTo(HaveOccurred())
		}

		By("Wait for elasticsearch to be deleted")
		f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeFalse())

		By("Wait for elasticsearch resources to be wipedOut")
		f.EventuallyWipedOut(elasticsearch.ObjectMeta).Should(Succeed())
	}

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			f.PrintDebugHelpers()
		}
	})

	AfterEach(func() {
		// Delete test resource
		deleteTestResource()

		for _, es := range garbageElasticsearch.Items {
			*elasticsearch = es
			// Delete test resource
			deleteTestResource()
		}

		if secret != nil {
			err := f.DeleteSecret(secret.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
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

					By("Halt Elasticsearch: Update elasticsearch to set spec.halted = true")
					_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.Halted = true
						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Wait for halted/paused elasticsearch")
					f.EventuallyElasticsearchPhase(elasticsearch.ObjectMeta).Should(Equal(api.DatabasePhaseHalted))

					By("Resume Elasticsearch: Update elasticsearch to set spec.halted = false")
					_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.Halted = false
						return in
					})
					Expect(err).NotTo(HaveOccurred())

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
									core.ResourceMemory: resource.MustParse("256Mi"),
								},
							}
							elasticsearch.Spec.Topology.Master.Resources = core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse("256Mi"),
								},
							}
							elasticsearch.Spec.Topology.Data.Resources = core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse("256Mi"),
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
					elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyHalt
				})

				It("should start and resume successfully", func() {
					//shouldTakeSnapshot()
					createAndWaitForRunning()
					if elasticsearch == nil {
						Skip("Skipping")
					}
					By("Check if Elasticsearch " + elasticsearch.Name + " exists.")
					_, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
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
							log.Infof("Skipping rest of cleanup. Reason: Elasticsearch %s is not found.", elasticsearch.Name)
							return
						}
						Expect(err).NotTo(HaveOccurred())
					}

					By("wait until elasticsearch is deleted")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeFalse())

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

		Context("Initialize", func() {

			// To run this test,
			// 1st: Deploy stash latest operator
			// https://github.com/stashed/elasticsearch
			Context("With Stash/Restic", func() {
				var bc *stashV1beta1.BackupConfiguration
				var rs *stashV1beta1.RestoreSession
				var repo *stashV1alpha1.Repository

				BeforeEach(func() {
					if !f.FoundStashCRDs() {
						Skip("Skipping tests for stash integration. reason: stash operator is not running.")
					}
				})

				AfterEach(func() {
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

					By("Check for Elastic client")
					f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())
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

					By("Delete BackupConfiguration to stop backup scheduling")
					err = f.DeleteBackupConfiguration(bc.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					oldElasticsearch, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					garbageElasticsearch.Items = append(garbageElasticsearch.Items, *oldElasticsearch)

					By("Create elasticsearch from stash")
					*elasticsearch = *f.CombinedElasticsearch()
					rs = f.RestoreSession(elasticsearch.ObjectMeta, repo)
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
						repo = f.Repository(elasticsearch.ObjectMeta)
						bc = f.BackupConfiguration(elasticsearch.ObjectMeta, repo)

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
							repo = f.Repository(elasticsearch.ObjectMeta)
							bc = f.BackupConfiguration(elasticsearch.ObjectMeta, repo)

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

				By("Halt Elasticsearch: Update elasticsearch to set spec.halted = true")
				_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
					in.Spec.Halted = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Wait for halted/paused elasticsearch")
				f.EventuallyElasticsearchPhase(elasticsearch.ObjectMeta).Should(Equal(api.DatabasePhaseHalted))

				By("Resume Elasticsearch: Update elasticsearch to set spec.halted = false")
				_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
					in.Spec.Halted = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

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

		Context("Termination Policy", func() {

			var shouldRunAndHalt = func() {
				// create elasticsearch and insert data
				createAndInsertData()

				By("Halt Elasticsearch: Update elasticsearch to set spec.halted = true")
				_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
					in.Spec.Halted = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Wait for halted/paused elasticsearch")
				f.EventuallyElasticsearchPhase(elasticsearch.ObjectMeta).Should(Equal(api.DatabasePhaseHalted))

				By("Resume Elasticsearch: Update elasticsearch to set spec.halted = false")
				_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
					in.Spec.Halted = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Wait for Running elasticsearch")
				f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

				By("Check for Elastic client")
				f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

				elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				defer elasticClient.Stop()
				defer f.Tunnel.Close()

				By("Checking existing indices")
				f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
			}

			Context("with TerminationPolicyDoNotTerminate", func() {

				BeforeEach(func() {
					elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyDoNotTerminate
				})

				It("should work successfully", func() {
					// create and wait for running Elasticsearch
					createAndWaitForRunning()

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).Should(HaveOccurred())

					By("Elasticsearch is not deleted. Check for elasticsearch")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Check for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Halt Elasticsearch: Update elasticsearch to set spec.halted = true")
					_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.Halted = true
						return in
					})
					Expect(err).To(HaveOccurred())

					By("Elasticsearch is not paused. Check for elasticsearch")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Check for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Update elasticsearch to set spec.terminationPolicy = halt")
					_, err := f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.TerminationPolicy = api.TerminationPolicyHalt
						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Halt Elasticsearch: Update elasticsearch to set spec.halted = true")
					_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.Halted = true
						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Wait for halted/paused elasticsearch")
					f.EventuallyElasticsearchPhase(elasticsearch.ObjectMeta).Should(Equal(api.DatabasePhaseHalted))

					By("Resume Elasticsearch: Update elasticsearch to set spec.halted = false")
					_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.Halted = false
						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Wait for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())
				})
			})

			Context("with TerminationPolicyHalt ", func() {

				var shouldRunWithTerminationHalt = func() {
					shouldRunAndHalt()

					By("Halt Elasticsearch: Update elasticsearch to set spec.halted = true")
					_, err := f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.Halted = true
						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Wait for halted/paused elasticsearch")
					f.EventuallyElasticsearchPhase(elasticsearch.ObjectMeta).Should(Equal(api.DatabasePhaseHalted))

					By("Resume Elasticsearch: Update elasticsearch to set spec.halted = false")
					_, err = f.PatchElasticsearch(elasticsearch.ObjectMeta, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.Halted = false
						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Wait for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("wait until elasticsearch is deleted")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeFalse())

					// create elasticsearch object again to resume it
					By("Create (pause) Elasticsearch: " + elasticsearch.Name)
					err = f.CreateElasticsearch(elasticsearch)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					By("Check for Elastic client")
					f.EventuallyElasticsearchClientReady(elasticsearch.ObjectMeta).Should(BeTrue())

					elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					defer elasticClient.Stop()
					defer f.Tunnel.Close()

					By("Checking existing indices")
					f.EventuallyElasticsearchIndicesCount(elasticClient).Should(Equal(f.IndicesCount(elasticsearch, indicesCount)))
				}

				It("should create dormantdatabase successfully", shouldRunWithTerminationHalt)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})

					It("should create dormantdatabase successfully", shouldRunWithTerminationHalt)
				})

				Context("with Dedicated elasticsearch", func() {

					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
					})

					It("should initialize database successfully", shouldRunWithTerminationHalt)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should initialize database successfully", shouldRunWithTerminationHalt)
					})
				})
			})

			Context("with TerminationPolicyDelete", func() {

				BeforeEach(func() {
					elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyDelete
				})

				var shouldRunWithTerminationDelete = func() {
					createAndInsertData()

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("wait until elasticsearch is deleted")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Check for deleted PVCs")
					f.EventuallyPVCCount(elasticsearch.ObjectMeta).Should(Equal(0))

					By("Check for intact Secrets")
					f.EventuallyDBSecretCount(elasticsearch.ObjectMeta).ShouldNot(Equal(0))
				}

				It("should run with TerminationPolicyDelete", shouldRunWithTerminationDelete)

				It("should pause with TerminationPolicyDelete", shouldRunAndHalt)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})
					It("should run with TerminationPolicyDelete", shouldRunWithTerminationDelete)

					It("should pause with TerminationPolicyDelete", shouldRunAndHalt)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyDelete
					})
					It("should initialize database successfully", shouldRunWithTerminationDelete)

					It("should pause with TerminationPolicyDelete", shouldRunAndHalt)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should initialize database successfully", shouldRunWithTerminationDelete)

						It("should pause with TerminationPolicyDelete", shouldRunAndHalt)
					})
				})
			})

			Context("with TerminationPolicyWipeOut", func() {

				BeforeEach(func() {
					elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
				})

				var shouldRunWithTerminationWipeOut = func() {
					createAndInsertData()

					By("Delete elasticsearch")
					err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("wait until elasticsearch is deleted")
					f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Check for deleted PVCs")
					f.EventuallyPVCCount(elasticsearch.ObjectMeta).Should(Equal(0))

					By("Check for deleted Secrets")
					f.EventuallyDBSecretCount(elasticsearch.ObjectMeta).Should(Equal(0))
				}

				It("should run with TerminationPolicyWipeOut", shouldRunWithTerminationWipeOut)

				It("should pause with TerminationPolicyWipeOut", shouldRunAndHalt)

				Context("with SSL disabled", func() {
					BeforeEach(func() {
						elasticsearch.Spec.EnableSSL = false
					})
					It("should run with TerminationPolicyDelete", shouldRunWithTerminationWipeOut)

					It("should pause with TerminationPolicyWipeOut", shouldRunAndHalt)
				})

				Context("with Dedicated elasticsearch", func() {
					BeforeEach(func() {
						elasticsearch = f.DedicatedElasticsearch()
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyWipeOut
					})
					It("should initialize database successfully", shouldRunWithTerminationWipeOut)

					It("should pause with TerminationPolicyWipeOut", shouldRunAndHalt)

					Context("with SSL disabled", func() {
						BeforeEach(func() {
							elasticsearch.Spec.EnableSSL = false
						})

						It("should initialize database successfully", shouldRunWithTerminationWipeOut)

						It("should pause with TerminationPolicyWipeOut", shouldRunAndHalt)
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

				By("wait until elasticsearch is deleted")
				f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeFalse())

				// create elasticsearch object again to resume it
				By("Create Elasticsearch: " + elasticsearch.Name)
				err = f.CreateElasticsearch(elasticsearch)
				Expect(err).NotTo(HaveOccurred())

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
					pod, err := f.KubeClient().CoreV1().Pods(elasticsearch.Namespace).Get(context.TODO(), podName, metav1.GetOptions{})
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
					_, _, err := util.PatchElasticsearch(context.TODO(), f.ExtClient().KubedbV1alpha1(), elasticsearch, func(in *api.Elasticsearch) *api.Elasticsearch {
						in.Spec.PodTemplate.Spec.Env = []core.EnvVar{
							{
								Name:  "CLUSTER_NAME",
								Value: "kubedb-es-e2e-cluster-patched",
							},
						}
						return in
					}, metav1.PatchOptions{})
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

				elasticClient, err := f.GetElasticClient(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				defer elasticClient.Stop()
				defer f.Tunnel.Close()

				By("Reading Nodes information")
				settings, err := elasticClient.GetAllNodesInfo()
				Expect(err).NotTo(HaveOccurred())

				By("Checking nodes are using provided config")
				Expect(f.IsUsingProvidedConfig(elasticsearch, settings)).Should(BeTrue())
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

				Context("With TerminationPolicyHalt", func() {

					BeforeEach(func() {
						elasticsearch.Spec.StorageType = api.StorageTypeEphemeral
						elasticsearch.Spec.Storage = nil
						elasticsearch.Spec.TerminationPolicy = api.TerminationPolicyHalt
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

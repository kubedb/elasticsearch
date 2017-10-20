package e2e_test

import (
	"os"

	"github.com/appscode/go/types"
	tapi "github.com/k8sdb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/k8sdb/elasticsearch/test/e2e/framework"
	"github.com/k8sdb/elasticsearch/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	S3_BUCKET_NAME       = "S3_BUCKET_NAME"
	GCS_BUCKET_NAME      = "GCS_BUCKET_NAME"
	AZURE_CONTAINER_NAME = "AZURE_CONTAINER_NAME"
	SWIFT_CONTAINER_NAME = "SWIFT_CONTAINER_NAME"
)

var _ = Describe("Elasticsearch", func() {
	var (
		err           error
		f             *framework.Invocation
		elasticsearch *tapi.Elasticsearch
		snapshot      *tapi.Snapshot
		secret        *apiv1.Secret
		skipMessage   string
	)

	BeforeEach(func() {
		f = root.Invoke()
		elasticsearch = f.Elasticsearch()
		snapshot = f.Snapshot()
		skipMessage = ""
	})

	var createAndWaitForRunning = func() {
		By("Create Elasticsearch: " + elasticsearch.Name)
		err = f.CreateElasticsearch(elasticsearch)
		Expect(err).NotTo(HaveOccurred())

		By("Wait for Running elasticsearch")
		f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())
	}

	var deleteTestResouce = func() {
		By("Delete elasticsearch")
		err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		By("Wait for elasticsearch to be paused")
		f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

		By("WipeOut elasticsearch")
		_, err := f.TryPatchDormantDatabase(elasticsearch.ObjectMeta, func(in *tapi.DormantDatabase) *tapi.DormantDatabase {
			in.Spec.WipeOut = true
			return in
		})
		Expect(err).NotTo(HaveOccurred())

		By("Wait for elasticsearch to be wipedOut")
		f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HaveWipedOut())

		err = f.DeleteDormantDatabase(elasticsearch.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	}

	var shouldSuccessfullyRunning = func() {
		if skipMessage != "" {
			Skip(skipMessage)
		}

		// Create Elasticsearch
		createAndWaitForRunning()

		// Delete test resource
		deleteTestResouce()
	}

	Describe("Test", func() {

		Context("General", func() {

			Context("-", func() {
				It("should run successfully", shouldSuccessfullyRunning)
			})

			Context("With PVC", func() {
				BeforeEach(func() {
					if f.StorageClass == "" {
						skipMessage = "Missing StorageClassName. Provide as flag to test this."
					}
					elasticsearch.Spec.Storage = &apiv1.PersistentVolumeClaimSpec{
						Resources: apiv1.ResourceRequirements{
							Requests: apiv1.ResourceList{
								apiv1.ResourceStorage: resource.MustParse("5Gi"),
							},
						},
						StorageClassName: types.StringP(f.StorageClass),
					}
				})
				It("should run successfully", shouldSuccessfullyRunning)
			})
		})

		Context("DoNotPause", func() {
			BeforeEach(func() {
				elasticsearch.Spec.DoNotPause = true
			})

			It("should work successfully", func() {
				// Create and wait for running Elasticsearch
				createAndWaitForRunning()

				By("Delete elasticsearch")
				err = f.DeleteElasticsearch(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Elasticsearch is not paused. Check for elasticsearch")
				f.EventuallyElasticsearch(elasticsearch.ObjectMeta).Should(BeTrue())

				By("Check for Running elasticsearch")
				f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

				By("Update elasticsearch to set DoNotPause=false")
				f.TryPatchElasticsearch(elasticsearch.ObjectMeta, func(in *tapi.Elasticsearch) *tapi.Elasticsearch {
					in.Spec.DoNotPause = false
					return in
				})

				// Delete test resource
				deleteTestResouce()
			})
		})

		Context("Snapshot", func() {
			var skipDataCheck bool

			AfterEach(func() {
				f.DeleteSecret(secret.ObjectMeta)
			})

			BeforeEach(func() {
				skipDataCheck = false
				snapshot.Spec.DatabaseName = elasticsearch.Name
			})

			var shouldTakeSnapshot = func() {
				// Create and wait for running Elasticsearch
				createAndWaitForRunning()

				By("Create Secret")
				f.CreateSecret(secret)

				By("Create Snapshot")
				f.CreateSnapshot(snapshot)

				By("Check for Successed snapshot")
				f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(tapi.SnapshotPhaseSuccessed))

				if !skipDataCheck {
					By("Check for snapshot data")
					f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())
				}

				// Delete test resource
				deleteTestResouce()

				if !skipDataCheck {
					By("Check for snapshot data")
					f.EventuallySnapshotDataFound(snapshot).Should(BeFalse())
				}
			}

			Context("In Local", func() {
				BeforeEach(func() {
					skipDataCheck = true
					secret = f.SecretForLocalBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.Local = &tapi.LocalSpec{
						Path: "/repo",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/repo",
							},
						},
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)
			})

			Context("In S3", func() {
				BeforeEach(func() {
					secret = f.SecretForS3Backend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.S3 = &tapi.S3Spec{
						Bucket: os.Getenv(S3_BUCKET_NAME),
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)
			})

			Context("In GCS", func() {
				BeforeEach(func() {
					secret = f.SecretForGCSBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.GCS = &tapi.GCSSpec{
						Bucket: os.Getenv(GCS_BUCKET_NAME),
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)
			})

			Context("In Azure", func() {
				BeforeEach(func() {
					secret = f.SecretForAzureBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.Azure = &tapi.AzureSpec{
						Container: os.Getenv(AZURE_CONTAINER_NAME),
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)
			})

			Context("In Swift", func() {
				BeforeEach(func() {
					secret = f.SecretForSwiftBackend()
					snapshot.Spec.StorageSecretName = secret.Name
					snapshot.Spec.Swift = &tapi.SwiftSpec{
						Container: os.Getenv(SWIFT_CONTAINER_NAME),
					}
				})

				It("should take Snapshot successfully", shouldTakeSnapshot)
			})
		})

		Context("Initialize", func() {
			AfterEach(func() {
				f.DeleteSecret(secret.ObjectMeta)
			})

			BeforeEach(func() {
				secret = f.SecretForS3Backend()
				snapshot.Spec.StorageSecretName = secret.Name
				snapshot.Spec.S3 = &tapi.S3Spec{
					Bucket: os.Getenv(S3_BUCKET_NAME),
				}
				snapshot.Spec.DatabaseName = elasticsearch.Name
			})

			It("should run successfully", func() {
				// Create and wait for running Elasticsearch
				createAndWaitForRunning()

				By("Create Secret")
				f.CreateSecret(secret)

				By("Create Snapshot")
				f.CreateSnapshot(snapshot)

				By("Check for Successed snapshot")
				f.EventuallySnapshotPhase(snapshot.ObjectMeta).Should(Equal(tapi.SnapshotPhaseSuccessed))

				By("Check for snapshot data")
				f.EventuallySnapshotDataFound(snapshot).Should(BeTrue())

				oldElasticsearch, err := f.GetElasticsearch(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create elasticsearch from snapshot")
				elasticsearch = f.Elasticsearch()
				elasticsearch.Spec.Init = &tapi.InitSpec{
					SnapshotSource: &tapi.SnapshotSourceSpec{
						Namespace: snapshot.Namespace,
						Name:      snapshot.Name,
					},
				}

				// Create and wait for running Elasticsearch
				createAndWaitForRunning()

				// Delete test resource
				deleteTestResouce()
				elasticsearch = oldElasticsearch
				// Delete test resource
				deleteTestResouce()
			})
		})

		Context("Resume", func() {
			var usedInitSpec bool
			BeforeEach(func() {
				usedInitSpec = false
			})

			var shouldResumeSuccessfully = func() {
				// Create and wait for running Elasticsearch
				createAndWaitForRunning()

				By("Delete elasticsearch")
				f.DeleteElasticsearch(elasticsearch.ObjectMeta)

				By("Wait for elasticsearch to be paused")
				f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

				_, err = f.TryPatchDormantDatabase(elasticsearch.ObjectMeta, func(in *tapi.DormantDatabase) *tapi.DormantDatabase {
					in.Spec.Resume = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Wait for DormantDatabase to be deleted")
				f.EventuallyDormantDatabase(elasticsearch.ObjectMeta).Should(BeFalse())

				By("Wait for Running elasticsearch")
				f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

				elasticsearch, err = f.GetElasticsearch(elasticsearch.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				if usedInitSpec {
					Expect(elasticsearch.Spec.Init).Should(BeNil())
					Expect(elasticsearch.Annotations[tapi.ElasticsearchInitSpec]).ShouldNot(BeEmpty())
				}

				// Delete test resource
				deleteTestResouce()
			}

			Context("-", func() {
				It("should resume DormantDatabase successfully", shouldResumeSuccessfully)
			})

			Context("With Init", func() {
				BeforeEach(func() {
					usedInitSpec = true
					elasticsearch.Spec.Init = &tapi.InitSpec{
						ScriptSource: &tapi.ScriptSourceSpec{
							ScriptPath: "elasticsearch-init-scripts/run.sh",
							VolumeSource: apiv1.VolumeSource{
								GitRepo: &apiv1.GitRepoVolumeSource{
									Repository: "https://github.com/k8sdb/elasticsearch-init-scripts.git",
								},
							},
						},
					}
				})

				It("should resume DormantDatabase successfully", shouldResumeSuccessfully)
			})

			Context("With original Elasticsearch", func() {
				It("should resume DormantDatabase successfully", func() {
					// Create and wait for running Elasticsearch
					createAndWaitForRunning()

					By("Delete elasticsearch")
					f.DeleteElasticsearch(elasticsearch.ObjectMeta)

					By("Wait for elasticsearch to be paused")
					f.EventuallyDormantDatabaseStatus(elasticsearch.ObjectMeta).Should(matcher.HavePaused())

					// Create Elasticsearch object again to resume it
					By("Create Elasticsearch: " + elasticsearch.Name)
					err = f.CreateElasticsearch(elasticsearch)
					Expect(err).NotTo(HaveOccurred())

					By("Wait for DormantDatabase to be deleted")
					f.EventuallyDormantDatabase(elasticsearch.ObjectMeta).Should(BeFalse())

					By("Wait for Running elasticsearch")
					f.EventuallyElasticsearchRunning(elasticsearch.ObjectMeta).Should(BeTrue())

					elasticsearch, err = f.GetElasticsearch(elasticsearch.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					// Delete test resource
					deleteTestResouce()
				})
			})
		})

		Context("SnapshotScheduler", func() {
			AfterEach(func() {
				f.DeleteSecret(secret.ObjectMeta)
			})

			BeforeEach(func() {
				secret = f.SecretForLocalBackend()
			})

			Context("With Startup", func() {
				BeforeEach(func() {
					elasticsearch.Spec.BackupSchedule = &tapi.BackupScheduleSpec{
						CronExpression: "@every 1m",
						SnapshotStorageSpec: tapi.SnapshotStorageSpec{
							StorageSecretName: secret.Name,
							Local: &tapi.LocalSpec{
								Path: "/repo",
								VolumeSource: apiv1.VolumeSource{
									HostPath: &apiv1.HostPathVolumeSource{
										Path: "/repo",
									},
								},
							},
						},
					}
				})

				It("should run schedular successfully", func() {
					By("Create Secret")
					f.CreateSecret(secret)

					// Create and wait for running Elasticsearch
					createAndWaitForRunning()

					By("Count multiple Snapshot")
					f.EventuallySnapshotCount(elasticsearch.ObjectMeta).Should(matcher.MoreThan(3))

					deleteTestResouce()
				})
			})

			Context("With Update", func() {
				It("should run schedular successfully", func() {
					// Create and wait for running Elasticsearch
					createAndWaitForRunning()

					By("Create Secret")
					f.CreateSecret(secret)

					By("Update elasticsearch")
					_, err = f.TryPatchElasticsearch(elasticsearch.ObjectMeta, func(in *tapi.Elasticsearch) *tapi.Elasticsearch {
						in.Spec.BackupSchedule = &tapi.BackupScheduleSpec{
							CronExpression: "@every 1m",
							SnapshotStorageSpec: tapi.SnapshotStorageSpec{
								StorageSecretName: secret.Name,
								Local: &tapi.LocalSpec{
									Path: "/repo",
									VolumeSource: apiv1.VolumeSource{
										HostPath: &apiv1.HostPathVolumeSource{
											Path: "/repo",
										},
									},
								},
							},
						}

						return in
					})
					Expect(err).NotTo(HaveOccurred())

					By("Count multiple Snapshot")
					f.EventuallySnapshotCount(elasticsearch.ObjectMeta).Should(matcher.MoreThan(3))

					deleteTestResouce()
				})
			})
		})

	})
})

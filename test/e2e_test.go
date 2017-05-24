package test

import (
	"fmt"
	"testing"
	"time"

	tapi "github.com/k8sdb/apimachinery/api"
	"github.com/k8sdb/elasticsearch/test/mini"
	"github.com/stretchr/testify/assert"
	kapi "k8s.io/kubernetes/pkg/api"
)

func TestCreate(t *testing.T) {
	controller, err := getController()
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("--> Running Elastic Controller")

	// elastic
	fmt.Println()
	fmt.Println("-- >> Testing elastic")
	fmt.Println("---- >> Creating elastic")
	elastic := mini.NewElastic()
	elastic, err = controller.ExtClient.Elastics("default").Create(elastic)
	if !assert.Nil(t, err) {
		return
	}

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err := mini.CheckElasticStatus(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic fails to be Ready")
		return
	} else {
		err := mini.CheckElasticWorkload(controller, elastic)
		if !assert.Nil(t, err) {
			return
		}
	}

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DormantDatabase")
	done, err := mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhasePaused)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> WipingOut Database")
	err = mini.WipeOutDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhaseWipedOut)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Deleting DormantDatabase")
	err = mini.DeleteDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

func TestDoNotPause(t *testing.T) {
	controller, err := getController()
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("--> Running elastic Controller")

	// elastic
	fmt.Println()
	fmt.Println("-- >> Testing elastic")
	fmt.Println("---- >> Creating elastic")
	elastic := mini.NewElastic()
	elastic.Spec.DoNotPause = true
	elastic, err = controller.ExtClient.Elastics("default").Create(elastic)
	if !assert.Nil(t, err) {
		return
	}

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err := mini.CheckElasticStatus(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic fails to be Ready")
	} else {
		err := mini.CheckElasticWorkload(controller, elastic)
		assert.Nil(t, err)
	}

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err = mini.CheckElasticStatus(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic fails to be Ready")
	} else {
		err := mini.CheckElasticWorkload(controller, elastic)
		assert.Nil(t, err)
	}

	elastic, _ = controller.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name)
	elastic.Spec.DoNotPause = false
	elastic, err = mini.UpdateElastic(controller, elastic)
	if !assert.Nil(t, err) {
		return
	}
	time.Sleep(time.Second * 10)

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DormantDatabase")
	done, err := mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhasePaused)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> WipingOut Database")
	err = mini.WipeOutDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhaseWipedOut)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Deleting DormantDatabase")
	err = mini.DeleteDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

func TestSnapshot(t *testing.T) {
	controller, err := getController()
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("--> Running elastic Controller")

	// elastic
	fmt.Println()
	fmt.Println("-- >> Testing elastic")
	fmt.Println("---- >> Creating elastic")
	elastic := mini.NewElastic()
	elastic, err = controller.ExtClient.Elastics("default").Create(elastic)
	if !assert.Nil(t, err) {
		return
	}

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err := mini.CheckElasticStatus(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic fails to be Ready")
		return
	} else {
		err := mini.CheckElasticWorkload(controller, elastic)
		if !assert.Nil(t, err) {
			fmt.Println("---- >> Failed to check elasticWorkload")
			return
		}
	}

	const (
		bucket     = ""
		secretName = ""
	)

	snapshotSpec := tapi.SnapshotSpec{
		DatabaseName: elastic.Name,
		SnapshotStorageSpec: tapi.SnapshotStorageSpec{
			BucketName: bucket,
			StorageSecret: &kapi.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}

	err = controller.CheckBucketAccess(snapshotSpec.SnapshotStorageSpec, elastic.Namespace)
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("---- >> Creating Snapshot")
	snapshot, err := mini.CreateSnapshot(controller, elastic.Namespace, snapshotSpec)
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("---- >> Checking Snapshot")
	done, err := mini.CheckSnapshot(controller, snapshot)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to take snapshot")
		return
	}

	fmt.Println("---- >> Checking Snapshot data")
	count, err := mini.CheckSnapshotData(controller, snapshot)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to check snapshot data")
		return
	}
	assert.NotZero(t, count)

	fmt.Println("---- >> Deleting Snapshot")
	err = controller.ExtClient.Snapshots(snapshot.Namespace).Delete(snapshot.Name)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to delete Snapshot")
		return
	}

	time.Sleep(time.Second * 30)

	fmt.Println("---- >> Checking Snapshot data")
	count, err = mini.CheckSnapshotData(controller, snapshot)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to check snapshot data")
		return
	}
	assert.Zero(t, count)

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhasePaused)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> WipingOut Database")
	err = mini.WipeOutDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhaseWipedOut)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Deleting DormantDatabase")
	err = mini.DeleteDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

func TestDatabaseResume(t *testing.T) {
	controller, err := getController()
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("--> Running elastic Controller")

	// elastic
	fmt.Println()
	fmt.Println("-- >> Testing elastic")
	fmt.Println("---- >> Creating elastic")
	elastic := mini.NewElastic()
	elastic, err = controller.ExtClient.Elastics("default").Create(elastic)
	if !assert.Nil(t, err) {
		return
	}

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err := mini.CheckElasticStatus(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic fails to be Ready")
		return
	} else {
		err := mini.CheckElasticWorkload(controller, elastic)
		if !assert.Nil(t, err) {
			fmt.Println("---- >> Failed to check elasticWorkload")
			return
		}
	}

	fmt.Println("---- >> Deleting elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DormantDatabase")
	done, err := mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhasePaused)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be delete")
	}

	fmt.Println("---- >> Updating DormantDatabase")
	dormantDb, err := controller.ExtClient.DormantDatabases(elastic.Namespace).Get(elastic.Name)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to get DormantDatabase")
		return
	}

	dormantDb.Spec.Resume = true
	_, err = controller.ExtClient.DormantDatabases(dormantDb.Namespace).Update(dormantDb)
	assert.Nil(t, err)

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err = mini.CheckElasticStatus(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic fails to be Ready")
		return
	} else {
		err := mini.CheckElasticWorkload(controller, elastic)
		if !assert.Nil(t, err) {
			fmt.Println("---- >> Failed to check elasticWorkload")
			return
		}
	}

	_, err = controller.ExtClient.DormantDatabases(dormantDb.Namespace).Get(dormantDb.Name)
	if !assert.NotNil(t, err) {
		fmt.Println("---- >> Failed to delete DormantDatabase")
		return
	}

	fmt.Println("---- >> Deleting elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhasePaused)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be delete")
	}

	fmt.Println("---- >> WipingOut Database")
	err = mini.WipeOutDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhaseWipedOut)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Deleting DormantDatabase")
	err = mini.DeleteDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

func TestInitialize(t *testing.T) {
	controller, err := getController()
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("--> Running elastic Controller")

	// elastic
	fmt.Println()
	fmt.Println("-- >> Testing elastic")
	fmt.Println("---- >> Creating elastic")
	elastic := mini.NewElastic()
	elastic, err = controller.ExtClient.Elastics("default").Create(elastic)
	if !assert.Nil(t, err) {
		return
	}

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err := mini.CheckElasticStatus(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic fails to be Ready")
		return
	} else {
		err := mini.CheckElasticWorkload(controller, elastic)
		if !assert.Nil(t, err) {
			fmt.Println("---- >> Failed to check elasticWorkload")
			return
		}
	}

	const (
		bucket     = ""
		secretName = ""
	)

	snapshotSpec := tapi.SnapshotSpec{
		DatabaseName: elastic.Name,
		SnapshotStorageSpec: tapi.SnapshotStorageSpec{
			BucketName: bucket,
			StorageSecret: &kapi.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}

	fmt.Println("---- >> Creating Snapshot")
	snapshot, err := mini.CreateSnapshot(controller, elastic.Namespace, snapshotSpec)
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("---- >> Checking Snapshot")
	done, err := mini.CheckSnapshot(controller, snapshot)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to take snapshot")
		return
	}

	fmt.Println("---- >> Checking Snapshot data")
	count, err := mini.CheckSnapshotData(controller, snapshot)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to check snapshot data")
		return
	}
	assert.NotZero(t, count)

	// elastic
	fmt.Println()
	fmt.Println("-- >> Testing elastic_init")
	fmt.Println("---- >> Creating elastic_init")
	elastic_init := mini.NewElastic()
	elastic_init.Spec.Init = &tapi.InitSpec{
		SnapshotSource: &tapi.SnapshotSourceSpec{
			Name: snapshot.Name,
		},
	}

	elastic_init, err = controller.ExtClient.Elastics("default").Create(elastic_init)
	if !assert.Nil(t, err) {
		return
	}

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err = mini.CheckElasticStatus(controller, elastic_init)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic_init fails to be Ready")
		return
	} else {
		err := mini.CheckElasticWorkload(controller, elastic_init)
		if !assert.Nil(t, err) {
			fmt.Println("---- >> Failed to check elasticWorkload")
			return
		}
	}

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhasePaused)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> WipingOut Database")
	err = mini.WipeOutDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhaseWipedOut)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Deleting DormantDatabase")
	err = mini.DeleteDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> Deleted elastic_init")
	err = mini.DeleteElastic(controller, elastic_init)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic_init, tapi.DormantDatabasePhasePaused)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> WipingOut Database")
	err = mini.WipeOutDormantDatabase(controller, elastic_init)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic_init, tapi.DormantDatabasePhaseWipedOut)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Deleting DormantDatabase")
	err = mini.DeleteDormantDatabase(controller, elastic_init)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

func TestUpdateScheduler(t *testing.T) {
	controller, err := getController()
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("--> Running elastic Controller")

	// elastic
	fmt.Println()
	fmt.Println("-- >> Testing elastic")
	fmt.Println("---- >> Creating elastic")
	elastic := mini.NewElastic()
	elastic, err = controller.ExtClient.Elastics("default").Create(elastic)
	if !assert.Nil(t, err) {
		return
	}

	time.Sleep(time.Second * 30)
	fmt.Println("---- >> Checking elastic")
	running, err := mini.CheckElasticStatus(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, running) {
		fmt.Println("---- >> elastic fails to be Ready")
		return
	} else {
		err := mini.CheckElasticWorkload(controller, elastic)
		if !assert.Nil(t, err) {
			return
		}
	}

	elastic, err = controller.ExtClient.Elastics("default").Get(elastic.Name)
	if !assert.Nil(t, err) {
		return
	}

	elastic.Spec.BackupSchedule = &tapi.BackupScheduleSpec{
		CronExpression: "@every 30s",
		SnapshotStorageSpec: tapi.SnapshotStorageSpec{
			BucketName: "",
			StorageSecret: &kapi.SecretVolumeSource{
				SecretName: "",
			},
		},
	}

	elastic, err = mini.UpdateElastic(controller, elastic)
	if !assert.Nil(t, err) {
		return
	}

	err = mini.CheckSnapshotScheduler(controller, elastic)
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("---- >> Deleted Postgres")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DormantDatabase")
	done, err := mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhasePaused)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> WipingOut Database")
	err = mini.WipeOutDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Checking DormantDatabase")
	done, err = mini.CheckDormantDatabasePhase(controller, elastic, tapi.DormantDatabasePhaseWipedOut)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be wipedout")
	}

	fmt.Println("---- >> Deleting DormantDatabase")
	err = mini.DeleteDormantDatabase(controller, elastic)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

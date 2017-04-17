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
		if !assert.Nil(t, err){
			return
		}
	}

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DeletedDatabase")
	done, err := mini.CheckDeletedDatabasePhase(controller, elastic, tapi.PhaseDatabaseDeleted)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> ReCreating elastic")
	elastic, err = mini.ReCreateElastic(controller, elastic)
	if !assert.Nil(t, err) {
		return
	}

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

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DeletedDatabase")
	done, err = mini.CheckDeletedDatabasePhase(controller, elastic, tapi.PhaseDatabaseDeleted)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

func TestDoNotDelete(t *testing.T) {
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
	elastic.Spec.DoNotDelete = true
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
	elastic.Spec.DoNotDelete = false
	elastic, err = mini.UpdateElastic(controller, elastic)
	if !assert.Nil(t, err) {
		return
	}
	time.Sleep(time.Second * 10)

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DeletedDatabase")
	done, err := mini.CheckDeletedDatabasePhase(controller, elastic, tapi.PhaseDatabaseDeleted)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

func TestDatabaseSnapshot(t *testing.T) {
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

	dbSnapshotSpec := tapi.DatabaseSnapshotSpec{
		DatabaseName: elastic.Name,
		SnapshotSpec: tapi.SnapshotSpec{
			BucketName: bucket,
			StorageSecret: &kapi.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}

	err = controller.CheckBucketAccess(dbSnapshotSpec.SnapshotSpec, elastic.Namespace)
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("---- >> Creating DatabaseSnapshot")
	dbSnapshot, err := mini.CreateDatabaseSnapshot(controller, elastic.Namespace, dbSnapshotSpec)
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("---- >> Checking DatabaseSnapshot")
	done, err := mini.CheckDatabaseSnapshot(controller, dbSnapshot)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to take snapshot")
		return
	}

	fmt.Println("---- >> Checking Snapshot data")
	count, err := mini.CheckSnapshotData(controller, dbSnapshot)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to check snapshot data")
		return
	}
	assert.NotZero(t, count)

	fmt.Println("---- >> Deleting DatabaseSnapshot")
	err = controller.ExtClient.DatabaseSnapshots(dbSnapshot.Namespace).Delete(dbSnapshot.Name)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to delete DatabaseSnapshot")
		return
	}

	time.Sleep(time.Second * 30)

	fmt.Println("---- >> Checking Snapshot data")
	count, err = mini.CheckSnapshotData(controller, dbSnapshot)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to check snapshot data")
		return
	}
	assert.Zero(t, count)

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DeletedDatabase")
	done, err = mini.CheckDeletedDatabasePhase(controller, elastic, tapi.PhaseDatabaseDeleted)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

func TestDeletedDatabase(t *testing.T) {
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

	fmt.Println("---- >> Checking DeletedDatabase")
	done, err := mini.CheckDeletedDatabasePhase(controller, elastic, tapi.PhaseDatabaseDeleted)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be delete")
	}

	fmt.Println("---- >> Updating DeletedDatabase")
	deletedDb, err := controller.ExtClient.DeletedDatabases(elastic.Namespace).Get(elastic.Name)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to get DeletedDatabase")
		return
	}

	deletedDb.Spec.Recover = true
	_, err = controller.ExtClient.DeletedDatabases(deletedDb.Namespace).Update(deletedDb)
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

	fmt.Println("---- >> Deleting elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DeletedDatabase")
	done, err = mini.CheckDeletedDatabasePhase(controller, elastic, tapi.PhaseDatabaseDeleted)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be delete")
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

	dbSnapshotSpec := tapi.DatabaseSnapshotSpec{
		DatabaseName: elastic.Name,
		SnapshotSpec: tapi.SnapshotSpec{
			BucketName: bucket,
			StorageSecret: &kapi.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}

	fmt.Println("---- >> Creating DatabaseSnapshot")
	dbSnapshot, err := mini.CreateDatabaseSnapshot(controller, elastic.Namespace, dbSnapshotSpec)
	if !assert.Nil(t, err) {
		return
	}

	fmt.Println("---- >> Checking DatabaseSnapshot")
	done, err := mini.CheckDatabaseSnapshot(controller, dbSnapshot)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to take snapshot")
		return
	}

	fmt.Println("---- >> Checking Snapshot data")
	count, err := mini.CheckSnapshotData(controller, dbSnapshot)
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
			Name: dbSnapshot.Name,
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


	fmt.Println("---- >> Deleting DatabaseSnapshot")
	err = controller.ExtClient.DatabaseSnapshots(dbSnapshot.Namespace).Delete(dbSnapshot.Name)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to delete DatabaseSnapshot")
		return
	}

	time.Sleep(time.Second * 30)

	fmt.Println("---- >> Checking Snapshot data")
	count, err = mini.CheckSnapshotData(controller, dbSnapshot)
	if !assert.Nil(t, err) {
		fmt.Println("---- >> Failed to check snapshot data")
		return
	}
	assert.Zero(t, count)

	fmt.Println("---- >> Deleted elastic")
	err = mini.DeleteElastic(controller, elastic)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DeletedDatabase")
	done, err = mini.CheckDeletedDatabasePhase(controller, elastic, tapi.PhaseDatabaseDeleted)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}

	fmt.Println("---- >> Deleted elastic_init")
	err = mini.DeleteElastic(controller, elastic_init)
	assert.Nil(t, err)

	fmt.Println("---- >> Checking DeletedDatabase")
	done, err = mini.CheckDeletedDatabasePhase(controller, elastic_init, tapi.PhaseDatabaseDeleted)
	assert.Nil(t, err)
	if !assert.True(t, done) {
		fmt.Println("---- >> Failed to be deleted")
	}
}

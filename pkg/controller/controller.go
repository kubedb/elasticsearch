package controller

import (
	"reflect"
	"time"

	"github.com/appscode/go/hold"
	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	cmap "github.com/orcaman/concurrent-map"
	"gopkg.in/robfig/cron.v2"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
)

type Controller struct {
	*amc.Controller
	// For Internal Cron Job
	cron *cron.Cron
	// Store Cron Job EntryID for further use
	cronEntryIDs cmap.ConcurrentMap
	// sync time to sync the list.
	SyncPeriod time.Duration
}

func New(c *rest.Config) *Controller {
	return &Controller{
		Controller:   amc.New(c),
		cron:         cron.New(),
		cronEntryIDs: cmap.New(),
		SyncPeriod:   time.Minute * 2,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (w *Controller) RunAndHold() {
	// Ensure all related ThirdPartyResource
	w.ensureThirdPartyResource()
	// Start Cron
	w.cron.Start()
	defer w.cron.Stop()
	// Watch Elastic TPR objects
	go w.watchElastic()
	// Watch DatabaseSnapshot with labelSelector only for Elastic
	go w.watchDatabaseSnapshot()

	hold.Hold()
}

func (w *Controller) watchElastic() {
	lw := &cache.ListWatch{
		ListFunc: func(opts kapi.ListOptions) (runtime.Object, error) {
			return w.ExtClient.Elastic(kapi.NamespaceAll).List(kapi.ListOptions{})
		},
		WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
			return w.ExtClient.Elastic(kapi.NamespaceAll).Watch(kapi.ListOptions{})
		},
	}
	_, cacheController := cache.NewInformer(lw,
		&tapi.Elastic{},
		w.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.create(obj.(*tapi.Elastic))
			},
			DeleteFunc: func(obj interface{}) {
				w.delete(obj.(*tapi.Elastic))
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*tapi.Elastic)
				if !ok {
					return
				}
				newObj, ok := new.(*tapi.Elastic)
				if !ok {
					return
				}
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
					w.update(oldObj, newObj)
				}
			},
		},
	)
	cacheController.Run(wait.NeverStop)
}

func (w *Controller) watchDatabaseSnapshot() {
	labelMap := map[string]string{
		LabelDatabaseType: DatabaseElasticsearch,
	}
	// Watch with label selector
	lw := &cache.ListWatch{
		ListFunc: func(opts kapi.ListOptions) (runtime.Object, error) {
			return w.Controller.ExtClient.DatabaseSnapshot(kapi.NamespaceAll).List(
				kapi.ListOptions{
					LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
				})
		},
		WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
			return w.Controller.ExtClient.DatabaseSnapshot(kapi.NamespaceAll).Watch(
				kapi.ListOptions{
					LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
				})
		},
	}

	_, cacheController := cache.NewInformer(lw,
		&tapi.DatabaseSnapshot{},
		w.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			// Only add operation is handled
			AddFunc: func(obj interface{}) {
				databaseSnapshot := obj.(*tapi.DatabaseSnapshot)
				if databaseSnapshot.Status.StartTime == nil {
					w.backup(databaseSnapshot)
				}
			},
		},
	)

	cacheController.Run(wait.NeverStop)
}

func (w *Controller) ensureThirdPartyResource() {
	log.Info("Ensuring ThirdPartyResource...")

	// Ensure Elastic TPR
	w.ensureElastic()

	// Ensure DatabaseSnapshot TPR
	w.Controller.EnsureDatabaseSnapshot()
}

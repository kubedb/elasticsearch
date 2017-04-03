package controller

import (
	"reflect"
	"time"

	"github.com/appscode/go/hold"
	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/apimachinery/pkg/eventer"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
)

type Controller struct {
	*amc.Controller
	// Cron Controller
	cronController amc.CronControllerInterface
	// Event Recorder
	eventRecorder eventer.EventRecorderInterface
	// sync time to sync the list.
	syncPeriod time.Duration
}

func New(c *rest.Config) *Controller {
	controller := amc.NewController(c)
	return &Controller{
		Controller:     controller,
		cronController: amc.NewCronController(controller.Client, controller.ExtClient),
		eventRecorder:  eventer.NewEventRecorder(controller.Client, "Elastic Controller"),
		syncPeriod:     time.Minute * 2,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) RunAndHold() {
	// Ensure Elastic TPR
	c.ensureThirdPartyResource()

	// Watch Elastic TPR objects
	go c.watchElastic()
	// Watch DatabaseSnapshot with labelSelector only for Elastic
	go c.watchDatabaseSnapshot()
	// Watch DeletedDatabase with labelSelector only for Elastic
	go c.watchDeletedDatabase()
	hold.Hold()
}

func (c *Controller) watchElastic() {
	lw := &cache.ListWatch{
		ListFunc: func(opts kapi.ListOptions) (runtime.Object, error) {
			return c.ExtClient.Elastics(kapi.NamespaceAll).List(kapi.ListOptions{})
		},
		WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
			return c.ExtClient.Elastics(kapi.NamespaceAll).Watch(kapi.ListOptions{})
		},
	}

	eController := &elasticController{c}
	_, cacheController := cache.NewInformer(
		lw,
		&tapi.Elastic{},
		c.syncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				elastic := obj.(*tapi.Elastic)
				/*
					TODO: set appropriate checking
					We do not want to handle same TPR objects multiple times
				*/
				if true {
					eController.create(elastic)
				}
			},
			DeleteFunc: func(obj interface{}) {
				eController.delete(obj.(*tapi.Elastic))
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
					eController.update(oldObj, newObj)
				}
			},
		},
	)
	cacheController.Run(wait.NeverStop)
}

func (c *Controller) watchDatabaseSnapshot() {
	labelMap := map[string]string{
		LabelDatabaseType: DatabaseElasticsearch,
	}
	// Watch with label selector
	lw := &cache.ListWatch{
		ListFunc: func(opts kapi.ListOptions) (runtime.Object, error) {
			return c.ExtClient.DatabaseSnapshots(kapi.NamespaceAll).List(
				kapi.ListOptions{
					LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
				})
		},
		WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
			return c.ExtClient.DatabaseSnapshots(kapi.NamespaceAll).Watch(
				kapi.ListOptions{
					LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
				})
		},
	}

	snapshotter := NewSnapshotter(c.Controller)
	amc.NewDatabaseSnapshotController(c.Client, c.ExtClient, snapshotter, lw, c.syncPeriod).Run()
}

func (c *Controller) watchDeletedDatabase() {
	labelMap := map[string]string{
		LabelDatabaseType: DatabaseElasticsearch,
	}
	// Watch with label selector
	lw := &cache.ListWatch{
		ListFunc: func(opts kapi.ListOptions) (runtime.Object, error) {
			return c.ExtClient.DeletedDatabases(kapi.NamespaceAll).List(
				kapi.ListOptions{
					LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
				})
		},
		WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
			return c.ExtClient.DeletedDatabases(kapi.NamespaceAll).Watch(
				kapi.ListOptions{
					LabelSelector: labels.SelectorFromSet(labels.Set(labelMap)),
				})
		},
	}

	deleter := NewDeleter(c.Controller)
	amc.NewDeletedDbController(c.Client, c.ExtClient, deleter, lw, c.syncPeriod).Run()
}

func (c *Controller) ensureThirdPartyResource() {
	log.Infoln("Ensuring ThirdPartyResource...")

	resourceName := tapi.ResourceNameElastic + "." + tapi.V1beta1SchemeGroupVersion.Group

	if _, err := c.Client.Extensions().ThirdPartyResources().Get(resourceName); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Fatalln(err)
		}
	} else {
		return
	}

	thirdPartyResource := &extensions.ThirdPartyResource{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "ThirdPartyResource",
		},
		ObjectMeta: kapi.ObjectMeta{
			Name: resourceName,
		},
		Versions: []extensions.APIVersion{
			{
				Name: tapi.V1beta1SchemeGroupVersion.Version,
			},
		},
	}

	if _, err := c.Client.Extensions().ThirdPartyResources().Create(thirdPartyResource); err != nil {
		log.Fatalln(err)
	}
}

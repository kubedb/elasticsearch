package controller

import (
	"reflect"
	"time"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/elasticsearch/api"
	tcs "github.com/k8sdb/elasticsearch/client/clientset"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
)

type Controller struct {
	// Kubernetes client to apiserver
	Client clientset.Interface
	// ThirdPartyExtension client to apiserver
	ExtClient tcs.ExtensionInterface
	// sync time to sync the list.
	SyncPeriod time.Duration
}

func New(c *rest.Config) *Controller {
	return &Controller{
		Client:     clientset.NewForConfigOrDie(c),
		ExtClient:  tcs.NewExtensionsForConfigOrDie(c),
		SyncPeriod: time.Minute * 2,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (w *Controller) RunAndHold() {
	log.Infoln("Ensuring ThirdPartyResource...")
	w.ensureThirdPartyResource()

	lw := &cache.ListWatch{
		ListFunc: func(opts kapi.ListOptions) (runtime.Object, error) {
			return w.ExtClient.Elasticsearch(kapi.NamespaceAll).List(kapi.ListOptions{})
		},
		WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
			return w.ExtClient.Elasticsearch(kapi.NamespaceAll).Watch(kapi.ListOptions{})
		},
	}
	_, controller := cache.NewInformer(lw,
		&tapi.Elasticsearch{},
		w.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.create(obj.(*tapi.Elasticsearch))
			},
			DeleteFunc: func(obj interface{}) {
				w.delete(obj.(*tapi.Elasticsearch))
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*tapi.Elasticsearch)
				if !ok {
					return
				}
				newObj, ok := new.(*tapi.Elasticsearch)
				if !ok {
					return
				}
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
					w.update(newObj)
				}
			},
		},
	)
	controller.Run(wait.NeverStop)
}

func (w *Controller) ensureThirdPartyResource() {
	resourceName := "elasticsearch" + "." + tapi.V1beta1SchemeGroupVersion.Group

	if _, err := w.Client.Extensions().ThirdPartyResources().Get(resourceName); err != nil {
		if !errors.IsNotFound(err) {
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

	if _, err := w.Client.Extensions().ThirdPartyResources().Create(thirdPartyResource); err != nil {
		log.Fatalln(err)
	}
}
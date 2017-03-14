package controller

import (
	"time"

	tcs "github.com/k8sdb/apimachinery/client/clientset"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
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

}

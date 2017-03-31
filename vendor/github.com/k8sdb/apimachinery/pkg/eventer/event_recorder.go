package eventer

import (
	"github.com/appscode/log"
	kapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/record"
	"k8s.io/kubernetes/pkg/runtime"
)

const (
	EventReasonDeleting           string = "Deleting"
	EventReasonDestroying         string = "Destroying"
	EventReasonFailedToCreate     string = "Failed"
	EventReasonFailedToDelete     string = "Failed"
	EventReasonFailedToDestroy    string = "Failed"
	EventReasonFailedToGet        string = "Failed"
	EventReasonFailedToList       string = "Failed"
	EventReasonFailedToUpdate     string = "Failed"
	EventReasonInvalid            string = "Invalid"
	EventReasonSnapshotFailed     string = "SnapshotFailed"
	EventReasonStarting           string = "Starting"
	EventReasonSuccessfulDelete   string = "SuccessfulDelete"
	EventReasonSuccessfulDestroy  string = "SuccessfulDestroy"
	EventReasonSuccessfulSnapshot string = "SuccessfulSnapshot"
	EventReasonIngoredSnapshot    string = "IngoredSnapshot"
)

type EventRecorderInterface interface {
	PushEvent(eventtype, reason, message string, objects ...runtime.Object)
}

type eventRecorder struct {
	// Event Recorder
	record.EventRecorder
}

func NewEventRecorder(client clientset.Interface, component string) EventRecorderInterface {
	// Event Broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartEventWatcher(
		func(event *kapi.Event) {
			_, err := client.Core().Events(event.Namespace).Create(event)
			log.Errorln(err)
		},
	)
	// Event Recorder
	return &eventRecorder{broadcaster.NewRecorder(kapi.EventSource{Component: component})}
}

func (e *eventRecorder) PushEvent(eventtype, reason, message string, objects ...runtime.Object) {
	for _, obj := range objects {
		e.Event(obj, eventtype, reason, message)
	}
}

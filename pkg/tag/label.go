package tag

import (
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// set by the webhook only
var TRACEY_WEBHOOK_LABEL = "tracey-uid"

// the ID of the reconcile invocation in which the object was acted upon
var TRACEY_RECONCILE_ID = "discrete.events/reconcile-id"

// the ID of the controller that acted upon the object
var TRACEY_CREATOR_ID = "discrete.events/creator-id"

// the ID of the root event that caused the object to be acted upon.
// the value originates from a TRACEY_WEBHOOK_LABEL value but we just
// use a different name when propagating the value.
var TRACEY_ROOT_ID = "discrete.events/root-event-id"

// deprecated... to be determined offline
var TRACEY_PARENT_ID = "discrete.events/parent-id"

var CHANGE_ID = "discrete.events/change-id"

// LabelChange sets a change-id on the object to associate an object's current value with the change event that produced it.
func LabelChange(obj client.Object) {
	labels := obj.GetLabels()
	// if map is nil, create a new one
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[CHANGE_ID] = uuid.New().String()
	obj.SetLabels(labels)
}

func GetChangeLabel() map[string]string {
	labels := make(map[string]string)
	labels[CHANGE_ID] = uuid.New().String()
	return labels
}

type LabelContext struct {
	RootID       string
	TraceID      string
	ParentID     string
	SourceObject string
}

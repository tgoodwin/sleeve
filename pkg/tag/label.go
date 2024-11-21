package tag

import (
	"fmt"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// set by the webhook only
	TraceyWebhookLabel = "tracey-uid"

	// the ID of the reconcile invocation in which the object was acted upon
	TraceyReconcileID = "discrete.events/prev-write-reconcile-id"

	// the ID of the controller that acted upon the object
	TraceyCreatorID = "discrete.events/creator-id"

	// the ID of the root event that caused the object to be acted upon.
	// the value originates from a TraceyWebhookLabel value but we just
	// use a different name when propagating the value.
	TraceyRootID = "discrete.events/root-event-id"

	// deprecated... to be determined offline
	TraceyParentID = "discrete.events/parent-id"

	ChangeID = "discrete.events/change-id"
)

// LabelChange sets a change-id on the object to associate an object's current value with the change event that produced it.
func LabelChange(obj client.Object) {
	labels := obj.GetLabels()
	// if map is nil, create a new one
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[ChangeID] = uuid.New().String()
	obj.SetLabels(labels)
}

func GetChangeLabel() map[string]string {
	labels := make(map[string]string)
	labels[ChangeID] = uuid.New().String()
	return labels
}

func SanityCheckLabels(obj client.Object) error {
	labels := obj.GetLabels()
	if labels == nil {
		return nil
	}
	if webhookLabel, ok := labels[TraceyWebhookLabel]; ok {
		if rootID, ok := labels[TraceyRootID]; ok {
			if webhookLabel != rootID {
				// logf.Log.WithValues("key", "val").Error(nil, "labeling assumptions violated")
				return fmt.Errorf("labeling assumptions violated: tracey-uid=%s, root-event-id=%s", webhookLabel, rootID)

			}
		}
	}
	return nil
}

type LabelContext struct {
	RootID       string
	TraceID      string
	ParentID     string
	SourceObject string
}

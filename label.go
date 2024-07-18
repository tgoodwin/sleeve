package client

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

type LabelContext struct {
	RootID       string
	TraceID      string
	ParentID     string
	SourceObject string
}

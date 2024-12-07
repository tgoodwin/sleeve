package event

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ChangeID corresponds to the discrete.events/change-id label on a k8s object
type ChangeID string

// CausalKey represents a unique identifier for a sleeve object at the version represented by its change-id label.
// note that the version is the sleeve change-id label, not the resource version. Using the sleeve change-id label
// lets us track the causal history of an object at "sleeve granularity", ignoring any object version changes
// produced by k8s-internal controllers/mechanisms that are not instrumented by sleeve.
type CausalKey struct {
	Kind     string
	ObjectID string
	Version  ChangeID
}

func (c CausalKey) String() string {
	return fmt.Sprintf("%s:%s@%s", c.Kind, c.ObjectID, c.Version)
}

func GetCausalKey(obj *unstructured.Unstructured) (CausalKey, error) {
	cid, err := GetChangeID(obj)
	if err != nil {
		return CausalKey{}, errors.New("object has no causal ID")
	}

	k := CausalKey{
		Kind:     obj.GetKind(),
		ObjectID: string(obj.GetUID()),
		Version:  cid,
	}
	return k, nil
}

func GetChangeID(obj *unstructured.Unstructured) (ChangeID, error) {
	if obj == nil {
		return "", fmt.Errorf("object is nil")
	}
	labels := obj.GetLabels()
	if labels == nil {
		return "", fmt.Errorf("object has no labels")
	}
	if causalID, ok := labels["discrete.events/change-id"]; ok {
		return ChangeID(causalID), nil
	}
	// case where its a top-level GET event from a declared resource that has only
	// been tagged by the webhook with a tracey-uid
	if rootID, ok := labels["tracey-uid"]; ok {
		return ChangeID(rootID), nil
	}
	if rootID, ok := labels["discrete.events/root-event-id"]; ok {
		return ChangeID(rootID), nil
	}
	return "", fmt.Errorf("object has no causal ID")
}

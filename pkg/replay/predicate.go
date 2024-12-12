package replay

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Predicate represents a boolean property of a given object in an execution trace.
// Predicates are used to specify some desired outcome of an execution, and to evaluate
// whether or not the outcome was achieved by perturbing the traced execution in some way.
type Predicate func(obj *unstructured.Unstructured) bool

func ConditionPredicate(resourceType string, conditionName, conditionValue string) Predicate {
	return func(obj *unstructured.Unstructured) bool {
		conditions := obj.GetAnnotations()
		for name, value := range conditions {
			if name == conditionName && value == conditionValue {
				return true
			}
		}
		return false
	}
}

type executionPredicate struct {
	satisfied bool
	evaluate  Predicate
}

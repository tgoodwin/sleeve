package replay

import "sigs.k8s.io/controller-runtime/pkg/client"

// Predicate represents a boolean property of a given object in an execution trace.
// Predicates are used to specify some desired outcome of an execution, and to evaluate
// whether or not the outcome was achieved by perturbing the traced execution in some way.
type Predicate func(obj client.Object) bool

func ConditionPredicate(resourceType string, conditionName, conditionValue string) Predicate {
	return func(obj client.Object) bool {
		conditions := obj.GetAnnotations()
		for name, value := range conditions {
			if name == conditionName && value == conditionValue {
				return true
			}
		}
		return false
	}
}

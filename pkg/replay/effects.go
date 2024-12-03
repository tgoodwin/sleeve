package replay

import (
	"context"

	sleeveclient "github.com/tgoodwin/sleeve/pkg/client"
	"github.com/tgoodwin/sleeve/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Recorder struct {
	reconcilerID    string
	effectContainer map[string]DataEffect

	predicates []Predicate
}

func (r *Recorder) RecordEffect(ctx context.Context, obj client.Object, opType sleeveclient.OperationType) error {
	reconcileID := frameIDFromContext(ctx)
	e := sleeveclient.Operation(obj, reconcileID, r.reconcilerID, "<REPLAY>", opType)

	de, exists := r.effectContainer[reconcileID]
	if !exists {
		de = DataEffect{}
	}

	if event.IsReadOp(*e) {
		de.Reads = append(de.Reads, *e)
	} else if event.IsWriteOp(*e) {
		de.Writes = append(de.Writes, *e)
		// in the case where we are recording a perturbed execution,
		// see if the perturbation produced the desired effect
		r.evaluatePredicates(ctx, obj)
	}

	r.effectContainer[reconcileID] = de
	return nil
}

func (r *Recorder) evaluatePredicates(_ context.Context, obj client.Object) {
	for _, predicate := range r.predicates {
		if predicate(obj) {
			panic("predicate returned true")
		}
	}
}

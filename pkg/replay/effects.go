package replay

import (
	"context"
	"errors"

	sleeveclient "github.com/tgoodwin/sleeve/pkg/client"
	"github.com/tgoodwin/sleeve/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DataEffect struct {
	Reads  []event.Event
	Writes []event.Event
}

type EffectHandler interface {
	Record(frameID string, de DataEffect) error
	Retrieve(frameID string) (DataEffect, bool)
}

type Recorder struct {
	reconcilerID    string
	effectContainer map[string]DataEffect

	predicates []*executionPredicate
}

func (r *Recorder) Record(frameID string, de DataEffect) error {
	if _, ok := r.effectContainer[frameID]; ok {
		return errors.New("effect already recorded for frame")
	}
	r.effectContainer[frameID] = de
	return nil
}

func (r *Recorder) Retrieve(frameID string) (DataEffect, bool) {
	de, ok := r.effectContainer[frameID]
	return de, ok
}

var _ EffectHandler = (*Recorder)(nil)

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
	for _, p := range r.predicates {
		if p.evaluate(obj) {
			p.satisfied = true
		}
	}
}

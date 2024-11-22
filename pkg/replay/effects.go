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
}

func (r *Recorder) RecordEffect(ctx context.Context, obj client.Object, opType sleeveclient.OperationType) error {
	reconcileID := frameIDFromContext(ctx)
	e := sleeveclient.Operation(obj, reconcileID, r.reconcilerID, "<REPLAY>", opType)
	if _, ok := r.effectContainer[reconcileID]; !ok {
		r.effectContainer[reconcileID] = DataEffect{}
	}
	de := r.effectContainer[reconcileID]
	if event.IsReadOp(*e) {
		de.Reads = append(r.effectContainer[reconcileID].Reads, *e)
		return nil
	} else if event.IsWriteOp(*e) {
		de.Writes = append(r.effectContainer[reconcileID].Writes, *e)
	}
	r.effectContainer[reconcileID] = de

	return nil
}

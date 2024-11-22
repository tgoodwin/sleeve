package replay

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcilerHarness struct {
	ReconcilerID       string
	frames             []Frame
	frameDataByFrameID map[string]CacheFrame

	// this is the reconciler that the frames will be replayed against
	// and it should be constructed with p.Client()
	// reconciler reconcile.Reconciler
}

func NewPlayer(reconcilerID string, frames []Frame, frameData map[string]CacheFrame) *ReconcilerHarness {
	return &ReconcilerHarness{
		frames:             frames,
		frameDataByFrameID: frameData,
		ReconcilerID:       reconcilerID,
	}
}

func (p *ReconcilerHarness) ReplayClient() *Client {
	// TODO implement effectRecorder
	return NewClient(p.frameDataByFrameID, nil)
}

func (p *ReconcilerHarness) Load(r reconcile.Reconciler) *Player {
	return &Player{
		reconciler: r,
		harness:    p,
	}
}

type Player struct {
	reconciler reconcile.Reconciler
	harness    *ReconcilerHarness
}

func (r *Player) Run() error {
	for _, f := range r.harness.frames {
		ctx := withFrameID(context.Background(), f.ID)
		fmt.Printf("Replaying frame %s for controller %s\n", f.ID, r.harness.ReconcilerID)
		if _, err := r.reconciler.Reconcile(ctx, f.Req); err != nil {
			return err
		}
	}
	return nil
}

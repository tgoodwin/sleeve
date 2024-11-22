package replay

import (
	"context"
	"fmt"

	"github.com/tgoodwin/sleeve/pkg/event"
	"github.com/tgoodwin/sleeve/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type DataEffect struct {
	Reads  []event.Event
	Writes []event.Event
}

type ReconcilerHarness struct {
	ReconcilerID       string
	frames             []Frame
	frameDataByFrameID map[string]CacheFrame

	// data effect by frameID (reconcileID)
	effects map[string]DataEffect

	// this is the reconciler that the frames will be replayed against
	// and it should be constructed with p.Client()
	// reconciler reconcile.Reconciler
}

func newHarness(reconcilerID string, frames []Frame, frameData map[string]CacheFrame, effects map[string]DataEffect) *ReconcilerHarness {
	return &ReconcilerHarness{
		frames:             frames,
		frameDataByFrameID: frameData,
		ReconcilerID:       reconcilerID,
		effects:            effects,
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
		fmt.Printf("Readset:\n%s\n", formatEventList(r.harness.effects[f.ID].Reads))
		fmt.Printf("Expected Writeset:\n%s\n", formatEventList(r.harness.effects[f.ID].Writes))
		if _, err := r.reconciler.Reconcile(ctx, f.Req); err != nil {
			return err
		}
	}
	return nil
}

func formatEventList(events []event.Event) string {
	if len(events) == 0 {
		return "\t<empty>\n"
	}
	s := ""
	for _, e := range events {
		if event.IsReadOp(e) {
			s += fmt.Sprintf("\t{kind: %s, id: %s, ver: %s}\n", e.Kind, util.Shorter(e.ObjectID), e.Version)
		} else {
			s += fmt.Sprintf("\t{kind: %s, id: %s, op: %s}\n", e.Kind, util.Shorter(e.ObjectID), e.OpType)
		}

	}
	return s
}

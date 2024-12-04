package replay

import (
	"context"
	"fmt"

	"github.com/tgoodwin/sleeve/pkg/event"
	"github.com/tgoodwin/sleeve/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type DataEffect struct {
	Reads  []event.Event
	Writes []event.Event
}

type ReplayHarness struct {
	ReconcilerID       string
	frames             []Frame
	frameDataByFrameID map[string]CacheFrame

	// trace data effect by frameID (reconcileID)
	tracedEffects map[string]DataEffect

	// container for the effects that are recorded during replay
	replayEffects map[string]DataEffect

	predicates []*executionPredicate
}

func newHarness(reconcilerID string, frames []Frame, frameData map[string]CacheFrame, effects map[string]DataEffect) *ReplayHarness {
	replayEffects := make(map[string]DataEffect)
	return &ReplayHarness{
		frames:             frames,
		frameDataByFrameID: frameData,
		ReconcilerID:       reconcilerID,
		tracedEffects:      effects,
		replayEffects:      replayEffects,
		predicates:         make([]*executionPredicate, 0),
	}
}

func (p *ReplayHarness) WithPredicate(predicate Predicate) *ReplayHarness {
	p.predicates = append(p.predicates, &executionPredicate{evaluate: predicate})
	return p
}

func (p *ReplayHarness) ReplayClient(scheme *runtime.Scheme) *Client {
	recorder := &Recorder{
		reconcilerID:    p.ReconcilerID,
		effectContainer: p.replayEffects,
		predicates:      p.predicates,
	}
	return NewClient(scheme, p.frameDataByFrameID, recorder)
}

func (p *ReplayHarness) Load(r reconcile.Reconciler) *Player {
	return &Player{
		reconciler: r,
		harness:    p,
	}
}

type Player struct {
	reconciler reconcile.Reconciler
	harness    *ReplayHarness
}

func (r *Player) Play() error {
	for _, f := range r.harness.frames {
		// skip frames with no writes
		if len(r.harness.tracedEffects[f.ID].Writes) == 0 {
			continue
		}
		ctx := withFrameID(context.Background(), f.ID)
		fmt.Printf("Replaying frame %s for controller %s\n", f.ID, r.harness.ReconcilerID)
		fmt.Printf("Traced Readset:\n%s\n", formatEventList(r.harness.tracedEffects[f.ID].Reads))
		fmt.Printf("Traced Writeset:\n%s\n", formatEventList(r.harness.tracedEffects[f.ID].Writes))

		if _, err := r.reconciler.Reconcile(ctx, f.Req); err != nil {
			fmt.Println("Error during replay:", err)
			return err
		}

		fmt.Printf("Actual Readset:\n%s\n", formatEventList(r.harness.replayEffects[f.ID].Reads))
		fmt.Printf("Actual Writeset:\n%s\n", formatEventList(r.harness.replayEffects[f.ID].Writes))

		// check predicates
		for _, p := range r.harness.predicates {
			if p.satisfied {
				fmt.Println("Predicate satisfied!!!")
				// TODO return
			}
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

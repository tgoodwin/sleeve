package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/tgoodwin/sleeve/pkg/event"
	"github.com/tgoodwin/sleeve/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type frameIDKey struct{}

func withFrameID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, frameIDKey{}, id)
}

func frameIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(frameIDKey{}).(string)
	if !ok {
		return ""
	}
	return id
}

// Like the frames of a movie, a Frame is a snapshot of the state of the world at a particular point in time.
type Frame struct {
	// for ordering
	sequenceID string
	// snaps map[schema.GroupVersionKind]map[string]client.Object
	ID  string
	Req reconcile.Request
	// Elements map[snapshot.VersionKey]struct{}
}

// keyed by Type (Kind) and then by NamespacedName
type CacheFrame map[string]map[types.NamespacedName]*unstructured.Unstructured

func DumpCacheFrameContents(c CacheFrame) {
	fmt.Println("CacheFrame contents:")
	for kind, objs := range c {
		for nn := range objs {
			fmt.Printf("\t%s/%s/%s\n", kind, nn.Namespace, nn.Name)
		}
	}
}

func ParseTrace(traceData []byte) (*Builder, error) {
	b := &Builder{}
	if err := b.fromTrace(traceData); err != nil {
		return nil, err
	}
	return b, nil
}

// TODO
// - need to index snapshot records by Kind somehow to support List operations
// - have the client hold some map of reconcileID -> map of NamespacedName => client.Object
// - probably want to have the Client's Get / List methods infer the Kind of the object from the object itself
// - and then use that to key into the map of snapshot records
// - and perhaps in the client we can pull the reconcileID out of the context
// - and we will set the reconcileID in the context in the Replayer

// Builder handles the replaying of a sequence of frames to a given reconciler.
type Builder struct {
	// object versions found in the trace
	*replayStore

	// controller operations found in the trace
	events []event.Event

	// for bookkeeping and validation
	reconcilerIDs map[string]struct{}
}

func (b *Builder) fromTrace(traceData []byte) error {
	rs := newReplayStore()
	if err := rs.HydrateFromTrace(traceData); err != nil {
		return err
	}
	b.replayStore = rs

	// track all reconciler IDs in the trace
	b.reconcilerIDs = make(map[string]struct{})

	lines := strings.Split(string(traceData), "\n")
	events, err := ParseEventsFromLines(lines)
	if err != nil {
		return err
	}
	fmt.Println("total events", len(events))

	// filter events to only include those that are reads
	readEvents := lo.Filter(events, func(e event.Event, _ int) bool {
		return e.OpType == "GET" || e.OpType == "LIST"
	})

	// for each read event, sanity check that the object is in the store
	// if not, return an error
	for _, e := range readEvents {
		key := snapshot.VersionKey{Kind: e.Kind, ObjectID: e.ObjectID, Version: e.Version}
		if _, ok := b.store[key]; !ok {
			return fmt.Errorf("object not found in store: %#v", key)
		}
		b.reconcilerIDs[e.ControllerID] = struct{}{}
	}

	b.events = events
	for controllerID := range b.reconcilerIDs {
		fmt.Println("Found controllerID in trace", controllerID)
	}

	return nil
}

func (b *Builder) BuildHarness(controllerID string) (*ReplayHarness, error) {
	if _, ok := b.reconcilerIDs[controllerID]; !ok {
		return nil, fmt.Errorf("controllerID not found in trace: %s", controllerID)
	}

	controllerEvents := lo.Filter(b.events, func(e event.Event, _ int) bool {
		return e.ControllerID == controllerID
	})
	byReconcileID := lo.GroupBy(controllerEvents, func(e event.Event) string {
		return e.ReconcileID
	})

	frameData := make(map[string]CacheFrame)
	frames := make([]Frame, 0)
	effects := make(map[string]DataEffect)

	for reconcileID, events := range byReconcileID {
		earliestTs := events[0].Timestamp

		reads, writes := event.FilterReadsWrites(events)
		effects[reconcileID] = DataEffect{Reads: reads, Writes: writes}
		req, err := b.inferReconcileRequestFromReadset(controllerID, reads)
		if err != nil {
			return nil, err
		}
		cacheFrame, err := b.generateCacheFrame(reads)
		if err != nil {
			return nil, err
		}
		frameData[reconcileID] = cacheFrame
		frames = append(frames, Frame{ID: reconcileID, Req: req, sequenceID: earliestTs})
	}

	// sort the frames by sequenceID
	sort.Slice(frames, func(i, j int) bool {
		return frames[i].sequenceID < frames[j].sequenceID
	})

	harness := newHarness(controllerID, frames, frameData, effects)
	return harness, nil
}

func (r *Builder) generateCacheFrame(events []event.Event) (CacheFrame, error) {
	cacheFrame := make(CacheFrame)
	for _, e := range events {
		key := snapshot.VersionKey{Kind: e.Kind, ObjectID: e.ObjectID, Version: e.Version}
		if obj, ok := r.store[key]; ok {
			if _, ok := cacheFrame[e.Kind]; !ok {
				cacheFrame[e.Kind] = make(map[types.NamespacedName]*unstructured.Unstructured)
			}
			cacheFrame[e.Kind][types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}] = obj
		} else {
			return nil, fmt.Errorf("generating cache frame: object not found in store: %#v", key)
		}
	}
	return cacheFrame, nil
}

func (r *Builder) inferReconcileRequestFromReadset(controllerID string, readset []event.Event) (reconcile.Request, error) {
	for _, e := range readset {
		if e.Kind == controllerID {
			objKey := snapshot.VersionKey{Kind: e.Kind, ObjectID: e.ObjectID, Version: e.Version}
			if obj, ok := r.store[objKey]; ok {
				name := obj.GetName()
				namespace := obj.GetNamespace()
				req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: name}}
				return req, nil
			}
		}
	}
	return reconcile.Request{}, fmt.Errorf("could not infer reconcile.Request from readset")
}

type replayStore struct {
	// indexes all of the objects in the trace
	store map[snapshot.VersionKey]*unstructured.Unstructured
	mu    sync.RWMutex
}

func newReplayStore() *replayStore {
	return &replayStore{
		store: make(map[snapshot.VersionKey]*unstructured.Unstructured),
	}
}

func (f *replayStore) Add(r snapshot.Record) error {
	// Unmarshal the value into an unstructured object
	key := snapshot.VersionKey{Kind: r.Kind, ObjectID: r.ObjectID, Version: r.Version}
	// fmt.Printf("adding key to store: %#v\n", key)
	obj := unstructured.Unstructured{}
	if err := json.Unmarshal([]byte(r.Value), &obj); err != nil {
		return errors.Wrap(err, "unmarshaling record value")
	}

	// Generate the key using namespace/name
	// key := obj.GetNamespace() + "/" + obj.GetName()
	f.mu.Lock()
	f.store[key] = &obj
	f.mu.Unlock()

	return nil
}

func (f *replayStore) HydrateFromTrace(traceData []byte) error {
	lines := strings.Split(string(traceData), "\n")
	records, err := ParseRecordsFromLines(lines)
	if err != nil {
		return err
	}

	for _, r := range records {
		if err := f.Add(r); err != nil {
			return err
		}
	}
	fmt.Println("total record observations in trace", len(records))
	fmt.Println("unique records in store after hydration", len(f.store))

	return nil
}

// func (f *replayStore) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
// 	f.mu.RLock()
// 	defer f.mu.RUnlock()

// 	// Construct the key as "namespace/name"
// 	cacheKey := fmt.Sprintf("%s/%s", key.Namespace, key.Name)
// 	if cachedObj, exists := f.store[cacheKey]; exists {
// 		// Use DeepCopyObject to create a deep copy
// 		deepCopiedObj, ok := cachedObj.DeepCopyObject().(client.Object)
// 		if !ok {
// 			return fmt.Errorf("failed to cast deep copied object to client.Object")
// 		}

// 		// Use json.Marshal and json.Unmarshal to populate the obj parameter
// 		data, err := json.Marshal(deepCopiedObj)
// 		if err != nil {
// 			return fmt.Errorf("failed to marshal cached object: %v", err)
// 		}

// 		if err := json.Unmarshal(data, obj); err != nil {
// 			return fmt.Errorf("failed to unmarshal into provided object: %v", err)
// 		}

// 		return nil
// 	}
// 	gvk := obj.GetObjectKind().GroupVersionKind()
// 	gr := schema.GroupResource{
// 		Group:    gvk.Group,
// 		Resource: gvk.Kind, // Assuming Kind is used as Resource here
// 	}
// 	return apierrors.NewNotFound(gr, key.Name)
// }

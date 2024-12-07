package replay

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/tgoodwin/sleeve/pkg/event"
	"github.com/tgoodwin/sleeve/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type replayStore struct {
	// indexes all of the objects in the trace
	store map[event.CausalKey]*unstructured.Unstructured
	mu    sync.RWMutex
}

func newReplayStore() *replayStore {
	return &replayStore{
		store: make(map[event.CausalKey]*unstructured.Unstructured),
	}
}

func (f *replayStore) Add(r snapshot.Record) error {
	// Unmarshal the value into an unstructured object
	// key := snapshot.VersionKey{Kind: r.Kind, ObjectID: r.ObjectID, Version: r.Version}
	obj := r.ToUnstructured()
	key, err := event.GetCausalKey(obj)
	if err != nil {
		return errors.Wrap(err, "inserting object into replay store")
	}

	f.mu.Lock()
	f.store[key] = obj
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
			fmt.Printf("error adding record to store: %v\n", r.ObjectID)
			continue
		}
	}
	fmt.Println("total record observations in trace", len(records))
	fmt.Println("unique records in store after hydration", len(f.store))

	return nil
}

func (f *replayStore) AllOfKind(kind string) []*unstructured.Unstructured {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var objs []*unstructured.Unstructured
	for _, obj := range f.store {
		if obj.GetKind() == kind {
			objs = append(objs, obj)
		}
	}
	sort.Slice(objs, func(i, j int) bool {
		return objs[i].GetResourceVersion() < objs[j].GetResourceVersion()
	})
	return objs
}

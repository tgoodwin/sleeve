package replay

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/tgoodwin/sleeve/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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

func (f *replayStore) AllForKind(kind string) []*unstructured.Unstructured {
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

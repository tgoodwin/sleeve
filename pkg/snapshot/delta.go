package snapshot

import (
	"bufio"
	"encoding/json"
	"io"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type uniqueKey struct {
	Kind     string
	ObjectID string
	Version  string
}

func ReadFile(f io.Reader) ([]Record, error) {
	seen := make(map[uniqueKey]struct{})
	scanner := bufio.NewScanner(f)
	records := make([]Record, 0)
	for scanner.Scan() {
		r, err := LoadFromString(scanner.Text())
		if err != nil {
			return nil, err
		}
		key := uniqueKey{r.Kind, r.ObjectID, r.Version}
		if _, ok := seen[key]; !ok {
			records = append(records, r)
			seen[key] = struct{}{}
		}
	}
	return records, nil
}

func GroupByID(records []Record) map[string][]Record {
	groups := make(map[string][]Record)
	for _, r := range records {
		if _, ok := groups[r.ObjectID]; !ok {
			groups[r.ObjectID] = make([]Record, 0)
		}
		groups[r.ObjectID] = append(groups[r.ObjectID], r)
	}
	return groups
}

func LoadFromString(s string) (Record, error) {
	var r Record
	err := json.Unmarshal([]byte(s), &r)
	return r, err
}

func (r Record) Diff(other Record) (string, error) {
	this := unstructured.Unstructured{}
	otherObj := unstructured.Unstructured{}
	if err := json.Unmarshal([]byte(r.Value), &this); err != nil {
		return "", err
	}
	if err := json.Unmarshal([]byte(other.Value), &otherObj); err != nil {
		return "", err
	}
	return computeDelta(&this, &otherObj), nil
}

var toIgnore = map[string]struct{}{
	"resourceVersion": {},
	"managedFields":   {},
	"generation":      {},

	// sleeve labels
	"discrete.events/change-id":               {},
	"discrete.events/creator-id":              {},
	"discrete.events/root-event-id":           {},
	"discrete.events/prev-write-reconcile-id": {},
}

func shouldIgnore(k string, v interface{}) bool {
	if _, ok := toIgnore[k]; ok {
		return true
	}
	return false
}

func computeDelta(old, new *unstructured.Unstructured) string {
	cmpOpt := cmpopts.IgnoreMapEntries(shouldIgnore)
	var r DiffReporter
	cmp.Diff(old, new, cmpOpt, cmp.Reporter(&r))
	rdiff := r.String()
	return rdiff
}

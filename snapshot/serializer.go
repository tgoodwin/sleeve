package snapshot

import (
	"encoding/json"
	"fmt"
	"log"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// fields to mask over:
// - UID
// - CreationTimestamp

type Value struct {
	Version string `json:"version"`
	Data    string `json:"data"`
}

var toMask = map[string]struct{}{
	"UID":               {},
	"ResourceVersion":   {},
	"Generation":        {},
	"CreationTimestamp": {},
	// TODO just distinguish between nil and not-nil for purposes of comparison
	"DeletionTimestamp": {},
}

func serialize(obj interface{}) map[string]string {
	data, err := json.Marshal(obj)
	if err != nil {
		log.Fatalf("Error marshaling struct to JSON: %v", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(data, &resultMap); err != nil {
		log.Fatalf("Error unmarshaling JSON to map: %v", err)
	}

	stringMap := make(map[string]string)
	for k, v := range resultMap {
		stringMap[k] = fmt.Sprintf("%v", v)
	}
	return stringMap
}

func maskFields(in map[string]string) map[string]string {
	masked := make(map[string]string)
	for k := range in {
		if _, ok := toMask[k]; ok {
			continue
		}
		masked[k] = in[k]
	}
	return masked
}

// TODO this needs to handle nested objects
func Serialize(obj client.Object) string {
	serialized := serialize(obj)
	masked := maskFields(serialized)
	asJSON, err := json.Marshal(masked)
	if err != nil {
		log.Fatalf("Error marshaling masked object to JSON: %v", err)
	}
	return string(asJSON)
}

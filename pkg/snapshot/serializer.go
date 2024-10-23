package snapshot

import (
	"encoding/json"
	"log"
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

func serialize(obj interface{}) map[string]interface{} {
	data, err := json.Marshal(obj)
	if err != nil {
		log.Fatalf("Error marshaling struct to JSON: %v", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(data, &resultMap); err != nil {
		log.Fatalf("Error unmarshaling JSON to map: %v", err)
	}
	return resultMap
}

func maskFields(in map[string]string) map[string]interface{} {
	masked := make(map[string]interface{})
	for k := range in {
		if _, ok := toMask[k]; ok {
			continue
		}
		masked[k] = in[k]
	}
	return masked
}

// check out https://github.com/cisco-open/k8s-objectmatcher

// TODO this needs to handle nested objects
func Serialize(obj interface{}) string {
	serialized := serialize(obj)
	asJSON, _ := json.Marshal(serialized)
	return string(asJSON)
}

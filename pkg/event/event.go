package event

import (
	"encoding/json"
	"fmt"
	"sort"
)

type Event struct {
	Timestamp    string            `json:"timestamp"`
	ReconcileID  string            `json:"reconcile_id"`
	ControllerID string            `json:"controller_id"`
	RootEventID  string            `json:"root_event_id"`
	OpType       string            `json:"op_type"`
	Kind         string            `json:"kind"`
	ObjectID     string            `json:"object_id"`
	Version      string            `json:"version"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// Ensure Event implements the json.Marshaler and json.Unmarshaler interfaces
var _ json.Marshaler = (*Event)(nil)
var _ json.Unmarshaler = (*Event)(nil)

func (e *Event) UnmarshalJSON(data []byte) error {
	type Alias Event
	aux := &struct {
		*Alias
		Labels map[string]string `json:"-"`
	}{
		Alias:  (*Alias)(e),
		Labels: make(map[string]string),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal Event: %w", err)
	}

	// Populate the Labels map with keys that have the prefix "label:"
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return fmt.Errorf("failed to unmarshal raw data: %w", err)
	}

	for key, value := range rawMap {
		if len(key) > 6 && key[:6] == "label:" {
			if strValue, ok := value.(string); ok {
				aux.Labels[key[6:]] = strValue
			}
		}
	}

	e.Labels = aux.Labels
	return nil
}

func (e Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(&e),
	}

	// Marshal the alias struct to JSON
	auxData, err := json.Marshal(aux)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Event: %w", err)
	}
	// Unmarshal the JSON back into a map
	var dataMap map[string]interface{}
	if err := json.Unmarshal(auxData, &dataMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal aux data: %w", err)
	}

	// Add the labels with the "label:" prefix in a sorted order
	labelKeys := make([]string, 0, len(e.Labels))
	for key := range e.Labels {
		labelKeys = append(labelKeys, key)
	}
	sort.Strings(labelKeys)

	for _, key := range labelKeys {
		dataMap["label:"+key] = e.Labels[key]
	}
	delete(dataMap, "labels")

	// Marshal the final map to JSON
	return json.Marshal(dataMap)
}
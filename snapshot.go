package client

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ObjectVersion represents the version of a Kubernetes resource
type ObjectVersion struct {
	Kind    string `json:"kind"`
	Uid     string `json:"uid"`
	Version string `json:"version"`
	TraceID string `json:"traceID"` // causal reference to some top-level declarative state change
}

// VersionSet represents a collection of Kubernetes resources and their versions
type VersionSet map[ObjectVersion]struct{}

type Observation struct {
	timestamp string
	elements  VersionSet
}

type LocalKnowledge = []Observation

func (o ObjectVersion) NewerThan(other ObjectVersion) bool {
	return o.Version > other.Version
}

func (s VersionSet) Contains(other VersionSet) bool {
	for k := range other {
		if _, ok := s[k]; !ok {
			return false
		}
	}
	return true
}

// Intersection can be used to represent common knowledge between two sets of versions.
func (s VersionSet) Intersection(other VersionSet) VersionSet {
	result := make(VersionSet)
	for k := range s {
		if _, ok := other[k]; ok {
			result[k] = struct{}{}
		}
	}
	return result
}

func (s VersionSet) Diff(o VersionSet) VersionSet {
	result := make(VersionSet)
	for k := range s {
		if _, ok := o[k]; !ok {
			result[k] = struct{}{}
		}
	}
	return result
}

// Observation represents the result of a List operation on a Kubernetes resource at a given point in time
func (o Observation) Precedes(other Observation) bool {
	return o.timestamp < other.timestamp
}

func (o Observation) Contains(other Observation) bool {
	return o.elements.Contains(other.elements)
}

type key schema.GroupVersionKind

type Snapshot struct {
	Objects         map[key]VersionSet
	Timestamp       time.Time
	ResourceVersion string
}

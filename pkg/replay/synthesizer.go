package replay

import (
	"fmt"
	"sort"

	"github.com/tgoodwin/sleeve/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// detect if a controller failed to observe certain states

// 1 - determine which read dependencies the controller has
// 2 - determine if there are any versions of the read dependencies that the controller failed to observe

func (b *Builder) FindMissedObservations(controllerID string) error {
	if _, ok := b.reconcilerIDs[controllerID]; !ok {
		return fmt.Errorf("controllerID not found in trace: %s", controllerID)
	}

	harness, err := b.BuildHarness(controllerID)
	if err != nil {
		return err
	}

	readDependencies := make(map[string]struct{})
	knowledgeByKind := make(map[string]snapshot.KnowledgeSet)
	for _, effects := range harness.tracedEffects {
		for _, e := range effects.Reads {
			readDependencies[e.Kind] = struct{}{}
			if _, ok := knowledgeByKind[e.Kind]; !ok {
				knowledgeByKind[e.Kind] = make(snapshot.KnowledgeSet)
			}
			knowledgeByKind[e.Kind].Add(e.VersionKey())
		}
	}

	for kind := range readDependencies {
		allOfKind := b.AllOfKind(kind)
		diffList, err := showDiffs(allOfKind)
		if err != nil {
			fmt.Printf("error showing diffs for kind %s: %v\n", kind, err)
		}

		diffByCurrVersion := make(map[snapshot.VersionKey]diff)
		for _, d := range diffList {
			diffByCurrVersion[d.curr] = d
		}

		allKnowledge := asKnowledge(allOfKind)
		localKnowledge := knowledgeByKind[kind]
		knowledgeDiff := allKnowledge.Diff(localKnowledge)
		if len(knowledgeDiff) > 0 {
			fmt.Printf("controller %s %d missed observations for %s\n", controllerID, len(knowledgeDiff), kind)
			for key := range knowledgeDiff {
				fmt.Printf("missed observation: %s\n", key)
				if d, ok := diffByCurrVersion[key]; ok {
					fmt.Printf("%s\n\t%s\n", key, d.delta)
				}
			}
		}
	}

	return nil
}

func asKnowledge(elems []*unstructured.Unstructured) snapshot.KnowledgeSet {
	versions := make(snapshot.KnowledgeSet)
	for _, elem := range elems {
		key := snapshot.VersionKey{
			Kind:     elem.GetKind(),
			ObjectID: string(elem.GetUID()),
			Version:  elem.GetResourceVersion(),
		}
		versions[key] = struct{}{}
	}
	return versions
}

type diff struct {
	prev  snapshot.VersionKey
	curr  snapshot.VersionKey
	delta string
}

func showDiffs(versions []*unstructured.Unstructured) ([]diff, error) {

	if len(versions) == 0 {
		return nil, nil
	}

	kind := versions[0].GetKind()
	for _, v := range versions {
		if v.GetKind() != kind {
			return nil, fmt.Errorf("error: versions contain different kinds: %s and %s", kind, v.GetKind())
		}
	}

	// defensively sort the versions vy resource version
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GetResourceVersion() < versions[j].GetResourceVersion()
	})

	diffList := make([]diff, 0)

	// iterate over the list, comparing each element to the next
	for i := 0; i < len(versions)-1; i++ {
		a := versions[i]
		b := versions[i+1]
		diffstr := snapshot.ComputeDelta(a, b)
		d := diff{
			// TODO ITS UNCLEAR WHETHER OR NOT WE SHOULD USE UID OR NAME HERE, AS UID IS NOT ALWAYS SET
			prev:  snapshot.VersionKey{Kind: kind, ObjectID: string(a.GetUID()), Version: a.GetResourceVersion()},
			curr:  snapshot.VersionKey{Kind: kind, ObjectID: string(b.GetUID()), Version: b.GetResourceVersion()},
			delta: diffstr,
		}
		diffList = append(diffList, d)
		fmt.Printf("diff between %s and %s:\n%s\n", a.GetResourceVersion(), b.GetResourceVersion(), diffstr)
	}
	return diffList, nil
}

func getCausalID(obj *unstructured.Unstructured) (string, error) {
	labels := obj.GetLabels()
	if labels == nil {
		return "", fmt.Errorf("object has no labels")
	}
	if causalID, ok := labels["discrete.events/change-id"]; ok {
		return causalID, nil
	}
	if rootID, ok := labels["discrete.events/root-event-id"]; ok {
		return rootID, nil
	}
	return "", fmt.Errorf("object has no causal ID")
}

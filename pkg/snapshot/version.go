package snapshot

// VersionKey uniquely identifies the state of an object at a given version
type VersionKey struct {
	Kind     string
	ObjectID string
	Version  string
}

type KnowledgeSet map[VersionKey]struct{}

func (ks KnowledgeSet) Add(vk VersionKey) {
	ks[vk] = struct{}{}
}

// Diff returns the set difference between ks and other
func (ks KnowledgeSet) Diff(other KnowledgeSet) KnowledgeSet {
	diff := make(KnowledgeSet)
	for key := range ks {
		if _, found := other[key]; !found {
			diff[key] = struct{}{}
		}
	}
	return diff
}

package snapshot

// VersionKey uniquely identifies the state of an object at a given version
type VersionKey struct {
	Kind     string
	ObjectID string
	Version  string
}

package replay

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type FrameType string

const (
	FrameTypeTraced    FrameType = "TRACED"
	FrameTypeSynthetic FrameType = "SYNTHETIC"
)

// Like the frames of a movie, a Frame is a snapshot of the state of the world at a particular point in time.
type Frame struct {
	ID   string
	Type FrameType

	// for ordering. In practice this is just a timestamp
	sequenceID string

	Req reconcile.Request

	TraceyRootID string
}

type frameIDKey struct{}

func withFrameID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, frameIDKey{}, id)
}

func frameIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(frameIDKey{}).(string)
	if !ok {
		return ""
	}
	return id
}

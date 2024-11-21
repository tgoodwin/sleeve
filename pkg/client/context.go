package client

import "sync"

type ReconcileContext struct {
	reconcileID string
	rootID      string

	mu sync.Mutex
}

func (rc *ReconcileContext) SetReconcileID(id string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.reconcileID = id
}

func (rc *ReconcileContext) SetRootID(id string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.rootID = id
}

func (rc *ReconcileContext) GetReconcileID() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.reconcileID
}

func (rc *ReconcileContext) GetRootID() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.rootID
}

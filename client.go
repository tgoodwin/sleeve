package client

import (
	"context"
	"fmt"

	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("sleeveless")

// enum for controller operation types
type OperationType string

var (
	GET    OperationType = "GET"
	LIST   OperationType = "LIST"
	CREATE OperationType = "CREATE"
	UPDATE OperationType = "UPDATE"
	DELETE OperationType = "DELETE"
	PATCH  OperationType = "PATCH"
)

func createFixedLengthHash() string {
	// Get the current time
	currentTime := time.Now()

	// Convert the current time to a byte slice
	timeBytes := []byte(currentTime.String())

	// Hash the byte slice using SHA256
	hash := sha256.Sum256(timeBytes)

	// Convert the hash to a fixed length string
	hashString := hex.EncodeToString(hash[:])

	// Take the first 6 characters of the hash string
	shortHash := hashString[:6]

	return shortHash
}

type Client struct {
	// this syntax is "embedding" the client.Client interface in the Client struct
	// this means that the Client struct will have all the methods of the client.Client interface.
	// below, we will override some of these methods to add our own behavior.
	client.Client

	// identifier for the reconciler (controller name)
	id string

	// used to scope observations to a given Reconcile invocation
	reconcileID string

	// root event ID
	rootID string

	logger logr.Logger
}

var _ client.Client = &Client{}

func newClient(wrapped client.Client) *Client {
	return &Client{
		Client: wrapped,
		logger: log,
	}
}

func Wrap(c client.Client) *Client {
	return newClient(c)
}

func (c *Client) WithName(name string) *Client {
	c.id = name
	return c
}

func (c *Client) StartReconcileContext() func() {
	if c.reconcileID != "" {
		// unsure if this should never happen or not.
		// if it does, then we should store reconcileIDs on the client struct as a map
		panic("concurrent reconcile invocations detected")
	}
	// set a reconcileID for this invocation
	c.reconcileID = createFixedLengthHash()
	c.logger.WithValues(
		"ReconcileID", c.reconcileID,
		"TimestampNS", fmt.Sprintf("%d", time.Now().UnixNano()),
	).Info("Reconcile context started")
	return func() {
		c.logger.WithValues(
			"ReconcileID", c.reconcileID,
			"TimestampNS", fmt.Sprintf("%d", time.Now().UnixNano()),
		).Info("Reconcile context ended")

		// reset temporary state
		c.reconcileID = ""
		c.rootID = ""
	}
}

func (c *Client) logObservation(ov ObjectVersion, op OperationType) {
	c.logger.WithValues(
		"Timestamp", fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond)),
		"ReconcileID", c.reconcileID,
		"CreatorID", c.id,
		"RootEventID", c.rootID,
		"OperationType", fmt.Sprintf("%v", op),
		"ObservedObjectKind", fmt.Sprintf("%+v", ov.Kind),
		"ObservedObjectUID", fmt.Sprintf("%+v", ov.Uid),
		"ObservedObjectVersion", fmt.Sprintf("%+v", ov.Version),
	).Info("log-observation")
}

func (c *Client) setRootContext(obj client.Object) {
	labels := obj.GetLabels()
	rootID, ok := labels[TRACEY_WEBHOOK_LABEL]
	if !ok {
		return
	}
	if c.rootID != "" && c.rootID != rootID {
		c.logger.WithValues(
			"RootID", c.rootID,
			"NewRootID", rootID,
		).Error(nil, "Root context changed")
	}
	c.rootID = rootID
	c.logger.WithValues(
		"RootID", c.rootID,
		"ObjectKind", obj.GetObjectKind().GroupVersionKind().String(),
		"ObjectUID", obj.GetUID(),
	).Info("Root context set")
}

// func (c *Client) setLabelContext(obj client.Object) {
// 	labels := obj.GetLabels()
// 	rootID, ok := labels[TRACEY_WEBHOOK_LABEL]
// 	c.lc.SourceObject = string(obj.GetUID())
// 	if !ok {
// 		return
// 	}
// 	c.lc.RootID = rootID
// 	if _, ok := labels[TRACEY_PARENT_ID]; !ok {
// 		c.lc.ParentID = rootID
// 	}
// 	if traceID, ok := labels[TRACEY_RECONCILE_ID]; ok {
// 		c.lc.TraceID = traceID
// 	}
// 	c.logger.WithValues(
// 		"RootID", c.lc.RootID,
// 		"ParentID", c.lc.ParentID,
// 		"TraceID", c.lc.TraceID,
// 	).Info("Label context set")
// }

func (c *Client) propagateLabels(obj client.Object) {
	currLabels := obj.GetLabels()
	out := make(map[string]string)
	for k, v := range currLabels {
		out[k] = v
	}
	out[TRACEY_CREATOR_ID] = c.id
	out[TRACEY_ROOT_ID] = c.rootID
	out[TRACEY_RECONCILE_ID] = c.reconcileID

	c.logger.WithValues(
		"RootID", c.rootID,
		"ReconcileID", c.reconcileID,
		"ObjectUID", obj.GetUID(),
	).Info("Propagating labels")
	obj.SetLabels(out)
}

func (c *Client) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.propagateLabels(obj)
	res := c.Client.Create(ctx, obj, opts...)
	c.logObservation(RecordSingle(obj), CREATE)
	return res

}

func (c *Client) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	c.propagateLabels(obj)
	res := c.Client.Delete(ctx, obj, opts...)
	c.logObservation(RecordSingle(obj), DELETE)
	return res
}

func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	res := c.Client.Get(ctx, key, obj, opts...)
	c.setRootContext(obj)
	c.logObservation(RecordSingle(obj), GET)
	return res
}

func (c *Client) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	// TODO log observation for each item in the list
	// this is hard cause we don't have access to list.Items without knowing the concrete type
	// so we may have to re-implement below the controller-runtime level to be able to do this.
	return c.Client.List(ctx, list, opts...)
}

func (c *Client) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	// need to record the knowledge snapshot this update is based on
	c.propagateLabels(obj)
	c.logObservation(RecordSingle(obj), UPDATE)
	res := c.Client.Update(ctx, obj, opts...)
	return res
}

func (c *Client) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	// TODO verify labels propagate correctly under patch
	c.propagateLabels(obj)
	res := c.Client.Patch(ctx, obj, patch, opts...)
	c.logObservation(RecordSingle(obj), PATCH)
	return res
}

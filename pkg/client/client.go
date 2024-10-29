package client

import (
	"context"
	"fmt"
	"reflect"

	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"github.com/tgoodwin/sleeve/pkg/event"
	"github.com/tgoodwin/sleeve/pkg/snapshot"
	"github.com/tgoodwin/sleeve/pkg/tag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ctrl "sigs.k8s.io/controller-runtime/pkg/controller"
)

var log = logf.Log.WithName("sleeveless")

// enum for controller operation types
type OperationType string

var (
	INIT   OperationType = "INIT"
	GET    OperationType = "GET"
	LIST   OperationType = "LIST"
	CREATE OperationType = "CREATE"
	UPDATE OperationType = "UPDATE"
	DELETE OperationType = "DELETE"
	PATCH  OperationType = "PATCH"
)

var OPERATION_KEY = "sleeve:controller-operation"
var OBJECT_VERSION_KEY = "sleeve:object-version"

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

	config *Config
}

var _ client.Client = (*Client)(nil)

func newClient(wrapped client.Client) *Client {
	return &Client{
		Client: wrapped,
		logger: log,
		config: NewConfig(),
	}
}

func Wrap(c client.Client) *Client {
	return newClient(c)
}

func (c *Client) WithName(name string) *Client {
	c.id = name
	return c
}

func StartReconcileContext(client client.Client) func() {
	c, ok := client.(*Client)
	if !ok {
		panic("client is not a tracey client")
	}
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

	cleanup := func() {
		c.logger.WithValues(
			"ReconcileID", c.reconcileID,
			"TimestampNS", fmt.Sprintf("%d", time.Now().UnixNano()),
		).Info("Reconcile context ended")

		// reset temporary state
		c.reconcileID = ""
		c.rootID = ""
	}
	return cleanup
}

func (c *Client) setReconcileID(ctx context.Context) {
	rid := string(ctrl.ReconcileIDFromContext(ctx))
	if rid == "" {
		// this should never happen given our assumptions
		panic("reconcileID not set in context")
	}

	if rid != c.reconcileID {
		// we are entering a new reconcile invocation
		// first, clear out stuff
		c.logger.V(2).Info("reconcileID changed", "old", c.reconcileID, "new", rid)
		c.rootID = ""
		// then, update to the new reconcileID.
		c.reconcileID = string(rid)
	}
}

func (c *Client) logOperation(obj client.Object, op OperationType) {
	event := &event.Event{
		Timestamp:    fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond)),
		ReconcileID:  c.reconcileID,
		ControllerID: c.id,
		RootEventID:  c.rootID,
		OpType:       string(op),
		Kind:         obj.GetObjectKind().GroupVersionKind().Kind,
		ObjectID:     string(obj.GetUID()),
		Version:      obj.GetResourceVersion(),
		Labels:       obj.GetLabels(),
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		panic("failed to marshal event")
	}
	c.logger.WithValues("LogType", OPERATION_KEY).Info(string(eventJSON))
}

func (c *Client) logObjectVersion(obj client.Object) {
	if c.config.LogObjectSnapshots {
		r := snapshot.RecordValue(obj)
		c.logger.WithValues("LogType", "object-version").Info(r)
	}
}

func (c *Client) setRootContext(obj client.Object) {
	labels := obj.GetLabels()
	// set by the webhook
	rootID, ok := labels[tag.TRACEY_WEBHOOK_LABEL]
	if !ok {
		rootID, ok = labels[tag.TRACEY_ROOT_ID]
		if !ok {
			// no root context to set
			c.logger.V(2).Error(nil, "Root context not set")
			return
		}
	}
	if c.rootID != "" && c.rootID != rootID {
		c.logger.WithValues(
			"RootID", c.rootID,
			"NewRootID", rootID,
		).V(2).Error(nil, "Root context changed")
	}
	c.rootID = rootID
	c.logger.V(2).WithValues(
		"RootID", c.rootID,
		"ObjectKind", obj.GetObjectKind().GroupVersionKind().String(),
		"ObjectUID", obj.GetUID(),
	).Info("Root context set")
}

func (c *Client) propagateLabels(obj client.Object) {
	currLabels := obj.GetLabels()
	out := make(map[string]string)
	for k, v := range currLabels {
		out[k] = v
	}
	out[tag.TRACEY_CREATOR_ID] = c.id
	out[tag.TRACEY_ROOT_ID] = c.rootID
	out[tag.TRACEY_RECONCILE_ID] = c.reconcileID

	obj.SetLabels(out)
}

func (c *Client) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.setReconcileID(ctx)
	tag.LabelChange(obj)
	c.logOperation(obj, CREATE)
	c.propagateLabels(obj)
	res := c.Client.Create(ctx, obj, opts...)
	return res
}

func (c *Client) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	c.setReconcileID(ctx)
	// its important taht we propagate AFTER logging so we update the labels with the latest reconcileID
	// after logging the prior reconcileID on the object

	// ACTUALLY maybe we dont need this since we should assume that all updates are preceded by a GET that has the prior reconcileID
	c.logOperation(obj, DELETE)
	c.propagateLabels(obj)
	res := c.Client.Delete(ctx, obj, opts...)
	return res
}

func (c *Client) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	c.setReconcileID(ctx)
	// TODO do we really need to propagate labels after logging the deletion?
	// - tgoodwin 10-23-2024
	c.logOperation(obj, DELETE)
	c.propagateLabels(obj)
	res := c.Client.DeleteAllOf(ctx, obj, opts...)
	return res

}

func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	c.setReconcileID(ctx)
	// cast back to a client.Ojbject
	objCopy, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		panic("object does not implement client.Object")
	}

	if err := c.Client.Get(ctx, key, objCopy, opts...); err != nil {
		return err
	}
	isVisible := c.isVisible(objCopy)
	if !isVisible {
		return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
	}
	err := c.Client.Get(ctx, key, obj, opts...)
	c.setRootContext(obj)
	c.logOperation(obj, GET)
	c.logObjectVersion(obj)
	return err
}

func (c *Client) isVisible(obj client.Object) bool {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if visDelay, ok := c.config.visibilityDelayByKind[kind]; ok {
		now := time.Now()
		created := obj.GetCreationTimestamp().Time
		if now.Sub(created) < visDelay {
			c.logger.WithValues(
				"ObjectKind", kind,
				"ObjectUID", obj.GetUID(),
				"TimeSinceCreated", now.Sub(created),
			).V(1).Info("Object not visible yet")
			return false
		}
		return true
	}
	return true
}

func (c *Client) Observe(ctx context.Context, obj client.Object) {
	c.setReconcileID(ctx)
	c.logOperation(obj, GET)
}

func (c *Client) filterVisible(objs []client.Object) []client.Object {
	visible := make([]client.Object, 0)
	for _, obj := range objs {
		if c.isVisible(obj) {
			visible = append(visible, obj)
		}
	}
	return visible
}

func (c *Client) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	c.setReconcileID(ctx)
	// Perform the List operation on the wrapped client
	lc := list.DeepCopyObject().(client.ObjectList)
	if err := c.Client.List(ctx, lc, opts...); err != nil {
		return err
	}

	// use reflection to get the Items field from the result
	itemsValue := reflect.ValueOf(lc).Elem().FieldByName("Items")
	if !itemsValue.IsValid() {
		return fmt.Errorf("unable to get Items field from list")
	}

	// create a new slice to hold the items
	out := reflect.MakeSlice(itemsValue.Type(), 0, itemsValue.Len())
	for i := 0; i < itemsValue.Len(); i++ {
		item := itemsValue.Index(i).Addr().Interface().(client.Object)
		c.logOperation(item, LIST)
		out = reflect.Append(out, itemsValue.Index(i))
	}

	// Set the items back to the original list
	originalItemsValue := reflect.ValueOf(list).Elem().FieldByName("Items")
	if !originalItemsValue.IsValid() {
		return fmt.Errorf("unable to get Items field from original list")
	}
	originalItemsValue.Set(out)

	return nil
}

func (c *Client) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	// need to record the knowledge snapshot this update is based on

	// log observation before propagating labels to capture the label values before the update
	c.logOperation(obj, UPDATE)
	c.propagateLabels(obj)
	res := c.Client.Update(ctx, obj, opts...)
	return res
}

func (c *Client) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	// TODO verify labels propagate correctly under patch
	c.logOperation(obj, PATCH)
	c.propagateLabels(obj)
	res := c.Client.Patch(ctx, obj, patch, opts...)
	return res
}

func (c *Client) LabelChange(obj client.Object) error {
	// labels := tag.GetChangeLabel()
	// patch := map[string]interface{}{
	// 	"metadata": map[string]interface{}{
	// 		"labels": labels,
	// 	},
	// }
	// patchBytes, err := json.Marshal(patch)
	// if err != nil {
	// 	panic("failed to marshal patch")
	// }
	objCopy := obj.DeepCopyObject().(client.Object)
	labels := objCopy.GetLabels()
	// if map is nil, create a new one
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[tag.CHANGE_ID] = tag.GetChangeLabel()[tag.CHANGE_ID]
	objCopy.SetLabels(labels)
	patch := client.MergeFrom(obj)
	if err := c.Patch(context.TODO(), objCopy, patch); err != nil {
		c.logger.Error(err, "failed to patch object")
		return err
	}
	return nil
}

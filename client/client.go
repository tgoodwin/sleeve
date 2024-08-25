package client

import (
	"context"
	"fmt"

	"crypto/sha256"
	"encoding/hex"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"github.com/tgoodwin/sleeve/snapshot"
	"github.com/tgoodwin/sleeve/tag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ctrl "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

var OBSERVATION_KEY = "log-observation"

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

	visibilityDelayByKind map[string]time.Duration
}

var _ client.Client = &Client{}

func newClient(wrapped client.Client) *Client {
	return &Client{
		Client:                wrapped,
		logger:                log,
		visibilityDelayByKind: make(map[string]time.Duration),
	}
}

func Wrap(c client.Client) *Client {
	return newClient(c)
}

func (c *Client) WithName(name string) *Client {
	c.id = name
	return c
}

func (c *Client) WithDelay(kind string, duration time.Duration) *Client {
	c.visibilityDelayByKind[kind] = duration
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

func (c *Client) logObservation(obj client.Object, op OperationType) {
	ov := snapshot.RecordSingle(obj)
	labels := obj.GetLabels()
	l := c.logger.WithValues(
		"Timestamp", fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond)),
		"ReconcileID", c.reconcileID,
		"CreatorID", c.id,
		"RootEventID", c.rootID,
		"OpType", fmt.Sprintf("%v", op),
		"Kind", fmt.Sprintf("%+v", ov.Kind),
		"UID", fmt.Sprintf("%+v", ov.Uid),
		"Version", fmt.Sprintf("%+v", ov.Version),
	)
	for k, v := range labels {
		l = l.WithValues("label:"+k, v)
	}
	l.Info(OBSERVATION_KEY)
}

// InitReconcile... TODO do we need this?
func (c *Client) InitReconcile(ctx context.Context, req reconcile.Request) {
	c.setReconcileID(ctx)
	var partial metav1.PartialObjectMetadata
	c.Client.Get(ctx, req.NamespacedName, &partial)
	if partial.GetUID() != "" {
		c.logObservation(&partial, INIT)
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
			c.logger.V(2).Info("no root context to set")
			return
		}
		c.logger.Info("no webhook label found, using root label", "RootID", rootID)
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
	out[tag.TRACEY_CREATOR_ID] = c.id
	out[tag.TRACEY_ROOT_ID] = c.rootID
	out[tag.TRACEY_RECONCILE_ID] = c.reconcileID

	obj.SetLabels(out)
}

func (c *Client) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.setReconcileID(ctx)
	c.logObservation(obj, CREATE)
	c.propagateLabels(obj)
	res := c.Client.Create(ctx, obj, opts...)
	return res
}

func (c *Client) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	c.setReconcileID(ctx)
	// its important taht we propagate AFTER logging so we update the labels with the latest reconcileID
	// after logging the prior reconcileID on the object
	c.logObservation(obj, DELETE)
	c.propagateLabels(obj)
	res := c.Client.Delete(ctx, obj, opts...)
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
	c.logObservation(obj, GET)
	return err
}

func (c *Client) isVisible(obj client.Object) bool {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if visDelay, ok := c.visibilityDelayByKind[kind]; ok {
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
	c.logObservation(obj, GET)
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

	switch l := list.(type) {
	case *corev1.PodList:
		lc := list.DeepCopyObject().(*corev1.PodList)
		if err := c.Client.List(ctx, lc, opts...); err != nil {
			return err
		}
		out := make([]corev1.Pod, 0)
		for _, pod := range lc.Items {
			if c.isVisible(&pod) {
				c.logObservation(&pod, GET)
				out = append(out, pod)
			}
		}
		l.Items = out
		return nil
	case *appsv1.DeploymentList:
		lc := list.DeepCopyObject().(*appsv1.DeploymentList)
		if err := c.Client.List(ctx, lc, opts...); err != nil {
			return err
		}
		out := make([]appsv1.Deployment, 0)
		for _, deployment := range lc.Items {
			if c.isVisible(&deployment) {
				c.logObservation(&deployment, GET)
				out = append(out, deployment)
			}
		}
		l.Items = out
		return nil
	default:
		c.logger.Info("warning: unhandled list type")
		// TODO dont panic
		// panic("unhandled list type")
		return c.Client.List(ctx, list, opts...)
	}

	// // TODO generalize
	// podList, ok := list.DeepCopyObject().(*corev1.PodList)
	// if ok {
	// 	err := c.Client.List(ctx, podList, opts...)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	visiblePods := make([]corev1.Pod, 0)
	// 	for _, pod := range podList.Items {
	// 		isVisible := c.isVisible(&pod)
	// 		if !isVisible {
	// 			continue
	// 		}
	// 		visiblePods = append(visiblePods, pod)
	// 	}
	// 	c.logger.WithValues(
	// 		"ListedPods", len(podList.Items),
	// 		"VisiblePods", len(visiblePods),
	// 	).Info("returning visible pods")
	// 	list.(*corev1.PodList).Items = visiblePods
	// 	return nil
	// }

	// // TODO log observation for each item in the list
	// // this is hard cause we don't have access to list.Items without knowing the concrete type
	// // so we may have to re-implement below the controller-runtime level to be able to do this.
	// return c.Client.List(ctx, list, opts...)
}

func (c *Client) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	// need to record the knowledge snapshot this update is based on

	// log observation before propagating labels to capture the label values before the update
	c.logObservation(obj, UPDATE)
	c.propagateLabels(obj)
	res := c.Client.Update(ctx, obj, opts...)
	return res
}

func (c *Client) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	// TODO verify labels propagate correctly under patch
	c.logObservation(obj, PATCH)
	c.propagateLabels(obj)
	res := c.Client.Patch(ctx, obj, patch, opts...)
	return res
}

func extractListItems(list client.ObjectList) []client.Object {
	items := make([]client.Object, 0)
	// register each type that can be extracted
	if podList, ok := list.(*corev1.PodList); ok {
		for _, pod := range podList.Items {
			items = append(items, &pod)
		}
		return items
	}
	if deploymentList, ok := list.(*appsv1.DeploymentList); ok {
		for _, deployment := range deploymentList.Items {
			items = append(items, &deployment)
		}
		return items
	}
	if serviceList, ok := list.(*corev1.ServiceList); ok {
		for _, service := range serviceList.Items {
			items = append(items, &service)
		}
		return items
	}
	// add more types here
	panic("unhandled list type... please register the type you need to extract")
}

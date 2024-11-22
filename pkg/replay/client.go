package replay

import (
	"context"
	"reflect"

	sleeveclient "github.com/tgoodwin/sleeve/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EffectRecorder interface {
	RecordEffect(ctx context.Context, obj client.Object, opType sleeveclient.OperationType) error
}

type Client struct {
	// dummyClient is a useless type that implements the remainder of the client.Client interface
	*dummyClient
	framesByID     map[string]CacheFrame
	effectRecorder EffectRecorder
}

func NewClient(frameData map[string]CacheFrame, effectRecorder EffectRecorder) *Client {
	return &Client{
		dummyClient:    &dummyClient{},
		framesByID:     frameData,
		effectRecorder: effectRecorder,
	}
}

var _ client.Client = (*Client)(nil)

func inferKind(obj client.Object) string {
	// assumption: the object is a pointer to a struct
	t := reflect.TypeOf(obj).Elem()
	return t.Name()
}

func inferListKind(list client.ObjectList) string {
	itemsValue := reflect.ValueOf(list).Elem().FieldByName("Items")
	if !itemsValue.IsValid() {
		panic("List object does not have Items field")
	}
	itemType := itemsValue.Type().Elem()
	return itemType.Name()
}

func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	frameID := frameIDFromContext(ctx)
	kind := inferKind(obj)
	if frame, ok := c.framesByID[frameID]; ok {
		if frozenObj, ok := frame[kind][key]; ok {
			reflect.ValueOf(obj).Elem().Set(reflect.ValueOf(frozenObj).Elem())
			return nil
		}
	}
	return nil
}

func (c *Client) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	// TODO
	frameID := frameIDFromContext(ctx)
	kind := inferListKind(list)

	if frame, ok := c.framesByID[frameID]; ok {
		if objsForKind, ok := frame[kind]; ok {
			// get a slice of the objects from the map values
			objs := make([]client.Object, 0, len(objsForKind))
			for _, obj := range objsForKind {
				objs = append(objs, obj)
			}
			// set the Items field of the list object to the slice of objects
			itemsValue := reflect.ValueOf(list).Elem().FieldByName("Items")
			itemsValue.Set(reflect.ValueOf(objs))
		}
	}

	return nil
}

func (c *Client) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.effectRecorder.RecordEffect(ctx, obj, sleeveclient.CREATE)
	return nil
}

func (c *Client) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	c.effectRecorder.RecordEffect(ctx, obj, sleeveclient.DELETE)
	return nil
}

func (c *Client) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	c.effectRecorder.RecordEffect(ctx, obj, sleeveclient.UPDATE)
	return nil
}

func (c *Client) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	c.effectRecorder.RecordEffect(ctx, obj, sleeveclient.DELETE)
	return nil
}

func (c *Client) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	c.effectRecorder.RecordEffect(ctx, obj, sleeveclient.PATCH)
	return nil
}

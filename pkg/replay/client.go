package replay

import (
	"context"

	sleeveclient "github.com/tgoodwin/sleeve/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EffectRecorder interface {
	RecordEffect(ctx context.Context, obj client.Object, opType sleeveclient.OperationType) error
}

type Client struct {
	// dummyClient is a useless type that implements the remainder of the client.Client interface
	*dummyClient
	store          *FrozenCache
	effectRecorder EffectRecorder
}

var _ client.Client = (*Client)(nil)

func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return c.store.Get(ctx, key, obj)
}

func (c *Client) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	// TODO
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

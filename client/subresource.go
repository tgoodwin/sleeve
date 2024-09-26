package client

import (
	"context"

	"github.com/tgoodwin/sleeve/tag"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type SubResourceClient struct {
	// reader kclient.SubResourceReader
	client *Client
	writer kclient.SubResourceWriter
}

func (c *Client) Status() kclient.SubResourceWriter {
	statusClient := c.Client.Status()
	return &SubResourceClient{writer: statusClient, client: c}
}

func (s *SubResourceClient) logObservation(obj kclient.Object, action OperationType) {
	s.client.logObservation(obj, action)
}

func (s *SubResourceClient) Update(ctx context.Context, obj kclient.Object, opts ...kclient.SubResourceUpdateOption) error {
	s.client.setReconcileID(ctx)
	tag.LabelChange(obj)
	s.logObservation(obj, UPDATE)
	s.client.propagateLabels(obj)
	// persist the labels to the object before updating status
	s.client.Update(ctx, obj)

	// update status
	return s.writer.Update(ctx, obj, opts...)
}

func (s *SubResourceClient) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.SubResourcePatchOption) error {
	s.client.setReconcileID(ctx)
	tag.LabelChange(obj)
	s.logObservation(obj, PATCH)
	// persist the labels to the object before updating status
	s.client.Update(ctx, obj)
	return s.writer.Patch(ctx, obj, patch, opts...)
}

func (s *SubResourceClient) Create(ctx context.Context, obj kclient.Object, sub kclient.Object, opts ...kclient.SubResourceCreateOption) error {
	s.client.setReconcileID(ctx)
	tag.LabelChange(obj)
	s.logObservation(obj, CREATE)
	s.client.propagateLabels(obj)
	s.client.Update(ctx, obj)

	return s.writer.Create(ctx, obj, sub, opts...)
}

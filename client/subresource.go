package client

import (
	"context"

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
	s.logObservation(obj, UPDATE)
	return s.writer.Update(ctx, obj, opts...)
}

func (s *SubResourceClient) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.SubResourcePatchOption) error {
	s.client.setReconcileID(ctx)
	s.logObservation(obj, PATCH)
	return s.writer.Patch(ctx, obj, patch, opts...)
}

func (s *SubResourceClient) Create(ctx context.Context, obj kclient.Object, sub kclient.Object, opts ...kclient.SubResourceCreateOption) error {
	s.client.setReconcileID(ctx)
	s.logObservation(obj, CREATE)
	return s.writer.Create(ctx, obj, sub, opts...)
}

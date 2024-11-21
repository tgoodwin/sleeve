package replay

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type dummyClient struct{}

func (d *dummyClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (d *dummyClient) Scheme() *runtime.Scheme {
	return nil
}

func (d *dummyClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (d *dummyClient) Status() client.StatusWriter {
	return nil
}

func (d *dummyClient) SubResource(subresource string) client.SubResourceClient {
	return nil
}

func (d *dummyClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return false, nil
}

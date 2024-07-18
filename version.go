package client

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO how to instantiate this dynamically?
type GenResourceLister[OBJ any, OBJPTR interface {
	client.Object
	*OBJ
}, LIST interface {
	GetItems() []OBJ
}, LISTPTR interface {
	client.ObjectList
	*LIST
}] struct{}

func itemsToObjectListt[T any, TPTR interface {
	client.Object
	*T
}](items []T) []client.Object {
	result := make([]client.Object, len(items))
	for _, resource := range items {
		r := resource
		result = append(result, TPTR(&r))
	}
	return result
}

func (l *GenResourceLister[OBJ, OBJPTR, LIST, LISTPTR]) List(
	ctx context.Context,
	c client.Client,
	listOptions *client.ListOptions,
) ([]client.Object, error) {
	resources := new(LIST)
	if err := c.List(ctx, LISTPTR(resources), listOptions); err != nil {
		return nil, err
	}
	items := (*resources).GetItems()
	return itemsToObjectListt[OBJ, OBJPTR](items), nil
}

func Observe() Observation {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return Observation{
		timestamp: timestamp,
	}
}

func RecordSingle(obj client.Object) ObjectVersion {
	return ObjectVersion{
		Kind:    obj.GetObjectKind().GroupVersionKind().Kind,
		Uid:     string(obj.GetUID()),
		Version: obj.GetResourceVersion(),
	}
}

func RecordMultiple(objs client.ObjectList) VersionSet {
	result := make(VersionSet)
	unstructuredList, ok := objs.(*unstructured.UnstructuredList)
	if !ok {

		// Handle the case when objs is not an instance of unstructured.UnstructuredList
		return result
	}
	// lister := GenResourceLister[YourObjectType, *YourObjectType, YourObjectListType, *YourObjectListType]{}
	for _, obj := range unstructuredList.Items {
		result[RecordSingle(&obj)] = struct{}{}
	}
	return result
}

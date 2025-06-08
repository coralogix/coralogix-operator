package monitoring

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// InstrumentedClient wraps a controller-runtime client to track Kubernetes API calls.
type InstrumentedClient struct {
	client.Client
	scheme *runtime.Scheme
}

func NewInstrumentedClient(inner client.Client, scheme *runtime.Scheme) client.Client {
	return &InstrumentedClient{Client: inner, scheme: scheme}
}

// extractGVK tries to extract the GroupVersionKind from a client.Object.
func extractGVK(obj client.Object) (schema.GroupVersionKind, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		return gvk, fmt.Errorf("GVK not explicitly set for object: %T", obj)
	}
	return gvk, nil
}

func (c *InstrumentedClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	incrementAPICall("list", list, c.scheme)
	return c.Client.List(ctx, list, opts...)
}

func (c *InstrumentedClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	incrementAPICall("get", obj, c.scheme)
	return c.Client.Get(ctx, key, obj, opts...)
}

func (c *InstrumentedClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	incrementAPICall("create", obj, c.scheme)
	err := c.Client.Create(ctx, obj, opts...)
	if err == nil {
		if gvk, gvkErr := extractGVK(obj); gvkErr == nil {
			MarkGVKStale(gvk)
		}
	}
	return err
}

func (c *InstrumentedClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	incrementAPICall("update", obj, c.scheme)
	err := c.Client.Update(ctx, obj, opts...)
	if err == nil {
		if gvk, gvkErr := extractGVK(obj); gvkErr == nil {
			MarkGVKStale(gvk)
		}
	}
	return err
}

func (c *InstrumentedClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	incrementAPICall("delete", obj, c.scheme)
	err := c.Client.Delete(ctx, obj, opts...)
	if err == nil {
		if gvk, gvkErr := extractGVK(obj); gvkErr == nil {
			MarkGVKStale(gvk)
		}
	}
	return err
}

func (c *InstrumentedClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	incrementAPICall("patch", obj, c.scheme)
	return c.Client.Patch(ctx, obj, patch, opts...)
}

func incrementAPICall(verb string, obj any, scheme *runtime.Scheme) {
	var gvk schema.GroupVersionKind

	switch v := obj.(type) {
	case client.Object:
		gvk, _ = apiutil.GVKForObject(v, scheme)
	case client.ObjectList:
		gvk, _ = apiutil.GVKForObject(v, scheme)
	default:
		gvk = schema.GroupVersionKind{Kind: "unknown"}
	}

	resource := gvk.String()
	if resource == "" {
		resource = "unknown"
	}

	apiCallCounter.WithLabelValues(verb, resource).Inc()
}

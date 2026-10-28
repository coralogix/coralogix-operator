// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// ResourceIDRef refers to another Coralogix resource: by its id, or by the custom resource that
// manages it. Generated kinds use it for a field that a ref line of the operator generator names.
// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type ResourceIDRef struct {
	// BackendRef is the id of the resource in Coralogix.
	// +optional
	BackendRef *NCBackendRef `json:"backendRef,omitempty"`

	// ResourceRef is the custom resource that manages the resource.
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

// ResolveResourceID returns the Coralogix id that ref names. A backendRef id is sent as is.
// A resourceRef names a custom resource of kind. Its namespace defaults to namespace, the
// namespace of the referring resource. It must match the selector of the operator, and its status
// id is sent. Until it has an id, the error makes the reconcile retry.
func ResolveResourceID(ctx context.Context, namespace string, ref ResourceIDRef, kind string) (string, error) {
	if ref.BackendRef != nil {
		return ref.BackendRef.ID, nil
	}
	if ref.ResourceRef == nil {
		return "", fmt.Errorf("the reference to a %s sets neither backendRef nor resourceRef", kind)
	}
	if ref.ResourceRef.Namespace != nil {
		namespace = *ref.ResourceRef.Namespace
	}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(GroupVersion.WithKind(kind))
	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: ref.ResourceRef.Name, Namespace: namespace}, obj); err != nil {
		return "", err
	}
	if !config.GetConfig().Selector.Matches(obj.GetLabels(), obj.GetNamespace()) {
		return "", fmt.Errorf("%s %s does not match selector", kind, obj.GetName())
	}
	id, _, err := unstructured.NestedString(obj.Object, "status", "id")
	if err != nil || id == "" {
		return "", fmt.Errorf("ID is not populated for %s %s", kind, obj.GetName())
	}
	return id, nil
}

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
	"math"
	"strconv"
)

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type Int32ResourceIDRef struct {
	// +optional
	BackendRef *Int32BackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

type Int32BackendRef struct {
	ID int32 `json:"id"`
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type Int64ResourceIDRef struct {
	// +optional
	BackendRef *Int64BackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

type Int64BackendRef struct {
	ID int64 `json:"id"`
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type UInt32ResourceIDRef struct {
	// +optional
	BackendRef *UInt32BackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

type UInt32BackendRef struct {
	ID uint32 `json:"id"`
}

// ResolveNumericResourceID reads a numeric backend id or the status id of a referenced CR.
func ResolveNumericResourceID(ctx context.Context, namespace string, backendID *int64, resourceRef *ResourceRef, kind, version string, bits int) (int64, error) {
	if backendID != nil {
		if bits == 32 && (*backendID < math.MinInt32 || *backendID > math.MaxInt32) {
			return 0, fmt.Errorf("backend id %d is outside int32 range", *backendID)
		}
		return *backendID, nil
	}
	if version == "" {
		version = GroupVersion.Version
	}
	id, err := ResolveResourceIDVersion(ctx, namespace, ResourceIDRef{ResourceRef: resourceRef}, kind, version)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseInt(id, 10, bits)
	if err != nil {
		return 0, fmt.Errorf("%s status id %q is not an int%d: %w", kind, id, bits, err)
	}
	return parsed, nil
}

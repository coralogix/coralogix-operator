// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

// ArchiveRetention is the released reference of a TCO policy to an archive retention. The generated
// TCO policies kinds keep it (ref form namedBackendOnly), so their CRDs keep its shape.
type ArchiveRetention struct {
	// Reference to the retention policy
	BackendRef ArchiveRetentionBackendRef `json:"backendRef"`
}

// Backend reference to the policy. Exactly one of name or id must be set.
// +kubebuilder:validation:XValidation:rule="has(self.name) != has(self.id)",message="exactly one of name or id must be set"
type ArchiveRetentionBackendRef struct {
	// Name of the policy.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name *string `json:"name,omitempty"`

	// ID of the policy.
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	Id *string `json:"id,omitempty"`
}

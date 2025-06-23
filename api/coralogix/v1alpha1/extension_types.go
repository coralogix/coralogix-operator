// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package v1alpha1

import (
	"fmt"

	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// ExtensionSpec defines the desired state of a Coralogix extension.
// See also https://coralogix.com/docs/user-guides/getting-started/packages-and-extensions/integration-packages/
type ExtensionSpec struct {
	// Id of the extension to deploy.
	Id string `json:"id"`

	// Desired version of the extension.
	Version string `json:"version"`

	// Item IDs to be used by the extension.
	ItemIds []string `json:"itemIds,omitempty"`
}

type ExtensionStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (s *ExtensionSpec) ExtractDeployExtensionRequest() (*cxsdk.DeployExtensionRequest, error) {
	itemIds := []*wrapperspb.StringValue{}
	if s.Id == "" {
		return nil, fmt.Errorf("extension ID is required for deployment")
	}
	if s.Version == "" {
		return nil, fmt.Errorf("extension version is required for deployment")
	}
	for _, itemId := range s.ItemIds {
		itemIds = append(itemIds, &wrapperspb.StringValue{Value: itemId})
	}
	return &cxsdk.DeployExtensionRequest{
		Id:      &wrapperspb.StringValue{Value: s.Id},
		Version: &wrapperspb.StringValue{Value: s.Version},
		ItemIds: itemIds,
	}, nil
}

func (s *ExtensionSpec) ExtractUpdateExtensionRequest(id string) (*cxsdk.UpdateExtensionRequest, error) {
	itemIds := []*wrapperspb.StringValue{}
	if s.Id == "" {
		return nil, fmt.Errorf("extension ID is required for deployment")
	}
	if s.Version == "" {
		return nil, fmt.Errorf("extension version is required for deployment")
	}
	for _, itemId := range s.ItemIds {
		itemIds = append(itemIds, &wrapperspb.StringValue{Value: itemId})
	}
	return &cxsdk.UpdateExtensionRequest{
		Id:      &wrapperspb.StringValue{Value: id},
		Version: &wrapperspb.StringValue{Value: s.Version},
		ItemIds: itemIds,
	}, nil
}

func (i *Extension) GetConditions() []metav1.Condition {
	return i.Status.Conditions
}

func (i *Extension) SetConditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Extension is the Schema for the extensions API.
type Extension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExtensionSpec   `json:"spec,omitempty"`
	Status ExtensionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ExtensionList contains a list of Extension.
type ExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Extension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Extension{}, &ExtensionList{})
}

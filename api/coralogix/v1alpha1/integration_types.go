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

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// IntegrationSpec defines the desired state of a Coralogix (managed) integration.
type IntegrationSpec struct {

	// Unique name of the integration.
	IntegrationKey string `json:"integrationKey"`

	// Desired version of the integration
	Version string `json:"version"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// Parameters required by the integration.
	Parameters runtime.RawExtension `json:"parameters"`
}

func (s *IntegrationSpec) ExtractCreateIntegrationRequest() (*cxsdk.SaveIntegrationRequest, error) {
	parameters, err := s.ExtractParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}
	return &cxsdk.SaveIntegrationRequest{
		Metadata: &cxsdk.IntegrationMetadata{
			IntegrationKey: wrapperspb.String(s.IntegrationKey),
			Version:        wrapperspb.String(s.Version),
			SpecificData: &cxsdk.IntegrationMetadataIntegrationParameters{
				IntegrationParameters: &cxsdk.GenericIntegrationParameters{
					Parameters: parameters,
				},
			},
		},
	}, nil
}

func (s *IntegrationSpec) ExtractUpdateIntegrationRequest(id string) (*cxsdk.UpdateIntegrationRequest, error) {
	parameters, err := s.ExtractParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}
	return &cxsdk.UpdateIntegrationRequest{
		Id: wrapperspb.String(id),
		Metadata: &cxsdk.IntegrationMetadata{
			IntegrationKey: wrapperspb.String(s.IntegrationKey),
			Version:        wrapperspb.String(s.Version),
			SpecificData: &cxsdk.IntegrationMetadataIntegrationParameters{
				IntegrationParameters: &cxsdk.GenericIntegrationParameters{
					Parameters: parameters,
				},
			},
		},
	}, nil
}

func (s *IntegrationSpec) ExtractParameters() ([]*cxsdk.IntegrationParameter, error) {
	var rawParams map[string]interface{}
	if err := json.Unmarshal(s.Parameters.Raw, &rawParams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	var parameters []*cxsdk.IntegrationParameter
	for key, value := range rawParams {
		switch v := value.(type) {
		case string:
			parameters = append(parameters, &cxsdk.IntegrationParameter{
				Key:   key,
				Value: &cxsdk.IntegrationParameterStringValue{StringValue: wrapperspb.String(v)},
			})
		case float64:
			parameters = append(parameters, &cxsdk.IntegrationParameter{
				Key:   key,
				Value: &cxsdk.IntegrationParameterNumericValue{NumericValue: wrapperspb.Double(v)},
			})
		case bool:
			parameters = append(parameters, &cxsdk.IntegrationParameter{
				Key:   key,
				Value: &cxsdk.IntegrationParameterBooleanValue{BooleanValue: wrapperspb.Bool(v)},
			})
		case []interface{}:
			var stringList []*wrapperspb.StringValue
			for _, item := range v {
				if str, ok := item.(string); ok {
					stringList = append(stringList, wrapperspb.String(str))
				}
			}
			parameters = append(parameters, &cxsdk.IntegrationParameter{
				Key: key,
				Value: &cxsdk.IntegrationParameterStringList{
					StringList: &cxsdk.IntegrationParameterStringListInner{
						Values: stringList,
					},
				},
			})
		default:
			return nil, fmt.Errorf("unsupported value type for parameter %s", key)
		}
	}
	return parameters, nil
}

// IntegrationStatus defines the observed state of Integration.
type IntegrationStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (i *Integration) GetConditions() []metav1.Condition {
	return i.Status.Conditions
}

func (i *Integration) SetConditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Integration is the Schema for the Integrations API.
// See also https://coralogix.com/docs/user-guides/getting-started/packages-and-extensions/integration-packages/
//
// For available integrations see https://coralogix.com/docs/developer-portal/infrastructure-as-code/terraform-provider/integrations/aws-metrics-collector/ or at https://github.com/coralogix/coralogix-operator/tree/main/config/samples/v1alpha1/integrations.
//
// **Added in v0.4.0**
type Integration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationSpec   `json:"spec,omitempty"`
	Status IntegrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IntegrationList contains a list of Integrations.
type IntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Integration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Integration{}, &IntegrationList{})
}

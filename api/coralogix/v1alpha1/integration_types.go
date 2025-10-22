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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integrations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/integration_service"
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

func (s *IntegrationSpec) ExtractCreateIntegrationRequest() (*integrations.SaveIntegrationRequest, error) {
	parameters, err := s.ExtractParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}
	return &integrations.SaveIntegrationRequest{
		Metadata: integrations.IntegrationMetadata{
			IntegrationKey: integrations.PtrString(s.IntegrationKey),
			Version:        integrations.PtrString(s.Version),
			IntegrationParameters: &integrations.GenericIntegrationParameters{
				Parameters: parameters,
			},
		},
	}, nil
}

func (s *IntegrationSpec) ExtractUpdateIntegrationRequest(id string) (*integrations.UpdateIntegrationRequest, error) {
	parameters, err := s.ExtractParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}

	return &integrations.UpdateIntegrationRequest{
		Id: id,
		Metadata: integrations.IntegrationMetadata{
			IntegrationKey: integrations.PtrString(s.IntegrationKey),
			Version:        integrations.PtrString(s.Version),
			IntegrationParameters: &integrations.GenericIntegrationParameters{
				Parameters: parameters,
			},
		},
	}, nil
}

func (s *IntegrationSpec) ExtractParameters() ([]integrations.Parameter, error) {
	var rawParams map[string]interface{}
	if err := json.Unmarshal(s.Parameters.Raw, &rawParams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	var parameters []integrations.Parameter
	for key, value := range rawParams {
		switch v := value.(type) {
		case string:
			parameters = append(parameters, integrations.Parameter{
				ParameterStringValue: &integrations.ParameterStringValue{
					Key:         integrations.PtrString(key),
					StringValue: integrations.PtrString(v),
				},
			})
		case float64:
			parameters = append(parameters, integrations.Parameter{
				ParameterNumericValue: &integrations.ParameterNumericValue{
					Key:          integrations.PtrString(key),
					NumericValue: integrations.PtrFloat64(v),
				},
			})
		case bool:
			parameters = append(parameters, integrations.Parameter{
				ParameterBooleanValue: &integrations.ParameterBooleanValue{
					Key:          integrations.PtrString(key),
					BooleanValue: integrations.PtrBool(v),
				},
			})
		case []interface{}:
			var stringList integrations.StringList
			for _, item := range v {
				if str, ok := item.(string); ok {
					stringList.Values = append(stringList.Values, str)
				}
			}
			parameters = append(parameters, integrations.Parameter{
				ParameterStringList: &integrations.ParameterStringList{
					Key:        integrations.PtrString(key),
					StringList: &stringList,
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

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (i *Integration) GetConditions() []metav1.Condition {
	return i.Status.Conditions
}

func (i *Integration) SetConditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}

func (i *Integration) GetPrintableStatus() string {
	return i.Status.PrintableStatus
}

func (i *Integration) SetPrintableStatus(printableStatus string) {
	i.Status.PrintableStatus = printableStatus
}

func (i *Integration) HasIDInStatus() bool {
	return i.Status.Id != nil && *i.Status.Id != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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

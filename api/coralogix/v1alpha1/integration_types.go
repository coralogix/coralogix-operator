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
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
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

	// ParametersFromSecret is a map of parameter names to references of Kubernetes
	// Secret keys whose values should be used as the parameter value at reconcile time.
	// Use this for sensitive parameters (API keys, service account keys, tokens, etc.)
	// so that secret material does not need to live in the manifest.
	//
	// A given parameter name must appear in either Parameters or ParametersFromSecret,
	// not both. Only string-valued parameters are supported via this field; numeric,
	// boolean, and list-valued parameters must be set inline in Parameters.
	// +optional
	ParametersFromSecret map[string]corev1.SecretKeySelector `json:"parametersFromSecret,omitempty"`
}

// ExtractCreateIntegrationRequest builds a SaveIntegrationRequest, resolving any
// ParametersFromSecret references against Kubernetes Secrets in the Integration's
// namespace.
func (i *Integration) ExtractCreateIntegrationRequest(ctx context.Context) (*integrations.SaveIntegrationRequest, error) {
	parameters, err := i.Spec.ExtractParameters(ctx, i.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}
	return &integrations.SaveIntegrationRequest{
		Metadata: &integrations.IntegrationMetadata{
			IntegrationKey: integrations.PtrString(i.Spec.IntegrationKey),
			Version:        integrations.PtrString(i.Spec.Version),
			IntegrationParameters: &integrations.GenericIntegrationParameters{
				Parameters: parameters,
			},
		},
	}, nil
}

// ExtractUpdateIntegrationRequest builds an UpdateIntegrationRequest, resolving any
// ParametersFromSecret references against Kubernetes Secrets in the Integration's
// namespace.
func (i *Integration) ExtractUpdateIntegrationRequest(ctx context.Context, id *string) (*integrations.UpdateIntegrationRequest, error) {
	parameters, err := i.Spec.ExtractParameters(ctx, i.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}

	return &integrations.UpdateIntegrationRequest{
		Id: id,
		Metadata: &integrations.IntegrationMetadata{
			IntegrationKey: integrations.PtrString(i.Spec.IntegrationKey),
			Version:        integrations.PtrString(i.Spec.Version),
			IntegrationParameters: &integrations.GenericIntegrationParameters{
				Parameters: parameters,
			},
		},
	}, nil
}

// ExtractParameters returns the merged parameter set for the Integration, with
// inline Parameters values combined with ParametersFromSecret references resolved
// from Kubernetes Secrets in the given namespace.
//
// A parameter name appearing in both Parameters and ParametersFromSecret is rejected.
func (s *IntegrationSpec) ExtractParameters(ctx context.Context, namespace string) ([]integrations.Parameter, error) {
	rawParams := map[string]interface{}{}
	if len(s.Parameters.Raw) > 0 {
		if err := json.Unmarshal(s.Parameters.Raw, &rawParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}

	for key, ref := range s.ParametersFromSecret {
		if _, exists := rawParams[key]; exists {
			return nil, fmt.Errorf("parameter %q is set in both parameters and parametersFromSecret; only one source is allowed", key)
		}
		value, err := readSecret(ctx, ref, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read secret for parameter %q: %w", key, err)
		}
		rawParams[key] = value
	}

	var parameters []integrations.Parameter
	for key, value := range rawParams {
		switch v := value.(type) {
		case string:
			parameters = append(parameters, integrations.Parameter{
				ParameterStringValue: &integrations.ParameterStringValue{
					Key:         integrations.PtrString(key),
					StringValue: v,
				},
			})
		case float64:
			parameters = append(parameters, integrations.Parameter{
				ParameterNumericValue: &integrations.ParameterNumericValue{
					Key:          integrations.PtrString(key),
					NumericValue: v,
				},
			})
		case bool:
			parameters = append(parameters, integrations.Parameter{
				ParameterBooleanValue: &integrations.ParameterBooleanValue{
					Key:          integrations.PtrString(key),
					BooleanValue: v,
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
					StringList: stringList,
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

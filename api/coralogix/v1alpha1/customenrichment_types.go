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
	"fmt"
	"hash/adler32"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	customenrichments "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/custom_enrichments_service"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// CustomEnrichmentSpec defines the desired state of CustomEnrichment.
// +kubebuilder:validation:XValidation:rule="has(self.csv) != has(self.configMapRef)", message="Exactly one of csv or configMapRef must be set"
type CustomEnrichmentSpec struct {
	// The name of the custom enrichment.
	Name string `json:"name"`

	// The description of the custom enrichment.
	Description string `json:"description"`

	// Inline CSV data. Conflicts with ConfigMapRef.
	// +optional
	CSV *string `json:"csv,omitempty"`

	// Reference to a ConfigMap that contains the CSV data. Conflicts with CSV.
	// +optional
	ConfigMapRef *corev1.ConfigMapKeySelector `json:"configMapRef,omitempty"`
}

func (c *CustomEnrichment) ExtractCreateCustomEnrichmentRequest(ctx context.Context) (*customenrichments.CreateCustomEnrichmentRequest, error) {
	var fileContent *string

	if c.Spec.CSV != nil {
		fileContent = c.Spec.CSV
	} else if c.Spec.ConfigMapRef != nil {
		cmContext, err := readConfigMap(ctx, *c.Spec.ConfigMapRef, c.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read configmap: %w", err)
		}
		fileContent = &cmContext
	} else {
		return nil, fmt.Errorf("either CSV or ConfigMapRef must be provided")
	}

	h := adler32.New()
	if _, err := h.Write([]byte(*fileContent)); err != nil {
		return nil, fmt.Errorf("failed to compute hash of file content: %w", err)
	}
	name := fmt.Sprintf("%x", h.Sum(nil))

	return &customenrichments.CreateCustomEnrichmentRequest{
		Name:        c.Spec.Name,
		Description: c.Spec.Description,
		File: customenrichments.File{
			FileTextual: &customenrichments.FileTextual{
				Name:      customenrichments.PtrString(name),
				Extension: customenrichments.PtrString("csv"),
				Textual:   fileContent,
			},
		},
	}, nil
}

func (c *CustomEnrichment) ExtractUpdateCustomEnrichmentRequest(ctx context.Context) (*customenrichments.UpdateCustomEnrichmentRequest, error) {
	if c.Status.Id == nil {
		return nil, fmt.Errorf("custom enrichment ID is missing in status; cannot create update request without ID")
	}

	id, err := strconv.Atoi(*c.Status.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ID to UInt32: %w", err)
	}

	var fileContent *string
	if c.Spec.CSV != nil {
		fileContent = c.Spec.CSV
	} else if c.Spec.ConfigMapRef != nil {
		cmContext, err := readConfigMap(ctx, *c.Spec.ConfigMapRef, c.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read configmap: %w", err)
		}
		fileContent = &cmContext
	} else {
		return nil, fmt.Errorf("either CSV or ConfigMapRef must be provided")
	}

	h := adler32.New()
	if _, err := h.Write([]byte(*fileContent)); err != nil {
		return nil, fmt.Errorf("failed to compute hash of file content: %w", err)
	}
	name := fmt.Sprintf("%x", h.Sum(nil))

	return &customenrichments.UpdateCustomEnrichmentRequest{
		CustomEnrichmentId: int64(id),
		Name:               c.Spec.Name,
		Description:        c.Spec.Description,
		File: customenrichments.File{
			FileTextual: &customenrichments.FileTextual{
				Name:      customenrichments.PtrString(name),
				Extension: customenrichments.PtrString("csv"),
				Textual:   fileContent,
			},
		},
	}, nil
}

func readConfigMap(ctx context.Context, configMapRef corev1.ConfigMapKeySelector, namespace string) (string, error) {
	cm := &corev1.ConfigMap{}
	if err := config.GetClient().Get(ctx, client.ObjectKey{Namespace: namespace, Name: configMapRef.Name}, cm); err != nil {
		return "", err
	}

	if content, ok := cm.Data[configMapRef.Key]; ok {
		return content, nil
	}

	return "", fmt.Errorf("cannot find key '%v' in config map '%v'", configMapRef.Key, configMapRef.Name)
}

// CustomEnrichmentStatus defines the observed state of CustomEnrichment.
type CustomEnrichmentStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (c *CustomEnrichment) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *CustomEnrichment) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

func (c *CustomEnrichment) GetPrintableStatus() string {
	return c.Status.PrintableStatus
}

func (c *CustomEnrichment) SetPrintableStatus(printableStatus string) {
	c.Status.PrintableStatus = printableStatus
}

func (c *CustomEnrichment) HasIDInStatus() bool {
	return c.Status.Id != nil && *c.Status.Id != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CustomEnrichment is the Schema for the customenrichments API.
// See also https://coralogix.com/docs/user-guides/data-transformation/enrichments/custom-enrichment/#configuration.
type CustomEnrichment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CustomEnrichmentSpec   `json:"spec,omitempty"`
	Status CustomEnrichmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CustomEnrichmentList contains a list of CustomEnrichment.
type CustomEnrichmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomEnrichment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CustomEnrichment{}, &CustomEnrichmentList{})
}

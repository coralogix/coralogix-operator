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
	"strconv"

	enrichments "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/enrichments_service"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EnrichmentSpec defines the desired state of Enrichment.
type EnrichmentSpec struct {
	// +kubebuilder:validation:MinItems=1
	// List of enrichments to apply. Each enrichment must have exactly one of GeoIp, SuspiciousIp, Aws, or Custom set.
	// Will overwrite the existing enrichments on the Coralogix side,
	// so it should contain all enrichments that should be applied, not just the new ones.
	Enrichments []EnrichmentType `json:"enrichments"`
}

// EnrichmentType must have exactly one of GeoIp, SuspiciousIp, Aws, or Custom set.
// +kubebuilder:validation:XValidation:rule="(has(self.geoIp) ? 1 : 0) + (has(self.suspiciousIp) ? 1 : 0) + (has(self.aws) ? 1 : 0) + (has(self.custom) ? 1 : 0) == 1", message="Exactly one of geoIp, suspiciousIp, aws, or custom must be set"
type EnrichmentType struct {
	// Set of fields to enrich with geo_ip information.
	// +optional
	GeoIp *GeoIpEnrichmentType `json:"geoIp,omitempty"`

	// Coralogix allows you to automatically discover threats on your web servers
	// by enriching your logs with the most updated IP blacklists.
	SuspiciousIp *SuspiciousIpEnrichmentType `json:"suspiciousIp,omitempty"`

	// Coralogix allows you to enrich your logs with the data from a chosen AWS resource.
	// The feature enriches every log that contains a particular resourceId,
	// associated with the metadata of a chosen AWS resource.
	// +optional
	Aws *AwsEnrichmentType `json:"aws,omitempty"`

	// Custom Log Enrichment with Coralogix enables you to easily enrich your log data.
	// +optional
	Custom *CustomEnrichmentType `json:"custom,omitempty"`
}

type GeoIpEnrichmentType struct {
	FieldName string `json:"fieldName"`

	// +optional
	WithAsn *bool `json:"withAsn,omitempty"`
}

type SuspiciousIpEnrichmentType struct {
	FieldName string `json:"fieldName"`
}

type AwsEnrichmentType struct {
	FieldName string `json:"fieldName"`

	ResourceType string `json:"resourceType"`
}

type CustomEnrichmentType struct {
	FieldName string `json:"fieldName"`

	// +optional
	EnrichedFieldName *string `json:"enrichedFieldName,omitempty"`

	// +optional
	SelectedColumns []string `json:"selectedColumns,omitempty"`

	// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)", message="Exactly one of backendRef or resourceRef must be set"
	CustomEnrichmentRef CustomEnrichmentRef `json:"customEnrichmentRef"`
}

type CustomEnrichmentRef struct {
	// BackendRef is a reference to a CustomEnrichment in the backend.
	// +optional
	BackendRef *CustomEnrichmentBackendRef `json:"backendRef,omitempty"`

	// ResourceRef is a reference to a CustomEnrichment resource in the cluster.
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

type CustomEnrichmentBackendRef struct {
	// ID of the CustomEnrichment in the backend.
	Id uint32 `json:"id"`
}

// EnrichmentStatus defines the observed state of Enrichment.
type EnrichmentStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Enrichment is the Schema for the enrichments API.
// Will overwrite the existing enrichments on the Coralogix side,
// so it should contain all enrichments that should be applied, not just the new ones.
// See also https://coralogix.com/docs/user-guides/data-transformation/enrichments/custom-enrichment/#configuration.
type Enrichment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnrichmentSpec   `json:"spec,omitempty"`
	Status EnrichmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnrichmentList contains a list of Enrichment.
type EnrichmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Enrichment `json:"items"`
}

func (e *Enrichment) GetConditions() []metav1.Condition {
	return e.Status.Conditions
}

func (e *Enrichment) SetConditions(conditions []metav1.Condition) {
	e.Status.Conditions = conditions
}

func (e *Enrichment) GetPrintableStatus() string {
	return e.Status.PrintableStatus
}

func (e *Enrichment) SetPrintableStatus(printableStatus string) {
	e.Status.PrintableStatus = printableStatus
}

func (e *Enrichment) HasIDInStatus() bool {
	return true
}

func (e *Enrichment) ExtractAtomicOverwriteRequest(ctx context.Context) (
	*enrichments.EnrichmentServiceAtomicOverwriteEnrichmentsRequest, error) {
	var reqs []enrichments.EnrichmentRequestModel
	for _, enrichment := range e.Spec.Enrichments {
		if enrichment.GeoIp != nil {
			reqs = append(reqs, enrichments.EnrichmentRequestModel{
				FieldName: enrichment.GeoIp.FieldName,
				EnrichmentType: enrichments.EnrichmentType{
					EnrichmentTypeGeoIp: &enrichments.EnrichmentTypeGeoIp{
						GeoIp: &enrichments.GeoIpType{
							WithAsn: enrichment.GeoIp.WithAsn,
						},
					},
				},
			})
		} else if enrichment.SuspiciousIp != nil {
			reqs = append(reqs, enrichments.EnrichmentRequestModel{
				FieldName: enrichment.SuspiciousIp.FieldName,
				EnrichmentType: enrichments.EnrichmentType{
					EnrichmentTypeSuspiciousIp: &enrichments.EnrichmentTypeSuspiciousIp{
						SuspiciousIp: map[string]any{},
					},
				},
			})
		} else if enrichment.Aws != nil {
			reqs = append(reqs, enrichments.EnrichmentRequestModel{
				FieldName: enrichment.Aws.FieldName,
				EnrichmentType: enrichments.EnrichmentType{
					EnrichmentTypeAws: &enrichments.EnrichmentTypeAws{
						Aws: &enrichments.AwsType{
							ResourceType: enrichments.PtrString(enrichment.Aws.ResourceType),
						},
					},
				},
			})
		} else if enrichment.Custom != nil {
			customEnrichmentID, err := e.ExtractCustomEnrichmentID(ctx, &enrichment.Custom.CustomEnrichmentRef)
			if err != nil {
				return nil, err
			}

			model := enrichments.EnrichmentRequestModel{
				FieldName: enrichment.Custom.FieldName,
				EnrichmentType: enrichments.EnrichmentType{
					EnrichmentTypeCustomEnrichment: &enrichments.EnrichmentTypeCustomEnrichment{
						CustomEnrichment: &enrichments.CustomEnrichmentType{
							Id: &customEnrichmentID,
						},
					},
				},
			}
			if enrichment.Custom.EnrichedFieldName != nil {
				model.EnrichedFieldName = enrichment.Custom.EnrichedFieldName
			}

			if len(enrichment.Custom.SelectedColumns) > 0 {
				model.SelectedColumns = enrichment.Custom.SelectedColumns
			}

			reqs = append(reqs, model)
		} else {
			return nil, fmt.Errorf("invalid spec: exactly one of geoIp, suspiciousIp, aws, or custom must be set")
		}
	}

	return &enrichments.EnrichmentServiceAtomicOverwriteEnrichmentsRequest{
		RequestEnrichments: reqs,
	}, nil
}

func (e *Enrichment) ExtractCustomEnrichmentID(ctx context.Context, customEnrichment *CustomEnrichmentRef) (int64, error) {
	if customEnrichment.BackendRef != nil {
		return int64(customEnrichment.BackendRef.Id), nil
	}
	if customEnrichment.ResourceRef == nil {
		return 0, fmt.Errorf("customEnrichment must have backendRef or resourceRef")
	}

	ref := customEnrichment.ResourceRef
	ns := e.Namespace
	if ref.Namespace != nil && *ref.Namespace != "" {
		ns = *ref.Namespace
	}
	var ce CustomEnrichment
	if err := config.GetClient().Get(ctx, types.NamespacedName{Namespace: ns, Name: ref.Name}, &ce); err != nil {
		return 0, fmt.Errorf("error getting CustomEnrichment %s/%s: %w", ns, ref.Name, err)
	}

	if ce.Status.Id == nil || *ce.Status.Id == "" {
		return 0, fmt.Errorf("CustomEnrichment %s/%s has no status.id", ns, ref.Name)
	}

	id, err := strconv.ParseInt(*ce.Status.Id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing CustomEnrichment status.id %q: %w", *ce.Status.Id, err)
	}

	return id, nil
}

func init() {
	SchemeBuilder.Register(&Enrichment{}, &EnrichmentList{})
}

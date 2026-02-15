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
// +kubebuilder:validation:XValidation:rule="(has(self.geoIp) ? 1 : 0) + (has(self.suspiciousIp) ? 1 : 0) + (has(self.aws) ? 1 : 0) + (has(self.custom) ? 1 : 0) == 1", message="Exactly one of geoIp, suspiciousIp, aws, or custom must be set"
type EnrichmentSpec struct {
	// Set of fields to enrich with geo_ip information.
	// +optional
	GeoIp *GeoIpEnrichmentType `json:"geoIp,omitempty"`

	// Coralogix allows you to automatically discover threats on your web servers
	// by enriching your logs with the most updated IP blacklists.
	SuspiciousIp *SuspiciousIpEnrichment `json:"suspiciousIp,omitempty"`

	// Coralogix allows you to enrich your logs with the data from a chosen AWS resource.
	// The feature enriches every log that contains a particular resourceId,
	// associated with the metadata of a chosen AWS resource.
	// +optional
	Aws *AwsEnrichment `json:"aws,omitempty"`

	// Custom Log Enrichment with Coralogix enables you to easily enrich your log data.
	// +optional
	Custom *CustomEnrichmentType `json:"custom,omitempty"`
}

type GeoIpEnrichmentType struct {
	FieldName string `json:"fieldName"`

	// +optional
	WithAsn *bool `json:"withAsn,omitempty"`
}

type SuspiciousIpEnrichment struct {
	FieldName string `json:"fieldName"`
}

type AwsEnrichment struct {
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
	DataSet DataSetRef `json:"dataSet"`
}

type DataSetRef struct {
	// BackendRef is a reference to a DataSet in the backend.
	// +optional
	BackendRef *DataSetBackendRef `json:"backendRef,omitempty"`

	// ResourceRef is a reference to a DataSet resource in the cluster.
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

type DataSetBackendRef struct {
	// ID of the DataSet in the backend.
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
	return e.Status.Id != nil && *e.Status.Id != ""
}

func (e *Enrichment) ExtractEnrichmentsCreationRequest(ctx context.Context) (*enrichments.EnrichmentsCreationRequest, error) {
	var reqs []enrichments.EnrichmentRequestModel
	if e.Spec.GeoIp != nil {
		reqs = []enrichments.EnrichmentRequestModel{{
			FieldName: e.Spec.GeoIp.FieldName,
			EnrichmentType: enrichments.EnrichmentType{
				EnrichmentTypeGeoIp: &enrichments.EnrichmentTypeGeoIp{
					GeoIp: &enrichments.GeoIpType{
						WithAsn: e.Spec.GeoIp.WithAsn,
					},
				},
			},
		}}
	} else if e.Spec.SuspiciousIp != nil {
		reqs = []enrichments.EnrichmentRequestModel{{
			FieldName: e.Spec.SuspiciousIp.FieldName,
			EnrichmentType: enrichments.EnrichmentType{
				EnrichmentTypeSuspiciousIp: &enrichments.EnrichmentTypeSuspiciousIp{},
			},
		}}
	} else if e.Spec.Aws != nil {
		reqs = []enrichments.EnrichmentRequestModel{{
			FieldName: e.Spec.Aws.FieldName,
			EnrichmentType: enrichments.EnrichmentType{
				EnrichmentTypeAws: &enrichments.EnrichmentTypeAws{
					Aws: &enrichments.AwsType{
						ResourceType: enrichments.PtrString(e.Spec.Aws.ResourceType),
					},
				},
			},
		}}
	} else if e.Spec.Custom != nil {
		customEnrichmentID, err := e.ExtractCustomEnrichmentID(ctx, &e.Spec.Custom.DataSet)
		if err != nil {
			return nil, err
		}

		model := enrichments.EnrichmentRequestModel{
			FieldName: e.Spec.Custom.FieldName,
			EnrichmentType: enrichments.EnrichmentType{
				EnrichmentTypeCustomEnrichment: &enrichments.EnrichmentTypeCustomEnrichment{
					CustomEnrichment: &enrichments.CustomEnrichmentType{
						Id: &customEnrichmentID,
					},
				},
			},
		}
		if e.Spec.Custom.EnrichedFieldName != nil {
			model.EnrichedFieldName = e.Spec.Custom.EnrichedFieldName
		}

		if len(e.Spec.Custom.SelectedColumns) > 0 {
			model.SelectedColumns = e.Spec.Custom.SelectedColumns
		}

		reqs = []enrichments.EnrichmentRequestModel{model}
	} else {
		return nil, fmt.Errorf("invalid spec: exactly one of geoIp, suspiciousIp, aws, or custom must be set")
	}

	return &enrichments.EnrichmentsCreationRequest{
		RequestEnrichments: reqs,
	}, nil
}

func (e *Enrichment) ExtractCustomEnrichmentID(ctx context.Context, dataSet *DataSetRef) (int64, error) {
	if dataSet.BackendRef != nil {
		return int64(dataSet.BackendRef.Id), nil
	}
	if dataSet.ResourceRef == nil {
		return 0, fmt.Errorf("dataSet must have backendRef or resourceRef")
	}

	ref := dataSet.ResourceRef
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

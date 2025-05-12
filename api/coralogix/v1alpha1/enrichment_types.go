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
	"context"
	"fmt"
	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/coralogix/coralogix-operator/internal/config"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// EnrichmentSpec defines the desired state of Enrichment.
// +kubebuilder:validation:XValidation:rule="(has(self.geoIp) ? 1 : 0) + (has(self.suspiciousIp) ? 1 : 0) + (has(self.aws) ? 1 : 0) + (has(self.custom) ? 1 : 0) == 1", message="Exactly one of geoIp, suspiciousIp, aws, or custom must be set"
type EnrichmentSpec struct {
	// Set of fields to enrich with geo_ip information.
	// +optional
	GeoIp *GeoIpEnrichment `json:"geoIp,omitempty"`

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
	Custom *CustomEnrichment `json:"custom,omitempty"`
}

type GeoIpEnrichment struct {
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

type CustomEnrichment struct {
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

func (e *Enrichment) ExtractCreateEnrichmentRequest(ctx context.Context) (*cxsdk.AddEnrichmentsRequest, error) {
	if e.Spec.GeoIp != nil {
		return e.extractGeoIp(), nil
	}
	if e.Spec.SuspiciousIp != nil {
		return e.extractSuspiciousIp(), nil
	}
	if e.Spec.Aws != nil {
		return e.extractAws(), nil
	}
	if e.Spec.Custom != nil {
		return e.extractCustom(ctx)
	}

	return nil, fmt.Errorf("no enrichment type specified")
}

func (e *Enrichment) extractGeoIp() *cxsdk.AddEnrichmentsRequest {
	return &cxsdk.AddEnrichmentsRequest{
		RequestEnrichments: []*cxsdk.EnrichmentRequestModel{
			{
				FieldName: wrapperspb.String(e.Spec.GeoIp.FieldName),
				EnrichmentType: &cxsdk.EnrichmentType{
					Type: &cxsdk.EnrichmentTypeGeoIP{
						GeoIp: &cxsdk.GeoIPType{
							WithAsn: e.Spec.GeoIp.WithAsn,
						},
					},
				},
			},
		},
	}
}

func (e *Enrichment) extractSuspiciousIp() *cxsdk.AddEnrichmentsRequest {
	return &cxsdk.AddEnrichmentsRequest{
		RequestEnrichments: []*cxsdk.EnrichmentRequestModel{
			{
				FieldName: wrapperspb.String(e.Spec.SuspiciousIp.FieldName),
				EnrichmentType: &cxsdk.EnrichmentType{
					Type: &cxsdk.EnrichmentTypeSuspiciousIP{
						SuspiciousIp: &cxsdk.SuspiciousIPType{},
					},
				},
			},
		},
	}
}

func (e *Enrichment) extractAws() *cxsdk.AddEnrichmentsRequest {
	return &cxsdk.AddEnrichmentsRequest{
		RequestEnrichments: []*cxsdk.EnrichmentRequestModel{
			{
				FieldName: wrapperspb.String(e.Spec.Aws.FieldName),
				EnrichmentType: &cxsdk.EnrichmentType{
					Type: &cxsdk.EnrichmentTypeAws{
						Aws: &cxsdk.AwsType{
							ResourceType: wrapperspb.String(e.Spec.Aws.ResourceType),
						},
					},
				},
			},
		},
	}
}

func (e *Enrichment) extractCustom(ctx context.Context) (*cxsdk.AddEnrichmentsRequest, error) {
	id, err := e.ExtractDataSetID(ctx, e.Spec.Custom.DataSet)
	if err != nil {
		return nil, fmt.Errorf("failed to extract DataSet ID: %w", err)
	}

	enrichmentRequestModel := cxsdk.EnrichmentRequestModel{
		FieldName: wrapperspb.String(e.Spec.Custom.FieldName),
		EnrichmentType: &cxsdk.EnrichmentType{
			Type: &cxsdk.EnrichmentTypeCustomEnrichment{
				CustomEnrichment: &cxsdk.CustomEnrichmentType{
					Id: wrapperspb.UInt32(id),
				},
			},
		},
		SelectedColumns: e.Spec.Custom.SelectedColumns,
	}

	if e.Spec.Custom.EnrichedFieldName != nil {
		enrichmentRequestModel.EnrichedFieldName = wrapperspb.String(*e.Spec.Custom.EnrichedFieldName)
	}

	return &cxsdk.AddEnrichmentsRequest{
		RequestEnrichments: []*cxsdk.EnrichmentRequestModel{
			&enrichmentRequestModel,
		},
	}, nil
}

func (e *Enrichment) ExtractDataSetID(ctx context.Context, dataSetRef DataSetRef) (uint32, error) {
	if dataSetRef.BackendRef != nil {
		return dataSetRef.BackendRef.Id, nil
	} else if dataSetRef.ResourceRef != nil {
		namespace := e.Namespace
		if dataSetRef.ResourceRef.Namespace != nil {
			namespace = *dataSetRef.ResourceRef.Namespace
		}
		return extractDataSetIdFromResourceRef(ctx, dataSetRef.ResourceRef, namespace)
	}

	return 0, fmt.Errorf("DataSetRef must have either BackendRef or ResourceRef set")
}

func extractDataSetIdFromResourceRef(ctx context.Context, ref *ResourceRef, namespace string) (uint32, error) {
	ds := &DataSet{}
	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: namespace}, ds); err != nil {
		return 0, err
	}

	if !config.GetConfig().Selector.Matches(ds.Labels, ds.Namespace) {
		return 0, fmt.Errorf("data set %s does not match selector", ds.Name)
	}

	if ds.Status.ID == nil {
		return 0, fmt.Errorf("ID is not populated for DataSet %s", ds.Name)
	}

	id, err := strconv.Atoi(*ds.Status.ID)
	if err != nil {
		return 0, err
	}

	return uint32(id), nil

}

// EnrichmentStatus defines the observed state of DataSet.
type EnrichmentStatus struct { // +optional
	ID *uint32 `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (e *Enrichment) GetConditions() []metav1.Condition {
	return e.Status.Conditions
}

func (e *Enrichment) SetConditions(conditions []metav1.Condition) {
	e.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Enrichment is the Schema for the enrichments API.
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

func init() {
	SchemeBuilder.Register(&Enrichment{}, &EnrichmentList{})
}

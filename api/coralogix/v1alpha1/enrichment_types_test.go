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
	"reflect"
	"testing"
)

func TestEnrichmentExtractAtomicOverwriteRequestIncludesNonCustomFieldOptions(t *testing.T) {
	withAsn := true
	enrichment := &Enrichment{
		Spec: EnrichmentSpec{
			Enrichments: []EnrichmentType{
				{
					GeoIp: &GeoIpEnrichmentType{
						FieldName:         "attributes.client_ip",
						EnrichedFieldName: stringPtr("client_geo"),
						SelectedColumns:   []string{"city", "country"},
						WithAsn:           &withAsn,
					},
				},
				{
					SuspiciousIp: &SuspiciousIpEnrichmentType{
						FieldName:         "attributes.source_ip",
						EnrichedFieldName: stringPtr("source_threat"),
						SelectedColumns:   []string{"classification", "threat_score"},
					},
				},
				{
					Aws: &AwsEnrichmentType{
						FieldName:         "attributes.aws_resource_id",
						EnrichedFieldName: stringPtr("aws_resource"),
						SelectedColumns:   []string{"resourceId", "accountId"},
						ResourceType:      "AWS::EC2::Instance",
					},
				},
			},
		},
	}

	req, err := enrichment.ExtractAtomicOverwriteRequest(context.Background())
	if err != nil {
		t.Fatalf("ExtractAtomicOverwriteRequest returned error: %v", err)
	}
	if got, want := len(req.RequestEnrichments), 3; got != want {
		t.Fatalf("RequestEnrichments length = %d, want %d", got, want)
	}

	assertRequestFieldOptions(t, &req.RequestEnrichments[0], "attributes.client_ip", "client_geo", []string{"city", "country"})
	if req.RequestEnrichments[0].EnrichmentType.EnrichmentTypeGeoIp == nil {
		t.Fatal("geo_ip enrichment type was not set")
	}
	if got := req.RequestEnrichments[0].EnrichmentType.EnrichmentTypeGeoIp.GeoIp.WithAsn; got == nil || !*got {
		t.Fatalf("geo_ip WithAsn = %v, want true", got)
	}

	assertRequestFieldOptions(
		t,
		&req.RequestEnrichments[1],
		"attributes.source_ip",
		"source_threat",
		[]string{"classification", "threat_score"},
	)
	if req.RequestEnrichments[1].EnrichmentType.EnrichmentTypeSuspiciousIp == nil {
		t.Fatal("suspicious_ip enrichment type was not set")
	}

	assertRequestFieldOptions(
		t,
		&req.RequestEnrichments[2],
		"attributes.aws_resource_id",
		"aws_resource",
		[]string{"resourceId", "accountId"},
	)
	awsType := req.RequestEnrichments[2].EnrichmentType.EnrichmentTypeAws
	if awsType == nil {
		t.Fatal("aws enrichment type was not set")
	}
	if got := awsType.Aws.ResourceType; got == nil || *got != "AWS::EC2::Instance" {
		t.Fatalf("aws ResourceType = %v, want AWS::EC2::Instance", got)
	}
}

func TestEnrichmentExtractAtomicOverwriteRequestOmitsEmptyFieldOptions(t *testing.T) {
	enrichment := &Enrichment{
		Spec: EnrichmentSpec{
			Enrichments: []EnrichmentType{
				{
					GeoIp: &GeoIpEnrichmentType{
						FieldName: "attributes.client_ip",
					},
				},
				{
					SuspiciousIp: &SuspiciousIpEnrichmentType{
						FieldName:       "attributes.source_ip",
						SelectedColumns: []string{},
					},
				},
				{
					Aws: &AwsEnrichmentType{
						FieldName:    "attributes.aws_resource_id",
						ResourceType: "AWS::EC2::Instance",
					},
				},
			},
		},
	}

	req, err := enrichment.ExtractAtomicOverwriteRequest(context.Background())
	if err != nil {
		t.Fatalf("ExtractAtomicOverwriteRequest returned error: %v", err)
	}

	for i, requestEnrichment := range req.RequestEnrichments {
		if requestEnrichment.EnrichedFieldName != nil {
			t.Fatalf("RequestEnrichments[%d].EnrichedFieldName = %q, want nil", i, *requestEnrichment.EnrichedFieldName)
		}
		if requestEnrichment.SelectedColumns != nil {
			t.Fatalf("RequestEnrichments[%d].SelectedColumns = %v, want nil", i, requestEnrichment.SelectedColumns)
		}
	}
}

func assertRequestFieldOptions(t *testing.T, got interface {
	GetFieldName() string
	GetEnrichedFieldNameOk() (*string, bool)
	GetSelectedColumns() []string
}, wantFieldName string, wantEnrichedFieldName string, wantSelectedColumns []string) {
	t.Helper()

	if got.GetFieldName() != wantFieldName {
		t.Fatalf("FieldName = %q, want %q", got.GetFieldName(), wantFieldName)
	}

	enrichedFieldName, ok := got.GetEnrichedFieldNameOk()
	if !ok {
		t.Fatal("EnrichedFieldName was not set")
	}
	if *enrichedFieldName != wantEnrichedFieldName {
		t.Fatalf("EnrichedFieldName = %q, want %q", *enrichedFieldName, wantEnrichedFieldName)
	}

	if !reflect.DeepEqual(got.GetSelectedColumns(), wantSelectedColumns) {
		t.Fatalf("SelectedColumns = %v, want %v", got.GetSelectedColumns(), wantSelectedColumns)
	}
}

func stringPtr(value string) *string {
	return &value
}

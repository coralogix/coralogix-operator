// Copyright 2025 Coralogix Ltd.
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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	archivemetrics "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/metrics_data_archive_service"
)

// ArchiveMetricsTargetSpec defines the desired state of a Coralogix archive logs target.
type ArchiveMetricsTargetSpec struct {
	// The S3 target configuration.
	// +optional
	S3Target *S3MetricsTarget `json:"s3Target,omitempty"`
	// The resolution policy for the metrics.
	ResolutionPolicy *ResolutionPolicy `json:"resolutionPolicy,omitempty"`
	// The retention days for the metrics.
	RetentionDays *int64 `json:"retentionDays,omitempty"`
}

type ResolutionPolicy struct {
	RawResolution         *int64 `json:"rawResolution,omitempty"`
	FiveMinutesResolution *int64 `json:"fiveMinutesResolution,omitempty"`
	OneHourResolution     *int64 `json:"oneHourResolution,omitempty"`
}

type S3MetricsTarget struct {
	// The region of the S3 bucket.
	Region     *string `json:"region,omitempty"`
	BucketName *string `json:"bucketName,omitempty"`
}

type ArchiveMetricsTargetStatus struct {
	// ID is the identifier of the archive metrics target.
	ID *string `json:"id,omitempty"` // The ID of the archive metrics target, if applicable.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (s *ArchiveMetricsTargetSpec) ExtractConfigureTenantRequest() (*archivemetrics.MetricsConfiguratorPublicServiceConfigureTenantRequest, error) {
	if s.S3Target != nil {
		return &archivemetrics.MetricsConfiguratorPublicServiceConfigureTenantRequest{
			ConfigureTenantRequestS3: &archivemetrics.ConfigureTenantRequestS3{
				RetentionPolicy: &archivemetrics.RetentionPolicyRequest{
					RawResolution:         s.ResolutionPolicy.RawResolution,
					FiveMinutesResolution: s.ResolutionPolicy.FiveMinutesResolution,
					OneHourResolution:     s.ResolutionPolicy.OneHourResolution,
				},
				S3: &archivemetrics.S3Config{
					Region: s.S3Target.Region,
					Bucket: s.S3Target.BucketName,
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("archive metrics target does not have a S3Target")
}

func (s *ArchiveMetricsTargetSpec) ExtractUpdateRequest() (*archivemetrics.MetricsConfiguratorPublicServiceUpdateRequest, error) {
	if s.S3Target != nil {
		return &archivemetrics.MetricsConfiguratorPublicServiceUpdateRequest{
			UpdateRequestS3: &archivemetrics.UpdateRequestS3{
				RetentionDays: s.RetentionDays,
				S3: &archivemetrics.S3Config{
					Region: s.S3Target.Region,
					Bucket: s.S3Target.BucketName,
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("archive metrics target does not have a S3Target")
}

func (a *ArchiveMetricsTarget) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *ArchiveMetricsTarget) SetConditions(conditions []metav1.Condition) {
	a.Status.Conditions = conditions
}

func (a *ArchiveMetricsTarget) GetPrintableStatus() string {
	return a.Status.PrintableStatus
}

func (a *ArchiveMetricsTarget) SetPrintableStatus(printableStatus string) {
	a.Status.PrintableStatus = printableStatus
}

func (a *ArchiveMetricsTarget) HasIDInStatus() bool {
	return a.Status.ID != nil && *a.Status.ID != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// ArchiveLogsTarget is the Schema for the archive logs targets API.
// See also https://coralogix.com/docs/archive-s3-bucket-forever
//
// **Added in v0.5.0**
type ArchiveMetricsTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArchiveMetricsTargetSpec   `json:"spec,omitempty"`
	Status ArchiveMetricsTargetStatus `json:"status,omitempty"`
}

// ArchiveMetricsTargetList contains a list of ArchiveMetricsTarget.
// +kubebuilder:object:root=true
type ArchiveMetricsTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArchiveMetricsTarget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArchiveMetricsTarget{}, &ArchiveMetricsTargetList{})
}

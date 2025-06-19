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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// ArchiveMetricsTargetSpec defines the desired state of a Coralogix archive logs target.
// +kubebuilder:validation:XValidation:rule="has(self.s3Target) != has(self.ibmCosTarget)",message="Exactly one of s3Target or ibmCosTarget must be specified"
type ArchiveMetricsTargetSpec struct {
	// The S3 target configuration.
	// +optional
	S3Target *S3MetricsTarget `json:"s3Target,omitempty"`
	// The IBM COS target configuration.
	// +optional
	IbmCosTarget *IbmCosMetricsTarget `json:"ibmCosTarget,omitempty"`
	// The resolution policy for the metrics.
	ResolutionPolicy *ResolutionPolicy `json:"resolutionPolicy,omitempty"`
	// The retention days for the metrics.
	RetentionDays uint32 `json:"retentionDays,omitempty"`
}

type ResolutionPolicy struct {
	RawResolution         uint32 `json:"rawResolution,omitempty"`
	FiveMinutesResolution uint32 `json:"fiveMinutesResolution,omitempty"`
	OneHourResolution     uint32 `json:"oneHourResolution,omitempty"`
}

type S3MetricsTarget struct {
	// The region of the S3 bucket.
	Region     string `json:"region,omitempty"`
	BucketName string `json:"bucketName,omitempty"`
}

type IbmCosMetricsTarget struct {
	// BucketCrn is the CRN of the IBM COS bucket.
	BucketCrn string `json:"bucketCrn,omitempty"`
	// Endpoint is the endpoint URL for the IBM COS service.
	Endpoint string `json:"endpoint,omitempty"`
	// ServiceCrn is the CRN of the service instance.
	// +optional
	ServiceCrn *string `json:"serviceCrn,omitempty"`
	// BucketType defines the type of the bucket.
	// +kubebuilder:validation:Enum=UNSPECIFIED;EXTERNAL;INTERNAL
	// +optional
	BucketType *string `json:"bucketType,omitempty"`
}

type ArchiveMetricsTargetStatus struct {
	// ID is the identifier of the archive metrics target.
	ID *string `json:"id,omitempty"` // The ID of the archive metrics target, if applicable.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (s *ArchiveMetricsTargetSpec) ExtractConfigureTenantRequest() (*cxsdk.ConfigureTenantRequest, error) {
	if s.S3Target != nil {
		return &cxsdk.ConfigureTenantRequest{
			RetentionPolicy: &cxsdk.RetentionPolicyRequest{
				RawResolution:         s.ResolutionPolicy.RawResolution,
				FiveMinutesResolution: s.ResolutionPolicy.FiveMinutesResolution,
				OneHourResolution:     s.ResolutionPolicy.OneHourResolution,
			},
			StorageConfig: &cxsdk.ConfigureTenantRequestS3{
				S3: &cxsdk.ArchiveS3Config{
					Region: s.S3Target.Region,
					Bucket: s.S3Target.BucketName,
				},
			},
		}, nil
	} else {

		return &cxsdk.ConfigureTenantRequest{
			RetentionPolicy: &cxsdk.RetentionPolicyRequest{
				RawResolution:         s.ResolutionPolicy.RawResolution,
				FiveMinutesResolution: s.ResolutionPolicy.FiveMinutesResolution,
				OneHourResolution:     s.ResolutionPolicy.OneHourResolution,
			},
			StorageConfig: &cxsdk.ConfigureTenantRequestIbm{
				Ibm: &cxsdk.ArchiveIbmConfigV2{
					Crn:        s.IbmCosTarget.BucketCrn,
					Endpoint:   s.IbmCosTarget.Endpoint,
					ServiceCrn: *s.IbmCosTarget.ServiceCrn,
				},
			},
		}, nil
	}
}

func (s *ArchiveMetricsTargetSpec) ExtractUpdateRequest() (*cxsdk.UpdateTenantRequest, error) {
	if s.S3Target != nil {
		return &cxsdk.UpdateTenantRequest{
			RetentionDays: &s.RetentionDays,
			StorageConfig: &cxsdk.UpdateRequestS3{
				S3: &cxsdk.ArchiveS3Config{
					Region: s.S3Target.Region,
					Bucket: s.S3Target.BucketName,
				},
			},
		}, nil
	} else {
		return &cxsdk.UpdateTenantRequest{
			RetentionDays: &s.RetentionDays,
			StorageConfig: &cxsdk.UpdateRequestIbm{
				Ibm: &cxsdk.ArchiveIbmConfigV2{
					Crn:        s.IbmCosTarget.BucketCrn,
					Endpoint:   s.IbmCosTarget.Endpoint,
					ServiceCrn: *s.IbmCosTarget.ServiceCrn,
				},
			},
		}, nil
	}
}

func (i *ArchiveMetricsTarget) GetConditions() []metav1.Condition {
	return i.Status.Conditions
}

func (i *ArchiveMetricsTarget) SetConditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
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

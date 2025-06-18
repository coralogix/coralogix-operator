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

// ArchiveLogsTargetSpec defines the desired state of a Coralogix archive logs target.
// See also https://coralogix.com/docs/user-guides/account-management/user-management/create-roles-and-permissions/
// +kubebuilder:validation:XValidation:rule="has(self.s3Target) != has(self.ibmCosTarget)",message="Exactly one of s3Target or ibmCosTarget must be specified"
//
// Added in version v1.0.0
type ArchiveLogsTargetSpec struct {
	// The S3 target configuration.
	// +optional
	S3Target *S3Target `json:"s3Target,omitempty"`
	// The IBM COS target configuration.
	// +optional
	IbmCosTarget *IbmCosTarget `json:"ibmCosTarget,omitempty"`
}

type S3Target struct {
	// The region of the S3 bucket.
	Region     string `json:"region,omitempty"`
	BucketName string `json:"bucketName,omitempty"`
}

type IbmCosTarget struct {
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

type ArchiveLogsTargetStatus struct {
	// ID is the identifier of the archive logs target.
	ID *string `json:"id,omitempty"` // The ID of the archive logs target, if applicable.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (s *ArchiveLogsTargetSpec) ExtractSetTargetRequest(isTargetActive bool) (*cxsdk.SetTargetRequest, error) {
	if s.S3Target != nil {
		return &cxsdk.SetTargetRequest{
			IsActive: isTargetActive,
			TargetSpec: &cxsdk.SetTargetRequestS3{
				S3: &cxsdk.S3TargetSpec{
					Region: &s.S3Target.Region,
					Bucket: s.S3Target.BucketName,
				},
			},
		}, nil
	} else {
		var bucketType cxsdk.IbmBucketType
		if s.IbmCosTarget.BucketType != nil {
			switch *s.IbmCosTarget.BucketType {
			case "UNSPECIFIED":
				bucketType = cxsdk.IbmBucketTypeUnspecified
			case "EXTERNAL":
				bucketType = cxsdk.IbmBucketTypeExternal
			case "INTERNAL":
				bucketType = cxsdk.IbmBucketTypeInternal

			}
		}

		return &cxsdk.SetTargetRequest{
			IsActive: true,
			TargetSpec: &cxsdk.SetTargetRequestIbmCos{
				IbmCos: &cxsdk.IBMCosTargetSpec{
					BucketCrn:  s.IbmCosTarget.BucketCrn,
					Endpoint:   s.IbmCosTarget.Endpoint,
					ServiceCrn: s.IbmCosTarget.ServiceCrn,
					BucketType: &bucketType,
				},
			},
		}, nil
	}
}

func (i *ArchiveLogsTarget) GetConditions() []metav1.Condition {
	return i.Status.Conditions
}

func (i *ArchiveLogsTarget) SetConditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// ArchiveLogsTarget is the Schema for the archive logs targets API.
//
// **Added in v0.5.0**
type ArchiveLogsTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArchiveLogsTargetSpec   `json:"spec,omitempty"`
	Status ArchiveLogsTargetStatus `json:"status,omitempty"`
}

// ArchiveLogsTargetList contains a list of ArchiveLogsTarget.
// +kubebuilder:object:root=true
type ArchiveLogsTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArchiveLogsTarget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArchiveLogsTarget{}, &ArchiveLogsTargetList{})
}

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

	targets "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/target_service"
)

// ArchiveLogsTargetSpec defines the desired state of a Coralogix archive logs target.
// +kubebuilder:validation:XValidation:rule="has(self.s3Target) != has(self.ibmCosTarget)",message="Exactly one of s3Target or ibmCosTarget must be specified"
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

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (s *ArchiveLogsTargetSpec) ExtractSetTargetRequest(isTargetActive bool) (*targets.SetTargetResponse, error) {
	if s.S3Target != nil {
		return &targets.SetTargetResponse{
			IsActive: isTargetActive,
			S3: &targets.S3TargetSpec{
				Region: &s.S3Target.Region,
				Bucket: s.S3Target.BucketName,
			},
		}, nil
	}

	return nil, fmt.Errorf("S3Target cannot be nil")
}

func (a *ArchiveLogsTarget) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *ArchiveLogsTarget) SetConditions(conditions []metav1.Condition) {
	a.Status.Conditions = conditions
}

func (a *ArchiveLogsTarget) GetPrintableStatus() string {
	return a.Status.PrintableStatus
}

func (a *ArchiveLogsTarget) SetPrintableStatus(printableStatus string) {
	a.Status.PrintableStatus = printableStatus
}

func (a *ArchiveLogsTarget) HasIDInStatus() bool {
	return a.Status.ID != nil && *a.Status.ID != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// ArchiveLogsTarget is the Schema for the Archive Logs API.
// See also https://coralogix.com/docs/user-guides/account-management/user-management/create-roles-and-permissions/
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

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

// The handwritten part of the generated ArchiveLogsTarget kind: the IBM COS target of the released
// CRD. The REST API sets only S3 targets, so the validate hook rejects a CR that holds one.

// IbmCosTarget is an IBM Cloud Object Storage target.
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

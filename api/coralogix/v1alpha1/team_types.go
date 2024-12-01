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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// TeamSpec defines the desired state of Team.
type TeamSpec struct {
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// +optional
	AdminsEmails []string `json:"adminsEmails,omitempty"`

	// +optional
	DailyQuota *float64 `json:"dailyQuota,omitempty"`
}

func (s *TeamSpec) ExtractCreateTeamRequest() *cxsdk.CreateTeamInOrgRequest {
	return &cxsdk.CreateTeamInOrgRequest{
		TeamName:        s.Name,
		TeamAdminsEmail: s.AdminsEmails,
		DailyQuota:      s.DailyQuota,
	}
}

func (s *TeamSpec) ExtractUpdateTeamRequest(id uint32) *cxsdk.UpdateTeamRequest {
	return &cxsdk.UpdateTeamRequest{
		TeamId: &cxsdk.TeamID{
			Id: id,
		},
		TeamName:   ptr.To(s.Name),
		DailyQuota: s.DailyQuota,
	}
}

// TeamStatus defines the observed state of Team.
type TeamStatus struct {
	Id *uint32 `json:"id"`

	Retention *int32 `json:"retention"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Team is the Schema for the teams API.
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team.
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}

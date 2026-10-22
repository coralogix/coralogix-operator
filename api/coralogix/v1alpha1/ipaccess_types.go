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
	"fmt"

	ipaccess "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ip_access_service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var SupportAccessToOpenAPI = map[string]ipaccess.CoralogixCustomerSupportAccess{
	"unspecified": ipaccess.CORALOGIXCUSTOMERSUPPORTACCESS_CORALOGIX_CUSTOMER_SUPPORT_ACCESS_UNSPECIFIED,
	"disabled":    ipaccess.CORALOGIXCUSTOMERSUPPORTACCESS_CORALOGIX_CUSTOMER_SUPPORT_ACCESS_DISABLED,
	"enabled":     ipaccess.CORALOGIXCUSTOMERSUPPORTACCESS_CORALOGIX_CUSTOMER_SUPPORT_ACCESS_ENABLED,
}

// IPAccessSpec defines the desired state of IPAccess.
type IPAccessSpec struct {
	// +kubebuilder:validation:Enum=unspecified;disabled;enabled
	// The Coralogix customer support access setting.
	EnableCoralogixCustomerSupportAccess string `json:"enableCoralogixCustomerSupportAccess"`

	// The list of IP access entries.
	IPAccess []IPAccessRule `json:"ipAccess,omitempty"`
}

// IPAccessRule represents a single IP access entry.
type IPAccessRule struct {
	// The name of the IP access entry.
	// +optional
	Name *string `json:"name,omitempty"`

	// The IP range in CIDR notation.
	IPRange string `json:"ipRange"`

	// Whether this IP access entry is enabled.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

func (i *IPAccess) ExtractCreateIPAccessRequest() (*ipaccess.CreateCompanyIPAccessSettingsRequest, error) {
	supportAccess, ok := SupportAccessToOpenAPI[i.Spec.EnableCoralogixCustomerSupportAccess]
	if !ok {
		return nil, fmt.Errorf("invalid enableCoralogixCustomerSupportAccess value: %q", i.Spec.EnableCoralogixCustomerSupportAccess)
	}

	req := &ipaccess.CreateCompanyIPAccessSettingsRequest{
		EnableCoralogixCustomerSupportAccess: supportAccess.Ptr(),
		IpAccess:                             ExtractIPAccessRules(i.Spec.IPAccess),
	}

	return req, nil
}

func (i *IPAccess) ExtractReplaceIPAccessRequest() (*ipaccess.ReplaceCompanyIPAccessSettingsRequest, error) {
	supportAccess, ok := SupportAccessToOpenAPI[i.Spec.EnableCoralogixCustomerSupportAccess]
	if !ok {
		return nil, fmt.Errorf("invalid enableCoralogixCustomerSupportAccess value: %q", i.Spec.EnableCoralogixCustomerSupportAccess)
	}

	req := &ipaccess.ReplaceCompanyIPAccessSettingsRequest{
		Id:                                   i.Status.ID,
		EnableCoralogixCustomerSupportAccess: supportAccess.Ptr(),
		IpAccess:                             ExtractIPAccessRules(i.Spec.IPAccess),
	}

	return req, nil
}

func ExtractIPAccessRules(specRules []IPAccessRule) []ipaccess.IpAccess {
	if len(specRules) == 0 {
		return nil
	}

	ipAccess := make([]ipaccess.IpAccess, len(specRules))
	for i, rule := range specRules {
		ipAccess[i] = ipaccess.IpAccess{
			Name:    rule.Name,
			IpRange: ipaccess.PtrString(rule.IPRange),
			Enabled: rule.Enabled,
		}
	}
	return ipAccess
}

// IPAccessStatus defines the observed state of IPAccess.
type IPAccessStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (i *IPAccess) GetConditions() []metav1.Condition {
	return i.Status.Conditions
}

func (i *IPAccess) SetConditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}

func (i *IPAccess) GetPrintableStatus() string {
	return i.Status.PrintableStatus
}

func (i *IPAccess) SetPrintableStatus(printableStatus string) {
	i.Status.PrintableStatus = printableStatus
}

func (i *IPAccess) HasIDInStatus() bool {
	return i.Status.ID != nil && *i.Status.ID != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// IPAccess is the Schema for the ipaccesses API.
// See also https://coralogix.com/docs/user-guides/account-management/account-settings/ip-access-control/
// **Added in v1.2.0**
type IPAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPAccessSpec   `json:"spec,omitempty"`
	Status IPAccessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPAccessList contains a list of IPAccess.
type IPAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPAccess{}, &IPAccessList{})
}

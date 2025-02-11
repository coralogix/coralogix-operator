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
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
)

// TCOTracesPoliciesSpec defines the desired state of TCOTracesPolicies.
type TCOTracesPoliciesSpec struct {
	Policies []TCOTracesPolicy `json:"policies"`
}

type TCOTracesPolicy struct {
	Name string `json:"name"`

	// +optional
	Description *string `json:"description,omitempty"`

	// +kubebuilder:validation:Enum=block;high;medium;low
	Priority string `json:"priority"`

	Severities []TCOPolicySeverity `json:"severities"`

	// +optional
	ArchiveRetention *ArchiveRetention `json:"archiveRetention,omitempty"`

	// +optional
	Applications *TCOPolicyRule `json:"applications,omitempty"`

	// +optional
	Subsystems *TCOPolicyRule `json:"subsystems,omitempty"`

	// +optional
	Actions *TCOPolicyRule `json:"actions,omitempty"`

	// +optional
	Services *TCOPolicyRule `json:"services,omitempty"`

	// +optional
	Tags []TCOPolicyTag `json:"tags,omitempty"`
}

type TCOPolicyTag struct {
	// +kubebuilder:validation:Pattern=`^tags\..*`
	Name string `json:"name"`

	Values []string `json:"values"`

	// +kubebuilder:validation:Enum=is;is_not;start_with;includes
	RuleType string `json:"ruleType"`
}

func (s *TCOTracesPoliciesSpec) ExtractOverwriteTracesPoliciesRequest(ctx context.Context, coralogixClientSet *cxsdk.ClientSet) (*cxsdk.AtomicOverwriteSpanPoliciesRequest, error) {
	var policies []*cxsdk.CreateSpanPolicyRequest
	var errs error

	for _, policy := range s.Policies {
		policyReq, err := policy.ExtractCreateSpanPolicyRequest(ctx, coralogixClientSet)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			policies = append(policies, policyReq)
		}
	}

	if errs != nil {
		return nil, errs
	}

	return &cxsdk.AtomicOverwriteSpanPoliciesRequest{Policies: policies}, nil
}

func (p *TCOTracesPolicy) ExtractCreateSpanPolicyRequest(ctx context.Context, coralogixClientSet *cxsdk.ClientSet) (*cxsdk.CreateSpanPolicyRequest, error) {
	var errs error
	priority, err := expandTCOPolicyPriority(p.Priority)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	applicationRule, err := expandTCOPolicyRule(p.Applications)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	subsystemRule, err := expandTCOPolicyRule(p.Subsystems)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	serviceRule, err := expandTCOPolicyRule(p.Services)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	actionRule, err := expandTCOPolicyRule(p.Actions)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	tagsRules, err := expandTCOPolicyTagRules(p.Tags)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	archiveRetentionID, err := expandArchiveRetention(ctx, coralogixClientSet, p.ArchiveRetention)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return nil, errs
	}

	req := &cxsdk.CreateSpanPolicyRequest{
		Policy: &cxsdk.CreateGenericPolicyRequest{
			Name:             wrapperspb.String(p.Name),
			Description:      utils.StringPointerToWrapperspbString(p.Description),
			Priority:         priority,
			ApplicationRule:  applicationRule,
			SubsystemRule:    subsystemRule,
			ArchiveRetention: archiveRetentionID,
		},
		SpanRules: &cxsdk.TCOSpanRules{
			ServiceRule: serviceRule,
			ActionRule:  actionRule,
			TagRules:    tagsRules,
		},
	}

	return req, nil
}

func expandTCOPolicyTagRules(tags []TCOPolicyTag) ([]*cxsdk.TCOPolicyTagRule, error) {
	var tagRules []*cxsdk.TCOPolicyTagRule
	var errs error

	for _, tag := range tags {
		tagRule, err := expandTCOPolicyTagRule(tag)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			tagRules = append(tagRules, tagRule)
		}
	}

	if errs != nil {
		return nil, errs
	}

	return tagRules, nil
}

func expandTCOPolicyTagRule(tag TCOPolicyTag) (*cxsdk.TCOPolicyTagRule, error) {
	ruleType, ok := cxsdk.TCOPolicyRuleTypeValueLookup["RULE_TYPE_ID_"+strings.ToUpper(tag.RuleType)]
	if !ok {
		return nil, fmt.Errorf("invalid rule type for TCO policy: %s", tag.RuleType)
	}

	return &cxsdk.TCOPolicyTagRule{
		TagName:    wrapperspb.String(tag.Name),
		TagValue:   wrapperspb.String(strings.Join(tag.Values, ",")),
		RuleTypeId: cxsdk.TCOPolicyRuleTypeID(ruleType),
	}, nil
}

// TCOTracesPoliciesStatus defines the observed state of TCOTracesPolicies.
type TCOTracesPoliciesStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TCOTracesPolicies is the Schema for the tcotracespolicies API.
// NOTE: This resource performs an atomic overwrite of all existing TCO traces policies
// in the backend. Any existing policies not defined in this resource will be
// removed. Use with caution as this operation is destructive.
type TCOTracesPolicies struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TCOTracesPoliciesSpec   `json:"spec,omitempty"`
	Status TCOTracesPoliciesStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TCOTracesPoliciesList contains a list of TCOTracesPolicies.
type TCOTracesPoliciesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TCOTracesPolicies `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TCOTracesPolicies{}, &TCOTracesPoliciesList{})
}

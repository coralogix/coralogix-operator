package alphacontrollers

import (
	"testing"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	rulesgroups "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/rules-groups/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestFlattenRuleGroupsErrorsOnBadResponse(t *testing.T) {
	ruleGroup := &rulesgroups.RuleGroup{
		Id:           wrapperspb.String("id"),
		Name:         wrapperspb.String("name"),
		Description:  wrapperspb.String("description"),
		Creator:      wrapperspb.String("creator"),
		Enabled:      wrapperspb.Bool(true),
		Hidden:       wrapperspb.Bool(false),
		RuleMatchers: []*rulesgroups.RuleMatcher{},
		RuleSubgroups: []*rulesgroups.RuleSubgroup{
			{
				Rules: []*rulesgroups.Rule{
					{
						Id:          wrapperspb.String("rule_id"),
						Name:        wrapperspb.String("rule_name"),
						Description: wrapperspb.String("rule_description"),
						SourceField: wrapperspb.String("text"),
						Parameters: &rulesgroups.RuleParameters{
							RuleParameters: nil,
						},
						Enabled: wrapperspb.Bool(true),
						Order:   wrapperspb.UInt32(1),
					},
				},
			},
		},
		Order: wrapperspb.UInt32(1),
	}

	status, err := flattenRuleGroup(ruleGroup)
	assert.Error(t, err)
	assert.Nil(t, status)
}

func TestFlattenRuleGroups(t *testing.T) {
	ruleGroup := &rulesgroups.RuleGroup{
		Id:           wrapperspb.String("id"),
		Name:         wrapperspb.String("name"),
		Description:  wrapperspb.String("description"),
		Creator:      wrapperspb.String("creator"),
		Enabled:      wrapperspb.Bool(true),
		Hidden:       wrapperspb.Bool(false),
		RuleMatchers: []*rulesgroups.RuleMatcher{},
		RuleSubgroups: []*rulesgroups.RuleSubgroup{
			{
				Id:    wrapperspb.String("subgroup_id"),
				Order: wrapperspb.UInt32(2),
				Rules: []*rulesgroups.Rule{
					{
						Id:          wrapperspb.String("rule_id"),
						Name:        wrapperspb.String("rule_name"),
						Description: wrapperspb.String("rule_description"),
						SourceField: wrapperspb.String("text"),
						Parameters: &rulesgroups.RuleParameters{
							RuleParameters: &rulesgroups.RuleParameters_JsonExtractParameters{
								JsonExtractParameters: &rulesgroups.JsonExtractParameters{
									DestinationField: rulesgroups.JsonExtractParameters_DESTINATION_FIELD_SEVERITY,
									Rule:             wrapperspb.String(`{"severity": "info"}`),
								},
							},
						},
						Enabled: wrapperspb.Bool(true),
						Order:   wrapperspb.UInt32(3),
					},
				},
			},
		},
		Order: wrapperspb.UInt32(1),
	}

	status, err := flattenRuleGroup(ruleGroup)
	assert.NoError(t, err)

	id := "id"
	subgroupId := "subgroup_id"
	expected := &coralogixv1alpha1.RuleGroupStatus{
		ID:           &id,
		Name:         "name",
		Description:  "description",
		Active:       true,
		Applications: nil,
		Subsystems:   nil,
		Severities:   nil,
		Hidden:       false,
		Creator:      "creator",
		Order:        int32Ptr(1),
		RuleSubgroups: []coralogixv1alpha1.RuleSubGroup{
			{
				ID:     &subgroupId,
				Active: false,
				Order:  int32Ptr(2),
				Rules: []coralogixv1alpha1.Rule{
					{
						Name:        "rule_name",
						Description: "rule_description",
						Active:      true,
						Parse:       nil,
						Block:       nil,
						JsonExtract: &coralogixv1alpha1.JsonExtract{
							DestinationField: coralogixv1alpha1.DestinationFieldRuleSeverity,
							JsonKey:          "{\"severity\": \"info\"}",
						},
					},
				},
			},
		},
	}

	assert.Equal(t, expected, status)
}

func int32Ptr(i int32) *int32 { return &i }

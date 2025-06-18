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
	"encoding/json"

	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
)

var (
	RulesSchemaSeverityToProtoSeverity = map[RuleSeverity]cxsdk.SeverityConstraintValue{
		RuleSeverityDebug:    cxsdk.SeverityConstraintValueDebugOrUnspecified,
		RuleSeverityVerbose:  cxsdk.SeverityConstraintValueVerbose,
		RuleSeverityInfo:     cxsdk.SeverityConstraintValueInfo,
		RuleSeverityWarning:  cxsdk.SeverityConstraintValueWarning,
		RuleSeverityError:    cxsdk.SeverityConstraintValueError,
		RuleSeverityCritical: cxsdk.SeverityConstraintValueCritical,
	}
	RulesProtoSeverityToSchemaSeverity                         = utils.ReverseMap(RulesSchemaSeverityToProtoSeverity)
	RulesSchemaDestinationFieldToProtoSeverityDestinationField = map[DestinationField]cxsdk.JSONExtractParametersDestinationField{
		DestinationFieldCategory:     cxsdk.JSONExtractParametersDestinationFieldCategoryOrUnspecified,
		DestinationFieldClassName:    cxsdk.JSONExtractParametersDestinationFieldClassName,
		DestinationFieldMethod:       cxsdk.JSONExtractParametersDestinationFieldMethodName,
		DestinationFieldThreadID:     cxsdk.JSONExtractParametersDestinationFieldThreadID,
		DestinationFieldRuleSeverity: cxsdk.JSONExtractParametersDestinationFieldSeverity,
	}
	RulesSchemaFormatStandardToProtoFormatStandard = map[FieldFormatStandard]cxsdk.ExtractTimestampParametersFormatStandard{
		FieldFormatStandardStrftime: cxsdk.ExtractTimestampParametersFormatStandardStrftimeOrUnspecified,
		FieldFormatStandardJavaSDF:  cxsdk.ExtractTimestampParametersFormatStandardJavasdf,
		FieldFormatStandardGolang:   cxsdk.ExtractTimestampParametersFormatStandardGolang,
		FieldFormatStandardSecondTS: cxsdk.ExtractTimestampParametersFormatStandardSecondsTS,
		FieldFormatStandardMilliTS:  cxsdk.ExtractTimestampParametersFormatStandardMilliTS,
		FieldFormatStandardMicroTS:  cxsdk.ExtractTimestampParametersFormatStandardMicroTS,
		FieldFormatStandardNanoTS:   cxsdk.ExtractTimestampParametersFormatStandardNanoTS,
	}
)

// A rule to change data extraction.
// +kubebuilder:validation:XValidation:rule="(has(self.parse) ? 1 : 0) + (has(self.block) ? 1 : 0) + (has(self.jsonExtract) ? 1 : 0) + (has(self.replace) ? 1 : 0) + (has(self.extractTimestamp) ? 1 : 0) + (has(self.removeFields) ? 1 : 0) + (has(self.jsonStringify) ? 1 : 0) + (has(self.extract) ? 1 : 0) + (has(self.parseJsonField) ? 1 : 0) == 1",message="Exactly one of the following fields should be set: parse, block, jsonExtract, replace, extractTimestamp, removeFields, jsonStringify, extract, parseJsonField"
type Rule struct {

	// Name of the rule.
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// Description of the rule.
	// +optional
	Description string `json:"description,omitempty"`

	// Whether the rule will be activated.
	//+kubebuilder:default=true
	Active bool `json:"active,omitempty"`

	// Parse unstructured logs into JSON format using named Regular Expression groups.
	// +optional
	Parse *Parse `json:"parse,omitempty"`

	// Block rules allow for refined filtering of incoming logs with a Regular Expression.
	// +optional
	Block *Block `json:"block,omitempty"`

	// Name a JSON field to extract its value directly into a Coralogix metadata field
	// +optional
	JsonExtract *JsonExtract `json:"jsonExtract,omitempty"`

	// Replace rules are used to strings in order to fix log structure, change log severity, or obscure information.
	// +optional
	Replace *Replace `json:"replace,omitempty"`

	// Replace rules are used to replace logs timestamp with JSON field.
	// +optional
	ExtractTimestamp *ExtractTimestamp `json:"extractTimestamp,omitempty"`

	// Remove Fields allows to select fields that will not be indexed.
	// +optional
	RemoveFields *RemoveFields `json:"removeFields,omitempty"`

	// Convert JSON object to JSON string.
	// +optional
	JsonStringify *JsonStringify `json:"jsonStringify,omitempty"`

	// Use a named Regular Expression group to extract specific values you need as JSON getKeysStrings without having to parse the entire log.
	// +optional
	Extract *Extract `json:"extract,omitempty"`

	// Convert JSON string to JSON object.
	// +optional
	ParseJsonField *ParseJsonField `json:"parseJsonField,omitempty"`
}

// Parsing instructions for unstructured fields.
type Parse struct {

	// The field on which the Regular Expression will operate on.
	SourceField string `json:"sourceField"`

	// The field that will be populated by the results of the Regular Expression operation.
	DestinationField string `json:"destinationField"`

	// Regular Expression. More info: https://coralogix.com/blog/regex-101/
	Regex string `json:"regex"`
}

// Blocking instructions
type Block struct {

	// The field on which the Regular Expression will operate on.
	SourceField string `json:"sourceField"`

	// Regular Expression. More info: https://coralogix.com/blog/regex-101/
	Regex string `json:"regex"`

	// Determines if to view blocked logs in LiveTail and archive to S3.
	//+kubebuilder:default=false
	KeepBlockedLogs bool `json:"keepBlockedLogs,omitempty"`

	// Block Logic. If true or nor set - blocking all matching blocks, if false - blocking all non-matching blocks.
	//+kubebuilder:default=true
	BlockingAllMatchingBlocks bool `json:"blockingAllMatchingBlocks,omitempty"`
}

// +kubebuilder:validation:Enum=Category;CLASSNAME;METHODNAME;THREADID;SEVERITY
// The field that will be populated by the results of the Regular Expression operation.
type DestinationField string

const (
	DestinationFieldCategory     DestinationField = "Category"
	DestinationFieldClassName    DestinationField = "CLASSNAME"
	DestinationFieldMethod       DestinationField = "METHODNAME"
	DestinationFieldThreadID     DestinationField = "THREADID"
	DestinationFieldRuleSeverity DestinationField = "SEVERITY"
)

// JsonExtract instructions.
type JsonExtract struct {
	// The field that will be populated by the results of the Regular Expression operation.
	DestinationField DestinationField `json:"destinationField"`

	// JSON key to extract its value directly into a Coralogix metadata field.
	JsonKey string `json:"jsonKey"`
}

// Instructions to replace data.
type Replace struct {

	// The field on which the Regular Expression will operate on.
	SourceField string `json:"sourceField"`

	// The field that will be populated by the results of the Regular Expression operation.
	DestinationField string `json:"destinationField"`

	// Regular Expression. More info: https://coralogix.com/blog/regex-101/
	Regex string `json:"regex"`

	// The string that will replace the matched Regular Expression
	ReplacementString string `json:"replacementString"`
}

// +kubebuilder:validation:Enum=Strftime;JavaSDF;Golang;SecondTS;MilliTS;MicroTS;NanoTS
// The format standard you want to use
type FieldFormatStandard string

const (
	FieldFormatStandardStrftime FieldFormatStandard = "Strftime"
	FieldFormatStandardJavaSDF  FieldFormatStandard = "JavaSDF"
	FieldFormatStandardGolang   FieldFormatStandard = "Golang"
	FieldFormatStandardSecondTS FieldFormatStandard = "SecondTS"
	FieldFormatStandardMilliTS  FieldFormatStandard = "MilliTS"
	FieldFormatStandardMicroTS  FieldFormatStandard = "MicroTS"
	FieldFormatStandardNanoTS   FieldFormatStandard = "NanoTS"
)

// Timestamp extraction instructions.
type ExtractTimestamp struct {
	// The field on which the Regular Expression will operate on.
	SourceField string `json:"sourceField"`

	// The format standard to parse the timestamp.
	FieldFormatStandard FieldFormatStandard `json:"fieldFormatStandard"`

	// A time formatting string that matches the field format standard.
	TimeFormat string `json:"timeFormat"`
}

// Instructions to remove fields from indexing.
type RemoveFields struct {
	// Excluded fields won't be indexed.
	ExcludedFields []string `json:"excludedFields"`
}

// Instructions to convert a JSON object to JSON string.
type JsonStringify struct {
	// The field on which the Regular Expression will operate on.
	SourceField string `json:"sourceField"`

	// The field that will be populated by the results of the Regular Expression
	DestinationField string `json:"destinationField"`

	//+kubebuilder:default=false
	KeepSourceField bool `json:"keepSourceField,omitempty"`
}

// Extract instructions.
type Extract struct {
	// The field on which the Regular Expression will operate on.
	SourceField string `json:"sourceField"`

	// Regular Expression. More info: https://coralogix.com/blog/regex-101/
	Regex string `json:"regex"`
}

// Parsing instructions for a JSON field from a JSON string.
type ParseJsonField struct {
	// The field on which the Regular Expression will operate on.
	SourceField string `json:"sourceField"`

	// The field that will be populated by the results of the Regular Expression
	DestinationField string `json:"destinationField"`

	// Determines whether to keep or to delete the source field.
	KeepSourceField bool `json:"keepSourceField"`

	// Determines whether to keep or to delete the destination field.
	KeepDestinationField bool `json:"keepDestinationField"`
}

// Sub group of rules.
type RuleSubGroup struct {
	// The rule id.
	// +optional
	ID *string `json:"id,omitempty"`

	// Determines whether to rule will be active or not.
	//+kubebuilder:default=true
	Active bool `json:"active,omitempty"`

	// Determines the index of the rule inside the rule-subgroup.
	// +optional
	Order *int32 `json:"order,omitempty"`

	// List of rules associated with the sub group.
	// +optional
	Rules []Rule `json:"rules,omitempty"`
}

// RuleGroupSpec defines the Desired state of RuleGroup
type RuleGroupSpec struct {

	// Name of the rule-group.
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// Description of the rule-group.
	// +optional
	Description string `json:"description,omitempty"`

	// Whether the rule-group is active.
	//+kubebuilder:default=true
	Active bool `json:"active,omitempty"`

	// Rules will execute on logs that match the these applications.
	// +optional
	Applications []string `json:"applications,omitempty"`

	// Rules will execute on logs that match the these subsystems.
	// +optional
	Subsystems []string `json:"subsystems,omitempty"`

	// Rules will execute on logs that match the these severities.
	// +optional
	Severities []RuleSeverity `json:"severities,omitempty"`

	// Hides the rule-group.
	//+kubebuilder:default=false
	Hidden bool `json:"hidden,omitempty"`

	// Rule-group creator
	// +optional
	Creator string `json:"creator,omitempty"`

	// +kubebuilder:validation:Minimum:=1
	// The index of the rule-group between the other rule-groups.
	// +optional
	Order *int32 `json:"order,omitempty"`

	// Rules within the same subgroup have an OR relationship,
	// while rules in different subgroups have an AND relationship.
	// Refer to https://github.com/coralogix/coralogix-operator/blob/main/config/samples/v1alpha1/rulegroups/mixed_rulegroup.yaml
	// for an example.
	// +optional
	RuleSubgroups []RuleSubGroup `json:"subgroups,omitempty"`
}

// +kubebuilder:validation:Enum=Debug;Verbose;Info;Warning;Error;Critical
// Severity to match to.
type RuleSeverity string

const (
	RuleSeverityDebug    RuleSeverity = "Debug"
	RuleSeverityVerbose  RuleSeverity = "Verbose"
	RuleSeverityInfo     RuleSeverity = "Info"
	RuleSeverityWarning  RuleSeverity = "Warning"
	RuleSeverityError    RuleSeverity = "Error"
	RuleSeverityCritical RuleSeverity = "Critical"
)

func (in *RuleGroupSpec) ToString() string {
	str, _ := json.Marshal(*in)
	return string(str)
}

func (in *RuleGroupSpec) ExtractUpdateRuleGroupRequest(id string) *cxsdk.UpdateRuleGroupRequest {
	ruleGroup := in.ExtractCreateRuleGroupRequest()
	return &cxsdk.UpdateRuleGroupRequest{
		GroupId:   wrapperspb.String(id),
		RuleGroup: ruleGroup,
	}
}

func (in *RuleGroupSpec) ExtractCreateRuleGroupRequest() *cxsdk.CreateRuleGroupRequest {
	name := wrapperspb.String(in.Name)
	description := wrapperspb.String(in.Description)
	enabled := wrapperspb.Bool(in.Active)
	hidden := wrapperspb.Bool(in.Hidden)
	creator := wrapperspb.String(in.Creator)
	ruleMatchers := expandRuleMatchers(in.Applications, in.Subsystems, in.Severities)
	ruleSubGroups := expandRuleSubGroups(in.RuleSubgroups)
	order := expandOrder(in.Order)

	return &cxsdk.CreateRuleGroupRequest{
		Name:          name,
		Description:   description,
		Enabled:       enabled,
		Hidden:        hidden,
		Creator:       creator,
		RuleMatchers:  ruleMatchers,
		RuleSubgroups: ruleSubGroups,
		Order:         order,
	}
}

func expandOrder(order *int32) *wrapperspb.UInt32Value {
	if order != nil {
		return wrapperspb.UInt32(uint32(*order))
	}
	return nil
}

func expandRuleSubGroups(subGroups []RuleSubGroup) []*cxsdk.CreateRuleGroupRequestCreateRuleSubgroup {
	ruleSubGroups := make([]*cxsdk.CreateRuleGroupRequestCreateRuleSubgroup, 0, len(subGroups))
	for i, subGroup := range subGroups {
		rsg := expandRuleSubGroup(subGroup)
		rsg.Order = wrapperspb.UInt32(uint32(i + 1))
		ruleSubGroups = append(ruleSubGroups, rsg)
	}
	return ruleSubGroups
}

func expandRuleSubGroup(subGroup RuleSubGroup) *cxsdk.CreateRuleGroupRequestCreateRuleSubgroup {
	enabled := wrapperspb.Bool(subGroup.Active)
	rules := expandRules(subGroup.Rules)
	return &cxsdk.CreateRuleGroupRequestCreateRuleSubgroup{
		Enabled: enabled,
		Rules:   rules,
	}
}

func expandRules(rules []Rule) []*cxsdk.CreateRuleGroupRequestCreateRuleSubgroupCreateRule {
	expandedRules := make([]*cxsdk.CreateRuleGroupRequestCreateRuleSubgroupCreateRule, 0, len(rules))
	for i, rule := range rules {
		r := expandRule(rule)
		r.Order = wrapperspb.UInt32(uint32(i + 1))
		expandedRules = append(expandedRules, r)
	}
	return expandedRules
}

func expandRule(rule Rule) *cxsdk.CreateRuleGroupRequestCreateRuleSubgroupCreateRule {
	name := wrapperspb.String(rule.Name)
	description := wrapperspb.String(rule.Description)
	enabled := wrapperspb.Bool(rule.Active)
	sourceFiled, parameters := expandSourceFiledAndParameters(rule)

	return &cxsdk.CreateRuleGroupRequestCreateRuleSubgroupCreateRule{
		Name:        name,
		Description: description,
		SourceField: sourceFiled,
		Parameters:  parameters,
		Enabled:     enabled,
	}
}

func expandSourceFiledAndParameters(rule Rule) (sourceField *wrapperspb.StringValue, parameters *cxsdk.RuleParameters) {
	if parse := rule.Parse; parse != nil {
		sourceField = wrapperspb.String(parse.SourceField)
		parameters = &cxsdk.RuleParameters{
			RuleParameters: &cxsdk.RuleParametersParseParameters{
				ParseParameters: &cxsdk.ParseParameters{
					DestinationField: wrapperspb.String(parse.DestinationField),
					Rule:             wrapperspb.String(parse.Regex),
				},
			},
		}
	} else if parseJsonField := rule.ParseJsonField; parseJsonField != nil {
		sourceField = wrapperspb.String(parseJsonField.SourceField)
		parameters = &cxsdk.RuleParameters{
			RuleParameters: &cxsdk.RuleParametersJSONParseParameters{
				JsonParseParameters: &cxsdk.JSONParseParameters{
					DestinationField: wrapperspb.String(parseJsonField.DestinationField),
					DeleteSource:     wrapperspb.Bool(!parseJsonField.KeepSourceField),
					OverrideDest:     wrapperspb.Bool(!parseJsonField.KeepDestinationField),
					EscapedValue:     wrapperspb.Bool(true),
				},
			},
		}
	} else if jsonStringify := rule.JsonStringify; jsonStringify != nil {
		sourceField = wrapperspb.String(jsonStringify.SourceField)
		parameters = &cxsdk.RuleParameters{
			RuleParameters: &cxsdk.RuleParametersJSONStringifyParameters{
				JsonStringifyParameters: &cxsdk.JSONStringifyParameters{
					DestinationField: wrapperspb.String(jsonStringify.DestinationField),
					DeleteSource:     wrapperspb.Bool(!jsonStringify.KeepSourceField),
				},
			},
		}
	} else if jsonExtract := rule.JsonExtract; jsonExtract != nil {
		sourceField = wrapperspb.String("text")
		destinationField := RulesSchemaDestinationFieldToProtoSeverityDestinationField[jsonExtract.DestinationField]
		jsonKey := wrapperspb.String(jsonExtract.JsonKey)
		parameters = &cxsdk.RuleParameters{
			RuleParameters: &cxsdk.RuleParametersJSONExtractParameters{
				JsonExtractParameters: &cxsdk.JSONExtractParameters{
					DestinationFieldType: destinationField,
					Rule:                 jsonKey,
				},
			},
		}
	} else if removeFields := rule.RemoveFields; removeFields != nil {
		sourceField = wrapperspb.String("text")
		parameters = &cxsdk.RuleParameters{
			RuleParameters: &cxsdk.RuleParametersRemoveFieldsParameters{
				RemoveFieldsParameters: &cxsdk.RemoveFieldsParameters{
					Fields: removeFields.ExcludedFields,
				},
			},
		}
	} else if extractTimestamp := rule.ExtractTimestamp; extractTimestamp != nil {
		sourceField = wrapperspb.String(extractTimestamp.SourceField)
		standard := RulesSchemaFormatStandardToProtoFormatStandard[extractTimestamp.FieldFormatStandard]
		format := wrapperspb.String(extractTimestamp.TimeFormat)
		parameters = &cxsdk.RuleParameters{
			RuleParameters: &cxsdk.RuleParametersExtractTimestampParameters{
				ExtractTimestampParameters: &cxsdk.ExtractTimestampParameters{
					Standard: standard,
					Format:   format,
				},
			},
		}
	} else if block := rule.Block; block != nil {
		sourceField = wrapperspb.String(block.SourceField)
		if block.BlockingAllMatchingBlocks {
			parameters = &cxsdk.RuleParameters{
				RuleParameters: &cxsdk.RuleParametersBlockParameters{
					BlockParameters: &cxsdk.BlockParameters{
						KeepBlockedLogs: wrapperspb.Bool(block.KeepBlockedLogs),
						Rule:            wrapperspb.String(block.Regex),
					},
				},
			}
		} else {
			parameters = &cxsdk.RuleParameters{
				RuleParameters: &cxsdk.RuleParametersAllowParameters{
					AllowParameters: &cxsdk.AllowParameters{
						KeepBlockedLogs: wrapperspb.Bool(block.KeepBlockedLogs),
						Rule:            wrapperspb.String(block.Regex),
					},
				},
			}
		}
	} else if replace := rule.Replace; replace != nil {
		sourceField = wrapperspb.String(replace.SourceField)
		parameters = &cxsdk.RuleParameters{
			RuleParameters: &cxsdk.RuleParametersReplaceParameters{
				ReplaceParameters: &cxsdk.ReplaceParameters{
					DestinationField: wrapperspb.String(replace.DestinationField),
					ReplaceNewVal:    wrapperspb.String(replace.ReplacementString),
					Rule:             wrapperspb.String(replace.Regex),
				},
			},
		}
	} else if extract := rule.Extract; extract != nil {
		sourceField = wrapperspb.String(extract.SourceField)
		parameters = &cxsdk.RuleParameters{
			RuleParameters: &cxsdk.RuleParametersExtractParameters{
				ExtractParameters: &cxsdk.ExtractParameters{
					Rule: wrapperspb.String(extract.Regex),
				},
			},
		}
	}

	return
}

func expandRuleMatchers(applications, subsystems []string, severities []RuleSeverity) []*cxsdk.RuleMatcher {
	ruleMatchers := make([]*cxsdk.RuleMatcher, 0, len(applications)+len(subsystems)+len(severities))

	for _, app := range applications {
		constraintStr := wrapperspb.String(app)
		applicationNameConstraint := cxsdk.ApplicationNameConstraint{Value: constraintStr}
		ruleMatcherApplicationName := cxsdk.RuleMatcherApplicationName{ApplicationName: &applicationNameConstraint}
		ruleMatchers = append(ruleMatchers, &cxsdk.RuleMatcher{Constraint: &ruleMatcherApplicationName})
	}

	for _, subSys := range subsystems {
		constraintStr := wrapperspb.String(subSys)
		subsystemNameConstraint := cxsdk.SubsystemNameConstraint{Value: constraintStr}
		ruleMatcherApplicationName := cxsdk.RuleMatcherSubsystemName{SubsystemName: &subsystemNameConstraint}
		ruleMatchers = append(ruleMatchers, &cxsdk.RuleMatcher{Constraint: &ruleMatcherApplicationName})
	}

	for _, sev := range severities {
		constraintEnum := RulesSchemaSeverityToProtoSeverity[sev]
		severityConstraint := cxsdk.SeverityConstraint{Value: constraintEnum}
		ruleMatcherSeverity := cxsdk.RuleMatcherSeverity{Severity: &severityConstraint}
		ruleMatchers = append(ruleMatchers, &cxsdk.RuleMatcher{Constraint: &ruleMatcherSeverity})
	}

	return ruleMatchers
}

func flattenRuleMatchers(matchers []*cxsdk.RuleMatcher) (applications []string, subsystems []string, severities []RuleSeverity) {
	applications = make([]string, 0)
	subsystems = make([]string, 0)
	severities = make([]RuleSeverity, 0)

	for _, m := range matchers {
		switch m.Constraint.(type) {
		case *cxsdk.RuleMatcherApplicationName:
			applications = append(applications, m.GetApplicationName().GetValue().GetValue())
		case *cxsdk.RuleMatcherSubsystemName:
			subsystems = append(subsystems, m.GetSubsystemName().GetValue().GetValue())
		case *cxsdk.RuleMatcherSeverity:
			severities = append(severities, RulesProtoSeverityToSchemaSeverity[m.GetSeverity().GetValue()])
		}
	}

	return applications, subsystems, severities
}

// RuleGroupStatus defines the observed state of RuleGroup
type RuleGroupStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (r *RuleGroup) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *RuleGroup) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// RuleGroup is the Schema for the RuleGroups API
// See also https://coralogix.com/docs/user-guides/data-transformation/metric-rules/recording-rules/
//
// **Added in v0.4.0**
type RuleGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuleGroupSpec   `json:"spec,omitempty"`
	Status RuleGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RuleGroupList contains a list of RuleGroups
type RuleGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RuleGroup{}, &RuleGroupList{})
}

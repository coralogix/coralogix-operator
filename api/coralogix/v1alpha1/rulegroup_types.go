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

	rulegroups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/rule_groups_service"

	utils "github.com/coralogix/coralogix-operator/v2/api/coralogix"
)

var (
	RulesSchemaSeverityToOpenAPISeverity = map[RuleSeverity]rulegroups.Value{
		RuleSeverityDebug:    rulegroups.VALUE_VALUE_DEBUG_OR_UNSPECIFIED,
		RuleSeverityVerbose:  rulegroups.VALUE_VALUE_VERBOSE,
		RuleSeverityInfo:     rulegroups.VALUE_VALUE_INFO,
		RuleSeverityWarning:  rulegroups.VALUE_VALUE_WARNING,
		RuleSeverityError:    rulegroups.VALUE_VALUE_ERROR,
		RuleSeverityCritical: rulegroups.VALUE_VALUE_CRITICAL,
	}
	RulesOpenAPISeverityToSchemaSeverity                         = utils.ReverseMap(RulesSchemaSeverityToOpenAPISeverity)
	RulesSchemaDestinationFieldToOpenAPISeverityDestinationField = map[DestinationField]rulegroups.DestinationField{
		DestinationFieldCategory:     rulegroups.DESTINATIONFIELD_DESTINATION_FIELD_CATEGORY_OR_UNSPECIFIED,
		DestinationFieldClassName:    rulegroups.DESTINATIONFIELD_DESTINATION_FIELD_CLASSNAME,
		DestinationFieldMethod:       rulegroups.DESTINATIONFIELD_DESTINATION_FIELD_METHODNAME,
		DestinationFieldThreadID:     rulegroups.DESTINATIONFIELD_DESTINATION_FIELD_THREADID,
		DestinationFieldRuleSeverity: rulegroups.DESTINATIONFIELD_DESTINATION_FIELD_SEVERITY,
	}
	RulesSchemaFormatStandardToOpenAPIFormatStandard = map[FieldFormatStandard]rulegroups.FormatStandard{
		FieldFormatStandardStrftime: rulegroups.FORMATSTANDARD_FORMAT_STANDARD_STRFTIME_OR_UNSPECIFIED,
		FieldFormatStandardJavaSDF:  rulegroups.FORMATSTANDARD_FORMAT_STANDARD_JAVASDF,
		FieldFormatStandardGolang:   rulegroups.FORMATSTANDARD_FORMAT_STANDARD_GOLANG,
		FieldFormatStandardSecondTS: rulegroups.FORMATSTANDARD_FORMAT_STANDARD_SECONDSTS,
		FieldFormatStandardMilliTS:  rulegroups.FORMATSTANDARD_FORMAT_STANDARD_MILLITS,
		FieldFormatStandardMicroTS:  rulegroups.FORMATSTANDARD_FORMAT_STANDARD_MICROTS,
		FieldFormatStandardNanoTS:   rulegroups.FORMATSTANDARD_FORMAT_STANDARD_NANOTS,
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
	// Refer to https://github.com/coralogix/coralogix-operator/v2/blob/main/config/samples/v1alpha1/rulegroups/mixed_rulegroup.yaml
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

func (in *RuleGroupSpec) ExtractCreateRuleGroupRequest() *rulegroups.RuleGroupsServiceCreateRuleGroupRequest {
	ruleMatchers := expandRuleMatchers(in.Applications, in.Subsystems, in.Severities)
	ruleSubGroups := expandRuleSubGroups(in.RuleSubgroups)
	order := expandOrder(in.Order)

	return &rulegroups.RuleGroupsServiceCreateRuleGroupRequest{
		Name:          rulegroups.PtrString(in.Name),
		Description:   rulegroups.PtrString(in.Description),
		Enabled:       rulegroups.PtrBool(in.Active),
		Hidden:        rulegroups.PtrBool(in.Hidden),
		Creator:       rulegroups.PtrString(in.Creator),
		RuleMatchers:  ruleMatchers,
		RuleSubgroups: ruleSubGroups,
		Order:         order,
	}
}

func expandOrder(order *int32) *int64 {
	if order != nil {
		return rulegroups.PtrInt64(int64(*order))
	}
	return nil
}

func expandRuleSubGroups(subGroups []RuleSubGroup) []rulegroups.CreateRuleGroupRequestCreateRuleSubgroup {
	ruleSubGroups := make([]rulegroups.CreateRuleGroupRequestCreateRuleSubgroup, 0, len(subGroups))
	for i, subGroup := range subGroups {
		rsg := expandRuleSubGroup(subGroup)
		rsg.Order = rulegroups.PtrInt64(int64(i + 1))
		ruleSubGroups = append(ruleSubGroups, *rsg)
	}
	return ruleSubGroups
}

func expandRuleSubGroup(subGroup RuleSubGroup) *rulegroups.CreateRuleGroupRequestCreateRuleSubgroup {
	rules := expandRules(subGroup.Rules)
	return &rulegroups.CreateRuleGroupRequestCreateRuleSubgroup{
		Enabled: rulegroups.PtrBool(subGroup.Active),
		Rules:   rules,
	}
}

func expandRules(rules []Rule) []rulegroups.CreateRuleGroupRequestCreateRuleSubgroupCreateRule {
	expandedRules := make([]rulegroups.CreateRuleGroupRequestCreateRuleSubgroupCreateRule, 0, len(rules))
	for i, rule := range rules {
		r := expandRule(rule)
		r.Order = rulegroups.PtrInt64(int64(i + 1))
		expandedRules = append(expandedRules, *r)
	}
	return expandedRules
}

func expandRule(rule Rule) *rulegroups.CreateRuleGroupRequestCreateRuleSubgroupCreateRule {
	sourceFiled, parameters := expandSourceFiledAndParameters(rule)

	return &rulegroups.CreateRuleGroupRequestCreateRuleSubgroupCreateRule{
		Name:        rulegroups.PtrString(rule.Name),
		Description: rulegroups.PtrString(rule.Description),
		SourceField: rulegroups.PtrString(sourceFiled),
		Parameters:  parameters,
		Enabled:     rulegroups.PtrBool(rule.Active),
	}
}

func expandSourceFiledAndParameters(rule Rule) (sourceField string, parameters *rulegroups.RuleParameters) {
	if parse := rule.Parse; parse != nil {
		sourceField = parse.SourceField
		parameters = &rulegroups.RuleParameters{
			RuleParametersParseParameters: &rulegroups.RuleParametersParseParameters{
				ParseParameters: &rulegroups.ParseParameters{
					DestinationField: rulegroups.PtrString(parse.DestinationField),
					Rule:             rulegroups.PtrString(parse.Regex),
				},
			},
		}
	} else if parseJsonField := rule.ParseJsonField; parseJsonField != nil {
		sourceField = parseJsonField.SourceField
		parameters = &rulegroups.RuleParameters{
			RuleParametersJsonParseParameters: &rulegroups.RuleParametersJsonParseParameters{
				JsonParseParameters: &rulegroups.JsonParseParameters{
					DestinationField: rulegroups.PtrString(parseJsonField.DestinationField),
					DeleteSource:     rulegroups.PtrBool(!parseJsonField.KeepSourceField),
					OverrideDest:     rulegroups.PtrBool(!parseJsonField.KeepDestinationField),
					EscapedValue:     rulegroups.PtrBool(true),
				},
			},
		}
	} else if jsonStringify := rule.JsonStringify; jsonStringify != nil {
		sourceField = jsonStringify.SourceField
		parameters = &rulegroups.RuleParameters{
			RuleParametersJsonStringifyParameters: &rulegroups.RuleParametersJsonStringifyParameters{
				JsonStringifyParameters: &rulegroups.JsonStringifyParameters{
					DestinationField: rulegroups.PtrString(jsonStringify.DestinationField),
					DeleteSource:     rulegroups.PtrBool(!jsonStringify.KeepSourceField),
				},
			},
		}
	} else if jsonExtract := rule.JsonExtract; jsonExtract != nil {
		sourceField = "text"
		destinationField := RulesSchemaDestinationFieldToOpenAPISeverityDestinationField[jsonExtract.DestinationField]
		jsonKey := rulegroups.PtrString(jsonExtract.JsonKey)
		parameters = &rulegroups.RuleParameters{
			RuleParametersJsonExtractParameters: &rulegroups.RuleParametersJsonExtractParameters{
				JsonExtractParameters: &rulegroups.JsonExtractParameters{
					DestinationFieldType: destinationField.Ptr(),
					Rule:                 jsonKey,
				},
			},
		}
	} else if removeFields := rule.RemoveFields; removeFields != nil {
		sourceField = "text"
		parameters = &rulegroups.RuleParameters{
			RuleParametersRemoveFieldsParameters: &rulegroups.RuleParametersRemoveFieldsParameters{
				RemoveFieldsParameters: &rulegroups.RemoveFieldsParameters{
					Fields: removeFields.ExcludedFields,
				},
			},
		}
	} else if extractTimestamp := rule.ExtractTimestamp; extractTimestamp != nil {
		sourceField = extractTimestamp.SourceField
		standard := RulesSchemaFormatStandardToOpenAPIFormatStandard[extractTimestamp.FieldFormatStandard]
		format := rulegroups.PtrString(extractTimestamp.TimeFormat)
		parameters = &rulegroups.RuleParameters{
			RuleParametersExtractTimestampParameters: &rulegroups.RuleParametersExtractTimestampParameters{
				ExtractTimestampParameters: &rulegroups.ExtractTimestampParameters{
					Standard: standard.Ptr(),
					Format:   format,
				},
			},
		}
	} else if block := rule.Block; block != nil {
		sourceField = block.SourceField
		if block.BlockingAllMatchingBlocks {
			parameters = &rulegroups.RuleParameters{
				RuleParametersBlockParameters: &rulegroups.RuleParametersBlockParameters{
					BlockParameters: &rulegroups.BlockParameters{
						KeepBlockedLogs: rulegroups.PtrBool(block.KeepBlockedLogs),
						Rule:            rulegroups.PtrString(block.Regex),
					},
				},
			}
		} else {
			parameters = &rulegroups.RuleParameters{
				RuleParametersAllowParameters: &rulegroups.RuleParametersAllowParameters{
					AllowParameters: &rulegroups.AllowParameters{
						KeepBlockedLogs: rulegroups.PtrBool(block.KeepBlockedLogs),
						Rule:            rulegroups.PtrString(block.Regex),
					},
				},
			}
		}
	} else if replace := rule.Replace; replace != nil {
		sourceField = replace.SourceField
		parameters = &rulegroups.RuleParameters{
			RuleParametersReplaceParameters: &rulegroups.RuleParametersReplaceParameters{
				ReplaceParameters: &rulegroups.ReplaceParameters{
					DestinationField: rulegroups.PtrString(replace.DestinationField),
					ReplaceNewVal:    rulegroups.PtrString(replace.ReplacementString),
					Rule:             rulegroups.PtrString(replace.Regex),
				},
			},
		}
	} else if extract := rule.Extract; extract != nil {
		sourceField = extract.SourceField
		parameters = &rulegroups.RuleParameters{
			RuleParametersExtractParameters: &rulegroups.RuleParametersExtractParameters{
				ExtractParameters: &rulegroups.ExtractParameters{
					Rule: rulegroups.PtrString(extract.Regex),
				},
			},
		}
	}

	return sourceField, parameters
}

func expandRuleMatchers(applications, subsystems []string, severities []RuleSeverity) []rulegroups.RuleMatcher {
	ruleMatchers := make([]rulegroups.RuleMatcher, 0, len(applications)+len(subsystems)+len(severities))

	for _, app := range applications {
		constraintStr := rulegroups.PtrString(app)
		applicationNameConstraint := rulegroups.ApplicationNameConstraint{Value: constraintStr}
		ruleMatcherApplicationName := rulegroups.RuleMatcherApplicationName{ApplicationName: &applicationNameConstraint}
		ruleMatchers = append(ruleMatchers, rulegroups.RuleMatcher{RuleMatcherApplicationName: &ruleMatcherApplicationName})
	}

	for _, subSys := range subsystems {
		constraintStr := rulegroups.PtrString(subSys)
		subsystemNameConstraint := rulegroups.SubsystemNameConstraint{Value: constraintStr}
		ruleMatcherApplicationName := rulegroups.RuleMatcherSubsystemName{SubsystemName: &subsystemNameConstraint}
		ruleMatchers = append(ruleMatchers, rulegroups.RuleMatcher{RuleMatcherSubsystemName: &ruleMatcherApplicationName})
	}

	for _, sev := range severities {
		constraintEnum := RulesSchemaSeverityToOpenAPISeverity[sev]
		severityConstraint := rulegroups.SeverityConstraint{Value: constraintEnum.Ptr()}
		ruleMatcherSeverity := rulegroups.RuleMatcherSeverity{Severity: &severityConstraint}
		ruleMatchers = append(ruleMatchers, rulegroups.RuleMatcher{RuleMatcherSeverity: &ruleMatcherSeverity})
	}

	return ruleMatchers
}

// RuleGroupStatus defines the observed state of RuleGroup
type RuleGroupStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (r *RuleGroup) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *RuleGroup) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

func (r *RuleGroup) GetPrintableStatus() string {
	return r.Status.PrintableStatus
}

func (r *RuleGroup) SetPrintableStatus(printableStatus string) {
	r.Status.PrintableStatus = printableStatus
}

func (r *RuleGroup) HasIDInStatus() bool {
	return r.Status.ID != nil && *r.Status.ID != ""
}

//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
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

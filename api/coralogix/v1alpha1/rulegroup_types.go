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
	"fmt"

	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api"
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
	RulesProtoSeverityDestinationFieldToSchemaDestinationField = utils.ReverseMap(RulesSchemaDestinationFieldToProtoSeverityDestinationField)
	RulesSchemaFormatStandardToProtoFormatStandard             = map[FieldFormatStandard]cxsdk.ExtractTimestampParametersFormatStandard{
		FieldFormatStandardStrftime: cxsdk.ExtractTimestampParametersFormatStandardStrftimeOrUnspecified,
		FieldFormatStandardJavaSDF:  cxsdk.ExtractTimestampParametersFormatStandardJavasdf,
		FieldFormatStandardGolang:   cxsdk.ExtractTimestampParametersFormatStandardGolang,
		FieldFormatStandardSecondTS: cxsdk.ExtractTimestampParametersFormatStandardSecondsTS,
		FieldFormatStandardMilliTS:  cxsdk.ExtractTimestampParametersFormatStandardMilliTS,
		FieldFormatStandardMicroTS:  cxsdk.ExtractTimestampParametersFormatStandardMicroTS,
		FieldFormatStandardNanoTS:   cxsdk.ExtractTimestampParametersFormatStandardNanoTS,
	}
	RulesProtoFormatStandardToSchemaFormatStandard = utils.ReverseMap(RulesSchemaFormatStandardToProtoFormatStandard)
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Rule struct {
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// +optional
	Description string `json:"description,omitempty"`

	//+kubebuilder:default=true
	Active bool `json:"active,omitempty"`

	// +optional
	Parse *Parse `json:"parse,omitempty"`

	// +optional
	Block *Block `json:"block,omitempty"`

	// +optional
	JsonExtract *JsonExtract `json:"jsonExtract,omitempty"`

	// +optional
	Replace *Replace `json:"replace,omitempty"`

	// +optional
	ExtractTimestamp *ExtractTimestamp `json:"extractTimestamp,omitempty"`

	// +optional
	RemoveFields *RemoveFields `json:"removeFields,omitempty"`

	// +optional
	JsonStringify *JsonStringify `json:"jsonStringify,omitempty"`

	// +optional
	Extract *Extract `json:"extract,omitempty"`

	// +optional
	ParseJsonField *ParseJsonField `json:"parseJsonField,omitempty"`
}

func (in *Rule) DeepEqual(actualRule Rule) (bool, utils.Diff) {
	if actualActive := actualRule.Active; in.Active != actualActive {
		return false, utils.Diff{
			Name:    "Active",
			Desired: in.Active,
			Actual:  actualActive,
		}
	}

	if actualDescription := actualRule.Description; in.Description != actualDescription {
		return false, utils.Diff{
			Name:    "Description",
			Desired: in.Description,
			Actual:  actualDescription,
		}
	}

	if actualName := actualRule.Name; in.Name != actualName {
		return false, utils.Diff{
			Name:    "Name",
			Desired: in.Name,
			Actual:  actualName,
		}
	}

	return in.DeepEqualRuleType(actualRule)
}

func (in *Rule) DeepEqualRuleType(rule Rule) (bool, utils.Diff) {
	if extract, actualExtract := in.Extract, rule.Extract; extract != nil {
		if actualExtract == nil {
			return false, utils.Diff{
				Name:    "Extract",
				Desired: *extract,
				Actual:  actualExtract,
			}
		} else if equal, diff := extract.DeepEqual(*actualExtract); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("Extract.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	if extractTimestamp, actualExtractTimestamp := in.ExtractTimestamp, rule.ExtractTimestamp; extractTimestamp != nil {
		if actualExtractTimestamp == nil {
			return false, utils.Diff{
				Name:    "ExtractTimestamp",
				Desired: *extractTimestamp,
				Actual:  actualExtractTimestamp,
			}
		} else if equal, diff := extractTimestamp.DeepEqual(*actualExtractTimestamp); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("ExtractTimestamp.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	if jsonExtract, actualJsonExtract := in.JsonExtract, rule.JsonExtract; jsonExtract != nil {
		if actualJsonExtract == nil {
			return false, utils.Diff{
				Name:    "JsonExtract",
				Desired: *jsonExtract,
				Actual:  actualJsonExtract,
			}
		} else if equal, diff := jsonExtract.DeepEqual(*actualJsonExtract); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("JsonExtract.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	if parseJsonField, actualParseJsonField := in.ParseJsonField, rule.ParseJsonField; parseJsonField != nil {
		if parseJsonField == nil {
			return false, utils.Diff{
				Name:    "ParseJsonField",
				Desired: *parseJsonField,
				Actual:  actualParseJsonField,
			}
		} else if equal, diff := parseJsonField.DeepEqual(*actualParseJsonField); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("ParseJsonField.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	if parse, actualParse := in.Parse, rule.Parse; parse != nil {
		if actualParse == nil {
			return false, utils.Diff{
				Name:    "Parse",
				Desired: *parse,
				Actual:  actualParse,
			}
		} else if equal, diff := parse.DeepEqual(*actualParse); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("Parse.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	if block, actualBlock := in.Block, rule.Block; block != nil {
		if actualBlock == nil {
			return false, utils.Diff{
				Name:    "Block",
				Desired: *block,
				Actual:  actualBlock,
			}
		} else if equal, diff := block.DeepEqual(*actualBlock); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("Block.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	if jsonStringify, actualJsonStringify := in.JsonStringify, rule.JsonStringify; jsonStringify != nil {
		if jsonStringify == nil {
			return false, utils.Diff{
				Name:    "JsonStringify",
				Desired: *jsonStringify,
				Actual:  actualJsonStringify,
			}
		} else if equal, diff := jsonStringify.DeepEqual(*actualJsonStringify); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("JsonStringify.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	if removeFields, actualRemoveFields := in.RemoveFields, rule.RemoveFields; removeFields != nil {
		if actualRemoveFields == nil {
			return false, utils.Diff{
				Name:    "RemoveFields",
				Desired: *removeFields,
				Actual:  actualRemoveFields,
			}
		} else if equal, diff := removeFields.DeepEqual(*actualRemoveFields); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("RemoveFields.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	if replace, actualReplace := in.Replace, rule.Replace; replace != nil {
		if actualReplace == nil {
			return false, utils.Diff{
				Name:    "Replace",
				Desired: *replace,
				Actual:  actualReplace,
			}
		} else if equal, diff := replace.DeepEqual(*actualReplace); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("Replace.%s", diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	return true, utils.Diff{}
}

type Parse struct {
	SourceField string `json:"sourceField"`

	DestinationField string `json:"destinationField"`

	Regex string `json:"regex"`
}

func (in *Parse) DeepEqual(parse Parse) (bool, utils.Diff) {
	if regex, actualRegex := in.Regex, parse.Regex; regex != actualRegex {
		return false, utils.Diff{
			Name:    "Regex",
			Desired: regex,
			Actual:  actualRegex,
		}
	}

	if sourceField, actualSourceField := in.SourceField, parse.SourceField; sourceField != actualSourceField {
		return false, utils.Diff{
			Name:    "SourceField",
			Desired: sourceField,
			Actual:  actualSourceField,
		}
	}

	if destinationField, actualDestinationField := in.DestinationField, parse.DestinationField; destinationField != actualDestinationField {
		return false, utils.Diff{
			Name:    "DestinationField",
			Desired: destinationField,
			Actual:  actualDestinationField,
		}
	}

	return true, utils.Diff{}
}

type Block struct {
	SourceField string `json:"sourceField"`

	Regex string `json:"regex"`

	//+kubebuilder:default=false
	KeepBlockedLogs bool `json:"keepBlockedLogs,omitempty"`

	//+kubebuilder:default=true
	BlockingAllMatchingBlocks bool `json:"blockingAllMatchingBlocks,omitempty"`
}

func (in *Block) DeepEqual(block Block) (bool, utils.Diff) {
	if keepBlockedLogs, actualKeepBlockedLogs := in.KeepBlockedLogs, block.KeepBlockedLogs; keepBlockedLogs != actualKeepBlockedLogs {
		return false, utils.Diff{
			Name:    "KeepBlockedLogs",
			Desired: keepBlockedLogs,
			Actual:  actualKeepBlockedLogs,
		}
	}

	if blockingAllMatchingBlocks, actualBlockingAllMatchingBlocks := in.BlockingAllMatchingBlocks, block.BlockingAllMatchingBlocks; blockingAllMatchingBlocks != actualBlockingAllMatchingBlocks {
		return false, utils.Diff{
			Name:    "BlockingAllMatchingBlocks",
			Desired: blockingAllMatchingBlocks,
			Actual:  actualBlockingAllMatchingBlocks,
		}
	}

	if regex, actualRegex := in.Regex, block.Regex; regex != actualRegex {
		return false, utils.Diff{
			Name:    "Regex",
			Desired: regex,
			Actual:  actualRegex,
		}
	}

	if sourceField, actualSourceField := in.SourceField, block.SourceField; sourceField != actualSourceField {
		return false, utils.Diff{
			Name:    "SourceField",
			Desired: sourceField,
			Actual:  actualSourceField,
		}
	}

	return true, utils.Diff{}
}

// +kubebuilder:validation:Enum=Category;CLASSNAME;METHODNAME;THREADID;SEVERITY
type DestinationField string

const (
	DestinationFieldCategory     DestinationField = "Category"
	DestinationFieldClassName    DestinationField = "CLASSNAME"
	DestinationFieldMethod       DestinationField = "METHODNAME"
	DestinationFieldThreadID     DestinationField = "THREADID"
	DestinationFieldRuleSeverity DestinationField = "SEVERITY"
)

type JsonExtract struct {
	DestinationField DestinationField `json:"destinationField"`

	JsonKey string `json:"jsonKey"`
}

func (in *JsonExtract) DeepEqual(jsonExtract JsonExtract) (bool, utils.Diff) {
	if destinationField, actualDestinationField := in.DestinationField, jsonExtract.DestinationField; destinationField != actualDestinationField {
		return false, utils.Diff{
			Name:    "DestinationField",
			Desired: destinationField,
			Actual:  actualDestinationField,
		}
	}

	if jsonKey, actualJsonKey := in.JsonKey, jsonExtract.JsonKey; jsonKey != actualJsonKey {
		return false, utils.Diff{
			Name:    "JsonKey",
			Desired: jsonKey,
			Actual:  actualJsonKey,
		}
	}

	return true, utils.Diff{}
}

type Replace struct {
	SourceField string `json:"sourceField"`

	DestinationField string `json:"destinationField"`

	Regex string `json:"regex"`

	ReplacementString string `json:"replacementString"`
}

func (in *Replace) DeepEqual(replace Replace) (bool, utils.Diff) {
	if regex, actualRegex := in.Regex, replace.Regex; regex != actualRegex {
		return false, utils.Diff{
			Name:    "Regex",
			Desired: regex,
			Actual:  actualRegex,
		}
	}

	if sourceField, actualSourceField := in.SourceField, replace.SourceField; sourceField != actualSourceField {
		return false, utils.Diff{
			Name:    "SourceField",
			Desired: sourceField,
			Actual:  actualSourceField,
		}
	}

	if destinationField, actualDestinationField := in.DestinationField, replace.DestinationField; destinationField != actualDestinationField {
		return false, utils.Diff{
			Name:    "DestinationField",
			Desired: destinationField,
			Actual:  actualDestinationField,
		}
	}

	if replacementString, actualReplacementString := in.ReplacementString, replace.ReplacementString; replacementString != actualReplacementString {
		return false, utils.Diff{
			Name:    "ReplacementString",
			Desired: replacementString,
			Actual:  actualReplacementString,
		}
	}

	return true, utils.Diff{}
}

// +kubebuilder:validation:Enum=Strftime;JavaSDF;Golang;SecondTS;MilliTS;MicroTS;NanoTS
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

type ExtractTimestamp struct {
	SourceField string `json:"sourceField"`

	FieldFormatStandard FieldFormatStandard `json:"fieldFormatStandard"`

	TimeFormat string `json:"timeFormat"`
}

func (in *ExtractTimestamp) DeepEqual(extractTimestamp ExtractTimestamp) (bool, utils.Diff) {
	if timeFormat, actualTimeFormat := extractTimestamp.TimeFormat, extractTimestamp.TimeFormat; timeFormat != actualTimeFormat {
		return false, utils.Diff{
			Name:    "TimeFormat",
			Desired: timeFormat,
			Actual:  actualTimeFormat,
		}
	}

	if fieldFormatStandard, actualFieldFormatStandard := extractTimestamp.FieldFormatStandard, extractTimestamp.FieldFormatStandard; fieldFormatStandard != actualFieldFormatStandard {
		return false, utils.Diff{
			Name:    "FieldFormatStandard",
			Desired: fieldFormatStandard,
			Actual:  actualFieldFormatStandard,
		}
	}

	if sourceField, actualTSourceField := extractTimestamp.SourceField, extractTimestamp.SourceField; sourceField != actualTSourceField {
		return false, utils.Diff{
			Name:    "SourceField",
			Desired: sourceField,
			Actual:  actualTSourceField,
		}
	}

	return true, utils.Diff{}
}

type RemoveFields struct {
	ExcludedFields []string `json:"excludedFields"`
}

func (in *RemoveFields) DeepEqual(removeFields RemoveFields) (bool, utils.Diff) {
	if excludedFields, actualExcludedFields := in.ExcludedFields, removeFields.ExcludedFields; !utils.SlicesWithUniqueValuesEqual(excludedFields, actualExcludedFields) {
		return false, utils.Diff{
			Name:    "ExcludedFields",
			Desired: excludedFields,
			Actual:  actualExcludedFields,
		}
	}

	return true, utils.Diff{}
}

type JsonStringify struct {
	SourceField string `json:"sourceField"`

	DestinationField string `json:"destinationField"`

	//+kubebuilder:default=false
	KeepSourceField bool `json:"keepSourceField,omitempty"`
}

func (in *JsonStringify) DeepEqual(jsonStringify JsonStringify) (bool, utils.Diff) {
	if keepSourceField, actualKeepSourceField := in.KeepSourceField, jsonStringify.KeepSourceField; keepSourceField != actualKeepSourceField {
		return false, utils.Diff{
			Name:    "KeepSourceField",
			Desired: keepSourceField,
			Actual:  actualKeepSourceField,
		}
	}

	if sourceField, actualSourceField := in.SourceField, jsonStringify.SourceField; sourceField != actualSourceField {
		return false, utils.Diff{
			Name:    "SourceField",
			Desired: sourceField,
			Actual:  actualSourceField,
		}
	}

	if destinationField, actualDestinationField := in.DestinationField, jsonStringify.DestinationField; destinationField != actualDestinationField {
		return false, utils.Diff{
			Name:    "DestinationField",
			Desired: destinationField,
			Actual:  actualDestinationField,
		}
	}

	return true, utils.Diff{}

}

type Extract struct {
	SourceField string `json:"sourceField"`

	Regex string `json:"regex"`
}

func (in *Extract) DeepEqual(extract Extract) (bool, utils.Diff) {
	if regex, actualRegex := in.Regex, extract.Regex; regex != actualRegex {
		return false, utils.Diff{
			Name:    "Regex",
			Desired: regex,
			Actual:  actualRegex,
		}
	}

	if sourceField, actualSourceField := in.SourceField, extract.SourceField; sourceField != actualSourceField {
		return false, utils.Diff{
			Name:    "SourceField",
			Desired: sourceField,
			Actual:  actualSourceField,
		}
	}

	return true, utils.Diff{}
}

type ParseJsonField struct {
	SourceField string `json:"sourceField"`

	DestinationField string `json:"destinationField"`

	KeepSourceField bool `json:"keepSourceField"`

	KeepDestinationField bool `json:"keepDestinationField"`
}

func (in *ParseJsonField) DeepEqual(field ParseJsonField) (bool, utils.Diff) {
	if keepDestinationField, actualKeepDestinationField := in.KeepDestinationField, field.KeepDestinationField; keepDestinationField != actualKeepDestinationField {
		return false, utils.Diff{
			Name:    "KeepDestinationField",
			Desired: keepDestinationField,
			Actual:  actualKeepDestinationField,
		}
	}

	if keepSourceField, actualKeepSourceField := in.KeepSourceField, field.KeepSourceField; keepSourceField != actualKeepSourceField {
		return false, utils.Diff{
			Name:    "KeepSourceField",
			Desired: keepSourceField,
			Actual:  actualKeepSourceField,
		}
	}

	if destinationField, actualDestinationField := in.DestinationField, field.DestinationField; destinationField != actualDestinationField {
		return false, utils.Diff{
			Name:    "DestinationField",
			Desired: destinationField,
			Actual:  actualDestinationField,
		}
	}

	if sourceField, actualSourceField := in.SourceField, field.SourceField; sourceField != actualSourceField {
		return false, utils.Diff{
			Name:    "SourceField",
			Desired: sourceField,
			Actual:  actualSourceField,
		}
	}

	return true, utils.Diff{}
}

type RuleSubGroup struct {
	// +optional
	ID *string `json:"id,omitempty"`

	//+kubebuilder:default=true
	Active bool `json:"active,omitempty"`

	// +optional
	Order *int32 `json:"order,omitempty"`

	// +optional
	Rules []Rule `json:"rules,omitempty"`
}

func (in *RuleSubGroup) DeepEqual(actualSubgroup RuleSubGroup) (bool, utils.Diff) {
	if actualActive := actualSubgroup.Active; in.Active != actualActive {
		return false, utils.Diff{
			Name:    "Active",
			Desired: in.Active,
			Actual:  actualActive,
		}
	}

	if len(in.Rules) != len(actualSubgroup.Rules) {
		return false, utils.Diff{
			Name:    "Rules.length",
			Desired: len(in.Rules),
			Actual:  len(actualSubgroup.Rules),
		}
	}

	for i := range in.Rules {
		if equal, diff := in.Rules[i].DeepEqual(actualSubgroup.Rules[i]); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("Rules[%d].%s", i, diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	return true, utils.Diff{}
}

// RuleGroupSpec defines the Desired state of RuleGroup
type RuleGroupSpec struct {
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// +optional
	Description string `json:"description,omitempty"`

	//+kubebuilder:default=true
	Active bool `json:"active,omitempty"`

	// +optional
	Applications []string `json:"applications,omitempty"`

	// +optional
	Subsystems []string `json:"subsystems,omitempty"`

	// +optional
	Severities []RuleSeverity `json:"severities,omitempty"`

	//+kubebuilder:default=false
	Hidden bool `json:"hidden,omitempty"`

	// +optional
	Creator string `json:"creator,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum:=1
	Order *int32 `json:"order,omitempty"`

	// +optional
	RuleSubgroups []RuleSubGroup `json:"subgroups,omitempty"`
}

// +kubebuilder:validation:Enum=Debug;Verbose;Info;Warning;Error;Critical
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
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ID *string `json:"id"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// RuleGroup is the Schema for the rulegroups API
type RuleGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuleGroupSpec   `json:"spec,omitempty"`
	Status RuleGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RuleGroupList contains a list of RuleGroup
type RuleGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RuleGroup{}, &RuleGroupList{})
}

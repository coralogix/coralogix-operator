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
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
)

// Events2MetricSpec defines the desired state of Events2Metric.
type Events2MetricSpec struct {
	// Name of the E2M
	Name string `json:"name"`
	// Description of the E2M
	// +optional
	Description *string `json:"description,omitempty"`
	// Represents the limit of the permutations
	// +optional
	PermutationsLimit *int32 `json:"permutationsLimit,omitempty"`
	// E2M metric labels
	// +optional
	MetricLabels []MetricLabel `json:"metricLabels,omitempty"`
	// E2M metric fields
	// +optional
	MetricFields []MetricField `json:"metricFields,omitempty"`
	// Spans or logs type query
	Query E2MQuery `json:"query"`
}

type MetricLabel struct {
	// Metric label target alias name
	TargetLabel string `json:"targetLabel"`
	// Metric label source field
	SourceField string `json:"sourceField"`
}

type MetricField struct {
	// Target metric field alias name
	TargetBaseMetricName string `json:"targetBaseMetricName"`
	// Source field
	SourceField string `json:"sourceField"`
	// Represents Aggregation type list
	// +optional
	Aggregations []MetricFieldAggregation `json:"aggregations,omitempty"`
}

type MetricFieldAggregation struct {
	// Is enabled. True by default
	// +default=true
	Enabled bool `json:"enabled"`
	// Aggregation type
	AggType AggregationType `json:"aggType"`
	// Target metric field alias name
	TargetMetricName string `json:"targetMetricName"`
	// Aggregate metadata, samples or histogram type
	// Types that are valid to be assigned to AggMetadata: AggregationTypeSamples, AggregationTypeHistogram
	AggMetadata AggregationMetadata `json:"aggMetadata"`
}

// AggregationType defines the type of aggregation to be performed.
// +kubebuilder:validation:Enum=min;max;count;avg;sum;histogram;samples
type AggregationType string

const (
	// AggregationTypeMin represents the minimum value aggregation.
	AggregationTypeMin AggregationType = "min"
	// AggregationTypeMax represents the maximum value aggregation.
	AggregationTypeMax AggregationType = "max"
	// AggregationTypeCount represents the count aggregation.
	AggregationTypeCount AggregationType = "count"
	// AggregationTypeAvg represents the average value aggregation.
	AggregationTypeAvg AggregationType = "avg"
	// AggregationTypeSum represents the sum aggregation.
	AggregationTypeSum AggregationType = "sum"
	// AggregationTypeHistogram represents the histogram aggregation.
	AggregationTypeHistogram AggregationType = "histogram"
	// AggregationTypeSamples represents the samples aggregation.
	AggregationTypeSamples AggregationType = "samples"
)

var AggregationTypeSchemaToProto = map[AggregationType]cxsdk.E2MAggregationType{
	AggregationTypeMin:       cxsdk.E2MAggregationTypeMin,
	AggregationTypeMax:       cxsdk.E2MAggregationTypeMax,
	AggregationTypeCount:     cxsdk.E2MAggregationTypeCount,
	AggregationTypeAvg:       cxsdk.E2MAggregationTypeAvg,
	AggregationTypeSum:       cxsdk.E2MAggregationTypeSum,
	AggregationTypeHistogram: cxsdk.E2MAggregationTypeHistogram,
	AggregationTypeSamples:   cxsdk.E2MAggregationTypeSamples,
}

// AggregationMetadata defines the metadata for aggregation.
// +kubebuilder:validation:XValidation:rule="has(self.samples) != has(self.histogram)",message="Exactly one of samples or histogram must be set"
type AggregationMetadata struct {
	// E2M sample type metadata
	// +optional
	Samples *SamplesMetadata `json:"samples,omitempty"`
	// E2M aggregate histogram type metadata
	// +optional
	Histogram *HistogramMetadata `json:"histogram,omitempty"`
}

// SamplesMetadata - E2M aggregate sample type
type SamplesMetadata struct {
	SampleType E2MAggSampleType `json:"sampleType"`
}

// E2MAggSamplesSampleType defines the type of sample aggregation to be performed.
// +kubebuilder:validation:Enum=min;max
type E2MAggSampleType string

const (
	E2MAggSamplesSampleTypeMin E2MAggSampleType = "min"
	E2MAggSamplesSampleTypeMax E2MAggSampleType = "max"
)

var E2MAggSamplesSampleTypeSchemaToProto = map[E2MAggSampleType]cxsdk.E2MAggSampleType{
	E2MAggSamplesSampleTypeMin: cxsdk.E2MAggSampleTypeMin,
	E2MAggSamplesSampleTypeMax: cxsdk.E2MAggSampleTypeMax,
}

// HistogramMetadata defines the metadata for histogram aggregation.
type HistogramMetadata struct {
	// Buckets of the E2M
	Buckets []resource.Quantity `json:"buckets"`
}

// E2MQuerySpans defines the query for spans2metrics E2M.
// +kubebuilder:validation:XValidation:rule="has(self.spans) != has(self.logs)",message="Exactly one of spans or logs must be set"
type E2MQuery struct {
	// Spans query for spans2metrics E2M
	// +optional
	Spans *E2MQuerySpans `json:"spans,omitempty"`
	// Logs query for logs2metrics E2M
	// +optional
	Logs *E2MQueryLogs `json:"logs,omitempty"`
}

// E2MQuerySpans defines the query for spans2metrics E2M.
type E2MQuerySpans struct {
	// lucene query
	// +optional
	Lucene *string `json:"lucene,omitempty"`
	// application name filters
	// +optional
	ApplicationNameFilters []string `json:"applicationNameFilters,omitempty"`
	// subsystem name filters
	// +optional
	SubsystemNameFilters []string `json:"subsystemNameFilters,omitempty"`
	// action filters
	// +optional
	ActionFilters []string `json:"actionFilters,omitempty"`
	// service filters
	// +optional
	ServiceFilters []string `json:"serviceFilters,omitempty"`
}

// E2MQueryLogs defines the query for logs2metrics E2M.
type E2MQueryLogs struct {
	// lucene query
	// +optional
	Lucene *string `json:"lucene,omitempty"`
	// alias
	// +optional
	Alias *string `json:"alias,omitempty"`
	// application name filters
	// +optional
	ApplicationNameFilters []string `json:"applicationNameFilters,omitempty"`
	// subsystem names filters
	// +optional
	SubsystemNameFilters []string `json:"subsystemNameFilters,omitempty"`
	// severity type filters
	// +optional
	SeverityFilters []L2MSeverity `json:"severityFilters,omitempty"`
}

// L2MSeverity defines the severity type for logs2metrics E2M.
// +kubebuilder:validation:Enum=debug;verbose;info;warn;error;critical
type L2MSeverity string

const (
	// L2MSeverityDebug represents the debug severity level.
	L2MSeverityDebug L2MSeverity = "debug"
	// L2MSeverityVerbose represents the verbose severity level.
	L2MSeverityVerbose L2MSeverity = "verbose"
	// L2MSeverityInfo represents the info severity level.
	L2MSeverityInfo L2MSeverity = "info"
	// L2MSeverityWarning represents the warning severity level.
	L2MSeverityWarning L2MSeverity = "warn"
	// L2MSeverityError represents the error severity level.
	L2MSeverityError L2MSeverity = "error"
	// L2MSeverityCritical represents the critical severity level.
	L2MSeverityCritical L2MSeverity = "critical"
)

var L2MSeveritySchemaToProto = map[L2MSeverity]cxsdk.L2MSeverity{
	L2MSeverityDebug:    cxsdk.L2MSeverityDebug,
	L2MSeverityVerbose:  cxsdk.L2MSeverityVerbose,
	L2MSeverityInfo:     cxsdk.L2MSeverityInfo,
	L2MSeverityWarning:  cxsdk.L2MSeverityWarning,
	L2MSeverityError:    cxsdk.L2MSeverityError,
	L2MSeverityCritical: cxsdk.L2MSeverityCritical,
}

// Events2MetricStatus defines the observed state of Events2Metric.
type Events2MetricStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (e2m *Events2Metric) GetConditions() []metav1.Condition {
	return e2m.Status.Conditions
}

func (e2m *Events2Metric) SetConditions(conditions []metav1.Condition) {
	e2m.Status.Conditions = conditions
}

func (e2m *Events2Metric) GetPrintableStatus() string {
	return e2m.Status.PrintableStatus
}

func (e2m *Events2Metric) SetPrintableStatus(printableStatus string) {
	e2m.Status.PrintableStatus = printableStatus
}

func (e2m *Events2Metric) HasIDInStatus() bool {
	return e2m.Status.Id != nil && *e2m.Status.Id != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// See also https://coralogix.com/docs/user-guides/monitoring-and-insights/events2metrics/
//
// **Added in v0.5.0**
// Events2Metric is the Schema for the events2metrics API.
type Events2Metric struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Events2MetricSpec   `json:"spec,omitempty"`
	Status Events2MetricStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Events2MetricList contains a list of Events2Metric.
type Events2MetricList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Events2Metric `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Events2Metric{}, &Events2MetricList{})
}

func (spec *Events2MetricSpec) ExtractCreateE2MRequest() *cxsdk.CreateE2MRequest {
	e2m := &cxsdk.E2MCreateParams{
		Name:              wrapperspb.String(spec.Name),
		Description:       utils.StringPointerToWrapperspbString(spec.Description),
		PermutationsLimit: utils.Int32PointerToWrapperspbInt32(spec.PermutationsLimit),
		MetricLabels:      extractE2mMetricLabels(spec.MetricLabels),
		MetricFields:      extractE2mMetricFields(spec.MetricFields),
	}
	e2m = expandE2MQuery(e2m, spec.Query)
	return &cxsdk.CreateE2MRequest{
		E2M: e2m,
	}
}

func (spec *Events2MetricSpec) ExtractReplaceE2MRequest() *cxsdk.ReplaceE2MRequest {
	e2m := &cxsdk.ReplaceE2MRequest{
		E2M: &cxsdk.E2M{
			Name:         wrapperspb.String(spec.Name),
			Description:  utils.StringPointerToWrapperspbString(spec.Description),
			Permutations: extractE2mPermutations(spec.PermutationsLimit),
			MetricLabels: extractE2mMetricLabels(spec.MetricLabels),
			MetricFields: extractE2mMetricFields(spec.MetricFields),
		},
	}
	e2m.E2M = expandUpdateE2MQuery(e2m.E2M, spec.Query)
	return e2m
}

func extractE2mPermutations(permutations *int32) *cxsdk.E2MPermutations {
	if permutations == nil {
		return nil
	}
	return &cxsdk.E2MPermutations{
		Limit: *permutations,
	}
}

func expandE2MQuery(e2m *cxsdk.E2MCreateParams, query E2MQuery) *cxsdk.E2MCreateParams {
	if spans := query.Spans; spans != nil {
		e2m.Query = &cxsdk.E2MCreateParamsSpansQuery{
			SpansQuery: &cxsdk.S2MSpansQuery{
				Lucene:                 utils.StringPointerToWrapperspbString(spans.Lucene),
				ApplicationnameFilters: utils.StringSliceToWrappedStringSlice(spans.ApplicationNameFilters),
				SubsystemnameFilters:   utils.StringSliceToWrappedStringSlice(spans.SubsystemNameFilters),
				ActionFilters:          utils.StringSliceToWrappedStringSlice(spans.ActionFilters),
				ServiceFilters:         utils.StringSliceToWrappedStringSlice(spans.ServiceFilters),
			},
		}
		e2m.Type = cxsdk.E2MTypeSpans2Metrics
	} else if logs := query.Logs; logs != nil {
		e2m.Query = &cxsdk.E2MCreateParamsLogsQuery{
			LogsQuery: &cxsdk.L2MLogsQuery{
				Lucene:                 utils.StringPointerToWrapperspbString(logs.Lucene),
				Alias:                  utils.StringPointerToWrapperspbString(logs.Alias),
				ApplicationnameFilters: utils.StringSliceToWrappedStringSlice(logs.ApplicationNameFilters),
				SubsystemnameFilters:   utils.StringSliceToWrappedStringSlice(logs.SubsystemNameFilters),
				SeverityFilters:        expandL2MSeverityFilters(logs.SeverityFilters),
			},
		}
		e2m.Type = cxsdk.E2MTypeLogs2Metrics
	}

	return e2m
}

func expandUpdateE2MQuery(e2m *cxsdk.E2M, query E2MQuery) *cxsdk.E2M {
	if spans := query.Spans; spans != nil {
		e2m.Query = &cxsdk.E2MSpansQuery{
			SpansQuery: &cxsdk.S2MSpansQuery{
				Lucene:                 utils.StringPointerToWrapperspbString(spans.Lucene),
				ApplicationnameFilters: utils.StringSliceToWrappedStringSlice(spans.ApplicationNameFilters),
				SubsystemnameFilters:   utils.StringSliceToWrappedStringSlice(spans.SubsystemNameFilters),
				ActionFilters:          utils.StringSliceToWrappedStringSlice(spans.ActionFilters),
				ServiceFilters:         utils.StringSliceToWrappedStringSlice(spans.ServiceFilters),
			},
		}
		e2m.Type = cxsdk.E2MTypeSpans2Metrics
	} else if logs := query.Logs; logs != nil {
		e2m.Query = &cxsdk.E2MLogsQuery{
			LogsQuery: &cxsdk.L2MLogsQuery{
				Lucene:                 utils.StringPointerToWrapperspbString(logs.Lucene),
				Alias:                  utils.StringPointerToWrapperspbString(logs.Alias),
				ApplicationnameFilters: utils.StringSliceToWrappedStringSlice(logs.ApplicationNameFilters),
				SubsystemnameFilters:   utils.StringSliceToWrappedStringSlice(logs.SubsystemNameFilters),
				SeverityFilters:        expandL2MSeverityFilters(logs.SeverityFilters),
			},
		}
		e2m.Type = cxsdk.E2MTypeLogs2Metrics
	}

	return e2m
}

func expandL2MSeverityFilters(severityFilters []L2MSeverity) []cxsdk.L2MSeverity {
	if severityFilters == nil {
		return nil
	}
	expanded := make([]cxsdk.L2MSeverity, 0, len(severityFilters))
	for _, severity := range severityFilters {
		if protoSeverity, ok := L2MSeveritySchemaToProto[severity]; ok {
			expanded = append(expanded, protoSeverity)
		}
	}
	return expanded
}

func extractE2mMetricLabels(labels []MetricLabel) []*cxsdk.MetricLabel {
	metricLabels := make([]*cxsdk.MetricLabel, 0, len(labels))
	for _, label := range labels {
		metricLabels = append(metricLabels, &cxsdk.MetricLabel{
			TargetLabel: wrapperspb.String(label.TargetLabel),
			SourceField: wrapperspb.String(label.SourceField),
		})
	}
	return metricLabels
}

func extractE2mMetricFields(fields []MetricField) []*cxsdk.MetricField {
	metricFields := make([]*cxsdk.MetricField, 0, len(fields))
	for _, field := range fields {
		metricField := &cxsdk.MetricField{
			TargetBaseMetricName: wrapperspb.String(field.TargetBaseMetricName),
			SourceField:          wrapperspb.String(field.SourceField),
			Aggregations:         extractE2mAggregations(field.Aggregations),
		}
		metricFields = append(metricFields, metricField)
	}
	return metricFields
}

func extractE2mAggregations(aggregations []MetricFieldAggregation) []*cxsdk.E2MAggregation {
	metricAggregations := make([]*cxsdk.E2MAggregation, 0, len(aggregations))
	for _, aggregation := range aggregations {
		metricAggregation := &cxsdk.E2MAggregation{
			Enabled:          aggregation.Enabled,
			AggType:          AggregationTypeSchemaToProto[aggregation.AggType],
			TargetMetricName: aggregation.TargetMetricName,
		}
		metricAggregation = expandE2MAggMetadata(metricAggregation, aggregation.AggMetadata)
		metricAggregations = append(metricAggregations, metricAggregation)
	}
	return metricAggregations
}

func expandE2MAggMetadata(metricAggregation *cxsdk.E2MAggregation, metadata AggregationMetadata) *cxsdk.E2MAggregation {
	if metadata.Samples != nil {
		metricAggregation.AggMetadata = &cxsdk.E2MAggregationSamples{
			Samples: &cxsdk.E2MAggSamples{
				SampleType: E2MAggSamplesSampleTypeSchemaToProto[metadata.Samples.SampleType],
			},
		}
	} else if metadata.Histogram != nil {
		metricAggregation.AggMetadata = &cxsdk.E2MAggregationHistogram{
			Histogram: &cxsdk.E2MAggHistogram{
				Buckets: utils.QuantitiesToFloats32(metadata.Histogram.Buckets),
			},
		}
	}

	return metricAggregation
}

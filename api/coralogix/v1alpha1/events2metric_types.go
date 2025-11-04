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

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	events2metrics "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/events2metrics_service"

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

var AggregationTypeSchemaToOpenAPI = map[AggregationType]events2metrics.AggType{
	AggregationTypeMin:       events2metrics.AGGTYPE_AGG_TYPE_MIN,
	AggregationTypeMax:       events2metrics.AGGTYPE_AGG_TYPE_MAX,
	AggregationTypeCount:     events2metrics.AGGTYPE_AGG_TYPE_COUNT,
	AggregationTypeAvg:       events2metrics.AGGTYPE_AGG_TYPE_AVG,
	AggregationTypeSum:       events2metrics.AGGTYPE_AGG_TYPE_SUM,
	AggregationTypeHistogram: events2metrics.AGGTYPE_AGG_TYPE_HISTOGRAM,
	AggregationTypeSamples:   events2metrics.AGGTYPE_AGG_TYPE_SAMPLES,
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

var E2MAggSamplesSampleTypeSchemaToOpenAPI = map[E2MAggSampleType]*events2metrics.SampleType{
	E2MAggSamplesSampleTypeMin: events2metrics.SAMPLETYPE_SAMPLE_TYPE_MIN.Ptr(),
	E2MAggSamplesSampleTypeMax: events2metrics.SAMPLETYPE_SAMPLE_TYPE_MAX.Ptr(),
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

var L2MSeveritySchemaToOpenAPI = map[L2MSeverity]events2metrics.Logs2metricsV2Severity{
	L2MSeverityDebug:    events2metrics.LOGS2METRICSV2SEVERITY_SEVERITY_DEBUG,
	L2MSeverityVerbose:  events2metrics.LOGS2METRICSV2SEVERITY_SEVERITY_VERBOSE,
	L2MSeverityInfo:     events2metrics.LOGS2METRICSV2SEVERITY_SEVERITY_INFO,
	L2MSeverityWarning:  events2metrics.LOGS2METRICSV2SEVERITY_SEVERITY_WARNING,
	L2MSeverityError:    events2metrics.LOGS2METRICSV2SEVERITY_SEVERITY_ERROR,
	L2MSeverityCritical: events2metrics.LOGS2METRICSV2SEVERITY_SEVERITY_CRITICAL,
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
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
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

func (spec *Events2MetricSpec) ExtractCreateE2MRequest() (*events2metrics.Events2MetricServiceCreateE2MRequest, error) {
	if spans := spec.Query.Spans; spans != nil {
		return &events2metrics.Events2MetricServiceCreateE2MRequest{
			E2MCreateParamsSpansQuery: &events2metrics.E2MCreateParamsSpansQuery{
				Name:              spec.Name,
				Description:       spec.Description,
				Type:              events2metrics.E2MTYPE_E2_M_TYPE_SPANS2_METRICS.Ptr(),
				PermutationsLimit: spec.PermutationsLimit,
				MetricLabels:      extractE2mMetricLabels(spec.MetricLabels),
				MetricFields:      extractE2mMetricFields(spec.MetricFields),
				SpansQuery: &events2metrics.V2SpansQuery{
					Lucene:                 spans.Lucene,
					ApplicationnameFilters: spans.ApplicationNameFilters,
					SubsystemnameFilters:   spans.SubsystemNameFilters,
					ActionFilters:          spans.ActionFilters,
					ServiceFilters:         spans.ServiceFilters,
				},
			},
		}, nil
	} else if logs := spec.Query.Logs; logs != nil {
		return &events2metrics.Events2MetricServiceCreateE2MRequest{
			E2MCreateParamsLogsQuery: &events2metrics.E2MCreateParamsLogsQuery{
				Name:              spec.Name,
				Description:       spec.Description,
				Type:              events2metrics.E2MTYPE_E2_M_TYPE_LOGS2_METRICS.Ptr(),
				PermutationsLimit: spec.PermutationsLimit,
				MetricLabels:      extractE2mMetricLabels(spec.MetricLabels),
				MetricFields:      extractE2mMetricFields(spec.MetricFields),
				LogsQuery: &events2metrics.V2LogsQuery{
					Lucene:                 logs.Lucene,
					Alias:                  logs.Alias,
					ApplicationnameFilters: logs.ApplicationNameFilters,
					SubsystemnameFilters:   logs.SubsystemNameFilters,
					SeverityFilters:        expandL2MSeverityFilters(logs.SeverityFilters),
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("neither spans nor logs query is defined")
}

func (spec *Events2MetricSpec) ExtractReplaceE2MRequest() (*events2metrics.Events2MetricServiceReplaceE2MRequest, error) {
	if spans := spec.Query.Spans; spans != nil {
		return &events2metrics.Events2MetricServiceReplaceE2MRequest{
			E2MSpansQuery: &events2metrics.E2MSpansQuery{
				Name:         spec.Name,
				Description:  spec.Description,
				Type:         events2metrics.E2MTYPE_E2_M_TYPE_SPANS2_METRICS,
				Permutations: extractE2mPermutations(spec.PermutationsLimit),
				MetricLabels: extractE2mMetricLabels(spec.MetricLabels),
				MetricFields: extractE2mMetricFields(spec.MetricFields),
				SpansQuery: &events2metrics.V2SpansQuery{
					Lucene:                 spans.Lucene,
					ApplicationnameFilters: spans.ApplicationNameFilters,
					SubsystemnameFilters:   spans.SubsystemNameFilters,
					ActionFilters:          spans.ActionFilters,
					ServiceFilters:         spans.ServiceFilters,
				},
			},
		}, nil
	} else if logs := spec.Query.Logs; logs != nil {
		return &events2metrics.Events2MetricServiceReplaceE2MRequest{
			E2MLogsQuery: &events2metrics.E2MLogsQuery{
				Name:         spec.Name,
				Description:  spec.Description,
				Type:         events2metrics.E2MTYPE_E2_M_TYPE_LOGS2_METRICS,
				Permutations: extractE2mPermutations(spec.PermutationsLimit),
				MetricLabels: extractE2mMetricLabels(spec.MetricLabels),
				MetricFields: extractE2mMetricFields(spec.MetricFields),
				LogsQuery: &events2metrics.V2LogsQuery{
					Lucene:                 logs.Lucene,
					Alias:                  logs.Alias,
					ApplicationnameFilters: logs.ApplicationNameFilters,
					SubsystemnameFilters:   logs.SubsystemNameFilters,
					SeverityFilters:        expandL2MSeverityFilters(logs.SeverityFilters),
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("neither spans nor logs query is defined")
}

func extractE2mPermutations(permutations *int32) *events2metrics.E2MPermutations {
	if permutations == nil {
		return nil
	}
	return &events2metrics.E2MPermutations{
		Limit: *permutations,
	}
}

func expandL2MSeverityFilters(severityFilters []L2MSeverity) []events2metrics.Logs2metricsV2Severity {
	if severityFilters == nil {
		return nil
	}
	expanded := make([]events2metrics.Logs2metricsV2Severity, 0, len(severityFilters))
	for _, severity := range severityFilters {
		if protoSeverity, ok := L2MSeveritySchemaToOpenAPI[severity]; ok {
			expanded = append(expanded, protoSeverity)
		}
	}
	return expanded
}

func extractE2mMetricLabels(labels []MetricLabel) []events2metrics.MetricLabel {
	metricLabels := make([]events2metrics.MetricLabel, 0, len(labels))
	for _, label := range labels {
		metricLabels = append(metricLabels, events2metrics.MetricLabel{
			TargetLabel: label.TargetLabel,
			SourceField: label.SourceField,
		})
	}
	return metricLabels
}

func extractE2mMetricFields(fields []MetricField) []events2metrics.V2MetricField {
	metricFields := make([]events2metrics.V2MetricField, 0, len(fields))
	for _, field := range fields {
		metricField := events2metrics.V2MetricField{
			TargetBaseMetricName: field.TargetBaseMetricName,
			SourceField:          field.SourceField,
			Aggregations:         extractE2mAggregations(field.Aggregations),
		}
		metricFields = append(metricFields, metricField)
	}
	return metricFields
}

func extractE2mAggregations(aggregations []MetricFieldAggregation) []events2metrics.V2Aggregation {
	metricAggregations := make([]events2metrics.V2Aggregation, 0, len(aggregations))
	for _, aggregation := range aggregations {
		if aggregation.AggMetadata.Samples != nil {
			metricAggregations = append(metricAggregations, events2metrics.V2Aggregation{
				V2AggregationSamples: &events2metrics.V2AggregationSamples{
					Enabled:          events2metrics.PtrBool(aggregation.Enabled),
					AggType:          AggregationTypeSchemaToOpenAPI[aggregation.AggType].Ptr(),
					TargetMetricName: events2metrics.PtrString(aggregation.TargetMetricName),
					Samples: &events2metrics.E2MAggSamples{
						SampleType: E2MAggSamplesSampleTypeSchemaToOpenAPI[aggregation.AggMetadata.Samples.SampleType],
					},
				},
			})
		} else if aggregation.AggMetadata.Histogram != nil {
			metricAggregations = append(metricAggregations, events2metrics.V2Aggregation{
				V2AggregationHistogram: &events2metrics.V2AggregationHistogram{
					Enabled:          events2metrics.PtrBool(aggregation.Enabled),
					AggType:          AggregationTypeSchemaToOpenAPI[aggregation.AggType].Ptr(),
					TargetMetricName: events2metrics.PtrString(aggregation.TargetMetricName),
					Histogram: &events2metrics.E2MAggHistogram{
						Buckets: utils.QuantitiesToFloats32(aggregation.AggMetadata.Histogram.Buckets),
					},
				},
			})
		}
	}

	return metricAggregations
}

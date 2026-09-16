/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	slos "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/slos_service"
)

var (
	// WindowSloWindowSchemaToOpenAPI has no "unspecified" entry on purpose. The Coralogix
	// API has no implementation for WINDOW_SLO_WINDOW_UNSPECIFIED and answers HTTP 500 for
	// it, so a CR that still carries the value is rejected in the extract path with a clear
	// message. This mirrors how "90d" is kept out of sloTimeFrameSchemaToOpenAPI.
	WindowSloWindowSchemaToOpenAPI = map[SloWindowEnum]slos.WindowSloWindow{
		"1m": slos.WINDOWSLOWINDOW_WINDOW_SLO_WINDOW_1_MINUTE,
		"5m": slos.WINDOWSLOWINDOW_WINDOW_SLO_WINDOW_5_MINUTES,
	}
	SloProductTypeSchemaToOpenAPI = map[SloProductType]slos.SloProductType{
		"unspecified": slos.SLOPRODUCTTYPE_SLO_PRODUCT_TYPE_UNSPECIFIED,
		"apm":         slos.SLOPRODUCTTYPE_SLO_PRODUCT_TYPE_APM,
	}
	ComparisonOperatorSchemaToOpenAPI = map[ComparisonOperator]slos.ComparisonOperator{
		"unspecified":         slos.COMPARISONOPERATOR_COMPARISON_OPERATOR_UNSPECIFIED,
		"greaterThan":         slos.COMPARISONOPERATOR_COMPARISON_OPERATOR_GREATER_THAN,
		"lessThan":            slos.COMPARISONOPERATOR_COMPARISON_OPERATOR_LESS_THAN,
		"greaterThanOrEquals": slos.COMPARISONOPERATOR_COMPARISON_OPERATOR_GREATER_THAN_OR_EQUALS,
		"lessThanOrEquals":    slos.COMPARISONOPERATOR_COMPARISON_OPERATOR_LESS_THAN_OR_EQUALS,
	}
)

// SLOSpec defines the desired state of SLO. For more information, see: https://coralogix.com/platform/apm/slo-management/
// +kubebuilder:validation:XValidation:rule="!(has(self.productType) && self.productType == 'apm') || has(self.sliType.apmSli)",message="productType 'apm' requires sliType.apmSli"
type SLOSpec struct {
	// SLO name
	Name string `json:"name"`
	// +optional
	// Optional SLO description
	Description *string `json:"description"`
	// +optional
	// Labels are additional labels to be added to the SLO.
	Labels *map[string]string `json:"labels,omitempty"`
	// SliType defines the type of SLI used for the SLO.
	// Exactly one of requestBasedMetric, windowBasedMetric or apmSli must be set.
	SliType SliType `json:"sliType"`
	// +optional
	// ProductType selects the Coralogix product the SLO is built from. Valid values are
	// "unspecified" and "apm". An apmSli requires "apm". When omitted, the API stores
	// SLO_PRODUCT_TYPE_UNSPECIFIED.
	ProductType *SloProductType `json:"productType,omitempty"`
	// Window defines the time window for the SLO.
	Window SloWindow `json:"window"`
	// TargetThresholdPercentage is the target threshold percentage for the SLO.
	TargetThresholdPercentage resource.Quantity `json:"targetThresholdPercentage"`
}

type SloGrouping struct {
	// Labels defines the labels to group the SLO by.
	Labels []string `json:"labels,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="[has(self.requestBasedMetric),has(self.windowBasedMetric),has(self.apmSli)].filter(x, x).size() == 1",message="Exactly one of requestBasedMetric, windowBasedMetric or apmSli must be set"
type SliType struct {
	// +optional
	RequestBasedMetricSli *RequestBasedMetricSli `json:"requestBasedMetric,omitempty"`
	// +optional
	WindowBasedMetricSli *WindowBasedMetricSli `json:"windowBasedMetric,omitempty"`
	// +optional
	// ApmSli builds the SLO from an APM Service Catalog service instead of a PromQL query.
	// It requires spec.productType "apm".
	ApmSli *ApmSli `json:"apmSli,omitempty"`
}

type RequestBasedMetricSli struct {
	// GoodEvents defines the good events metric.
	GoodEvents SloMetricEvent `json:"goodEvents"`
	// TotalEvents defines the total events metric.
	TotalEvents SloMetricEvent `json:"totalEvents"`
	// +optional
	// GroupByLabels defines the labels to group the SLI by.
	GroupByLabels []string `json:"groupByLabels,omitempty"`
}

type WindowBasedMetricSli struct {
	// +optional
	// Optional query for the metric.
	Query *SloMetricEvent `json:"query,omitempty"`
	// Window defines the time window for the SLO. Valid values are "1m" and "5m".
	Window SloWindowEnum `json:"window"`
	// ComparisonOperator defines the comparison operator for the SLO. Valid values are "unspecified", "greaterThan", "lessThan", "greaterThanOrEquals", and "lessThanOrEquals".
	ComparisonOperator ComparisonOperator `json:"comparisonOperator,omitempty"`
	// Threshold defines the threshold for the SLO.
	Threshold resource.Quantity `json:"threshold,omitempty"`
}

// ApmSli defines an SLI over an APM Service Catalog service.
// +kubebuilder:validation:XValidation:rule="has(self.errorConfig) != has(self.latencyConfig)",message="Exactly one of errorConfig or latencyConfig must be set"
type ApmSli struct {
	// Services lists the APM Service Catalog services the SLO covers. The Coralogix API
	// accepts exactly one service today and rejects names that are not in the catalog.
	// The count is deliberately not capped here, so a server-side relaxation needs no
	// operator release.
	// +kubebuilder:validation:MinItems=1
	Services []string `json:"services"`
	// +optional
	// Filters narrows the SLI to spans matching every filter.
	Filters []ApmFilter `json:"filters,omitempty"`
	// +optional
	// GroupingKeys splits the SLI by the given span attributes.
	GroupingKeys []string `json:"groupingKeys,omitempty"`
	// +optional
	// ErrorConfig makes this an APM error-rate SLI. It carries no settings.
	ErrorConfig *ApmErrorSli `json:"errorConfig,omitempty"`
	// +optional
	// LatencyConfig makes this an APM latency SLI.
	LatencyConfig *ApmLatencySli `json:"latencyConfig,omitempty"`
}

// ApmErrorSli selects the APM error-rate SLI. It has no fields.
type ApmErrorSli struct{}

// ApmLatencySli defines an APM latency SLI.
// +kubebuilder:validation:XValidation:rule="has(self.quantile) != has(self.average)",message="Exactly one of quantile or average must be set"
type ApmLatencySli struct {
	// TimeWindow defines the evaluation window. Valid values are "1m" and "5m".
	TimeWindow SloWindowEnum `json:"timeWindow"`
	// +optional
	// Threshold is the latency threshold in seconds. The API stores 0 when omitted.
	Threshold *resource.Quantity `json:"threshold,omitempty"`
	// +optional
	// Quantile measures a latency quantile.
	Quantile *ApmLatencyQuantile `json:"quantile,omitempty"`
	// +optional
	// Average measures average latency. It carries no settings.
	Average *ApmLatencyAverage `json:"average,omitempty"`
}

// ApmLatencyQuantile selects the quantile latency query type.
type ApmLatencyQuantile struct {
	// +optional
	// Percentile is a fraction, so 0.95 means P95. The API stores 0 when omitted.
	// The accepted range is not validated here because the API's own bounds are unverified.
	Percentile *resource.Quantity `json:"percentile,omitempty"`
}

// ApmLatencyAverage selects the average latency query type. It has no fields.
type ApmLatencyAverage struct{}

// ApmFilter matches a span attribute against a set of values.
type ApmFilter struct {
	// Key is the span attribute name.
	Key string `json:"key"`
	// Values are the accepted values for Key.
	// +kubebuilder:validation:MinItems=1
	Values []string `json:"values"`
}

// +kubebuilder:validation:Enum={"unspecified","apm"}
type SloProductType string

// +kubebuilder:validation:Enum={"1m","5m"}
type SloWindowEnum string

// +kubebuilder:validation:Enum={"unspecified","greaterThan","lessThan","greaterThanOrEquals","lessThanOrEquals"}
type ComparisonOperator string

type SloMetricEvent struct {
	// Query is the metric query string.
	Query string `json:"query"`
}

type SloWindow struct {
	// +optional
	// TimeFrame defines the time frame for the SLO window. Valid values are "unspecified", "7d", "14d", "21d", and "28d".
	// Deprecated: "90d" is no longer supported by the Coralogix API and will be rejected by the operator.
	TimeFrame *SloTimeFrame `json:"timeFrame,omitempty"`
}

// +kubebuilder:validation:Enum={"unspecified","7d","14d","21d","28d","90d"}
type SloTimeFrame string

const (
	SloTimeFrameUnspecified SloTimeFrame = "unspecified"
	SloTimeFrame7d          SloTimeFrame = "7d"
	SloTimeFrame14d         SloTimeFrame = "14d"
	SloTimeFrame21d         SloTimeFrame = "21d"
	SloTimeFrame28d         SloTimeFrame = "28d"
	SloTimeFrame90d         SloTimeFrame = "90d"
)

var sloTimeFrameSchemaToOpenAPI = map[SloTimeFrame]slos.SloTimeFrame{
	SloTimeFrameUnspecified: slos.SLOTIMEFRAME_SLO_TIME_FRAME_UNSPECIFIED,
	SloTimeFrame7d:          slos.SLOTIMEFRAME_SLO_TIME_FRAME_7_DAYS,
	SloTimeFrame14d:         slos.SLOTIMEFRAME_SLO_TIME_FRAME_14_DAYS,
	SloTimeFrame21d:         slos.SLOTIMEFRAME_SLO_TIME_FRAME_21_DAYS,
	SloTimeFrame28d:         slos.SLOTIMEFRAME_SLO_TIME_FRAME_28_DAYS,
}

// SLOStatus defines the observed state of SLO.
type SLOStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Revision *int32 `json:"revision,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// SLO is the Schema for the slos API.
// See also https://coralogix.com/platform/apm/slo-management/
type SLO struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SLOSpec   `json:"spec,omitempty"`
	Status SLOStatus `json:"status,omitempty"`
}

func (s *SLO) ExtractSLOCreateRequest() (*slos.Slo1, error) {
	return s.Spec.extractSLO()
}

func (s *SLO) ExtractSLOUpdateRequest() (*slos.Slo1, error) {
	slo, err := s.Spec.extractSLO()
	if err != nil {
		return nil, err
	}

	slo.Id = s.Status.ID
	return slo, nil
}

// extractSLO builds the request body for the SLI arm the spec selects. A CEL rule on
// SliType already requires exactly one arm, so the default case only guards against a CR
// stored before that rule existed.
func (s *SLOSpec) extractSLO() (*slos.Slo1, error) {
	switch {
	case s.SliType.RequestBasedMetricSli != nil:
		return s.ExtractRequestBasedMetricSli()
	case s.SliType.WindowBasedMetricSli != nil:
		return s.ExtractWindowBasedMetricSli()
	case s.SliType.ApmSli != nil:
		return s.ExtractApmSli()
	default:
		return nil, fmt.Errorf("sliType must be set to one of requestBasedMetric, windowBasedMetric or apmSli")
	}
}

// extractCommon builds the fields every SLI arm shares.
func (s *SLOSpec) extractCommon() (*slos.Slo1, error) {
	timeFrame, err := s.Window.ExpandTimeFrame()
	if err != nil {
		return nil, fmt.Errorf("error expanding time frame: %w", err)
	}

	productType, err := s.ExpandProductType()
	if err != nil {
		return nil, fmt.Errorf("error expanding product type: %w", err)
	}

	return &slos.Slo1{
		Name:                      slos.PtrString(s.Name),
		Description:               s.Description,
		Labels:                    ptr.Deref(s.Labels, nil),
		SloTimeFrame:              timeFrame,
		ProductType:               productType,
		TargetThresholdPercentage: slos.PtrFloat32(float32(s.TargetThresholdPercentage.AsApproximateFloat64())),
	}, nil
}

// ExpandProductType maps the spec value to the SDK enum. The field is optional, so an
// absent value leaves it out of the request and lets the API pick its own default.
func (s *SLOSpec) ExpandProductType() (*slos.SloProductType, error) {
	if s.ProductType == nil {
		return nil, nil
	}

	productType, ok := SloProductTypeSchemaToOpenAPI[*s.ProductType]
	if !ok {
		return nil, fmt.Errorf("invalid SLO product type: %s", *s.ProductType)
	}
	return productType.Ptr(), nil
}

func (s *SLOSpec) ExtractRequestBasedMetricSli() (*slos.Slo1, error) {
	slo, err := s.extractCommon()
	if err != nil {
		return nil, err
	}

	sli := s.SliType.RequestBasedMetricSli
	slo.RequestBasedMetricSli = &slos.RequestBasedMetricSli{
		GoodEvents: &slos.Metric{
			Query: slos.PtrString(sli.GoodEvents.Query),
		},
		TotalEvents: &slos.Metric{
			Query: slos.PtrString(sli.TotalEvents.Query),
		},
	}
	return slo, nil
}

func (s *SLOSpec) ExtractWindowBasedMetricSli() (*slos.Slo1, error) {
	slo, err := s.extractCommon()
	if err != nil {
		return nil, err
	}

	sli := s.SliType.WindowBasedMetricSli
	window, ok := WindowSloWindowSchemaToOpenAPI[sli.Window]
	if !ok {
		return nil, fmt.Errorf("invalid SLO window: %s", sli.Window)
	}

	comparisonOperator, err := sli.ExpandComparisonOperator()
	if err != nil {
		return nil, fmt.Errorf("error expanding comparison operator: %w", err)
	}

	slo.WindowBasedMetricSli = &slos.WindowBasedMetricSli{
		Query: &slos.Metric{
			Query: slos.PtrString(sli.Query.Query),
		},
		Window:             window.Ptr(),
		ComparisonOperator: comparisonOperator,
		Threshold:          slos.PtrFloat32(float32(sli.Threshold.AsApproximateFloat64())),
	}
	return slo, nil
}

func (s *SLOSpec) ExtractApmSli() (*slos.Slo1, error) {
	slo, err := s.extractCommon()
	if err != nil {
		return nil, err
	}

	apmSli, err := s.SliType.ApmSli.ExpandApmSli()
	if err != nil {
		return nil, fmt.Errorf("error expanding apm SLI: %w", err)
	}

	// apmSliMetadata is a server-side mirror of apmSli. The API rejects it on write, so
	// it is never part of the request.
	slo.ApmSli = apmSli
	return slo, nil
}

func (a *ApmSli) ExpandApmSli() (*slos.ApmSli, error) {
	apmSli := &slos.ApmSli{
		Services:     a.Services,
		GroupingKeys: a.GroupingKeys,
		Filters:      expandApmFilters(a.Filters),
	}

	if a.ErrorConfig != nil {
		// errorConfig is an empty JSON object. The SDK drops a nil map and serializes a
		// non-nil empty map as {}, so the map must be allocated.
		apmSli.ErrorConfig = map[string]interface{}{}
	}

	if a.LatencyConfig != nil {
		latencyConfig, err := a.LatencyConfig.ExpandApmLatencySli()
		if err != nil {
			return nil, fmt.Errorf("error expanding latency config: %w", err)
		}
		apmSli.LatencyConfig = latencyConfig
	}

	return apmSli, nil
}

func (l *ApmLatencySli) ExpandApmLatencySli() (*slos.ApmLatencySli, error) {
	timeWindow, ok := WindowSloWindowSchemaToOpenAPI[l.TimeWindow]
	if !ok {
		return nil, fmt.Errorf("invalid APM latency time window: %s", l.TimeWindow)
	}

	latencyConfig := &slos.ApmLatencySli{
		TimeWindow: timeWindow.Ptr(),
	}

	if l.Threshold != nil {
		latencyConfig.Threshold = slos.PtrFloat32(float32(l.Threshold.AsApproximateFloat64()))
	}

	if l.Quantile != nil {
		latencyConfig.Quantile = &slos.ApmLatencyQuantile{}
		if percentile := l.Quantile.Percentile; percentile != nil {
			latencyConfig.Quantile.Percentile = slos.PtrFloat32(float32(percentile.AsApproximateFloat64()))
		}
	}

	if l.Average != nil {
		// average is an empty JSON object, same as errorConfig.
		latencyConfig.Average = map[string]interface{}{}
	}

	return latencyConfig, nil
}

// expandApmFilters writes the plural values field only. The scalar value field is
// deprecated in the API.
func expandApmFilters(filters []ApmFilter) []slos.ApmFilter {
	if filters == nil {
		return nil
	}

	expanded := make([]slos.ApmFilter, 0, len(filters))
	for _, filter := range filters {
		expanded = append(expanded, slos.ApmFilter{
			Key:    slos.PtrString(filter.Key),
			Values: filter.Values,
		})
	}
	return expanded
}

// ExpandComparisonOperator maps the spec value to the SDK enum. comparisonOperator is
// optional in the CRD, so an empty value leaves the field out of the request instead of
// sending an empty enum string, which the API rejects.
func (w *WindowBasedMetricSli) ExpandComparisonOperator() (*slos.ComparisonOperator, error) {
	if w.ComparisonOperator == "" {
		return nil, nil
	}

	op, ok := ComparisonOperatorSchemaToOpenAPI[w.ComparisonOperator]
	if !ok {
		return nil, fmt.Errorf("invalid SLO comparison operator: %s", w.ComparisonOperator)
	}
	return op.Ptr(), nil
}

func (w *SloWindow) ExpandTimeFrame() (*slos.SloTimeFrame, error) {
	if w.TimeFrame != nil {
		tf, ok := sloTimeFrameSchemaToOpenAPI[*w.TimeFrame]
		if !ok {
			return nil, fmt.Errorf("invalid SLO time frame: %s", *w.TimeFrame)
		}
		return tf.Ptr(), nil
	}

	return nil, nil
}

func (s *SLO) SetConditions(conditions []metav1.Condition) {
	s.Status.Conditions = conditions
}

func (s *SLO) GetConditions() []metav1.Condition {
	return s.Status.Conditions
}

func (s *SLO) GetPrintableStatus() string {
	return s.Status.PrintableStatus
}

func (s *SLO) SetPrintableStatus(status string) {
	s.Status.PrintableStatus = status
}

func (s *SLO) HasIDInStatus() bool {
	return s.Status.ID != nil && *s.Status.ID != ""
}

// +kubebuilder:object:root=true

// SLOList contains a list of SLO.
type SLOList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SLO `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SLO{}, &SLOList{})
}

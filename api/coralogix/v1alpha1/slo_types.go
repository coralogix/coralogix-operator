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

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	WindowSloWindowSchemaToProto = map[SloWindowEnum]cxsdk.SloWindow{
		"unspecified": cxsdk.SloWindowUnspecified,
		"1m":          cxsdk.SloWindow1Minute,
		"5m":          cxsdk.SloWindow5Minutes,
	}
	ComparisonOperatorSchemaToProto = map[ComparisonOperator]cxsdk.SloComparisonOperator{
		"unspecified":         cxsdk.SloComparisonOperatorUnspecified,
		"greaterThan":         cxsdk.SloComparisonOperatorGreaterThan,
		"lessThan":            cxsdk.SloComparisonOperatorLessThan,
		"greaterThanOrEquals": cxsdk.SloComparisonOperatorGreaterThanOrEquals,
		"lessThanOrEquals":    cxsdk.SloComparisonOperatorLessThanOrEquals,
	}
)

// SLOSpec defines the desired state of SLO. For more information, see: https://coralogix.com/platform/apm/slo-management/
type SLOSpec struct {
	// SLO name
	Name string `json:"name"`
	// +optional
	// Optional SLO description
	Description *string `json:"description"`
	// +optional
	// Labels are additional labels to be added to the SLO.
	Labels map[string]string `json:"labels,omitempty"`
	// SliType defines the type of SLI used for the SLO. Exactly one of metric or windowBasedMetric must be set.
	SliType SliType `json:"sliType"`
	// Window defines the time window for the SLO.
	Window SloWindow `json:"window"`
	// TargetThresholdPercentage is the target threshold percentage for the SLO.
	TargetThresholdPercentage resource.Quantity `json:"targetThresholdPercentage"`
}

type SloGrouping struct {
	// Labels defines the labels to group the SLO by.
	Labels []string `json:"labels,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.requestBasedMetric) != has(self.windowBasedMetric)",message="Exactly one of requestBasedMetricSli or windowBasedMetric must be set"
type SliType struct {
	// +optional
	RequestBasedMetricSli *RequestBasedMetricSli `json:"requestBasedMetric,omitempty"`
	// +optional
	WindowBasedMetricSli *WindowBasedMetricSli `json:"windowBasedMetric,omitempty"`
}

type RequestBasedMetricSli struct {
	// +optional
	// GoodEvents defines the good events metric.
	GoodEvents *SloMetricEvent `json:"goodEvents,omitempty"`
	// +optional
	// TotalEvents defines the total events metric.
	TotalEvents *SloMetricEvent `json:"totalEvents,omitempty"`
	// +optional
	// GroupByLabels defines the labels to group the SLI by.
	GroupByLabels []string `json:"groupByLabels,omitempty"`
}

type WindowBasedMetricSli struct {
	// +optional
	// Optional query for the metric.
	Query *SloMetricEvent `json:"query,omitempty"`
	// Window defines the time window for the SLO. Valid values are "unspecified", "1m", and "5m".
	Window SloWindowEnum `json:"window,omitempty"`
	// ComparisonOperator defines the comparison operator for the SLO. Valid values are "unspecified", "greaterThan", "lessThan", "greaterThanOrEquals", and "lessThanOrEquals".
	ComparisonOperator ComparisonOperator `json:"comparisonOperator,omitempty"`
	// Threshold defines the threshold for the SLO.
	Threshold resource.Quantity `json:"threshold,omitempty"`
}

// +kubebuilder:validation:Enum={"unspecified","1m","5m"}
type SloWindowEnum string

// +kubebuilder:validation:Enum={"unspecified","greaterThan","lessThan","greaterThanOrEquals","lessThanOrEquals"}
type ComparisonOperator string

type SloMetricEvent struct {
	// Query is the metric query string.
	Query string `json:"query"`
}

type SloWindow struct {
	// +optional
	// TimeFrame defines the time frame for the SLO window. Valid values are "unspecified", "7d", "14d", "21d", "28d", and "90d".
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

var sloTimeFrameMap = map[SloTimeFrame]cxsdk.SloTimeframeEnum{
	SloTimeFrameUnspecified: cxsdk.SloTimeframeUnspecified,
	SloTimeFrame7d:          cxsdk.SloTimeframe7Days,
	SloTimeFrame14d:         cxsdk.SloTimeframe14Days,
	SloTimeFrame21d:         cxsdk.SloTimeframe21Days,
	SloTimeFrame28d:         cxsdk.SloTimeframe28Days,
}

// SLOStatus defines the observed state of SLO.
type SLOStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Revision *int32 `json:"revision,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SLO is the Schema for the slos API.
type SLO struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec            SLOSpec   `json:"spec,omitempty"`
	Status          SLOStatus `json:"status,omitempty"`
	PrintableStatus string    `json:"printableStatus,omitempty"`
}

func (spec *SLOSpec) ExtractSLO() (*cxsdk.Slo, error) {
	slo := &cxsdk.Slo{
		Name:                      spec.Name,
		Description:               spec.Description,
		Labels:                    spec.Labels,
		TargetThresholdPercentage: float32(spec.TargetThresholdPercentage.AsApproximateFloat64()),
	}

	slo, err := spec.SliType.ExpandSliType(slo)
	if err != nil {
		return nil, err
	}

	slo, err = spec.Window.ExpandSloWindow(slo)
	if err != nil {
		return nil, err
	}

	return slo, nil
}

func (in *SliType) ExpandSliType(slo *cxsdk.Slo) (*cxsdk.Slo, error) {
	if requestBasedMetricSli := in.RequestBasedMetricSli; requestBasedMetricSli != nil {
		slo.Sli = &cxsdk.SloRequestBasedMetricSli{
			RequestBasedMetricSli: &cxsdk.RequestBasedMetricSli{
				GoodEvents:  extractMetricEvent(requestBasedMetricSli.GoodEvents),
				TotalEvents: extractMetricEvent(requestBasedMetricSli.TotalEvents),
			},
		}
	} else if windowBasedMetricSli := in.WindowBasedMetricSli; windowBasedMetricSli != nil {
		slo.Sli = &cxsdk.SloWindowBasedMetricSli{
			WindowBasedMetricSli: &cxsdk.WindowBasedMetricSli{
				Query:              extractMetricEvent(windowBasedMetricSli.Query),
				Window:             WindowSloWindowSchemaToProto[windowBasedMetricSli.Window],
				ComparisonOperator: ComparisonOperatorSchemaToProto[windowBasedMetricSli.ComparisonOperator],
				Threshold:          float32(windowBasedMetricSli.Threshold.AsApproximateFloat64()),
			},
		}
	}

	return slo, nil
}

func extractMetricEvent(metricEvent *SloMetricEvent) *cxsdk.Metric {
	if metricEvent == nil {
		return nil
	}
	return &cxsdk.Metric{
		Query: metricEvent.Query,
	}
}

func (in SloWindow) ExpandSloWindow(slo *cxsdk.Slo) (*cxsdk.Slo, error) {
	if timeFrame := in.TimeFrame; timeFrame != nil {
		sloTimeFrame, ok := sloTimeFrameMap[*timeFrame]
		if !ok {
			return nil, fmt.Errorf("invalid SLO time frame: %s", *timeFrame)
		}
		slo.Window = &cxsdk.SloTimeframe{
			SloTimeFrame: sloTimeFrame,
		}
	}

	return slo, nil
}

func (s *SLO) SetConditions(conditions []metav1.Condition) {
	s.Status.Conditions = conditions
}

func (s *SLO) GetConditions() []metav1.Condition {
	return s.Status.Conditions
}

func (s *SLO) GetPrintableStatus() string {
	return s.PrintableStatus
}

func (s *SLO) SetPrintableStatus(status string) {
	s.PrintableStatus = status
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

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

	slos "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/slos_service"
)

var (
	WindowSloWindowSchemaToOpenAPI = map[SloWindowEnum]slos.WindowSloWindow{
		"unspecified": slos.WINDOWSLOWINDOW_WINDOW_SLO_WINDOW_UNSPECIFIED,
		"1m":          slos.WINDOWSLOWINDOW_WINDOW_SLO_WINDOW_1_MINUTE,
		"5m":          slos.WINDOWSLOWINDOW_WINDOW_SLO_WINDOW_5_MINUTES,
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
type SLOSpec struct {
	// SLO name
	Name string `json:"name"`
	// +optional
	// Optional SLO description
	Description *string `json:"description"`
	// +optional
	// Labels are additional labels to be added to the SLO.
	Labels *map[string]string `json:"labels,omitempty"`
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

func (s *SLO) ExtractSLOCreateRequest() (*slos.SlosServiceCreateSloRequest, error) {
	if requestBasedMetricSli := s.Spec.SliType.RequestBasedMetricSli; requestBasedMetricSli != nil {
		requestBased, err := s.Spec.ExtractRequestBasedMetricSli()
		if err != nil {
			return nil, fmt.Errorf("error extracting request based metric SLI: %w", err)
		}

		return &slos.SlosServiceCreateSloRequest{
			SloRequestBasedMetricSli: requestBased,
		}, nil
	} else if windowBasedMetricSli := s.Spec.SliType.WindowBasedMetricSli; windowBasedMetricSli != nil {
		windowBased, err := s.Spec.ExtractWindowBasedMetricSli()
		if err != nil {
			return nil, fmt.Errorf("error extracting window based metric SLI: %w", err)
		}
		return &slos.SlosServiceCreateSloRequest{
			SloWindowBasedMetricSli: windowBased,
		}, nil
	}

	return nil, fmt.Errorf("sliType must be set to either requestBasedMetricSli or windowBasedMetricSli")
}

func (s *SLO) ExtractSLOUpdateRequest() (*slos.SlosServiceReplaceSloRequest, error) {
	if requestBasedMetricSli := s.Spec.SliType.RequestBasedMetricSli; requestBasedMetricSli != nil {
		requestBased, err := s.Spec.ExtractRequestBasedMetricSli()
		if err != nil {
			return nil, fmt.Errorf("error extracting request based metric SLI: %w", err)
		}

		requestBased.Id = s.Status.ID
		return &slos.SlosServiceReplaceSloRequest{
			SloRequestBasedMetricSli: requestBased,
		}, nil
	} else if windowBasedMetricSli := s.Spec.SliType.WindowBasedMetricSli; windowBasedMetricSli != nil {
		windowBased, err := s.Spec.ExtractWindowBasedMetricSli()
		if err != nil {
			return nil, fmt.Errorf("error extracting window based metric SLI: %w", err)
		}

		windowBased.Id = s.Status.ID
		return &slos.SlosServiceReplaceSloRequest{
			SloWindowBasedMetricSli: windowBased,
		}, nil
	}

	return nil, fmt.Errorf("sliType must be set to either requestBasedMetricSli or windowBasedMetricSli")
}

func (s *SLOSpec) ExtractRequestBasedMetricSli() (*slos.SloRequestBasedMetricSli, error) {
	timeFrame, err := s.Window.ExpandTimeFrame()
	if err != nil {
		return nil, fmt.Errorf("error expanding time frame: %w", err)
	}

	return &slos.SloRequestBasedMetricSli{
		Name:                      s.Name,
		Description:               s.Description,
		Labels:                    s.Labels,
		SloTimeFrame:              timeFrame,
		TargetThresholdPercentage: float32(s.TargetThresholdPercentage.AsApproximateFloat64()),
		RequestBasedMetricSli: &slos.RequestBasedMetricSli{
			GoodEvents: slos.Metric{
				Query: s.SliType.RequestBasedMetricSli.GoodEvents.Query,
			},
			TotalEvents: slos.Metric{
				Query: s.SliType.RequestBasedMetricSli.TotalEvents.Query,
			},
		},
	}, nil
}

func (s *SLOSpec) ExtractWindowBasedMetricSli() (*slos.SloWindowBasedMetricSli, error) {
	timeFrame, err := s.Window.ExpandTimeFrame()
	if err != nil {
		return nil, fmt.Errorf("error expanding time frame: %w", err)
	}

	return &slos.SloWindowBasedMetricSli{
		Name:                      s.Name,
		Description:               s.Description,
		Labels:                    s.Labels,
		SloTimeFrame:              timeFrame,
		TargetThresholdPercentage: float32(s.TargetThresholdPercentage.AsApproximateFloat64()),
		WindowBasedMetricSli: &slos.WindowBasedMetricSli{
			Query: slos.Metric{
				Query: s.SliType.WindowBasedMetricSli.Query.Query,
			},
			Window:             WindowSloWindowSchemaToOpenAPI[s.SliType.WindowBasedMetricSli.Window],
			ComparisonOperator: ComparisonOperatorSchemaToOpenAPI[s.SliType.WindowBasedMetricSli.ComparisonOperator],
			Threshold:          float32(s.SliType.WindowBasedMetricSli.Threshold.AsApproximateFloat64()),
		},
	}, nil
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

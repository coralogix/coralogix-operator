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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SLOSpec defines the desired state of SLO.
type SLOSpec struct {
	Name        string `json:"name"`
	ServiceName string `json:"serviceName"`
	// +optional
	Description *string `json:"description"`
	// +optional
	Labels  map[string]string `json:"labels,omitempty"`
	SliType SliType           `json:"sliType"`
	Window  SloWindow         `json:"window"`
	// +kubebuilder:validation:Maximum:=100
	TargetThresholdPercentage int32 `json:"targetThresholdPercentage"`
}

type SliType struct {
	// +optional
	Metric *SloMetricType `json:"metric,omitempty"`
}

type SloMetricType struct {
	// +optional
	GoodEvents *SloMetricEvent `json:"goodEvents,omitempty"`
	// +optional
	TotalEvents *SloMetricEvent `json:"totalEvents,omitempty"`
	// +optional
	GroupByLabels []string `json:"groupByLabels,omitempty"`
}

type SloMetricEvent struct {
	Query string `json:"query"`
}

type SloWindow struct {
	// +optional
	TimeFrame *SloTimeFrame `json:"timeFrame"`
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
	SloTimeFrame90d:         cxsdk.SloTimeframe90Days,
}

// SLOStatus defines the observed state of SLO.
type SLOStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SLO is the Schema for the slos API.
type SLO struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SLOSpec   `json:"spec,omitempty"`
	Status SLOStatus `json:"status,omitempty"`
}

func (spec *SLOSpec) ExtractSLO() (*cxsdk.Slo, error) {
	slo := &cxsdk.Slo{
		Name:                      spec.Name,
		Description:               spec.Description,
		Labels:                    spec.Labels,
		TargetThresholdPercentage: spec.TargetThresholdPercentage,
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
	if typeMetric := in.Metric; typeMetric != nil {
		slo.Sli = &cxsdk.SloMetricSli{
			MetricSli: &cxsdk.MetricSli{
				GoodEvents:    extractMetricEvent(typeMetric.GoodEvents),
				TotalEvents:   extractMetricEvent(typeMetric.TotalEvents),
				GroupByLabels: typeMetric.GroupByLabels,
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

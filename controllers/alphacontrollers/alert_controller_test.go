package alphacontrollers

import (
	"context"
	"testing"

	utils "github.com/coralogix/coralogix-operator/apis"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	alerts "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/alerts/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/pointer"
)

func TestFlattenAlerts(t *testing.T) {
	alert := &alerts.Alert{
		UniqueIdentifier: wrapperspb.String("id"),
		Name:             wrapperspb.String("name"),
		Description:      wrapperspb.String("description"),
		IsActive:         wrapperspb.Bool(true),
		Severity:         alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL,
		MetaLabels:       []*alerts.MetaLabel{{Key: wrapperspb.String("key"), Value: wrapperspb.String("value")}},
		Condition: &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThanUsual{
				MoreThanUsual: &alerts.MoreThanUsualCondition{
					Parameters: &alerts.ConditionParameters{
						Threshold: wrapperspb.Double(3),
						Timeframe: alerts.Timeframe_TIMEFRAME_12_H,
						MetricAlertPromqlParameters: &alerts.MetricAlertPromqlConditionParameters{
							PromqlText:        wrapperspb.String("http_requests_total{status!~\"4..\"}"),
							NonNullPercentage: wrapperspb.UInt32(10),
							SwapNullValues:    wrapperspb.Bool(false),
						},
						NotifyGroupByOnlyAlerts: wrapperspb.Bool(false),
					},
				},
			},
		},
		Filters: &alerts.AlertFilters{
			FilterType: alerts.AlertFilters_FILTER_TYPE_METRIC,
		},
	}

	spec := coralogixv1alpha1.AlertSpec{
		Scheduling: &coralogixv1alpha1.Scheduling{
			TimeZone: coralogixv1alpha1.TimeZone("UTC+02"),
		},
	}
	status, err := flattenAlert(context.Background(), alert, spec)
	assert.NoError(t, err)

	minNonNullValuesPercentage := 10
	expected := &coralogixv1alpha1.AlertStatus{
		ID:          pointer.String("id"),
		Name:        "name",
		Description: "description",
		Active:      true,
		Severity:    "Critical",
		Labels:      map[string]string{"key": "value"},
		AlertType: coralogixv1alpha1.AlertType{
			Metric: &coralogixv1alpha1.Metric{
				Promql: &coralogixv1alpha1.Promql{
					SearchQuery: "http_requests_total{status!~\"4..\"}",
					Conditions: coralogixv1alpha1.PromqlConditions{
						AlertWhen:                   "MoreThanUsual",
						Threshold:                   utils.FloatToQuantity(3.0),
						TimeWindow:                  coralogixv1alpha1.MetricTimeWindow("TwelveHours"),
						MinNonNullValuesPercentage:  &minNonNullValuesPercentage,
						ReplaceMissingValueWithZero: false,
					},
				},
			},
		},
		NotificationGroups: []coralogixv1alpha1.NotificationGroup{},
		PayloadFilters:     []string{},
	}

	assert.EqualValues(t, expected, status)
}

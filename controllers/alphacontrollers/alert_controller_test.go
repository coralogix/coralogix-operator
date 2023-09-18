package alphacontrollers

import (
	"context"
	"testing"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	alerts "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/alerts/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestFlattenAlerts(t *testing.T) {
	alert := &alerts.Alert{
		Id:          wrapperspb.String("id"),
		Name:        wrapperspb.String("name"),
		Description: wrapperspb.String("description"),
		IsActive:    wrapperspb.Bool(true),
		Severity:    alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL,
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
	}

	spec := coralogixv1alpha1.AlertSpec{
		Scheduling: &coralogixv1alpha1.Scheduling{
			TimeZone: coralogixv1alpha1.TimeZone("UTC+02"),
		},
	}
	status, err := flattenAlert(context.Background(), alert, spec)
	assert.NoError(t, err)

	id := "id"
	subgroupId := "subgroup_id"
	expected := &coralogixv1alpha1.AlertStatus{
		ID:          &id,
		Name:        "name",
		Description: "description",
		Active:      true,
		Severity:    "critical",
		Condition:   "more_than_usual",
	}

	assert.Equal(t, expected, status)
}

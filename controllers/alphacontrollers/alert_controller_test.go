package alphacontrollers

import (
	"context"
	"os"
	"testing"

	utils "github.com/coralogix/coralogix-operator/apis"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	alerts "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/alerts/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
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

type mockCoralogixClientSet struct {
}

func (m *mockCoralogixClientSet) RuleGroups() clientset.RuleGroupsClientInterface {
	return nil
}

func (m *mockCoralogixClientSet) Alerts() clientset.AlertsClientInterface {
	return &mockAlertsClient{}
}

func (m *mockCoralogixClientSet) RecordingRulesGroups() clientset.RecordingRulesGroupsClientInterface {
	return nil
}

func (m *mockCoralogixClientSet) Webhooks() clientset.WebhooksClientInterface {
	return &mockWebhooksClient{}
}

type mockAlertsClient struct {
	alerts map[string]*alerts.Alert
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (m *mockAlertsClient) CreateAlert(_ context.Context, req *alerts.CreateAlertRequest) (*alerts.CreateAlertResponse, error) {
	if m.alerts == nil {
		m.alerts = make(map[string]*alerts.Alert)
	}

	alert := &alerts.Alert{
		UniqueIdentifier:           wrapperspb.String(RandStringBytes(10)),
		Name:                       req.GetName(),
		Description:                req.GetDescription(),
		IsActive:                   req.GetIsActive(),
		Severity:                   req.GetSeverity(),
		Expiration:                 req.GetExpiration(),
		MetaLabels:                 req.GetMetaLabels(),
		Condition:                  req.GetCondition(),
		Filters:                    req.GetFilters(),
		ShowInInsight:              req.GetShowInInsight(),
		NotificationGroups:         req.GetNotificationGroups(),
		ActiveWhen:                 req.GetActiveWhen(),
		TracingAlert:               req.GetTracingAlert(),
		NotificationPayloadFilters: req.GetNotificationPayloadFilters(),
	}

	m.alerts[alert.GetUniqueIdentifier().GetValue()] = alert

	return &alerts.CreateAlertResponse{
		Alert: alert,
	}, nil
}

func (m *mockAlertsClient) GetAlert(_ context.Context, req *alerts.GetAlertByUniqueIdRequest) (*alerts.GetAlertByUniqueIdResponse, error) {
	if m.alerts == nil {
		m.alerts = make(map[string]*alerts.Alert)
	}

	alert, ok := m.alerts[req.GetId().GetValue()]
	if !ok {
		return nil, errors.NewNotFound(schema.GroupResource{}, req.GetId().GetValue())
	}

	return &alerts.GetAlertByUniqueIdResponse{
		Alert: alert,
	}, nil
}

func (m *mockAlertsClient) UpdateAlert(_ context.Context, req *alerts.UpdateAlertByUniqueIdRequest) (*alerts.UpdateAlertByUniqueIdResponse, error) {
	if m.alerts == nil {
		m.alerts = make(map[string]*alerts.Alert)
	}

	_, ok := m.alerts[req.GetAlert().GetUniqueIdentifier().GetValue()]
	if !ok {
		return nil, errors.NewNotFound(schema.GroupResource{}, req.GetAlert().GetUniqueIdentifier().GetValue())
	}

	alert := req.GetAlert()
	m.alerts[req.GetAlert().GetUniqueIdentifier().GetValue()] = alert

	return &alerts.UpdateAlertByUniqueIdResponse{
		Alert: alert,
	}, nil
}

func (m *mockAlertsClient) DeleteAlert(_ context.Context, req *alerts.DeleteAlertByUniqueIdRequest) (*alerts.DeleteAlertByUniqueIdResponse, error) {
	if m.alerts == nil {
		m.alerts = make(map[string]*alerts.Alert)
	}

	_, ok := m.alerts[req.GetId().GetValue()]
	if !ok {
		return nil, errors.NewNotFound(schema.GroupResource{}, req.GetId().GetValue())
	}

	delete(m.alerts, req.GetId().GetValue())

	return &alerts.DeleteAlertByUniqueIdResponse{}, nil
}

type mockWebhooksClient struct {
}

func (m mockWebhooksClient) CreateWebhook(ctx context.Context, body string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockWebhooksClient) GetWebhook(ctx context.Context, webhookId string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockWebhooksClient) GetWebhooks(ctx context.Context) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockWebhooksClient) UpdateWebhook(ctx context.Context, body string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockWebhooksClient) DeleteWebhook(ctx context.Context, webhookId string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func TestAlertReconciler_Reconcile(t *testing.T) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: runtime.NewScheme(),
	})
	if err != nil {
		os.Exit(1)
	}

	r := AlertReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: &mockCoralogixClientSet{},
	}
}

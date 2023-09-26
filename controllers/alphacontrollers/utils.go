package alphacontrollers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coralogix/coralogix-operator/controllers/clientset"
	alerts "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/alerts/v2"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	defaultRequeuePeriod    = 30 * time.Second
	defaultErrRequeuePeriod = 20 * time.Second
)

type mockCoralogixClientSet struct {
	alertsClient   *mockAlertsClient
	webhooksClient *mockWebhooksClient
}

func NewMockCoralogixClientSet() clientset.ClientSetInterface {
	return &mockCoralogixClientSet{
		alertsClient:   &mockAlertsClient{},
		webhooksClient: &mockWebhooksClient{},
	}
}

func (m *mockCoralogixClientSet) RuleGroups() clientset.RuleGroupsClientInterface {
	return nil
}

func (m *mockCoralogixClientSet) Alerts() clientset.AlertsClientInterface {
	return m.alertsClient
}

func (m *mockCoralogixClientSet) RecordingRuleGroups() clientset.RecordingRulesGroupsClientInterface {
	return nil
}

func (m *mockCoralogixClientSet) Webhooks() clientset.WebhooksClientInterface {
	return m.webhooksClient
}

func (m *mockAlertsClient) CreateAlert(_ context.Context, req *alerts.CreateAlertRequest) (*alerts.CreateAlertResponse, error) {
	if m.Alerts == nil {
		m.Alerts = make(map[string]*alerts.Alert)
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

	m.Alerts[alert.GetUniqueIdentifier().GetValue()] = alert

	return &alerts.CreateAlertResponse{
		Alert: alert,
	}, nil
}

func (m *mockAlertsClient) GetAlert(_ context.Context, req *alerts.GetAlertByUniqueIdRequest) (*alerts.GetAlertByUniqueIdResponse, error) {
	if m.Alerts == nil {
		m.Alerts = make(map[string]*alerts.Alert)
	}

	alert, ok := m.Alerts[req.GetId().GetValue()]
	if !ok {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("couldn't find alert %s", req.GetId().GetValue()))
	}

	return &alerts.GetAlertByUniqueIdResponse{
		Alert: alert,
	}, nil
}

func (m *mockAlertsClient) UpdateAlert(_ context.Context, req *alerts.UpdateAlertByUniqueIdRequest) (*alerts.UpdateAlertByUniqueIdResponse, error) {
	if m.Alerts == nil {
		m.Alerts = make(map[string]*alerts.Alert)
	}

	if _, ok := m.Alerts[req.GetAlert().GetUniqueIdentifier().GetValue()]; !ok {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("couldn't find alert %s", req.GetAlert().GetUniqueIdentifier().GetValue()))
	}

	alert := req.GetAlert()
	m.Alerts[req.GetAlert().GetUniqueIdentifier().GetValue()] = alert

	return &alerts.UpdateAlertByUniqueIdResponse{
		Alert: alert,
	}, nil
}

func (m *mockAlertsClient) DeleteAlert(_ context.Context, req *alerts.DeleteAlertByUniqueIdRequest) (*alerts.DeleteAlertByUniqueIdResponse, error) {
	if m.Alerts == nil {
		m.Alerts = make(map[string]*alerts.Alert)
	}

	if _, ok := m.Alerts[req.GetId().GetValue()]; !ok {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("couldn't find alert %s", req.GetId().GetValue()))
	}

	delete(m.Alerts, req.GetId().GetValue())

	return &alerts.DeleteAlertByUniqueIdResponse{}, nil
}

type mockWebhooksClient struct {
	webhooks map[string]map[string]interface{}
}

func (m mockWebhooksClient) CreateWebhook(_ context.Context, body string) (string, error) {
	if m.webhooks == nil {
		m.webhooks = map[string]map[string]interface{}{}
	}
	var webhook map[string]interface{}
	json.Unmarshal([]byte(body), &webhook)

	id := RandStringBytes(10)
	webhook["id"] = id
	m.webhooks[id] = webhook

	bytes, _ := json.Marshal(webhook)
	return string(bytes), nil
}

func (m mockWebhooksClient) GetWebhook(_ context.Context, webhookId string) (string, error) {
	if m.webhooks == nil {
		m.webhooks = map[string]map[string]interface{}{}
	}
	webhook, ok := m.webhooks[webhookId]
	if !ok {
		return "", errors.NewNotFound(schema.GroupResource{}, webhookId)
	}
	bytes, _ := json.Marshal(webhook)
	return string(bytes), nil
}

func (m mockWebhooksClient) GetWebhooks(_ context.Context) (string, error) {
	webhooks := make([]map[string]interface{}, 0, len(m.webhooks))

	for _, w := range m.webhooks {
		webhooks = append(webhooks, w)
	}

	bytes, _ := json.Marshal(webhooks)
	return string(bytes), nil
}

func (m mockWebhooksClient) UpdateWebhook(_ context.Context, body string) (string, error) {
	if m.webhooks == nil {
		m.webhooks = map[string]map[string]interface{}{}
	}
	var webhook map[string]interface{}
	json.Unmarshal([]byte(body), &webhook)

	id := webhook["id"].(string)
	m.webhooks[id] = webhook

	bytes, _ := json.Marshal(webhook)
	return string(bytes), nil
}

func (m mockWebhooksClient) DeleteWebhook(_ context.Context, webhookId string) (string, error) {
	if m.webhooks == nil {
		m.webhooks = map[string]map[string]interface{}{}
	}
	_, ok := m.webhooks[webhookId]
	if !ok {
		return "", errors.NewNotFound(schema.GroupResource{}, webhookId)
	}
	delete(m.webhooks, webhookId)
	return "", nil
}

type mockAlertsClient struct {
	Alerts map[string]*alerts.Alert
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

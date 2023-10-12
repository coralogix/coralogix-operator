package alphacontrollers

import (
	"context"
	"encoding/json"
	"testing"

	utils "github.com/coralogix/coralogix-operator/apis"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	alerts "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/alerts/v2"
	"github.com/coralogix/coralogix-operator/controllers/mock_clientset"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var expectedAlertBackendSchema = &alerts.Alert{
	UniqueIdentifier: wrapperspb.String("id"),
	Name:             wrapperspb.String("name"),
	Description:      wrapperspb.String("description"),
	IsActive:         wrapperspb.Bool(true),
	Severity:         alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL,
	MetaLabels: []*alerts.MetaLabel{
		{Key: wrapperspb.String("key"), Value: wrapperspb.String("value")},
		{Key: wrapperspb.String("managed-by"), Value: wrapperspb.String("coralogix-operator")},
	},
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
	NotificationGroups: []*alerts.AlertNotificationGroups{
		{
			Notifications: []*alerts.AlertNotification{
				{
					RetriggeringPeriodSeconds: wrapperspb.UInt32(600),
					NotifyOn: func() *alerts.NotifyOn {
						notifyOn := new(alerts.NotifyOn)
						*notifyOn = alerts.NotifyOn_TRIGGERED_AND_RESOLVED
						return notifyOn
					}(),
					IntegrationType: &alerts.AlertNotification_Recipients{
						Recipients: &alerts.Recipients{
							Emails: []*wrapperspb.StringValue{wrapperspb.String("example@coralogix.com")},
						},
					},
				},
			},
		},
	},
	Filters: &alerts.AlertFilters{
		FilterType: alerts.AlertFilters_FILTER_TYPE_METRIC,
	},
	NotificationPayloadFilters: []*wrapperspb.StringValue{wrapperspb.String("filter")},
}
var expectedAlertCRD = &coralogixv1alpha1.Alert{
	TypeMeta:   metav1.TypeMeta{Kind: "Alert", APIVersion: "coralogix.com/v1alpha1"},
	ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
	Spec: coralogixv1alpha1.AlertSpec{
		Name:        expectedAlertBackendSchema.GetName().GetValue(),
		Description: expectedAlertBackendSchema.GetDescription().GetValue(),
		Active:      expectedAlertBackendSchema.GetIsActive().GetValue(),
		Severity:    alertProtoSeverityToSchemaSeverity[expectedAlertBackendSchema.GetSeverity()],
		Labels:      map[string]string{"key": "value", "managed-by": "coralogix-operator"},
		NotificationGroups: []coralogixv1alpha1.NotificationGroup{
			{
				Notifications: []coralogixv1alpha1.Notification{
					{
						RetriggeringPeriodMinutes: 10,
						NotifyOn:                  coralogixv1alpha1.NotifyOnTriggeredAndResolved,
						EmailRecipients:           []string{"example@coralogix.com"},
					},
				},
			},
		},
		PayloadFilters: []string{"filter"},
		AlertType: coralogixv1alpha1.AlertType{
			Metric: &coralogixv1alpha1.Metric{
				Promql: &coralogixv1alpha1.Promql{
					SearchQuery: "http_requests_total{status!~\"4..\"}",
					Conditions: coralogixv1alpha1.PromqlConditions{
						AlertWhen:                   "MoreThanUsual",
						Threshold:                   utils.FloatToQuantity(3.0),
						TimeWindow:                  coralogixv1alpha1.MetricTimeWindow("TwelveHours"),
						MinNonNullValuesPercentage:  pointer.Int(10),
						ReplaceMissingValueWithZero: false,
					},
				},
			},
		},
	},
}
var expectedAlertStatus = &coralogixv1alpha1.AlertStatus{
	ID:          pointer.String("id"),
	Name:        "name",
	Description: "description",
	Active:      true,
	Severity:    "Critical",
	Labels:      map[string]string{"key": "value", "managed-by": "coralogix-operator"},
	AlertType: coralogixv1alpha1.AlertType{
		Metric: &coralogixv1alpha1.Metric{
			Promql: &coralogixv1alpha1.Promql{
				SearchQuery: "http_requests_total{status!~\"4..\"}",
				Conditions: coralogixv1alpha1.PromqlConditions{
					AlertWhen:                   "MoreThanUsual",
					Threshold:                   utils.FloatToQuantity(3.0),
					TimeWindow:                  coralogixv1alpha1.MetricTimeWindow("TwelveHours"),
					MinNonNullValuesPercentage:  pointer.Int(10),
					ReplaceMissingValueWithZero: false,
				},
			},
		},
	},
	NotificationGroups: []coralogixv1alpha1.NotificationGroup{
		{
			Notifications: []coralogixv1alpha1.Notification{
				{
					RetriggeringPeriodMinutes: 10,
					NotifyOn:                  coralogixv1alpha1.NotifyOnTriggeredAndResolved,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
			},
		},
	},
	PayloadFilters: []string{"filter"},
}

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
						MinNonNullValuesPercentage:  pointer.Int(10),
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

func TestAlertReconciler_Reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockAlertsClient := createSimpleMockAlertsClient(mockCtrl, expectedAlertBackendSchema)
	mockWebhooksClient := createSimpleWebhooksClient(mockCtrl)
	mockClientSet := mock_clientset.NewMockClientSetInterface(mockCtrl)
	mockClientSet.EXPECT().Alerts().Return(mockAlertsClient).AnyTimes()
	mockClientSet.EXPECT().Webhooks().Return(mockWebhooksClient).AnyTimes()

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	ctx := context.Background()
	go mgr.GetCache().Start(ctx)
	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	assert.NoError(t, err)
	r := AlertReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.AlertList{})
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	err = r.Client.Create(ctx, expectedAlertCRD)
	assert.NoError(t, err)
	<-watcher.ResultChan()

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, result.RequeueAfter, defaultRequeuePeriod)

	namespacedName := types.NamespacedName{Namespace: "default", Name: "test"}
	actualAlertCRD := &coralogixv1alpha1.Alert{}
	err = r.Client.Get(ctx, namespacedName, actualAlertCRD)
	assert.NoError(t, err)

	id := actualAlertCRD.Status.ID
	if !assert.NotNil(t, id) {
		return
	}
	getAlertRequest := &alerts.GetAlertByUniqueIdRequest{Id: wrapperspb.String(*id)}
	alert, err := r.CoralogixClientSet.Alerts().GetAlert(ctx, getAlertRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, expectedAlertBackendSchema, alert.GetAlert())

	err = r.Client.Delete(ctx, actualAlertCRD)
	<-watcher.ResultChan()

	result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, false, result.Requeue)

	alert, err = r.CoralogixClientSet.Alerts().GetAlert(ctx, getAlertRequest)
	assert.Nil(t, alert)
	assert.Error(t, err)
}

func TestAlertReconciler_Reconcile_5XX_StatusError(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockAlertsClient := createMockAlertsClientWith5XXStatusError(mockCtrl, expectedAlertBackendSchema)
	mockWebhooksClient := createSimpleWebhooksClient(mockCtrl)
	mockClientSet := mock_clientset.NewMockClientSetInterface(mockCtrl)
	mockClientSet.EXPECT().Alerts().Return(mockAlertsClient).AnyTimes()
	mockClientSet.EXPECT().Webhooks().Return(mockWebhooksClient).AnyTimes()

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	ctx := context.Background()
	go mgr.GetCache().Start(ctx)
	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	assert.NoError(t, err)
	r := AlertReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.AlertList{})
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	err = r.Client.Create(ctx, expectedAlertCRD)
	assert.NoError(t, err)
	<-watcher.ResultChan()

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.Error(t, err)
	assert.Equal(t, result.RequeueAfter, defaultErrRequeuePeriod)

	result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, result.RequeueAfter, defaultRequeuePeriod)

	namespacedName := types.NamespacedName{Namespace: "default", Name: "test"}
	actualAlertCRD := &coralogixv1alpha1.Alert{}
	err = r.Client.Get(ctx, namespacedName, actualAlertCRD)
	assert.NoError(t, err)
}

// Creates a mock webhooks client that contains a single webhook with id "id1".
func createSimpleWebhooksClient(mockCtrl *gomock.Controller) *mock_clientset.MockWebhooksClientInterface {
	mockWebhooksClient := mock_clientset.NewMockWebhooksClientInterface(mockCtrl)
	webhooks := []map[string]interface{}{{"id": 1}}
	bytes, _ := json.Marshal(webhooks)
	var nilErr error
	mockWebhooksClient.EXPECT().GetWebhooks(gomock.Any()).Return(string(bytes), nilErr).AnyTimes()
	return mockWebhooksClient
}

// Creates a mock alerts client that returns the given alert when creating an alert with name "name1" and id "id1".
func createSimpleMockAlertsClient(mockCtrl *gomock.Controller, alert *alerts.Alert) *mock_clientset.MockAlertsClientInterface {
	mockAlertsClient := mock_clientset.NewMockAlertsClientInterface(mockCtrl)

	var alertExist bool

	mockAlertsClient.EXPECT().
		CreateAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *alerts.CreateAlertRequest) (*alerts.CreateAlertResponse, error) {
		alertExist = true
		return &alerts.CreateAlertResponse{Alert: alert}, nil
	}).AnyTimes()

	mockAlertsClient.EXPECT().
		GetAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *alerts.GetAlertByUniqueIdRequest) (*alerts.GetAlertByUniqueIdResponse, error) {
		if alertExist {
			return &alerts.GetAlertByUniqueIdResponse{Alert: alert}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	mockAlertsClient.EXPECT().
		DeleteAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *alerts.DeleteAlertByUniqueIdRequest) (*alerts.DeleteAlertByUniqueIdResponse, error) {
		if alertExist {
			alertExist = false
			return &alerts.DeleteAlertByUniqueIdResponse{}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	return mockAlertsClient
}

// Creates a mock alerts client that first time fails on creating alert, then returns the given alert when creating an alert with name "name1" and id "id1" .
func createMockAlertsClientWith5XXStatusError(mockCtrl *gomock.Controller, alert *alerts.Alert) *mock_clientset.MockAlertsClientInterface {
	mockAlertsClient := mock_clientset.NewMockAlertsClientInterface(mockCtrl)

	var alertExist bool
	var wasCalled bool
	mockAlertsClient.EXPECT().
		CreateAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *alerts.CreateAlertRequest) (*alerts.CreateAlertResponse, error) {
		if !wasCalled {
			wasCalled = true
			return nil, errors.NewBadRequest("bad request")
		}
		alertExist = true
		return &alerts.CreateAlertResponse{Alert: alert}, nil
	}).AnyTimes()

	mockAlertsClient.EXPECT().
		CreateAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *alerts.CreateAlertRequest) (*alerts.CreateAlertResponse, error) {
		alertExist = true
		return &alerts.CreateAlertResponse{Alert: alert}, nil
	}).AnyTimes()

	mockAlertsClient.EXPECT().
		GetAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *alerts.GetAlertByUniqueIdRequest) (*alerts.GetAlertByUniqueIdResponse, error) {
		if alertExist {
			return &alerts.GetAlertByUniqueIdResponse{Alert: alert}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	mockAlertsClient.EXPECT().
		DeleteAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *alerts.DeleteAlertByUniqueIdRequest) (*alerts.DeleteAlertByUniqueIdResponse, error) {
		if alertExist {
			alertExist = false
			return &alerts.DeleteAlertByUniqueIdResponse{}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	return mockAlertsClient
}

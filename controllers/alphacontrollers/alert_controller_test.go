package alphacontrollers

import (
	"context"
	"testing"

	utils "github.com/coralogix/coralogix-operator/apis"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	alerts "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/alerts/v2"
	"github.com/coralogix/coralogix-operator/controllers/mock_clientset"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func setupReconciler(t *testing.T, ctx context.Context, clientSet *mock_clientset.MockClientSetInterface) (AlertReconciler, watch.Interface) {
	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))

	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})

	go mgr.GetCache().Start(ctx)

	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})

	assert.NoError(t, err)
	r := AlertReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: clientSet,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.AlertList{})
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	return r, watcher
}

type PrepareParams struct {
	ctx            context.Context
	clientSet      *mock_clientset.MockClientSetInterface
	alertsClient   *mock_clientset.MockAlertsClientInterface
	webhooksClient *mock_clientset.MockWebhooksClientInterface
	alert          *coralogixv1alpha1.Alert
	remoteAlert    *alerts.Alert
}

func TestAlertCreation(t *testing.T) {
	defaultNotificationGroups := []coralogixv1alpha1.NotificationGroup{
		{
			Notifications: []coralogixv1alpha1.Notification{
				{
					RetriggeringPeriodMinutes: 10,
					NotifyOn:                  coralogixv1alpha1.NotifyOnTriggeredAndResolved,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
			},
		},
	}

	defaultAlertType := coralogixv1alpha1.AlertType{
		Metric: &coralogixv1alpha1.Metric{
			Promql: &coralogixv1alpha1.Promql{
				SearchQuery: "http_requests_total{status!~\"4..\"}",
				Conditions: coralogixv1alpha1.PromqlConditions{
					AlertWhen:                   "MoreThanUsual",
					Threshold:                   utils.FloatToQuantity(3.0),
					TimeWindow:                  "TwelveHours",
					MinNonNullValuesPercentage:  pointer.Int(10),
					ReplaceMissingValueWithZero: false,
				},
			},
		},
	}

	defaultRemoteNotificationGroups := []*alerts.AlertNotificationGroups{
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
	}

	defaultRemoteCondition := &alerts.AlertCondition{
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
	}

	tests := []struct {
		name        string
		prepare     func(params PrepareParams)
		alert       *coralogixv1alpha1.Alert
		remoteAlert *alerts.Alert
		shouldFail  bool
	}{
		{
			name:       "Alert creation success",
			shouldFail: false,
			alert: &coralogixv1alpha1.Alert{
				TypeMeta:   metav1.TypeMeta{Kind: "Alert", APIVersion: "coralogix.com/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: "alert-creation-success", Namespace: "default"},
				Spec: coralogixv1alpha1.AlertSpec{
					Name:               "AlertCreationSuccess",
					Description:        "AlertCreationSuccess",
					Active:             true,
					Severity:           alertProtoSeverityToSchemaSeverity[alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL],
					NotificationGroups: defaultNotificationGroups,
					PayloadFilters:     []string{"filter"},
					AlertType:          defaultAlertType,
				},
			},
			remoteAlert: &alerts.Alert{
				UniqueIdentifier: wrapperspb.String("AlertCreationSuccess"),
				Name:             wrapperspb.String("AlertCreationSuccess"),
				Description:      wrapperspb.String("AlertCreationSuccess"),
				IsActive:         wrapperspb.Bool(true),
				Severity:         alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL,
				MetaLabels: []*alerts.MetaLabel{
					{Key: wrapperspb.String("key"), Value: wrapperspb.String("value")},
					{Key: wrapperspb.String("managed-by"), Value: wrapperspb.String("coralogix-operator")},
				},
				Condition:          defaultRemoteCondition,
				NotificationGroups: defaultRemoteNotificationGroups,
				Filters: &alerts.AlertFilters{
					FilterType: alerts.AlertFilters_FILTER_TYPE_METRIC,
				},
				NotificationPayloadFilters: []*wrapperspb.StringValue{wrapperspb.String("filter")},
			},
			prepare: func(params PrepareParams) {

				params.alertsClient.EXPECT().
					GetAlert(params.alert.Namespace, coralogixv1alpha1.NewAlert()).
					Return(&alerts.GetAlertByUniqueIdResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)

				params.alertsClient.EXPECT().CreateAlert(params.ctx, gomock.Any()).
					Return(&alerts.CreateAlertResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			// Creating client set.
			clientSet := mock_clientset.NewMockClientSetInterface(controller)

			// Creating alerts client.
			alertsClient := mock_clientset.NewMockAlertsClientInterface(controller)

			// Creating webhooks client.
			webhooksClient := mock_clientset.NewMockWebhooksClientInterface(controller)

			// Preparing common mocks.
			clientSet.EXPECT().Alerts().MaxTimes(1).MinTimes(1).Return(alertsClient)
			clientSet.EXPECT().Webhooks().Return(webhooksClient).AnyTimes()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.prepare != nil {
				tt.prepare(PrepareParams{
					ctx:            ctx,
					clientSet:      clientSet,
					alertsClient:   alertsClient,
					webhooksClient: webhooksClient,
					alert:          tt.alert,
					remoteAlert:    tt.remoteAlert,
				})
			}

			reconciler, watcher := setupReconciler(t, ctx, clientSet)

			err := reconciler.Client.Create(ctx, tt.alert)

			assert.NoError(t, err)

			<-watcher.ResultChan()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.alert.Namespace,
					Name:      tt.alert.Name,
				},
			})

			if tt.shouldFail {
				assert.Error(t, err)
				assert.Equal(t, defaultErrRequeuePeriod, result.RequeueAfter)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, defaultRequeuePeriod, result.RequeueAfter)
			}
		})
	}

}

func TestAlertUpdate(t *testing.T) {
	defaultNotificationGroups := []coralogixv1alpha1.NotificationGroup{
		{
			Notifications: []coralogixv1alpha1.Notification{
				{
					RetriggeringPeriodMinutes: 10,
					NotifyOn:                  coralogixv1alpha1.NotifyOnTriggeredAndResolved,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
			},
		},
	}

	defaultAlertType := coralogixv1alpha1.AlertType{
		Metric: &coralogixv1alpha1.Metric{
			Promql: &coralogixv1alpha1.Promql{
				SearchQuery: "http_requests_total{status!~\"4..\"}",
				Conditions: coralogixv1alpha1.PromqlConditions{
					AlertWhen:                   "MoreThanUsual",
					Threshold:                   utils.FloatToQuantity(3.0),
					TimeWindow:                  "TwelveHours",
					MinNonNullValuesPercentage:  pointer.Int(10),
					ReplaceMissingValueWithZero: false,
				},
			},
		},
	}

	defaultRemoteNotificationGroups := []*alerts.AlertNotificationGroups{
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
	}

	defaultRemoteCondition := &alerts.AlertCondition{
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
	}

	tests := []struct {
		name        string
		prepare     func(params PrepareParams)
		alert       *coralogixv1alpha1.Alert
		remoteAlert *alerts.Alert
		shouldFail  bool
	}{
		{
			name:       "Alert update success",
			shouldFail: false,
			alert: &coralogixv1alpha1.Alert{
				TypeMeta:   metav1.TypeMeta{Kind: "Alert", APIVersion: "coralogix.com/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: "alert-update-success", Namespace: "default"},
				Spec: coralogixv1alpha1.AlertSpec{
					Name:               "AlertUpdateSuccess",
					Description:        "AlertUpdateSuccess",
					Active:             true,
					Severity:           alertProtoSeverityToSchemaSeverity[alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL],
					NotificationGroups: defaultNotificationGroups,
					PayloadFilters:     []string{"filter"},
					AlertType:          defaultAlertType,
				},
				Status: coralogixv1alpha1.AlertStatus{
					ID:          pointer.String("AlertUpdateSuccess"),
					Name:        "AlertUpdateSuccess",
					Description: "AlertUpdateSuccess",
					Active:      true,
					Severity:    "Critical",
				},
			},
			remoteAlert: &alerts.Alert{
				UniqueIdentifier: wrapperspb.String("AlertUpdateSuccess"),
				Name:             wrapperspb.String("AlertUpdateSuccess"),
				Description:      wrapperspb.String("AlertUpdateSuccess"),
				IsActive:         wrapperspb.Bool(true),
				Severity:         alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL,
				MetaLabels: []*alerts.MetaLabel{
					{Key: wrapperspb.String("key"), Value: wrapperspb.String("value")},
					{Key: wrapperspb.String("managed-by"), Value: wrapperspb.String("coralogix-operator")},
				},
				Condition:          defaultRemoteCondition,
				NotificationGroups: defaultRemoteNotificationGroups,
				Filters: &alerts.AlertFilters{
					FilterType: alerts.AlertFilters_FILTER_TYPE_METRIC,
				},
				NotificationPayloadFilters: []*wrapperspb.StringValue{wrapperspb.String("filter")},
			},
			prepare: func(params PrepareParams) {
				params.alertsClient.EXPECT().
					GetAlert(params.alert.Namespace, coralogixv1alpha1.NewAlert()).
					Return(&alerts.GetAlertByUniqueIdResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)

				params.alertsClient.EXPECT().CreateAlert(params.ctx, gomock.Any()).
					Return(&alerts.CreateAlertResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)

				params.alertsClient.EXPECT().UpdateAlert(params.ctx, gomock.Any()).
					Return(&alerts.UpdateAlertByUniqueIdResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)

				params.alertsClient.EXPECT().GetAlert(params.ctx, gomock.Any()).
					Return(&alerts.GetAlertByUniqueIdResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			// Creating client set.
			clientSet := mock_clientset.NewMockClientSetInterface(controller)

			// Creating alerts client.
			alertsClient := mock_clientset.NewMockAlertsClientInterface(controller)

			// Creating webhooks client.
			webhooksClient := mock_clientset.NewMockWebhooksClientInterface(controller)

			// Preparing common mocks.
			clientSet.EXPECT().Alerts().MaxTimes(1).MinTimes(1).Return(alertsClient)
			clientSet.EXPECT().Webhooks().Return(webhooksClient).AnyTimes()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.prepare != nil {
				tt.prepare(PrepareParams{
					ctx:            ctx,
					clientSet:      clientSet,
					alertsClient:   alertsClient,
					webhooksClient: webhooksClient,
					alert:          tt.alert,
					remoteAlert:    tt.remoteAlert,
				})
			}

			reconciler, watcher := setupReconciler(t, ctx, clientSet)

			err := reconciler.Client.Create(ctx, tt.alert)
			assert.NoError(t, err)

			<-watcher.ResultChan()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.alert.Namespace,
					Name:      tt.alert.Name,
				},
			})
			assert.NoError(t, err)

			currentAlert := &coralogixv1alpha1.Alert{}

			err = reconciler.Get(ctx, types.NamespacedName{
				Namespace: tt.alert.Namespace,
				Name:      tt.alert.Name,
			}, currentAlert)

			assert.NoError(t, err)

			err = reconciler.Client.Update(ctx, currentAlert)
			assert.NoError(t, err)

			result, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.alert.Namespace,
					Name:      tt.alert.Name,
				},
			})

			if tt.shouldFail {
				assert.Error(t, err)
				assert.Equal(t, defaultErrRequeuePeriod, result.RequeueAfter)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, defaultRequeuePeriod, result.RequeueAfter)
			}
		})
	}

}

func TestAlertDelete(t *testing.T) {
	defaultNotificationGroups := []coralogixv1alpha1.NotificationGroup{
		{
			Notifications: []coralogixv1alpha1.Notification{
				{
					RetriggeringPeriodMinutes: 10,
					NotifyOn:                  coralogixv1alpha1.NotifyOnTriggeredAndResolved,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
			},
		},
	}

	defaultAlertType := coralogixv1alpha1.AlertType{
		Metric: &coralogixv1alpha1.Metric{
			Promql: &coralogixv1alpha1.Promql{
				SearchQuery: "http_requests_total{status!~\"4..\"}",
				Conditions: coralogixv1alpha1.PromqlConditions{
					AlertWhen:                   "MoreThanUsual",
					Threshold:                   utils.FloatToQuantity(3.0),
					TimeWindow:                  "TwelveHours",
					MinNonNullValuesPercentage:  pointer.Int(10),
					ReplaceMissingValueWithZero: false,
				},
			},
		},
	}

	defaultRemoteNotificationGroups := []*alerts.AlertNotificationGroups{
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
	}

	defaultRemoteCondition := &alerts.AlertCondition{
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
	}

	tests := []struct {
		name        string
		prepare     func(params PrepareParams)
		alert       *coralogixv1alpha1.Alert
		remoteAlert *alerts.Alert
		shouldFail  bool
	}{
		{
			name:       "Alert delete success",
			shouldFail: false,
			alert: &coralogixv1alpha1.Alert{
				TypeMeta:   metav1.TypeMeta{Kind: "Alert", APIVersion: "coralogix.com/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: "alert-delete-success", Namespace: "default"},
				Spec: coralogixv1alpha1.AlertSpec{
					Name:               "AlertDeleteSuccess",
					Description:        "AlertDeleteSuccess",
					Active:             true,
					Severity:           alertProtoSeverityToSchemaSeverity[alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL],
					NotificationGroups: defaultNotificationGroups,
					PayloadFilters:     []string{"filter"},
					AlertType:          defaultAlertType,
				},
				Status: coralogixv1alpha1.AlertStatus{
					ID:          pointer.String("AlertDeleteSuccess"),
					Name:        "AlertDeleteSuccess",
					Description: "AlertDeleteSuccess",
					Active:      true,
					Severity:    "Critical",
				},
			},
			remoteAlert: &alerts.Alert{
				UniqueIdentifier: wrapperspb.String("AlertDeleteSuccess"),
				Name:             wrapperspb.String("AlertDeleteSuccess"),
				Description:      wrapperspb.String("AlertDeleteSuccess"),
				IsActive:         wrapperspb.Bool(true),
				Severity:         alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL,
				MetaLabels: []*alerts.MetaLabel{
					{Key: wrapperspb.String("key"), Value: wrapperspb.String("value")},
					{Key: wrapperspb.String("managed-by"), Value: wrapperspb.String("coralogix-operator")},
				},
				Condition:          defaultRemoteCondition,
				NotificationGroups: defaultRemoteNotificationGroups,
				Filters: &alerts.AlertFilters{
					FilterType: alerts.AlertFilters_FILTER_TYPE_METRIC,
				},
				NotificationPayloadFilters: []*wrapperspb.StringValue{wrapperspb.String("filter")},
			},
			prepare: func(params PrepareParams) {
				params.alertsClient.EXPECT().
					GetAlert(params.alert.Namespace, coralogixv1alpha1.NewAlert()).
					Return(&alerts.GetAlertByUniqueIdResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)

				params.alertsClient.EXPECT().CreateAlert(params.ctx, gomock.Any()).
					Return(&alerts.CreateAlertResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)

				params.alertsClient.EXPECT().DeleteAlert(params.ctx, gomock.Any()).
					Return(&alerts.DeleteAlertByUniqueIdResponse{}, nil).
					MinTimes(1).MaxTimes(1)

				params.alertsClient.EXPECT().GetAlert(params.ctx, gomock.Any()).
					Return(&alerts.GetAlertByUniqueIdResponse{Alert: params.remoteAlert}, nil).
					MinTimes(1).MaxTimes(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			// Creating client set.
			clientSet := mock_clientset.NewMockClientSetInterface(controller)

			// Creating alerts client.
			alertsClient := mock_clientset.NewMockAlertsClientInterface(controller)

			// Creating webhooks client.
			webhooksClient := mock_clientset.NewMockWebhooksClientInterface(controller)

			// Preparing common mocks.
			clientSet.EXPECT().Alerts().MaxTimes(1).MinTimes(1).Return(alertsClient)
			clientSet.EXPECT().Webhooks().Return(webhooksClient).AnyTimes()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.prepare != nil {
				tt.prepare(PrepareParams{
					ctx:            ctx,
					clientSet:      clientSet,
					alertsClient:   alertsClient,
					webhooksClient: webhooksClient,
					alert:          tt.alert,
					remoteAlert:    tt.remoteAlert,
				})
			}

			reconciler, watcher := setupReconciler(t, ctx, clientSet)

			err := reconciler.Client.Create(ctx, tt.alert)
			assert.NoError(t, err)

			<-watcher.ResultChan()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.alert.Namespace,
					Name:      tt.alert.Name,
				},
			})
			assert.NoError(t, err)

			currentAlert := &coralogixv1alpha1.Alert{}

			err = reconciler.Get(ctx, types.NamespacedName{
				Namespace: tt.alert.Namespace,
				Name:      tt.alert.Name,
			}, currentAlert)

			assert.NoError(t, err)

			err = reconciler.Client.Delete(ctx, currentAlert)
			assert.NoError(t, err)

			result, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.alert.Namespace,
					Name:      tt.alert.Name,
				},
			})

			if tt.shouldFail {
				assert.Error(t, err)
				assert.Equal(t, defaultErrRequeuePeriod, result.RequeueAfter)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, defaultRequeuePeriod, result.RequeueAfter)
			}
		})
	}

}

func TestFlattenAlerts(t *testing.T) {
	alert := &alerts.Alert{
		UniqueIdentifier: wrapperspb.String("id1"),
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
	status, err := getStatus(context.Background(), alert, spec)
	assert.NoError(t, err)

	expected := &coralogixv1alpha1.AlertStatus{
		ID:          pointer.String("id1"),
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

	assert.EqualValues(t, expected, &status)
}

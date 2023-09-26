package alphacontrollers

import (
	"context"
	"testing"
	"time"

	utils "github.com/coralogix/coralogix-operator/apis"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	alerts "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/alerts/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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
	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	assert.NoError(t, err)
	r := AlertReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: NewMockCoralogixClientSet(),
	}
	r.SetupWithManager(mgr)
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	go func() {
		err := mgr.Start(ctrl.SetupSignalHandler())
		assert.NoError(t, err)
	}()
	time.Sleep(2 * time.Second)

	alert := &coralogixv1alpha1.Alert{
		TypeMeta:   metav1.TypeMeta{Kind: "Alert", APIVersion: "coralogix.com/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: coralogixv1alpha1.AlertSpec{
			Name:        "name",
			Description: "description",
			Active:      true,
			Severity:    "Critical",
			Labels:      map[string]string{"key": "value"},
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
	ctx := context.Background()
	err = r.Client.Create(ctx, alert)
	time.Sleep(2 * time.Second)

	namespacedName := types.NamespacedName{Namespace: "default", Name: "test"}
	alertCRD := &coralogixv1alpha1.Alert{}
	err = r.Client.Get(ctx, namespacedName, alertCRD)
	assert.NoError(t, err)

	err = r.Client.Get(ctx, namespacedName, alertCRD)
	assert.NoError(t, err)

	id := alertCRD.Status.ID
	assert.NotNil(t, id)

	getAlertRequest := &alerts.GetAlertByUniqueIdRequest{Id: wrapperspb.String(*id)}
	_, err = r.CoralogixClientSet.Alerts().GetAlert(ctx, getAlertRequest)
	assert.NoError(t, err)

	r.Client.Delete(ctx, alert)
	time.Sleep(2 * time.Second)

	_, err = r.CoralogixClientSet.Alerts().GetAlert(ctx, getAlertRequest)
	assert.Error(t, err)
}

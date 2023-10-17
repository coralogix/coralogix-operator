package controllers

import (
	"context"
	"fmt"
	"log"
	"testing"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/alphacontrollers"
	alerts "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/alerts/v2"
	rrg "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/recording-rules-groups/v2"
	"github.com/coralogix/coralogix-operator/controllers/mock_clientset"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestPrometheusRuleReconciler_Reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockAlertsClient := createAlertsClientMock(mockCtrl)
	mockRecordingRuleGroupsClient := createRecordingRuleGroupClientMock(mockCtrl)
	mockWebhooksClient := mock_clientset.NewMockWebhooksClientInterface(mockCtrl)
	mockClientSet := mock_clientset.NewMockClientSetInterface(mockCtrl)
	mockClientSet.EXPECT().Alerts().Return(mockAlertsClient).AnyTimes()
	mockClientSet.EXPECT().RecordingRuleGroups().Return(mockRecordingRuleGroupsClient).AnyTimes()
	mockClientSet.EXPECT().Webhooks().Return(mockWebhooksClient).AnyTimes()
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	utilruntime.Must(prometheus.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.GetCache().Start(ctx)
	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme:     mgr.GetScheme(),
		HTTPClient: mgr.GetHTTPClient(),
		Mapper:     mgr.GetRESTMapper(),
		Cache:      &client.CacheOptions{Reader: mgr.GetCache()},
	})

	alertReconciler := alphacontrollers.AlertReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	alertReconciler.SetupWithManager(mgr)
	recordingRuleGroupReconciler := alphacontrollers.RecordingRuleGroupSetReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	recordingRuleGroupReconciler.SetupWithManager(mgr)
	prometheusRuleReconciler := PrometheusRuleReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	prometheusRuleReconciler.SetupWithManager(mgr)

	alertsWatcher, _ := alertReconciler.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.AlertList{})
	recordingRuleGroupWatcher, _ := recordingRuleGroupReconciler.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.RecordingRuleGroupSetList{})
	prometheusRulesWatcher, _ := prometheusRuleReconciler.Client.(client.WithWatch).Watch(ctx, &prometheus.PrometheusRuleList{})
	err = prometheusRuleReconciler.Client.Create(ctx, expectedPrometheusRuleCRD())
	assert.NoError(t, err)
	<-prometheusRulesWatcher.ResultChan()
	prometheusRuleCrd := &prometheus.PrometheusRule{}
	prometheusRuleReconciler.Client.Get(ctx, types.NamespacedName{Namespace: "default", Name: "test"}, prometheusRuleCrd)
	result, err := prometheusRuleReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, defaultRequeuePeriod, result.RequeueAfter)

	<-recordingRuleGroupWatcher.ResultChan()
	result, err = recordingRuleGroupReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, defaultRequeuePeriod, result.RequeueAfter)
	actualRecordingRuleGroupSetCRD := &coralogixv1alpha1.RecordingRuleGroupSet{}
	err = recordingRuleGroupReconciler.Client.Get(ctx, types.NamespacedName{Namespace: "default", Name: "test"}, actualRecordingRuleGroupSetCRD)
	assert.NoError(t, err)

	<-alertsWatcher.ResultChan()
	<-alertsWatcher.ResultChan()
	_, err = alertReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-app-latency-0"}})
	assert.NoError(t, err)
	_, err = alertReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-app-latency-1"}})
	assert.NoError(t, err)

	actualAlertCRD1 := &coralogixv1alpha1.Alert{}
	err = alertReconciler.Client.Get(ctx, types.NamespacedName{Namespace: "default", Name: "test-app-latency-0"}, actualAlertCRD1)
	assert.NoError(t, err)

	actualAlertCRD2 := &coralogixv1alpha1.Alert{}
	err = alertReconciler.Client.Get(ctx, types.NamespacedName{Namespace: "default", Name: "test-app-latency-1"}, actualAlertCRD2)
	assert.NoError(t, err)

	err = prometheusRuleReconciler.Client.Delete(ctx, prometheusRuleCrd)
	assert.NoError(t, err)
	<-prometheusRulesWatcher.ResultChan()
	result, err = prometheusRuleReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)

	assert.Equal(t, defaultRequeuePeriod, result.RequeueAfter)
	err = prometheusRuleReconciler.Client.Get(ctx, types.NamespacedName{Namespace: "default", Name: "test"}, prometheusRuleCrd)
	assert.Error(t, err)

	for event := <-recordingRuleGroupWatcher.ResultChan(); event.Type != watch.Deleted; event = <-recordingRuleGroupWatcher.ResultChan() {
		result, err = recordingRuleGroupReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
		assert.NoError(t, err)
	}

	rrg, err := recordingRuleGroupReconciler.CoralogixClientSet.RecordingRuleGroups().GetRecordingRuleGroupSet(ctx, &rrg.FetchRuleGroupSet{Id: *actualRecordingRuleGroupSetCRD.Status.ID})
	assert.Error(t, err)
	assert.Nil(t, rrg)

	for deletedAlertsEvents := 0; deletedAlertsEvents < 2; {
		event := <-alertsWatcher.ResultChan()
		result, err = alertReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: event.Object.(*coralogixv1alpha1.Alert).Name}})
		if event.Type == watch.Deleted {
			deletedAlertsEvents++
		}
		log.Print(fmt.Sprintf("#%v", event))
	}

	alert, err := alertReconciler.CoralogixClientSet.Alerts().GetAlert(ctx, &alerts.GetAlertByUniqueIdRequest{Id: wrapperspb.String(*actualAlertCRD1.Status.ID)})
	assert.Error(t, err)
	assert.Nil(t, alert)
	alert, err = alertReconciler.CoralogixClientSet.Alerts().GetAlert(ctx, &alerts.GetAlertByUniqueIdRequest{Id: wrapperspb.String(*actualAlertCRD2.Status.ID)})
	assert.Error(t, err)
	assert.Nil(t, alert)
}

func expectedPrometheusRuleCRD() *prometheus.PrometheusRule {
	return &prometheus.PrometheusRule{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels: map[string]string{
				"app.coralogix.com/track-recording-rules": "true",
				"app.coralogix.com/track-alerting-rules":  "true",
			},
		},
		Spec: prometheus.PrometheusRuleSpec{
			Groups: []prometheus.RuleGroup{
				{
					Name:     "test_1",
					Interval: "60s",
					Rules: []prometheus.Rule{
						{
							Record: "ExampleRecord",
							Expr:   intstr.FromString("vector(1)"),
						},
						{
							Record: "ExampleRecord2",
							Expr:   intstr.FromString("vector(2)"),
						},
						{
							Alert: "app-latency",
							Expr:  intstr.FromString("histogram_quantile(0.99, sum(irate(istio_request_duration_seconds_bucket{reporter=\"source\",destination_service=~\"ingress-annotation-test-svc.example-app.svc.cluster.local\"}[1m])) by (le, destination_workload)) > 0.2"),
							For:   "5m",
							Annotations: map[string]string{
								"cxMinNonNullValuesPercentage": "20",
							},
						},
					},
				},
				{
					Name:     "test_2",
					Interval: "70s",
					Rules: []prometheus.Rule{
						{
							Record: "ExampleRecord",
							Expr:   intstr.FromString("vector(3)"),
						},
						{
							Record: "ExampleRecord",
							Expr:   intstr.FromString("vector(4)"),
						},
						{
							Alert: "app-latency",
							Expr:  intstr.FromString("histogram_quantile(0.99, sum(irate(istio_request_duration_seconds_bucket{reporter=\"source\",destination_service=~\"ingress-annotation-test-svc.example-app.svc.cluster.local\"}[1m])) by (le, destination_workload)) > 0.2"),
							For:   "5m",
							Annotations: map[string]string{
								"cxMinNonNullValuesPercentage": "20",
							},
						},
					},
				},
			},
		},
	}
}

func createAlertsClientMock(mockCtrl *gomock.Controller) *mock_clientset.MockAlertsClientInterface {
	mockAlertsClient := mock_clientset.NewMockAlertsClientInterface(mockCtrl)
	alertsMap := make(map[string]*alerts.Alert)

	mockAlertsClient.EXPECT().
		CreateAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *alerts.CreateAlertRequest) (*alerts.CreateAlertResponse, error) {
		alert := flattenAlertCreateRequest(req)
		alertsMap[alert.UniqueIdentifier.GetValue()] = alert
		return &alerts.CreateAlertResponse{Alert: alert}, nil
	}).AnyTimes()

	mockAlertsClient.EXPECT().
		GetAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *alerts.GetAlertByUniqueIdRequest) (*alerts.GetAlertByUniqueIdResponse, error) {
		if alert, ok := alertsMap[req.GetId().GetValue()]; ok {
			return &alerts.GetAlertByUniqueIdResponse{Alert: alert}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, req.GetId().GetValue())
	}).AnyTimes()

	mockAlertsClient.EXPECT().
		DeleteAlert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *alerts.DeleteAlertByUniqueIdRequest) (*alerts.DeleteAlertByUniqueIdResponse, error) {
		if _, ok := alertsMap[req.GetId().GetValue()]; ok {
			delete(alertsMap, req.GetId().GetValue())
			return &alerts.DeleteAlertByUniqueIdResponse{}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, req.GetId().GetValue())
	}).AnyTimes()

	return mockAlertsClient
}

func flattenAlertCreateRequest(req *alerts.CreateAlertRequest) *alerts.Alert {
	return &alerts.Alert{
		Name:                       req.Name,
		Description:                req.Description,
		IsActive:                   req.IsActive,
		Severity:                   req.Severity,
		Expiration:                 req.Expiration,
		Condition:                  req.Condition,
		ShowInInsight:              req.ShowInInsight,
		NotificationGroups:         req.NotificationGroups,
		Filters:                    req.Filters,
		ActiveWhen:                 req.ActiveWhen,
		NotificationPayloadFilters: req.NotificationPayloadFilters,
		MetaLabels:                 req.MetaLabels,
		MetaLabelsStrings:          req.MetaLabelsStrings,
		TracingAlert:               req.TracingAlert,
		UniqueIdentifier:           wrapperspb.String(randomId()),
	}
}

func randomId() string {
	return fmt.Sprintf("%d", rand.Int())
}

func createRecordingRuleGroupClientMock(mockCtrl *gomock.Controller) *mock_clientset.MockRecordingRulesGroupsClientInterface {
	mockRecordingRuleGroupsClient := mock_clientset.NewMockRecordingRulesGroupsClientInterface(mockCtrl)
	recordingRuleGroupsMap := make(map[string]*rrg.OutRuleGroupSet)

	mockRecordingRuleGroupsClient.EXPECT().
		CreateRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *rrg.CreateRuleGroupSet) (*rrg.CreateRuleGroupSetResult, error) {
		id := randomId()
		recordingRuleGroupsMap[id] = &rrg.OutRuleGroupSet{
			Id:     id,
			Groups: inToOutRecordingRuleGroup(req.Groups),
		}
		return &rrg.CreateRuleGroupSetResult{Id: id}, nil
	}).AnyTimes()

	mockRecordingRuleGroupsClient.EXPECT().
		GetRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *rrg.FetchRuleGroupSet) (*rrg.OutRuleGroupSet, error) {
		if recordingRuleGroup, ok := recordingRuleGroupsMap[req.GetId()]; ok {
			return recordingRuleGroup, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, req.GetId())
	}).AnyTimes()

	mockRecordingRuleGroupsClient.EXPECT().
		DeleteRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *rrg.DeleteRuleGroupSet) (*emptypb.Empty, error) {
		if _, ok := recordingRuleGroupsMap[req.GetId()]; ok {
			delete(recordingRuleGroupsMap, req.GetId())
			return &emptypb.Empty{}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, req.GetId())
	}).AnyTimes()

	return mockRecordingRuleGroupsClient
}

func inToOutRecordingRuleGroup(in []*rrg.InRuleGroup) []*rrg.OutRuleGroup {
	out := make([]*rrg.OutRuleGroup, 0, len(in))
	for _, group := range in {
		out = append(out, &rrg.OutRuleGroup{
			Name:     group.Name,
			Interval: group.Interval,
			Limit:    group.Limit,
			Rules:    inToOutRecordingRule(group.Rules),
		})
	}
	return out
}

func inToOutRecordingRule(in []*rrg.InRule) []*rrg.OutRule {
	out := make([]*rrg.OutRule, len(in))
	for i, rule := range in {
		out[i] = &rrg.OutRule{
			Record: rule.Record,
			Expr:   rule.Expr,
			Labels: rule.Labels,
		}
	}
	return out
}

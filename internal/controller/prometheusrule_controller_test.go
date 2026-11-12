// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"fmt"
	"k8s.io/utils/ptr"
	"testing"
	"time"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crconfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

func setupReconciler(ctx context.Context, t *testing.T) (PrometheusRuleReconciler, watch.Interface) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	utilruntime.Must(prometheus.AddToScheme(scheme))
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	utilruntime.Must(coralogixv1beta1.AddToScheme(scheme))

	mgr, _ := ctrl.NewManager(testCfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
		Controller: crconfig.Controller{
			SkipNameValidation: ptr.To(true),
		},
	})

	go func() {
		if err := mgr.GetCache().Start(ctx); err != nil {
			t.Errorf("failed to start cache: %v", err)
			return
		}
	}()

	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme:     mgr.GetScheme(),
		HTTPClient: mgr.GetHTTPClient(),
		Mapper:     mgr.GetRESTMapper(),
		Cache:      &client.CacheOptions{Reader: mgr.GetCache()},
	})
	config.InitClient(mgr.GetClient())
	config.InitScheme(mgr.GetScheme())

	assert.NoError(t, err)
	r := PrometheusRuleReconciler{}
	err = r.SetupWithManager(mgr)
	assert.NoError(t, err)

	watcher, _ := withWatch.Watch(ctx, &prometheus.PrometheusRuleList{})
	return r, watcher
}

func TestPrometheusRulesConversionToCxParsingRules(t *testing.T) {
	tests := []struct {
		name           string
		prometheusRule *prometheus.PrometheusRule
		shouldFail     bool
		shouldCreate   bool
	}{
		{
			name:         "PrometheusRule with empty groups",
			shouldFail:   false,
			shouldCreate: true,
			prometheusRule: &prometheus.PrometheusRule{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
					Labels: map[string]string{
						utils.TrackPrometheusRuleRecordingRulesLabelKey: "true",
					},
				},
				Spec: prometheus.PrometheusRuleSpec{
					Groups: []prometheus.RuleGroup{},
				},
			},
		},
		{
			name:         "New PrometheusRule with one group and one rule",
			shouldFail:   false,
			shouldCreate: true,
			prometheusRule: &prometheus.PrometheusRule{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "new-with-rules",
					Namespace: "default",
					Labels: map[string]string{
						utils.TrackPrometheusRuleRecordingRulesLabelKey: "true",
					},
				},
				Spec: prometheus.PrometheusRuleSpec{
					Groups: []prometheus.RuleGroup{
						{
							Name:     "test_1",
							Interval: prometheus.DurationPointer("60s"),
							Rules: []prometheus.Rule{
								{
									Record: "ExampleRecord",
									Expr:   intstr.FromString("vector(1)"),
								},
							},
						},
					},
				},
			},
		},
		{
			name:       "Existing PrometheusRule with one group and one rule",
			shouldFail: false,
			prometheusRule: &prometheus.PrometheusRule{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "new-with-rules",
					Namespace: "default",
					Labels: map[string]string{
						utils.TrackPrometheusRuleRecordingRulesLabelKey: "true",
					},
				},
				Spec: prometheus.PrometheusRuleSpec{
					Groups: []prometheus.RuleGroup{
						{
							Name:     "test_1",
							Interval: prometheus.DurationPointer("60s"),
							Rules: []prometheus.Rule{
								{
									Record: "ExampleRecord",
									Expr:   intstr.FromString("vector(1)"),
								},
								{
									Record: "ExampleRecord",
									Expr:   intstr.FromString("vector(2)"),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			reconciler, watcher := setupReconciler(ctx, t)

			var err error
			if tt.shouldCreate {
				err = config.GetClient().Create(ctx, tt.prometheusRule)
				assert.NoError(t, err)
			}

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.prometheusRule.Namespace,
					Name:      tt.prometheusRule.Name,
				},
			})

			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrometheusRulesConvertionToCxAlert(t *testing.T) {
	tests := []struct {
		name           string
		prometheusRule *prometheus.PrometheusRule
		shouldFail     bool
		shouldCreate   bool
	}{
		{
			name:         "PrometheusRule with empty groups",
			shouldFail:   false,
			shouldCreate: true,
			prometheusRule: &prometheus.PrometheusRule{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test-alert",
					Namespace: "default",
					Labels: map[string]string{
						utils.TrackPrometheusRuleAlertsLabelKey: "true",
					},
				},
				Spec: prometheus.PrometheusRuleSpec{
					Groups: []prometheus.RuleGroup{},
				},
			},
		},
		{
			name:         "New PrometheusRule with one group and one rule",
			shouldFail:   false,
			shouldCreate: true,
			prometheusRule: &prometheus.PrometheusRule{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "new-with-alerting-rules",
					Namespace: "default",
					Labels: map[string]string{
						utils.TrackPrometheusRuleAlertsLabelKey: "true",
					},
				},
				Spec: prometheus.PrometheusRuleSpec{
					Groups: []prometheus.RuleGroup{
						{
							Name:     "test_1",
							Interval: prometheus.DurationPointer("60s"),
							Rules: []prometheus.Rule{
								{
									Alert: "ExampleAlert",
									Expr:  intstr.FromString("vector(1)"),
								},
							},
						},
					},
				},
			},
		},
		{
			name:       "Existing PrometheusRule with one group and one rule",
			shouldFail: false,
			prometheusRule: &prometheus.PrometheusRule{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "new-with-alerting-rules",
					Namespace: "default",
					Labels: map[string]string{
						utils.TrackPrometheusRuleAlertsLabelKey: "true",
					},
				},
				Spec: prometheus.PrometheusRuleSpec{
					Groups: []prometheus.RuleGroup{
						{
							Name:     "test_1",
							Interval: prometheus.DurationPointer("60s"),
							Rules: []prometheus.Rule{
								{
									Alert: "ExampleAlert",
									Expr:  intstr.FromString("vector(1)"),
								},
								{
									Alert: "ExampleAlert",
									Expr:  intstr.FromString("vector(2)"),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			reconciler, watcher := setupReconciler(ctx, t)
			var err error
			if tt.shouldCreate {
				err = config.GetClient().Create(ctx, tt.prometheusRule)
				assert.NoError(t, err)
			}

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.prometheusRule.Namespace,
					Name:      tt.prometheusRule.Name,
				},
			})

			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrometheusRuleUpdatePreservesAlertAccessPolicy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reconciler, watcher := setupReconciler(ctx, t)

	prometheusRule := &prometheus.PrometheusRule{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "access-policy-preservation",
			Namespace: "default",
			Labels: map[string]string{
				utils.TrackPrometheusRuleAlertsLabelKey: "true",
			},
		},
		Spec: prometheus.PrometheusRuleSpec{
			Groups: []prometheus.RuleGroup{
				{
					Name:     "test_1",
					Interval: prometheus.DurationPointer("60s"),
					Rules: []prometheus.Rule{
						{
							Alert: "ExampleAlert",
							Expr:  intstr.FromString("vector(1)"),
						},
					},
				},
			},
		},
	}
	assert.NoError(t, config.GetClient().Create(ctx, prometheusRule))
	<-watcher.ResultChan()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: prometheusRule.Namespace,
			Name:      prometheusRule.Name,
		},
	})
	assert.NoError(t, err)

	getManagedAlert := func() *coralogixv1beta1.Alert {
		alertList := &coralogixv1beta1.AlertList{}
		if err := config.GetClient().List(ctx, alertList, client.InNamespace("default")); err != nil {
			return nil
		}
		for i := range alertList.Items {
			if alertList.Items[i].Labels[managedByLabelKey] == prometheusRule.Name {
				return &alertList.Items[i]
			}
		}
		return nil
	}

	var generated *coralogixv1beta1.Alert
	for i := 0; i < 50; i++ {
		if generated = getManagedAlert(); generated != nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if generated == nil {
		t.Fatal("generated Alert was not created")
	}

	// Registered as t.Cleanup (not inline at the end) so a fail-fast above
	// cannot skip it. Best-effort: the shared envtest binary is left clean.
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		alertList := &coralogixv1beta1.AlertList{}
		if err := config.GetClient().List(cleanupCtx, alertList, client.InNamespace("default")); err == nil {
			for i := range alertList.Items {
				if alertList.Items[i].Labels[managedByLabelKey] != prometheusRule.Name {
					continue
				}
				item := &coralogixv1beta1.Alert{}
				if err := config.GetClient().Get(cleanupCtx, types.NamespacedName{
					Namespace: alertList.Items[i].Namespace,
					Name:      alertList.Items[i].Name,
				}, item); err != nil {
					continue
				}
				// No Alert controller runs in this test, so nothing would
				// remove the generated Alert's finalizer.
				item.Finalizers = nil
				_ = config.GetClient().Update(cleanupCtx, item)
				_ = config.GetClient().Delete(cleanupCtx, item)
			}
		}
		fr := &prometheus.PrometheusRule{}
		if err := config.GetClient().Get(cleanupCtx, types.NamespacedName{
			Namespace: prometheusRule.Namespace,
			Name:      prometheusRule.Name,
		}, fr); err == nil {
			fr.Finalizers = nil
			_ = config.GetClient().Update(cleanupCtx, fr)
			_ = config.GetClient().Delete(cleanupCtx, fr)
		}
	})

	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`
	generated.Spec.AccessPolicy = ptr.To(policy)
	assert.NoError(t, config.GetClient().Update(ctx, generated))

	freshRule := &prometheus.PrometheusRule{}
	assert.NoError(t, config.GetClient().Get(ctx, types.NamespacedName{
		Namespace: prometheusRule.Namespace,
		Name:      prometheusRule.Name,
	}, freshRule))
	freshRule.Spec.Groups[0].Rules[0].Annotations = map[string]string{"description": "updated description"}
	assert.NoError(t, config.GetClient().Update(ctx, freshRule))
	<-watcher.ResultChan()

	// Reconcile until the rule update is visible through the manager cache.
	var updated *coralogixv1beta1.Alert
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, err = reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: prometheusRule.Namespace,
				Name:      prometheusRule.Name,
			},
		})
		require.NoError(t, err)
		if updated = getManagedAlert(); updated != nil && updated.Spec.Description == "updated description" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("generated Alert description was not updated")
		}
		time.Sleep(500 * time.Millisecond)
	}
	if updated.Spec.AccessPolicy == nil || *updated.Spec.AccessPolicy != policy {
		t.Fatalf("generated Alert access policy was not preserved: %v", updated.Spec.AccessPolicy)
	}
}

func newAlertingPrometheusRule(name string, annotations map[string]string) *prometheus.PrometheusRule {
	return &prometheus.PrometheusRule{
		ObjectMeta: ctrl.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Annotations: annotations,
			Labels: map[string]string{
				utils.TrackPrometheusRuleAlertsLabelKey: "true",
			},
		},
		Spec: prometheus.PrometheusRuleSpec{
			Groups: []prometheus.RuleGroup{
				{
					Name:     "test_1",
					Interval: prometheus.DurationPointer("60s"),
					Rules: []prometheus.Rule{
						{
							Alert:       "ExampleAlert",
							Expr:        intstr.FromString("vector(1)"),
							Annotations: map[string]string{"description": "initial"},
						},
					},
				},
			},
		},
	}
}

// uncachedClient reads straight from the API server. Assertions use it so they see the
// result of the Reconcile that just ran, not a later one: the manager cache can lag behind
// writes, and a Reconcile that finds the Alert already created takes the update path.
func uncachedClient(t *testing.T) client.Client {
	t.Helper()
	c, err := client.New(testCfg, client.Options{Scheme: config.GetScheme()})
	require.NoError(t, err)
	return c
}

// generatedAlertKey names the Alert generated from the first rule of newAlertingPrometheusRule.
func generatedAlertKey(rule *prometheus.PrometheusRule) types.NamespacedName {
	return types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name + "-examplealert-0"}
}

// reconcilePrometheusRule reconciles rule until the Alert generated from its first rule
// satisfies done, or fails the test after 30s. Reconcile reads the PrometheusRule through
// the manager cache, which can lag behind writes, hence the retries.
func reconcilePrometheusRule(ctx context.Context, t *testing.T, reconciler PrometheusRuleReconciler,
	rule *prometheus.PrometheusRule, done func(*coralogixv1beta1.Alert) bool) *coralogixv1beta1.Alert {
	t.Helper()
	reader := uncachedClient(t)
	key := types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}
	alertKey := generatedAlertKey(rule)
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
		require.NoError(t, err)
		alert := &coralogixv1beta1.Alert{}
		if err := reader.Get(ctx, alertKey, alert); err == nil && done(alert) {
			return alert
		}
		if time.Now().After(deadline) {
			t.Fatalf("generated Alert %s did not reach the expected state", alertKey.Name)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// cleanupPrometheusRule removes rule and its generated Alerts. No Alert controller runs in
// these tests, so nothing else would remove the Alerts' finalizers.
func cleanupPrometheusRule(t *testing.T, rule *prometheus.PrometheusRule) {
	t.Cleanup(func() {
		ctx := context.Background()
		alertList := &coralogixv1beta1.AlertList{}
		if err := config.GetClient().List(ctx, alertList, client.InNamespace(rule.Namespace),
			client.MatchingLabels{managedByLabelKey: rule.Name}); err == nil {
			for i := range alertList.Items {
				item := &alertList.Items[i]
				item.Finalizers = nil
				_ = config.GetClient().Update(ctx, item)
				_ = config.GetClient().Delete(ctx, item)
			}
		}
		fresh := &prometheus.PrometheusRule{}
		if err := config.GetClient().Get(ctx, types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, fresh); err == nil {
			fresh.Finalizers = nil
			_ = config.GetClient().Update(ctx, fresh)
			_ = config.GetClient().Delete(ctx, fresh)
		}
	})
}

func TestPrometheusRuleAlertRouterAnnotationOnCreate(t *testing.T) {
	tests := []struct {
		name         string
		annotations  map[string]string
		wantNotifyOn *coralogixv1beta1.NotifyOn
	}{
		{
			name:         "annotation routes generated Alerts through the router",
			annotations:  map[string]string{utils.PrometheusRuleAlertRouterNotifyOnAnnotationKey: "triggeredAndResolved"},
			wantNotifyOn: ptr.To(coralogixv1beta1.NotifyOnTriggeredAndResolved),
		},
		{
			name: "no annotation leaves the notification group unset",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			reconciler, watcher := setupReconciler(ctx, t)

			rule := newAlertingPrometheusRule(fmt.Sprintf("router-on-create-%d", i), tt.annotations)
			require.NoError(t, config.GetClient().Create(ctx, rule))
			cleanupPrometheusRule(t, rule)
			<-watcher.ResultChan()

			alert := reconcilePrometheusRule(ctx, t, reconciler, rule, func(*coralogixv1beta1.Alert) bool { return true })

			if tt.wantNotifyOn == nil {
				assert.Nil(t, alert.Spec.NotificationGroup)
				return
			}
			require.NotNil(t, alert.Spec.NotificationGroup)
			require.NotNil(t, alert.Spec.NotificationGroup.Router)
			assert.Equal(t, *tt.wantNotifyOn, alert.Spec.NotificationGroup.Router.NotifyOn)
			assert.Equal(t, coralogixv1beta1.NotificationRouterModeRoutingLabels, alert.Spec.NotificationGroup.Router.Mode)
		})
	}
}

func TestPrometheusRuleAlertRouterAnnotationAddedLaterKeepsGroupByKeys(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cleanups run last-in first-out: cancelling first would stop the cache the
	// cleanup reads through.
	t.Cleanup(cancel)
	reconciler, watcher := setupReconciler(ctx, t)

	rule := newAlertingPrometheusRule("router-added-later", nil)
	require.NoError(t, config.GetClient().Create(ctx, rule))
	cleanupPrometheusRule(t, rule)
	<-watcher.ResultChan()

	generated := reconcilePrometheusRule(ctx, t, reconciler, rule, func(*coralogixv1beta1.Alert) bool { return true })
	generated.Spec.NotificationGroup = &coralogixv1beta1.NotificationGroup{GroupByKeys: []string{"service"}}
	require.NoError(t, config.GetClient().Update(ctx, generated))

	fresh := &prometheus.PrometheusRule{}
	require.NoError(t, config.GetClient().Get(ctx, types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, fresh))
	fresh.Annotations = map[string]string{utils.PrometheusRuleAlertRouterNotifyOnAnnotationKey: "triggeredOnly"}
	require.NoError(t, config.GetClient().Update(ctx, fresh))

	updated := reconcilePrometheusRule(ctx, t, reconciler, rule, func(a *coralogixv1beta1.Alert) bool {
		return a.Spec.NotificationGroup != nil && a.Spec.NotificationGroup.Router != nil
	})
	assert.Equal(t, coralogixv1beta1.NotifyOnTriggeredOnly, updated.Spec.NotificationGroup.Router.NotifyOn)
	assert.Equal(t, coralogixv1beta1.NotificationRouterModeRoutingLabels, updated.Spec.NotificationGroup.Router.Mode)
	assert.Equal(t, []string{"service"}, updated.Spec.NotificationGroup.GroupByKeys)
}

func TestPrometheusRuleAlertRouterAnnotationRejectsUnknownValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cleanups run last-in first-out: cancelling first would stop the cache the
	// cleanup reads through.
	t.Cleanup(cancel)
	reconciler, watcher := setupReconciler(ctx, t)

	rule := newAlertingPrometheusRule("router-invalid-value",
		map[string]string{utils.PrometheusRuleAlertRouterNotifyOnAnnotationKey: "always"})
	require.NoError(t, config.GetClient().Create(ctx, rule))
	cleanupPrometheusRule(t, rule)
	<-watcher.ResultChan()

	var err error
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}})
		if err != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.Error(t, err)
	assert.Contains(t, err.Error(), utils.PrometheusRuleAlertRouterNotifyOnAnnotationKey)
	assert.Contains(t, err.Error(), `"always"`)
}

func TestPrometheusRuleRoutingLabelsBecomeAlertEntityLabels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	reconciler, watcher := setupReconciler(ctx, t)

	rule := newAlertingPrometheusRule("routing-labels", nil)
	rule.Labels["routing.environment"] = "production"
	rule.Labels["team"] = "not-a-routing-label"
	rule.Spec.Groups[0].Rules[0].Labels = map[string]string{"severity": "warning"}
	require.NoError(t, config.GetClient().Create(ctx, rule))
	cleanupPrometheusRule(t, rule)
	<-watcher.ResultChan()

	created := reconcilePrometheusRule(ctx, t, reconciler, rule, func(*coralogixv1beta1.Alert) bool { return true })
	assert.Equal(t, map[string]string{"severity": "warning", "routing.environment": "production"}, created.Spec.EntityLabels)

	fresh := &prometheus.PrometheusRule{}
	require.NoError(t, config.GetClient().Get(ctx, types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, fresh))
	fresh.Labels["routing.team"] = "platform"
	require.NoError(t, config.GetClient().Update(ctx, fresh))

	updated := reconcilePrometheusRule(ctx, t, reconciler, rule, func(a *coralogixv1beta1.Alert) bool {
		return a.Spec.EntityLabels["routing.team"] == "platform"
	})
	assert.Equal(t, map[string]string{
		"severity":            "warning",
		"routing.environment": "production",
		"routing.team":        "platform",
	}, updated.Spec.EntityLabels)
}

func TestPrometheusRuleInvalidRouterAnnotationStillSyncsAlerts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	reconciler, watcher := setupReconciler(ctx, t)

	rule := newAlertingPrometheusRule("router-invalid-still-syncs", nil)
	require.NoError(t, config.GetClient().Create(ctx, rule))
	cleanupPrometheusRule(t, rule)
	<-watcher.ResultChan()
	reconcilePrometheusRule(ctx, t, reconciler, rule, func(*coralogixv1beta1.Alert) bool { return true })

	fresh := &prometheus.PrometheusRule{}
	require.NoError(t, config.GetClient().Get(ctx, types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, fresh))
	fresh.Annotations = map[string]string{utils.PrometheusRuleAlertRouterNotifyOnAnnotationKey: "always"}
	fresh.Spec.Groups[0].Rules[0].Annotations["description"] = "updated"
	require.NoError(t, config.GetClient().Update(ctx, fresh))

	// The invalid value fails the reconcile, but the rule's other changes still reach its Alert.
	reader := uncachedClient(t)
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}})
		alert := &coralogixv1beta1.Alert{}
		require.NoError(t, reader.Get(ctx, generatedAlertKey(rule), alert))
		if err != nil && alert.Spec.Description == "updated" {
			assert.Contains(t, err.Error(), utils.PrometheusRuleAlertRouterNotifyOnAnnotationKey)
			assert.Nil(t, alert.Spec.NotificationGroup)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("want a reconcile error and the updated description; got error %v, description %q", err, alert.Spec.Description)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func TestPrometheusRuleRouterAnnotationSkipsPhantomAlerts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	reconciler, watcher := setupReconciler(ctx, t)

	rule := newAlertingPrometheusRule("router-phantom", nil)
	require.NoError(t, config.GetClient().Create(ctx, rule))
	cleanupPrometheusRule(t, rule)
	<-watcher.ResultChan()

	generated := reconcilePrometheusRule(ctx, t, reconciler, rule, func(*coralogixv1beta1.Alert) bool { return true })
	generated.Spec.PhantomMode = true
	require.NoError(t, config.GetClient().Update(ctx, generated))

	fresh := &prometheus.PrometheusRule{}
	require.NoError(t, config.GetClient().Get(ctx, types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, fresh))
	fresh.Annotations = map[string]string{utils.PrometheusRuleAlertRouterNotifyOnAnnotationKey: "triggeredOnly"}
	fresh.Spec.Groups[0].Rules[0].Annotations["description"] = "updated"
	require.NoError(t, config.GetClient().Update(ctx, fresh))

	// A phantom Alert can't have a notification group, so it gets no router, and the rule's
	// other changes still apply.
	updated := reconcilePrometheusRule(ctx, t, reconciler, rule, func(a *coralogixv1beta1.Alert) bool {
		return a.Spec.Description == "updated"
	})
	assert.True(t, updated.Spec.PhantomMode)
	assert.Nil(t, updated.Spec.NotificationGroup)
}

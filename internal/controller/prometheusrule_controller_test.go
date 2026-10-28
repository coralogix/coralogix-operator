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

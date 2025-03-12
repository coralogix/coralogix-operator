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
	"testing"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

func setupReconciler(t *testing.T, ctx context.Context) (PrometheusRuleReconciler, watch.Interface) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	utilruntime.Must(prometheus.AddToScheme(scheme))
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	utilruntime.Must(coralogixv1beta1.AddToScheme(scheme))

	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})

	go mgr.GetCache().Start(ctx)

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
	r.SetupWithManager(mgr)

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
							Interval: "60s",
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
							Interval: "60s",
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

			reconciler, watcher := setupReconciler(t, ctx)

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
							Interval: "60s",
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
							Interval: "60s",
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

			reconciler, watcher := setupReconciler(t, ctx)
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

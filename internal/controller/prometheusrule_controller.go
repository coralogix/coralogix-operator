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
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"go.uber.org/zap/zapcore"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
)

const (
	defaultCoralogixNotificationPeriod int32 = 5
)

//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch

//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/finalizers,verbs=update

//+kubebuilder:rbac:groups=coralogix.com,resources=alerts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=alerts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=alerts/finalizers,verbs=update

// PrometheusRuleReconciler reconciles a PrometheusRule object
type PrometheusRuleReconciler struct {
	client.Client
	CoralogixClientSet clientset.ClientSetInterface
	Scheme             *runtime.Scheme
}

func (r *PrometheusRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	prometheusRule := &prometheus.PrometheusRule{}
	if err := r.Get(ctx, req.NamespacedName, prometheusRule); err != nil {
		if k8serrors.IsNotFound(err) {
			monitoring.PrometheusRuleInfoMetric.DeleteLabelValues(req.Name, req.Namespace)
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}
	monitoring.PrometheusRuleInfoMetric.WithLabelValues(prometheusRule.Name, prometheusRule.Namespace).Set(1)

	if shouldTrackRecordingRules(prometheusRule) {
		err := r.convertPrometheusRuleRecordingRuleToCxRecordingRule(ctx, log, prometheusRule, req)
		if err != nil {
			log.Error(err, "Received an error while trying to convert PrometheusRule to RecordingRule CRD")
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
	}

	if shouldTrackAlerts(prometheusRule) {
		err := r.convertPrometheusRuleAlertToCxAlert(ctx, prometheusRule)
		if err != nil {
			log.Error(err, "Received an error while trying to convert PrometheusRule to Alert CRD")
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *PrometheusRuleReconciler) convertPrometheusRuleRecordingRuleToCxRecordingRule(ctx context.Context, log logr.Logger, prometheusRule *prometheus.PrometheusRule, req reconcile.Request) error {
	recordingRuleGroupSetSpec := prometheusRuleToRecordingRuleToRuleGroupSet(log, prometheusRule)
	if len(recordingRuleGroupSetSpec.Groups) == 0 {
		return nil
	}

	recordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       prometheusRule.Namespace,
			Name:            prometheusRule.Name,
			OwnerReferences: []metav1.OwnerReference{getOwnerReference(prometheusRule)},
		},
		Spec: recordingRuleGroupSetSpec,
	}

	if err := r.Client.Get(ctx, req.NamespacedName, recordingRuleGroupSet); err != nil {
		if k8serrors.IsNotFound(err) {
			if err = r.Create(ctx, recordingRuleGroupSet); err != nil {
				return fmt.Errorf("received an error while trying to create RecordingRuleGroupSet CRD: %w", err)
			}
			return nil
		}

		return fmt.Errorf("received an error while trying to get RecordingRuleGroupSet CRD: %w", err)
	}

	recordingRuleGroupSet.Spec = recordingRuleGroupSetSpec
	if err := r.Client.Update(ctx, recordingRuleGroupSet); err != nil {
		return fmt.Errorf("received an error while trying to update RecordingRuleGroupSet CRD: %w", err)
	}

	return nil
}

func (r *PrometheusRuleReconciler) convertPrometheusRuleAlertToCxAlert(ctx context.Context, prometheusRule *prometheus.PrometheusRule) error {
	// A single PrometheusRule can have multiple alerts with the same name, while the Alert CRD from coralogix can only manage one alert.
	// alertMap is used to map an alert name with potentially multiple alerts from the promrule CRD. For example:
	//
	// A prometheusRule with the following rules:
	// rules:
	//   - alert: Example
	//     expr: metric > 10
	//   - alert: Example
	//     expr: metric > 20
	//
	// Would be mapped into:
	//   map[string][]prometheus.Rule{
	// 	   "Example": []prometheus.Rule{
	// 		 {
	//          Alert: Example,
	//          Expr: "metric > 10"
	// 		 },
	// 		 {
	//          Alert: Example,
	//          Expr: "metric > 100"
	// 		 },
	// 	   },
	//   }
	//
	// To later on generate coralogix Alert CRDs using the alert name followed by it's index on the array, making sure we don't clash names.
	alertMap := make(map[string][]prometheus.Rule)
	var a string
	for _, group := range prometheusRule.Spec.Groups {
		for _, rule := range group.Rules {
			if rule.Alert != "" {
				a = strings.ToLower(rule.Alert)
				if _, ok := alertMap[a]; !ok {
					alertMap[a] = []prometheus.Rule{rule}
					continue
				}
				alertMap[a] = append(alertMap[a], rule)
			}
		}
	}

	alertsToKeep := make(map[string]bool)
	var errorsEncountered []error
	for alertName, rules := range alertMap {
		for i, rule := range rules {
			alert := &coralogixv1alpha1.Alert{}
			alertName := fmt.Sprintf("%s-%s-%d", prometheusRule.Name, alertName, i)
			alertsToKeep[alertName] = true
			if err := r.Client.Get(ctx, client.ObjectKey{Namespace: prometheusRule.Namespace, Name: alertName}, alert); err != nil {
				if k8serrors.IsNotFound(err) {
					alert.Name = alertName
					alert.Namespace = prometheusRule.Namespace
					alert.Labels = map[string]string{"app.kubernetes.io/managed-by": prometheusRule.Name}
					if val, ok := prometheusRule.Labels["app.coralogix.com/managed-by-alertmanager-config"]; ok {
						alert.Labels["app.coralogix.com/managed-by-alertmanager-config"] = val
					}
					alert.OwnerReferences = []metav1.OwnerReference{getOwnerReference(prometheusRule)}
					alert.Spec.Name = rule.Alert
					alert.Spec.Description = rule.Annotations["description"]
					alert.Spec.Labels = rule.Labels
					alert.Spec.Severity = getSeverity(rule)
					alert.Spec.AlertType = coralogixv1alpha1.AlertType{
						Metric: &coralogixv1alpha1.Metric{
							Promql: prometheusAlertToPromqlTypeSpec(rule),
						},
					}
					if err = r.Create(ctx, alert); err != nil {
						errorsEncountered = append(errorsEncountered, fmt.Errorf("error creating Alert CRD %s: %w", alertName, err))
					}
					continue
				} else {
					errorsEncountered = append(errorsEncountered, fmt.Errorf("error getting Alert CRD %s: %w", alertName, err))
					continue
				}
			}

			updated := false
			desiredDescription := rule.Annotations["description"]
			if alert.Spec.Description != desiredDescription {
				alert.Spec.Description = desiredDescription
				updated = true
			}

			desiredLabels := rule.Labels
			if !reflect.DeepEqual(alert.Spec.Labels, desiredLabels) {
				alert.Spec.Labels = desiredLabels
				updated = true
			}

			desiredSeverity := getSeverity(rule)
			if alert.Spec.Severity != desiredSeverity {
				alert.Spec.Severity = desiredSeverity
				updated = true
			}

			desiredAlertTypeSpec := coralogixv1alpha1.AlertType{
				Metric: &coralogixv1alpha1.Metric{
					Promql: prometheusAlertToPromqlTypeSpec(rule),
				},
			}
			desiredAlertTypeSpec.Metric.Promql.Conditions.MinNonNullValuesPercentage = alert.Spec.AlertType.Metric.Promql.Conditions.MinNonNullValuesPercentage
			if !reflect.DeepEqual(alert.Spec.AlertType, desiredAlertTypeSpec) {
				alert.Spec.AlertType = desiredAlertTypeSpec
				updated = true
			}

			desiredOwnerReferences := []metav1.OwnerReference{getOwnerReference(prometheusRule)}
			if !reflect.DeepEqual(alert.OwnerReferences, desiredOwnerReferences) {
				alert.OwnerReferences = desiredOwnerReferences
				updated = true
			}

			if promRuleVal, ok := prometheusRule.Labels["app.coralogix.com/managed-by-alertmanager-config"]; ok {
				if alertVal, ok := alert.Labels["app.coralogix.com/managed-by-alertmanager-config"]; !ok || alertVal != promRuleVal {
					alert.Labels["app.coralogix.com/managed-by-alertmanager-config"] = promRuleVal
					updated = true
				}
			}

			if updated {
				if err := r.Update(ctx, alert); err != nil {
					errorsEncountered = append(errorsEncountered, fmt.Errorf("error updating Alert CRD %s: %w", alertName, err))
				}
			}
		}
	}

	var childAlerts coralogixv1alpha1.AlertList
	if err := r.List(ctx, &childAlerts, client.InNamespace(prometheusRule.Namespace), client.MatchingLabels{"app.kubernetes.io/managed-by": prometheusRule.Name}); err != nil {
		return fmt.Errorf("received an error while trying to list Alerts: %w", err)
	}

	for _, alert := range childAlerts.Items {
		if !alertsToKeep[alert.Name] {
			if err := r.Delete(ctx, &alert); err != nil {
				errorsEncountered = append(errorsEncountered, fmt.Errorf("error deleting Alert CRD %s: %w", alert.Name, err))
			}
		}
	}

	if len(errorsEncountered) > 0 {
		return errors.Join(errorsEncountered...)
	}
	return nil
}

func shouldTrackRecordingRules(prometheusRule *prometheus.PrometheusRule) bool {
	if value, ok := prometheusRule.Labels["app.coralogix.com/track-recording-rules"]; ok && value == "true" {
		return true
	}
	return false
}

func shouldTrackAlerts(prometheusRule *prometheus.PrometheusRule) bool {
	if value, ok := prometheusRule.Labels["app.coralogix.com/track-alerting-rules"]; ok && value == "true" {
		return true
	}
	return false
}

func prometheusRuleToRecordingRuleToRuleGroupSet(log logr.Logger, prometheusRule *prometheus.PrometheusRule) coralogixv1alpha1.RecordingRuleGroupSetSpec {
	groups := make([]coralogixv1alpha1.RecordingRuleGroup, 0)
	for _, group := range prometheusRule.Spec.Groups {
		var interval int32 = 60

		if group.Interval != "" {
			duration, err := time.ParseDuration(string(group.Interval))
			if err != nil {
				log.V(int(zapcore.WarnLevel)).Info("Failed to parse interval duration", "interval", group.Interval, "error", err, "using default interval")
			}

			// Convert duration to seconds
			durationSeconds := int32(duration.Seconds())

			if durationSeconds < interval {
				log.V(int(zapcore.WarnLevel)).Info("Recording rule interval is lower than the default interval", "interval", durationSeconds, "default interval", interval, "using the greater interval")
			} else {
				// Update interval if parsed duration is greater
				interval = durationSeconds
			}
		}

		if rules := prometheusInnerRulesToCoralogixInnerRules(group.Rules); len(rules) > 0 {
			groups = append(groups, coralogixv1alpha1.RecordingRuleGroup{
				Name:            group.Name,
				IntervalSeconds: interval,
				Rules:           rules,
			})
		}
	}

	return coralogixv1alpha1.RecordingRuleGroupSetSpec{
		Groups: groups,
	}
}

func prometheusInnerRulesToCoralogixInnerRules(rules []prometheus.Rule) []coralogixv1alpha1.RecordingRule {
	result := make([]coralogixv1alpha1.RecordingRule, 0)
	for _, rule := range rules {
		if rule.Record == "" {
			continue
		}

		result = append(result, coralogixv1alpha1.RecordingRule{
			Record: rule.Record,
			Expr:   rule.Expr.StrVal,
			Labels: rule.Labels,
		})
	}
	return result
}

func prometheusAlertToPromqlTypeSpec(rule prometheus.Rule) *coralogixv1alpha1.Promql {
	promqlType := &coralogixv1alpha1.Promql{
		SearchQuery: rule.Expr.StrVal,
		Conditions: coralogixv1alpha1.PromqlConditions{
			TimeWindow:                 getTimeWindow(rule),
			AlertWhen:                  coralogixv1alpha1.PromqlAlertWhenMoreThan,
			Threshold:                  resource.MustParse("0"),
			SampleThresholdPercentage:  100,
			MinNonNullValuesPercentage: ptr.To(0),
		},
	}

	return promqlType
}

func getSeverity(rule prometheus.Rule) coralogixv1alpha1.AlertSeverity {
	severity := coralogixv1alpha1.AlertSeverityInfo
	if severityStr, ok := rule.Labels["severity"]; ok && severityStr != "" {
		severityStr = strings.ToUpper(severityStr[:1]) + strings.ToLower(severityStr[1:])
		severity = coralogixv1alpha1.AlertSeverity(severityStr)
	}
	return severity
}

func getTimeWindow(rule prometheus.Rule) coralogixv1alpha1.MetricTimeWindow {
	if timeWindow, ok := prometheusAlertForToCoralogixPromqlAlertTimeWindow[rule.For]; ok {
		return timeWindow
	}
	return prometheusAlertForToCoralogixPromqlAlertTimeWindow["1m"]
}

var prometheusAlertForToCoralogixPromqlAlertTimeWindow = map[prometheus.Duration]coralogixv1alpha1.MetricTimeWindow{
	"1m":  coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowMinute),
	"5m":  coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowFiveMinutes),
	"10m": coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowTenMinutes),
	"15m": coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowFifteenMinutes),
	"20m": coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowTwentyMinutes),
	"30m": coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowThirtyMinutes),
	"1h":  coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowHour),
	"2h":  coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowTwelveHours),
	"4h":  coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowFourHours),
	"6h":  coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowSixHours),
	"12":  coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowTwelveHours),
	"24h": coralogixv1alpha1.MetricTimeWindow(coralogixv1alpha1.TimeWindowTwentyFourHours),
}

func getOwnerReference(promRule *prometheus.PrometheusRule) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: promRule.APIVersion,
		Kind:       promRule.Kind,
		Name:       promRule.Name,
		UID:        promRule.UID,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrometheusRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	shouldTrackPrometheusRules := func(labels map[string]string) bool {
		if value, ok := labels["app.coralogix.com/track-recording-rules"]; ok && value == "true" {
			return true
		}
		if value, ok := labels["app.coralogix.com/track-alerting-rules"]; ok && value == "true" {
			return true
		}
		return false
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&prometheus.PrometheusRule{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return shouldTrackPrometheusRules(e.Object.GetLabels())
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return shouldTrackPrometheusRules(e.ObjectNew.GetLabels()) || shouldTrackPrometheusRules(e.ObjectOld.GetLabels())
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return shouldTrackPrometheusRules(e.Object.GetLabels())
			},
		}).
		Complete(r)
}

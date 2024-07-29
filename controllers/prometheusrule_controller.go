package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	if err := r.Get(ctx, req.NamespacedName, prometheusRule); err != nil && !errors.IsNotFound(err) {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

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
		log.V(int(zapcore.DebugLevel)).Info("No recording rules found in PrometheusRule")
		return nil
	}

	recordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: prometheusRule.Namespace,
			Name:      prometheusRule.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: prometheusRule.APIVersion,
					Kind:       prometheusRule.Kind,
					Name:       prometheusRule.Name,
					UID:        prometheusRule.UID,
				},
			},
		},
		Spec: recordingRuleGroupSetSpec,
	}

	if err := r.Client.Get(ctx, req.NamespacedName, recordingRuleGroupSet); err != nil {
		if errors.IsNotFound(err) {
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
	prometheusRuleAlerts := make(map[string]bool)
	for _, group := range prometheusRule.Spec.Groups {
		for _, rule := range group.Rules {
			if rule.Alert == "" {
				continue
			}
			alert := &coralogixv1alpha1.Alert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: prometheusRule.Namespace,
					Name:      fmt.Sprintf("%s-%s", prometheusRule.Name, strings.ToLower(rule.Alert)),
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": prometheusRule.Name,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: prometheusRule.APIVersion,
							Kind:       prometheusRule.Kind,
							Name:       prometheusRule.Name,
							UID:        prometheusRule.UID,
						},
					},
				},
				Spec: prometheusRuleToCoralogixAlertSpec(rule),
			}

			prometheusRuleAlerts[alert.Name] = true

			if err := r.Client.Get(ctx, client.ObjectKeyFromObject(alert), alert); err != nil {
				if errors.IsNotFound(err) {
					if err = r.Create(ctx, alert); err != nil {
						return fmt.Errorf("received an error while trying to create Alert CRD from PrometheusRule: %w", err)
					}
					continue
				}
				return err
			}

			if err := r.Client.Update(ctx, alert); err != nil {
				return fmt.Errorf("received an error while trying to update Alert CRD from PrometheusRule: %w", err)
			}
		}
	}

	var alerts coralogixv1alpha1.AlertList
	if err := r.List(ctx, &alerts, client.InNamespace(prometheusRule.Namespace), client.MatchingLabels{"app.kubernetes.io/managed-by": prometheusRule.Name}); err != nil {
		return fmt.Errorf("received an error while trying to list child Alerts: %w", err)
	}

	// Remove alerts that are not present in the PrometheusRule anymore.
	for _, alert := range alerts.Items {
		if !prometheusRuleAlerts[alert.Name] {
			if err := r.Delete(ctx, &alert); err != nil {
				return fmt.Errorf("received an error while trying to remove child Alert: %w", err)
			}
		}
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
		// Default Coralogix interval is 30 seconds according to the documentation.
		// https://coralogix.com/docs/recordingrules/
		var interval int32 = 30

		if group.Interval != "" {
			duration, err := time.ParseDuration(string(group.Interval))
			if err != nil {
				log.V(int(zapcore.WarnLevel)).Info("failed to parse interval duration", "interval", group.Interval, "error", err, "using default interval")
			}
			interval = int32(duration.Seconds())
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

func prometheusRuleToCoralogixAlertSpec(prometheusRule prometheus.Rule) coralogixv1alpha1.AlertSpec {
	return coralogixv1alpha1.AlertSpec{
		Severity: getSeverity(prometheusRule),
		NotificationGroups: []coralogixv1alpha1.NotificationGroup{
			{
				Notifications: []coralogixv1alpha1.Notification{
					{
						RetriggeringPeriodMinutes: getNotificationPeriod(prometheusRule),
					},
				},
			},
		},
		Name: prometheusRule.Alert,
		AlertType: coralogixv1alpha1.AlertType{
			Metric: &coralogixv1alpha1.Metric{
				Promql: &coralogixv1alpha1.Promql{
					SearchQuery: prometheusRule.Expr.StrVal,
					Conditions: coralogixv1alpha1.PromqlConditions{
						TimeWindow:                 getTimeWindow(prometheusRule),
						AlertWhen:                  coralogixv1alpha1.PromqlAlertWhenMoreThan,
						Threshold:                  resource.MustParse("0"),
						SampleThresholdPercentage:  100,
						MinNonNullValuesPercentage: ptr.To(0),
					},
				},
			},
		},
	}
}

func getSeverity(prometheusRule prometheus.Rule) coralogixv1alpha1.AlertSeverity {
	severity := coralogixv1alpha1.AlertSeverityInfo
	if severityStr, ok := prometheusRule.Labels["severity"]; ok && severityStr != "" {
		severityStr = strings.ToUpper(severityStr[:1]) + strings.ToLower(severityStr[1:])
		severity = coralogixv1alpha1.AlertSeverity(severityStr)
	}
	return severity
}

func getTimeWindow(prometheusRule prometheus.Rule) coralogixv1alpha1.MetricTimeWindow {
	if timeWindow, ok := prometheusAlertForToCoralogixPromqlAlertTimeWindow[prometheusRule.For]; ok {
		return timeWindow
	}
	return prometheusAlertForToCoralogixPromqlAlertTimeWindow["1m"]
}

func getNotificationPeriod(prometheusRule prometheus.Rule) int32 {
	if cxNotifyEveryMin, ok := prometheusRule.Annotations["cxNotifyEveryMin"]; ok {
		if notificationPeriod, err := strconv.Atoi(cxNotifyEveryMin); err == nil {
			if notificationPeriod > 0 {
				return int32(notificationPeriod)
			}
		}
	}

	if duration, err := time.ParseDuration(string(prometheusRule.For)); err == nil {
		notificationPeriod := int(duration.Minutes())
		if notificationPeriod > 0 {
			return int32(notificationPeriod)
		}
	}

	return defaultCoralogixNotificationPeriod
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

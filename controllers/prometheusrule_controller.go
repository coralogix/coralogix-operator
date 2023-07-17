package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1alpha12 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultCoralogixNotificationPeriod int = 5
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
	CoralogixClientSet *clientset.ClientSet
	Scheme             *runtime.Scheme
}

func (r *PrometheusRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	prometheusRuleCRD := &prometheus.PrometheusRule{}
	if err := r.Get(ctx, req.NamespacedName, prometheusRuleCRD); err != nil && !errors.IsNotFound(err) {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	if shouldTrackRecordingRules(prometheusRuleCRD) {
		ruleGroupSetCRD := &v1alpha12.RecordingRuleGroupSet{}
		if err := r.Client.Get(ctx, req.NamespacedName, ruleGroupSetCRD); err != nil {
			if errors.IsNotFound(err) {
				log.V(1).Info(fmt.Sprintf("Couldn't find RecordingRuleSet Namespace: %s, Name: %s. Trying to create.", req.Namespace, req.Name))
				//Meaning there's a PrometheusRule with that NamespacedName but not RecordingRuleGroupSet accordingly (so creating it).
				if ruleGroupSetCRD.Spec, err = prometheusRuleToRuleGroupSet(prometheusRuleCRD); err != nil {
					log.Error(err, "Received an error while Converting PrometheusRule to RecordingRuleGroupSet", "PrometheusRule Name", prometheusRuleCRD.Name)
					return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
				}
				ruleGroupSetCRD.Namespace = req.Namespace
				ruleGroupSetCRD.Name = req.Name
				ruleGroupSetCRD.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: prometheusRuleCRD.APIVersion,
						Kind:       prometheusRuleCRD.Kind,
						Name:       prometheusRuleCRD.Name,
						UID:        prometheusRuleCRD.UID,
					},
				}
				if err = r.Create(ctx, ruleGroupSetCRD); err != nil {
					log.Error(err, "Received an error while trying to create RecordingRuleGroupSet CRD", "RecordingRuleGroupSet Name", ruleGroupSetCRD.Name)
					return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
				}

			} else {
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
		}

		//Converting the PrometheusRule to the desired RecordingRuleGroupSet.
		var err error
		if ruleGroupSetCRD.Spec, err = prometheusRuleToRuleGroupSet(prometheusRuleCRD); err != nil {
			log.Error(err, "Received an error while Converting PrometheusRule to RecordingRuleGroupSet", "PrometheusRule Name", prometheusRuleCRD.Name)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
		ruleGroupSetCRD.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: prometheusRuleCRD.APIVersion,
				Kind:       prometheusRuleCRD.Kind,
				Name:       prometheusRuleCRD.Name,
				UID:        prometheusRuleCRD.UID,
			},
		}

		if err = r.Client.Update(ctx, ruleGroupSetCRD); err != nil {
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
	}

	if shouldTrackAlerts(prometheusRuleCRD) {
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
		for _, group := range prometheusRuleCRD.Spec.Groups {
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
		for alertName, rules := range alertMap {
			for i, rule := range rules {
				alertCRD := &v1alpha12.Alert{}
				req.Name = fmt.Sprintf("%s-%s-%d", prometheusRuleCRD.Name, alertName, i)
				alertsToKeep[req.Name] = true
				if err := r.Client.Get(ctx, req.NamespacedName, alertCRD); err != nil {
					if errors.IsNotFound(err) {
						log.V(1).Info(fmt.Sprintf("Couldn't find Alert Namespace: %s, Name: %s. Trying to create.", req.Namespace, req.Name))
						alertCRD.Spec = prometheusInnerRuleToCoralogixAlert(rule)
						alertCRD.Namespace = req.Namespace
						alertCRD.Name = req.Name
						alertCRD.OwnerReferences = []metav1.OwnerReference{
							{
								APIVersion: prometheusRuleCRD.APIVersion,
								Kind:       prometheusRuleCRD.Kind,
								Name:       prometheusRuleCRD.Name,
								UID:        prometheusRuleCRD.UID,
							},
						}
						alertCRD.Labels = map[string]string{"app.kubernetes.io/managed-by": prometheusRuleCRD.Name}
						if err = r.Create(ctx, alertCRD); err != nil {
							log.Error(err, "Received an error while trying to create Alert CRD", "Alert Name", alertCRD.Name)
							return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
						}
					} else {
						return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
					}
				}

				//Converting the PrometheusRule to the desired Alert.
				alertCRD.Spec = prometheusInnerRuleToCoralogixAlert(rule)
				alertCRD.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: prometheusRuleCRD.APIVersion,
						Kind:       prometheusRuleCRD.Kind,
						Name:       prometheusRuleCRD.Name,
						UID:        prometheusRuleCRD.UID,
					},
				}
			}
		}

		var childAlerts v1alpha12.AlertList
		if err := r.List(ctx, &childAlerts, client.InNamespace(req.Namespace), client.MatchingLabels{"app.kubernetes.io/managed-by": prometheusRuleCRD.Name}); err != nil {
			log.Error(err, "unable to list child Alerts")
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}

		for _, alert := range childAlerts.Items {
			if !alertsToKeep[alert.Name] {
				if err := r.Delete(ctx, &alert); err != nil {
					log.Error(err, "unable to remove child Alert")
					return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
				}
			}
		}
	}

	return ctrl.Result{RequeueAfter: defaultRequeuePeriod}, nil
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

func prometheusRuleToRuleGroupSet(prometheusRule *prometheus.PrometheusRule) (v1alpha12.RecordingRuleGroupSetSpec, error) {
	groups := make([]v1alpha12.RecordingRuleGroup, 0)
	for _, group := range prometheusRule.Spec.Groups {
		rules := prometheusInnerRulesToCoralogixInnerRules(group.Rules)

		ruleGroup := v1alpha12.RecordingRuleGroup{
			Name:  group.Name,
			Rules: rules,
		}

		if interval := string(group.Interval); interval != "" {
			if duration, err := time.ParseDuration(interval); err != nil {
				return v1alpha12.RecordingRuleGroupSetSpec{}, err
			} else {
				ruleGroup.IntervalSeconds = int32(duration.Seconds())
			}
		}

		groups = append(groups, ruleGroup)
	}

	return v1alpha12.RecordingRuleGroupSetSpec{
		Groups: groups,
	}, nil
}

func prometheusInnerRuleToCoralogixAlert(prometheusRule prometheus.Rule) v1alpha12.AlertSpec {
	var notificationPeriod int
	if cxNotifyEveryMin, ok := prometheusRule.Annotations["cxNotifyEveryMin"]; ok {
		notificationPeriod, _ = strconv.Atoi(cxNotifyEveryMin)
	} else {
		duration, _ := time.ParseDuration(string(prometheusRule.For))
		notificationPeriod = int(duration.Minutes())
	}

	if notificationPeriod == 0 {
		notificationPeriod = defaultCoralogixNotificationPeriod
	}

	timeWindow, ok := prometheusAlertForToCoralogixPromqlAlertTimeWindow[prometheusRule.For]
	if !ok {
		timeWindow = prometheusAlertForToCoralogixPromqlAlertTimeWindow["1m"]
	}

	return v1alpha12.AlertSpec{
		Severity: v1alpha12.AlertSeverityInfo,
		NotificationGroups: []v1alpha12.NotificationGroup{
			{
				Notifications: []v1alpha12.Notification{
					{
						RetriggeringPeriodMinutes: int32(notificationPeriod),
					},
				},
			},
		},
		Name: prometheusRule.Alert,
		AlertType: v1alpha12.AlertType{
			Metric: &v1alpha12.Metric{
				Promql: &v1alpha12.Promql{
					SearchQuery: prometheusRule.Expr.StrVal,
					Conditions: v1alpha12.PromqlConditions{
						TimeWindow:                 timeWindow,
						AlertWhen:                  v1alpha12.AlertWhenMoreThan,
						Threshold:                  resource.MustParse("0"),
						SampleThresholdPercentage:  100,
						MinNonNullValuesPercentage: pointer.Int(0),
					},
				},
			},
		},
	}
}

var prometheusAlertForToCoralogixPromqlAlertTimeWindow = map[prometheus.Duration]v1alpha12.MetricTimeWindow{
	"1m":  v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowMinute),
	"5m":  v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowFiveMinutes),
	"10m": v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowTenMinutes),
	"15m": v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowFifteenMinutes),
	"20m": v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowTwentyMinutes),
	"30m": v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowThirtyMinutes),
	"1h":  v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowHour),
	"2h":  v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowTwelveHours),
	"4h":  v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowFourHours),
	"6h":  v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowSixHours),
	"12":  v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowTwelveHours),
	"24h": v1alpha12.MetricTimeWindow(v1alpha12.TimeWindowTwentyFourHours),
}

func prometheusInnerRulesToCoralogixInnerRules(rules []prometheus.Rule) []v1alpha12.RecordingRule {
	result := make([]v1alpha12.RecordingRule, 0)
	for _, r := range rules {
		if r.Record != "" {
			rule := prometheusInnerRuleToCoralogixInnerRule(r)
			result = append(result, rule)
		}
	}
	return result
}

func prometheusInnerRuleToCoralogixInnerRule(rule prometheus.Rule) v1alpha12.RecordingRule {
	return v1alpha12.RecordingRule{
		Record: rule.Record,
		Expr:   rule.Expr.StrVal,
		Labels: rule.Labels,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrometheusRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&prometheus.PrometheusRule{}).
		Complete(r)
}

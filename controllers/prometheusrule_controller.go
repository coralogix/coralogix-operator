package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
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
		ruleGroupSetCRD := &coralogixv1alpha1.RecordingRuleGroupSet{}
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
				alertCRD := &coralogixv1alpha1.Alert{}
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

		var childAlerts coralogixv1alpha1.AlertList
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

func prometheusRuleToRuleGroupSet(prometheusRule *prometheus.PrometheusRule) (coralogixv1alpha1.RecordingRuleGroupSetSpec, error) {
	groups := make([]coralogixv1alpha1.RecordingRuleGroup, 0)
	for _, group := range prometheusRule.Spec.Groups {
		rules := prometheusInnerRulesToCoralogixInnerRules(group.Rules)

		ruleGroup := coralogixv1alpha1.RecordingRuleGroup{
			Name:  group.Name,
			Rules: rules,
		}

		if interval := string(group.Interval); interval != "" {
			if duration, err := time.ParseDuration(interval); err != nil {
				return coralogixv1alpha1.RecordingRuleGroupSetSpec{}, err
			} else {
				ruleGroup.IntervalSeconds = int32(duration.Seconds())
			}
		}

		groups = append(groups, ruleGroup)
	}

	return coralogixv1alpha1.RecordingRuleGroupSetSpec{
		Groups: groups,
	}, nil
}

func prometheusInnerRuleToCoralogixAlert(prometheusRule prometheus.Rule) coralogixv1alpha1.AlertSpec {
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

	timeWindow := prometheusAlertForToCoralogixPromqlAlertTimeWindow[prometheusRule.For]

	return coralogixv1alpha1.AlertSpec{
		Severity: coralogixv1alpha1.AlertSeverityInfo,
		NotificationGroups: []coralogixv1alpha1.NotificationGroup{
			{
				Notifications: []coralogixv1alpha1.Notification{
					{
						RetriggeringPeriodMinutes: int32(notificationPeriod),
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
						TimeWindow:                 timeWindow,
						AlertWhen:                  coralogixv1alpha1.AlertWhenMoreThan,
						Threshold:                  resource.MustParse("0"),
						SampleThresholdPercentage:  100,
						MinNonNullValuesPercentage: pointer.Int(0),
					},
				},
			},
		},
	}
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

func prometheusInnerRulesToCoralogixInnerRules(rules []prometheus.Rule) []coralogixv1alpha1.RecordingRule {
	result := make([]coralogixv1alpha1.RecordingRule, 0)
	for _, r := range rules {
		if r.Record != "" {
			rule := prometheusInnerRuleToCoralogixInnerRule(r)
			result = append(result, rule)
		}
	}
	return result
}

func prometheusInnerRuleToCoralogixInnerRule(rule prometheus.Rule) coralogixv1alpha1.RecordingRule {
	return coralogixv1alpha1.RecordingRule{
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

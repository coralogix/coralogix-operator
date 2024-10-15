package controllers

import (
	"context"
	"fmt"
	"regexp"
	"time"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	"github.com/go-logr/logr"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=alertmanagerconfigs,verbs=get;list;watch

//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/finalizers,verbs=update

//+kubebuilder:rbac:groups=coralogix.com,resources=alerts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=alerts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=alerts/finalizers,verbs=update

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// AlertmanagerConfigReconciler reconciles a AlertmanagerConfig object
type AlertmanagerConfigReconciler struct {
	client.Client
	CoralogixClientSet clientset.ClientSetInterface
	Scheme             *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertmanagerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	shouldTrackAlertmanagerConfigs := func(labels map[string]string) bool {
		if value, ok := labels["app.coralogix.com/track-alertmanger-config"]; ok && value == "true" {
			return true
		}
		return false
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&prometheus.AlertmanagerConfig{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return shouldTrackAlertmanagerConfigs(e.Object.GetLabels())
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return shouldTrackAlertmanagerConfigs(e.ObjectNew.GetLabels()) || shouldTrackAlertmanagerConfigs(e.ObjectOld.GetLabels())
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return shouldTrackAlertmanagerConfigs(e.Object.GetLabels())
			},
		}).
		Complete(r)
}

func (r *AlertmanagerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	alertmanagerConfig := &prometheus.AlertmanagerConfig{}
	if err := r.Get(ctx, req.NamespacedName, alertmanagerConfig); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			if err = r.deleteWebhooksFromRelatedAlerts(ctx, alertmanagerConfig); err != nil {
				log.Error(err, "Received an error while trying to delete webhooks from related Alerts")
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	succeedConvertAlertmanager := r.convertAlertmanagerConfigToCxIntegrations(ctx, log, alertmanagerConfig)
	succeedLinkAlerts := r.linkCxAlertToCxIntegrations(ctx, log, alertmanagerConfig)
	if !succeedConvertAlertmanager || !succeedLinkAlerts {
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, fmt.Errorf("received an error while trying to convert AlertmanagerConfig to OutboundWebhook CRD")
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *AlertmanagerConfigReconciler) convertAlertmanagerConfigToCxIntegrations(ctx context.Context, log logr.Logger, alertmanagerConfig *prometheus.AlertmanagerConfig) (succeed bool) {
	succeed = true
	outboundWebhook := &coralogixv1alpha1.OutboundWebhook{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: alertmanagerConfig.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: alertmanagerConfig.APIVersion,
					Kind:       alertmanagerConfig.Kind,
					Name:       alertmanagerConfig.Name,
					UID:        alertmanagerConfig.UID,
				},
			},
		},
	}
	for _, receiver := range alertmanagerConfig.Spec.Receivers {
		for i, opsGenieConfig := range receiver.OpsGenieConfigs {
			opsGenieWebhook := outboundWebhook.DeepCopy()
			opsGenieWebhook.Name = fmt.Sprintf("%s.%s.%d", receiver.Name, "opsgenie", i)
			if err := r.Get(ctx, client.ObjectKeyFromObject(opsGenieWebhook), opsGenieWebhook); err != nil {
				if errors.IsNotFound(err) {
					opsGenieWebhookType := opsgenieToOutboundWebhookType(opsGenieConfig)
					opsGenieWebhook.Spec = coralogixv1alpha1.OutboundWebhookSpec{
						Name:                opsGenieWebhook.Name,
						OutboundWebhookType: opsGenieWebhookType,
					}
					if err = r.Create(ctx, opsGenieWebhook); err != nil {
						succeed = false
						log.Error(err, "Received an error while trying to create OutboundWebhook CRD from alertmanagerConfig")
						continue
					}
				} else {
					succeed = false
					log.Error(err, "Received an error while trying to get OutboundWebhook CRD from alertmanagerConfig")
					continue
				}
			} else {
				if err = r.Update(ctx, opsGenieWebhook); err != nil {
					succeed = false
					log.Error(err, "Received an error while trying to update OutboundWebhook CRD from alertmanagerConfig")
					continue
				}
			}
		}
		for i, slackConfig := range receiver.SlackConfigs {
			slackWebhook := outboundWebhook.DeepCopy()
			slackWebhook.Name = fmt.Sprintf("%s.%s.%d", receiver.Name, "slack", i)
			if err := r.Get(ctx, client.ObjectKeyFromObject(slackWebhook), slackWebhook); err != nil {
				if errors.IsNotFound(err) {
					outboundWebhookType, err := r.slackConfigToOutboundWebhookType(ctx, slackConfig, alertmanagerConfig.Namespace)
					if err != nil {
						succeed = false
						log.Error(err, "Received an error while trying to convert SlackConfig to OutboundWebhookType")
						continue
					}
					slackWebhook.Spec = coralogixv1alpha1.OutboundWebhookSpec{
						Name:                slackWebhook.Name,
						OutboundWebhookType: outboundWebhookType,
					}
					if err = r.Create(ctx, slackWebhook); err != nil {
						succeed = false
						log.Error(err, "Received an error while trying to create OutboundWebhook CRD from alertmanagerConfig")
						continue
					}
				} else {
					succeed = false
					log.Error(err, "Received an error while trying to get OutboundWebhook CRD from alertmanagerConfig")
					continue
				}
			} else {
				if err = r.Update(ctx, slackWebhook); err != nil {
					succeed = false
					log.Error(err, "Received an error while trying to update OutboundWebhook CRD from alertmanagerConfig")
					continue
				}
			}
		}
	}

	return
}

func (r *AlertmanagerConfigReconciler) slackConfigToOutboundWebhookType(ctx context.Context, config prometheus.SlackConfig, namespace string) (coralogixv1alpha1.OutboundWebhookType, error) {
	url, err := r.getSecret(ctx, config.APIURL, namespace)
	if err != nil {
		return coralogixv1alpha1.OutboundWebhookType{}, fmt.Errorf("received an error while trying to get API URL from secret: %w", err)
	}
	return coralogixv1alpha1.OutboundWebhookType{
		Slack: &coralogixv1alpha1.Slack{
			Url: url,
		},
	}, nil
}

func (r *AlertmanagerConfigReconciler) getSecret(ctx context.Context, secretKeySelector *v1.SecretKeySelector, namespace string) (string, error) {
	if secretKeySelector == nil {
		return "", nil
	}
	// Get the secret name and key
	secretName := secretKeySelector.Name
	secretKey := secretKeySelector.Key

	// Retrieve the secret from Kubernetes
	var secret v1.Secret
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, &secret)
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %v", err)
	}

	// Extract the value of the API URL from the secret data
	apiURLValue, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", secretKey, secretName)
	}

	return string(apiURLValue), nil
}

func (r *AlertmanagerConfigReconciler) linkCxAlertToCxIntegrations(ctx context.Context, log logr.Logger, config *prometheus.AlertmanagerConfig) (succeed bool) {
	succeed = true
	if config.Spec.Route == nil {
		return false
	}

	var alerts coralogixv1alpha1.AlertList
	if err := r.List(ctx, &alerts, client.InNamespace(config.Namespace), client.MatchingLabels{"app.coralogix.com/managed-by-alertmanger-config": "true"}); err != nil {
		log.Error(err, "Received an error while trying to list Alerts")
		return false
	}

	for _, alert := range alerts.Items {
		lset := getLabelSet(&alert)
		matchRoutes, err := Match(config.Spec.Route, lset)
		if err != nil {
			succeed = false
			log.Error(err, "Received an error while trying to match routes")
			continue
		}

		matchedReceiversMap := matchedRoutesToMatchedReceiversMap(matchRoutes, config.Spec.Receivers)
		alert.Spec.NotificationGroups, err = generateNotificationGroupFromRoutes(matchRoutes, matchedReceiversMap)
		if err != nil {
			succeed = false
			log.Error(err, "Received an error while trying to generate NotificationGroup from routes")
			continue
		}
		if err = r.Update(ctx, &alert); err != nil {
			succeed = false
			log.Error(err, "Received an error while trying to update Alert CRD from AlertmanagerConfig")
			continue
		}
	}

	return succeed
}

func (r *AlertmanagerConfigReconciler) deleteWebhooksFromRelatedAlerts(ctx context.Context, config *prometheus.AlertmanagerConfig) error {
	var alerts coralogixv1alpha1.AlertList
	if err := r.List(ctx, &alerts, client.InNamespace(config.Namespace), client.MatchingLabels{"app.coralogix.com/managed-by-alertmanger-config": "true"}); err != nil {
		return fmt.Errorf("received an error while trying to list Alerts: %w", err)
	}

	for _, alert := range alerts.Items {
		alert.Spec.NotificationGroups = nil
		if err := r.Update(ctx, &alert); err != nil {
			return fmt.Errorf("received an error while trying to update Alert CRD from AlertmanagerConfig: %w", err)
		}
	}

	return nil
}

func opsgenieToOutboundWebhookType(opsGenieConfig prometheus.OpsGenieConfig) coralogixv1alpha1.OutboundWebhookType {
	return coralogixv1alpha1.OutboundWebhookType{
		Opsgenie: &coralogixv1alpha1.Opsgenie{
			Url: opsGenieConfig.APIURL,
		},
	}
}

func matchedRoutesToMatchedReceiversMap(matchedRoutes []*prometheus.Route, allReceivers []prometheus.Receiver) map[string]*prometheus.Receiver {
	matchedReceiversMap := make(map[string]*prometheus.Receiver)
	for _, route := range matchedRoutes {
		if route.Receiver == "" {
			continue
		}
		receiver := getReceiverByName(allReceivers, route.Receiver)
		matchedReceiversMap[route.Receiver] = receiver
	}
	return matchedReceiversMap
}

func generateNotificationGroupFromRoutes(matchedRoutes []*prometheus.Route, matchedReceivers map[string]*prometheus.Receiver) ([]coralogixv1alpha1.NotificationGroup, error) {
	var notificationsGroups []coralogixv1alpha1.NotificationGroup
	for _, route := range matchedRoutes {
		receiver, ok := matchedReceivers[route.Receiver]
		if !ok || receiver == nil {
			continue
		}

		retriggeringPeriodMinutes, err := getRetriggeringPeriodMinutes(route)
		if err != nil {
			return nil, err
		}

		var notificationsGroup = coralogixv1alpha1.NotificationGroup{
			GroupByFields: route.GroupBy,
			Notifications: []coralogixv1alpha1.Notification{},
		}

		for i, conf := range receiver.SlackConfigs {
			webhookName := fmt.Sprintf("%s.%s.%d", receiver.Name, "slack", i)
			notificationsGroup.Notifications = append(notificationsGroup.Notifications, webhookNameToAlertNotification(webhookName, retriggeringPeriodMinutes, conf.SendResolved))
		}
		for i, conf := range receiver.OpsGenieConfigs {
			webhookName := fmt.Sprintf("%s.%s.%d", receiver.Name, "opsgenie", i)
			notificationsGroup.Notifications = append(notificationsGroup.Notifications, webhookNameToAlertNotification(webhookName, retriggeringPeriodMinutes, conf.SendResolved))
		}

		notificationsGroups = append(notificationsGroups, notificationsGroup)
	}
	if len(notificationsGroups) == 0 {
		return nil, nil
	}
	return notificationsGroups, nil
}

func getRetriggeringPeriodMinutes(route *prometheus.Route) (int32, error) {
	if route.RepeatInterval == "" {
		route.RepeatInterval = "4h"
	}
	RepeatIntervalDuration, err := time.ParseDuration(route.RepeatInterval)
	if err != nil {
		return 0, fmt.Errorf("received an error while trying to parse RepeatInterval: %w", err)
	}
	retriggeringPeriodMinutes := int32(RepeatIntervalDuration.Minutes())
	return retriggeringPeriodMinutes, nil
}

func webhookNameToAlertNotification(webhookName string, retriggeringPeriodMinutes int32, notifyOnResolve *bool) coralogixv1alpha1.Notification {
	var notifyOn string
	if notifyOnResolve != nil && *notifyOnResolve {
		notifyOn = coralogixv1alpha1.NotifyOnTriggeredAndResolved
	} else {
		notifyOn = coralogixv1alpha1.NotifyOnTriggeredOnly
	}
	return coralogixv1alpha1.Notification{
		IntegrationName:           pointer.String(webhookName),
		RetriggeringPeriodMinutes: retriggeringPeriodMinutes,
		NotifyOn:                  coralogixv1alpha1.NotifyOn(notifyOn),
	}
}

func getReceiverByName(receivers []prometheus.Receiver, receiver string) *prometheus.Receiver {
	for _, r := range receivers {
		if r.Name == receiver {
			return &r
		}
	}
	return nil
}

func getLabelSet(a *coralogixv1alpha1.Alert) model.LabelSet {
	lset := model.LabelSet{}
	for k, v := range a.Spec.Labels {
		lset[model.LabelName(k)] = model.LabelValue(v)
	}
	return lset
}

// Match does a depth-first left-to-right search through the route tree
// and returns the matching routing nodes.
func Match(r *prometheus.Route, lset model.LabelSet) ([]*prometheus.Route, error) {
	if r == nil {
		return nil, fmt.Errorf("match: nil route")
	}
	if match, err := AllMatches(r.Matchers, lset); err != nil {
		return nil, err
	} else if !match {
		return nil, nil
	}

	var all []*prometheus.Route
	crs, err := r.ChildRoutes()
	if err != nil {
		return nil, err
	}

	for _, cr := range crs {
		if cr.RepeatInterval == "" {
			cr.RepeatInterval = r.RepeatInterval
		}
		if cr.GroupBy == nil {
			cr.GroupBy = append([]string{}, r.GroupBy...)
		}
		matches, err := Match(&cr, lset)
		if err != nil {
			return nil, err
		}

		all = append(all, matches...)

		if matches != nil && !cr.Continue {
			break
		}
	}

	//If no child nodes were matches, the current node itself is a match.
	if len(all) == 0 {
		all = append(all, r)
	}

	return all, nil
}

// AllMatches checks whether all matchers are fulfilled against the given label set.
func AllMatches(ms []prometheus.Matcher, lset model.LabelSet) (bool, error) {
	for _, m := range ms {
		if match, err := Matches(&m, string(lset[model.LabelName(m.Name)])); err != nil {
			return false, err
		} else if !match {
			return false, nil
		}
	}
	return true, nil
}

// Matches returns whether the matcher matches the given string value.
func Matches(m *prometheus.Matcher, s string) (bool, error) {
	switch m.MatchType {
	case prometheus.MatchEqual:
		return s == m.Value, nil
	case prometheus.MatchNotEqual:
		return s != m.Value, nil
	case prometheus.MatchRegexp:
		re, err := regexp.Compile("^(?:" + m.Value + ")$")
		if err != nil {
			return false, fmt.Errorf("labels.Matcher.Matches: invalid regular expression %q: %v", s, err)
		}
		return re.MatchString(s), nil
	case prometheus.MatchNotRegexp:
		re, err := regexp.Compile("^(?:" + m.Value + ")$")
		if err != nil {
			return false, fmt.Errorf("labels.Matcher.Matches: invalid regular expression %q: %v", s, err)
		}
		return !re.MatchString(s), nil
	}
	return false, fmt.Errorf("labels.Matcher.Matches: invalid match type %v", m.MatchType)
}

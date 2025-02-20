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
	"reflect"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
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
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertmanagerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	shouldTrackAlertmanagerConfigs := func(labels map[string]string) bool {
		if value, ok := labels[utils.TrackAlertmanagerConfigLabelKey]; ok && value == "true" {
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
	if err := coralogixreconciler.GetClient().Get(ctx, req.NamespacedName, alertmanagerConfig); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			if err = r.deleteWebhooksFromRelatedAlerts(ctx, alertmanagerConfig); err != nil {
				log.Error(err, "Received an error while trying to delete webhooks from related Alerts")
				return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
			}
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	succeedConvertAlertmanager := r.convertAlertmanagerConfigToCxIntegrations(ctx, log, alertmanagerConfig)
	succeedLinkAlerts := r.linkCxAlertToCxIntegrations(ctx, log, alertmanagerConfig)
	if !succeedConvertAlertmanager || !succeedLinkAlerts {
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("received an error while trying to convert AlertmanagerConfig to OutboundWebhook CRD")
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
			if err := coralogixreconciler.GetClient().Get(ctx, client.ObjectKeyFromObject(opsGenieWebhook), opsGenieWebhook); err != nil {
				if errors.IsNotFound(err) {
					opsGenieWebhookType := opsgenieToOutboundWebhookType(opsGenieConfig)
					opsGenieWebhook.Spec = coralogixv1alpha1.OutboundWebhookSpec{
						Name:                opsGenieWebhook.Name,
						OutboundWebhookType: opsGenieWebhookType,
					}
					if err = coralogixreconciler.GetClient().Create(ctx, opsGenieWebhook); err != nil {
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
				if err = coralogixreconciler.GetClient().Update(ctx, opsGenieWebhook); err != nil {
					succeed = false
					log.Error(err, "Received an error while trying to update OutboundWebhook CRD from alertmanagerConfig")
					continue
				}
			}
		}
		for i, slackConfig := range receiver.SlackConfigs {
			slackWebhook := outboundWebhook.DeepCopy()
			slackWebhook.Name = fmt.Sprintf("%s.%s.%d", receiver.Name, "slack", i)
			if err := coralogixreconciler.GetClient().Get(ctx, client.ObjectKeyFromObject(slackWebhook), slackWebhook); err != nil {
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
					if err = coralogixreconciler.GetClient().Create(ctx, slackWebhook); err != nil {
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
				if err = coralogixreconciler.GetClient().Update(ctx, slackWebhook); err != nil {
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
	err := coralogixreconciler.GetClient().Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, &secret)
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

	var alerts coralogixv1beta1.AlertList
	if err := coralogixreconciler.GetClient().List(ctx, &alerts, client.InNamespace(config.Namespace), client.MatchingLabels{utils.ManagedByAlertmanagerConfigLabelKey: "true"}); err != nil {
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
		notificationGroups, err := generateNotificationGroupFromRoutes(matchRoutes, matchedReceiversMap)
		if err != nil {
			succeed = false
			log.Error(err, "Received an error while trying to generate NotificationGroup from routes")
			continue
		}

		notificationGroupsSpec := alert.Spec.NotificationGroupExcess
		if alert.Spec.NotificationGroup != nil {
			notificationGroupsSpec = append([]coralogixv1beta1.NotificationGroup{*alert.Spec.NotificationGroup}, notificationGroupsSpec...)
		}
		if !reflect.DeepEqual(notificationGroupsSpec, notificationGroups) {
			if err = coralogixreconciler.GetClient().Get(ctx, client.ObjectKey{Namespace: alert.Namespace, Name: alert.Name}, &alert); err != nil {
				succeed = false
				log.Error(err, "Received an error while trying to get Alert CRD from AlertmanagerConfig")
			}

			if len(notificationGroups) > 0 {
				alert.Spec.NotificationGroup = &notificationGroups[0]
			}
			if len(notificationGroups) > 1 {
				alert.Spec.NotificationGroupExcess = notificationGroups[1:]
			}

			if err = coralogixreconciler.GetClient().Update(ctx, &alert); err != nil {
				succeed = false
				log.Error(err, "Received an error while trying to update Alert CRD from AlertmanagerConfig")
				continue
			}
		}
	}

	return succeed
}

func (r *AlertmanagerConfigReconciler) deleteWebhooksFromRelatedAlerts(ctx context.Context, config *prometheus.AlertmanagerConfig) error {
	var alerts coralogixv1beta1.AlertList
	if err := coralogixreconciler.GetClient().List(ctx, &alerts, client.InNamespace(config.Namespace), client.MatchingLabels{utils.ManagedByAlertmanagerConfigLabelKey: "true"}); err != nil {
		return fmt.Errorf("received an error while trying to list Alerts: %w", err)
	}

	for _, alert := range alerts.Items {
		if alert.Spec.NotificationGroup != nil || len(alert.Spec.NotificationGroupExcess) > 0 {
			alert.Spec.NotificationGroup = nil
			alert.Spec.NotificationGroupExcess = nil
			if err := coralogixreconciler.GetClient().Update(ctx, &alert); err != nil {
				return fmt.Errorf("received an error while trying to update Alert CRD from AlertmanagerConfig: %w", err)
			}
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

func generateNotificationGroupFromRoutes(matchedRoutes []*prometheus.Route, matchedReceivers map[string]*prometheus.Receiver) ([]coralogixv1beta1.NotificationGroup, error) {
	var notificationsGroups []coralogixv1beta1.NotificationGroup
	for _, route := range matchedRoutes {
		receiver, ok := matchedReceivers[route.Receiver]
		if !ok || receiver == nil {
			continue
		}

		retriggeringPeriodMinutes, err := getRetriggeringPeriodMinutes(route)
		if err != nil {
			return nil, err
		}

		var notificationsGroup = coralogixv1beta1.NotificationGroup{
			GroupByKeys: route.GroupBy,
			Webhooks:    []coralogixv1beta1.WebhookSettings{},
		}

		for i, conf := range receiver.SlackConfigs {
			webhookName := fmt.Sprintf("%s.%s.%d", receiver.Name, "slack", i)
			notificationsGroup.Webhooks = append(notificationsGroup.Webhooks, webhookNameToAlertWebhookSettings(webhookName, retriggeringPeriodMinutes, conf.SendResolved))
		}
		for i, conf := range receiver.OpsGenieConfigs {
			webhookName := fmt.Sprintf("%s.%s.%d", receiver.Name, "opsgenie", i)
			notificationsGroup.Webhooks = append(notificationsGroup.Webhooks, webhookNameToAlertWebhookSettings(webhookName, retriggeringPeriodMinutes, conf.SendResolved))
		}

		notificationsGroups = append(notificationsGroups, notificationsGroup)
	}
	if len(notificationsGroups) == 0 {
		return nil, nil
	}
	return notificationsGroups, nil
}

func getRetriggeringPeriodMinutes(route *prometheus.Route) (uint32, error) {
	if route.RepeatInterval == "" {
		route.RepeatInterval = "4h"
	}
	RepeatIntervalDuration, err := time.ParseDuration(route.RepeatInterval)
	if err != nil {
		return 0, fmt.Errorf("received an error while trying to parse RepeatInterval: %w", err)
	}
	retriggeringPeriodMinutes := uint32(RepeatIntervalDuration.Minutes())
	return retriggeringPeriodMinutes, nil
}

func webhookNameToAlertWebhookSettings(webhookName string, retriggeringPeriodMinutes uint32, notifyOnResolve *bool) coralogixv1beta1.WebhookSettings {
	notifyOn := coralogixv1beta1.NotifyOnTriggeredOnly
	if notifyOnResolve != nil && *notifyOnResolve {
		notifyOn = coralogixv1beta1.NotifyOnTriggeredAndResolved
	}
	return coralogixv1beta1.WebhookSettings{
		Integration: coralogixv1beta1.IntegrationType{
			IntegrationRef: &coralogixv1beta1.IntegrationRef{
				BackendRef: &coralogixv1beta1.OutboundWebhookBackendRef{
					Name: pointer.String(webhookName),
				},
			},
		},
		RetriggeringPeriod: coralogixv1beta1.RetriggeringPeriod{
			Minutes: pointer.Uint32(retriggeringPeriodMinutes),
		},
		NotifyOn: notifyOn,
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

func getLabelSet(a *coralogixv1beta1.Alert) model.LabelSet {
	lset := model.LabelSet{}
	for k, v := range a.Spec.EntityLabels {
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

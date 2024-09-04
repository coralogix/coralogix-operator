package controllers

import (
	"context"
	"fmt"
	"regexp"
	"time"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/common/model"
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

//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/finalizers,verbs=update

//+kubebuilder:rbac:groups=coralogix.com,resources=alerts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=alerts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=alerts/finalizers,verbs=update

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

	if err := r.convertAlertmanagerConfigToCxIntegrations(ctx, alertmanagerConfig); err != nil {
		log.Error(err, "Received an error while trying to convert AlertmanagerConfig to Integration CRD")
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	if err := r.linkCxAlertToCxIntegrations(ctx, alertmanagerConfig); err != nil {
		log.Error(err, "Received an error while trying to link Alert to Integration CRD")
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
}

func (r *AlertmanagerConfigReconciler) convertAlertmanagerConfigToCxIntegrations(ctx context.Context, alertmanagerConfig *prometheus.AlertmanagerConfig) error {
	for _, receiver := range alertmanagerConfig.Spec.Receivers {
		for i, opsGenieConfig := range receiver.OpsGenieConfigs {
			name := fmt.Sprintf("%s.%s.%d", receiver.Name, "opsgenie", i)
			outboundWebhook := &coralogixv1alpha1.OutboundWebhook{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: alertmanagerConfig.Namespace}}
			if err := r.Get(ctx, client.ObjectKeyFromObject(outboundWebhook), outboundWebhook); err != nil {
				if errors.IsNotFound(err) {
					outboundWebhook.Spec = coralogixv1alpha1.OutboundWebhookSpec{
						Name: name,

						OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
							Opsgenie: &coralogixv1alpha1.Opsgenie{
								Url: opsGenieConfig.APIURL,
							},
						},
					}
					outboundWebhook.OwnerReferences = []metav1.OwnerReference{
						{
							APIVersion: alertmanagerConfig.APIVersion,
							Kind:       alertmanagerConfig.Kind,
							Name:       alertmanagerConfig.Name,
							UID:        alertmanagerConfig.UID,
						},
					}
					if err = r.Create(ctx, outboundWebhook); err != nil {
						return fmt.Errorf("received an error while trying to create OutboundWebhook CRD from alertmanagerConfig: %w", err)
					}
				} else {
					return fmt.Errorf("received an error while trying to get OutboundWebhook CRD from alertmanagerConfig: %w", err)
				}
			} else {
				if err = r.Update(ctx, outboundWebhook); err != nil {
					return fmt.Errorf("received an error while trying to update OutboundWebhook CRD from alertmanagerConfig: %w", err)
				}
			}
		}
		for i := range receiver.SlackConfigs {
			name := fmt.Sprintf("%s.%s.%d", receiver.Name, "slack", i)
			outboundWebhook := &coralogixv1alpha1.OutboundWebhook{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: alertmanagerConfig.Namespace}}
			if err := r.Get(ctx, client.ObjectKeyFromObject(outboundWebhook), outboundWebhook); err != nil {
				if errors.IsNotFound(err) {
					outboundWebhook.Spec = coralogixv1alpha1.OutboundWebhookSpec{
						Name: name,
						OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
							Slack: &coralogixv1alpha1.Slack{
								Url: "https://slack.com/api/chat.postMessage",
							},
						},
					}
					outboundWebhook.OwnerReferences = []metav1.OwnerReference{
						{
							APIVersion: alertmanagerConfig.APIVersion,
							Kind:       alertmanagerConfig.Kind,
							Name:       alertmanagerConfig.Name,
							UID:        alertmanagerConfig.UID,
						},
					}
					if err = r.Create(ctx, outboundWebhook); err != nil {
						return fmt.Errorf("received an error while trying to create OutboundWebhook CRD from alertmanagerConfig: %w", err)
					}
				} else {
					return fmt.Errorf("received an error while trying to get OutboundWebhook CRD from alertmanagerConfig: %w", err)
				}
			} else {
				if err = r.Update(ctx, outboundWebhook); err != nil {
					return fmt.Errorf("received an error while trying to update OutboundWebhook CRD from alertmanagerConfig: %w", err)
				}
			}
		}
	}
	return nil
}

func (r *AlertmanagerConfigReconciler) linkCxAlertToCxIntegrations(ctx context.Context, config *prometheus.AlertmanagerConfig) error {
	if config.Spec.Route == nil {
		return nil
	}

	var alerts coralogixv1alpha1.AlertList
	if err := r.List(ctx, &alerts, client.InNamespace(config.Namespace), client.MatchingLabels{"app.coralogix.com/managed-by-alertmanger-config": "true"}); err != nil {
		return fmt.Errorf("received an error while trying to list Alerts: %w", err)
	}

	for _, alert := range alerts.Items {
		lset := getLabelSet(&alert)
		matchRoutes, err := Match(config.Spec.Route, lset)
		if err != nil {
			return fmt.Errorf("received an error while trying to match routes: %w", err)
		}

		matchedReceivers := matchedRoutesToMatchedReceivers(matchRoutes, config.Spec.Receivers)
		alert.Spec.NotificationGroups = generateNotificationGroupOutOfMatchedReceivers(matchedReceivers)
		if err = r.Update(ctx, &alert); err != nil {
			return fmt.Errorf("received an error while trying to update OutboundWebhook CRD from AlertmanagerConfig: %w", err)
		}
	}

	return nil
}

func (r *AlertmanagerConfigReconciler) deleteWebhooksFromRelatedAlerts(ctx context.Context, config *prometheus.AlertmanagerConfig) error {
	var alerts coralogixv1alpha1.AlertList
	if err := r.List(ctx, &alerts, client.InNamespace(config.Namespace), client.MatchingLabels{"app.coralogix.com/managed-by-alertmanger-config": "true"}); err != nil {
		return fmt.Errorf("received an error while trying to list Alerts: %w", err)
	}

	for _, alert := range alerts.Items {
		alert.Spec.NotificationGroups = []coralogixv1alpha1.NotificationGroup{{}}
		if err := r.Update(ctx, &alert); err != nil {
			return fmt.Errorf("received an error while trying to update Alert CRD from AlertmanagerConfig: %w", err)
		}
	}

	return nil
}

func matchedRoutesToMatchedReceivers(matchedRoutes []*prometheus.Route, allReceivers []prometheus.Receiver) []*prometheus.Receiver {
	var matchedReceivers []*prometheus.Receiver
	for _, route := range matchedRoutes {
		if route.Receiver == "" {
			continue
		}
		receiver := getReceiverByName(allReceivers, route.Receiver)
		matchedReceivers = append(matchedReceivers, receiver)
	}
	return matchedReceivers
}

func generateNotificationGroupOutOfMatchedReceivers(matchedReceivers []*prometheus.Receiver) []coralogixv1alpha1.NotificationGroup {
	var notifications []coralogixv1alpha1.Notification
	for _, receiver := range matchedReceivers {
		if receiver == nil {
			continue
		}
		for i := range receiver.SlackConfigs {
			webhookName := fmt.Sprintf("%s.%s.%d", receiver.Name, "slack", i)
			notifications = append(notifications, webhookNameToAlertNotification(webhookName))
		}
		for i := range receiver.OpsGenieConfigs {
			webhookName := fmt.Sprintf("%s.%s.%d", receiver.Name, "opsgenie", i)
			notifications = append(notifications, webhookNameToAlertNotification(webhookName))
		}
	}

	return []coralogixv1alpha1.NotificationGroup{{Notifications: notifications}}
}

func webhookNameToAlertNotification(webhookName string) coralogixv1alpha1.Notification {
	return coralogixv1alpha1.Notification{
		IntegrationName:           pointer.String(webhookName),
		RetriggeringPeriodMinutes: 5,
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

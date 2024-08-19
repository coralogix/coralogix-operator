package controllers

import (
	"context"
	"fmt"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

func (r *AlertmanagerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	alertmanagerConfig := &prometheus.AlertmanagerConfig{}
	if err := r.Get(ctx, req.NamespacedName, alertmanagerConfig); err != nil && !errors.IsNotFound(err) {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	if shouldTrackIntegrations(alertmanagerConfig) {
		err := r.convertAlertmanagerConfigToCxIntegrations(ctx, alertmanagerConfig)
		if err != nil {
			log.Error(err, "Received an error while trying to convert AlertmanagerConfig to Integration CRD")
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *AlertmanagerConfigReconciler) convertAlertmanagerConfigToCxIntegrations(ctx context.Context, config *prometheus.AlertmanagerConfig) error {
	for _, receiver := range config.Spec.Receivers {
		for i, opsGenieConfig := range receiver.OpsGenieConfigs {
			name := receiver.Name + ".OpsGenie." + string(i)
			outboundWebhook := &coralogixv1alpha1.OutboundWebhook{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: config.Namespace}}
			if err := r.Get(ctx, client.ObjectKeyFromObject(outboundWebhook), outboundWebhook); err != nil {
				if errors.IsNotFound(err) {
					outboundWebhook.Spec = coralogixv1alpha1.OutboundWebhookSpec{
						Name: name, //todo: check if this is correct
						OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
							Opsgenie: &coralogixv1alpha1.Opsgenie{
								Url: opsGenieConfig.APIURL,
							},
						},
					}
					if err = r.Create(ctx, outboundWebhook); err != nil {
						return fmt.Errorf("received an error while trying to create OutboundWebhook CRD from AlertmanagerConfig: %w", err)
					}
				} else {
					return fmt.Errorf("received an error while trying to get OutboundWebhook CRD from AlertmanagerConfig: %w", err)
				}
			} else {
				if err = r.Update(ctx, outboundWebhook); err != nil {
					return fmt.Errorf("received an error while trying to update OutboundWebhook CRD from AlertmanagerConfig: %w", err)
				}
			}
		}
		for i, slackConfig := range receiver.SlackConfigs {
			slackConfig = slackConfig
			name := receiver.Name + ".Slack." + string(i)
			outboundWebhook := &coralogixv1alpha1.OutboundWebhook{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: config.Namespace}}
			if err := r.Get(ctx, client.ObjectKeyFromObject(outboundWebhook), outboundWebhook); err != nil {
				if errors.IsNotFound(err) {
					outboundWebhook.Spec = coralogixv1alpha1.OutboundWebhookSpec{
						Name: name, //todo: check if this is correct
						OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
							Slack: &coralogixv1alpha1.Slack{},
						},
					}
					if err = r.Create(ctx, outboundWebhook); err != nil {
						return fmt.Errorf("received an error while trying to create OutboundWebhook CRD from AlertmanagerConfig: %w", err)
					}
				} else {
					return fmt.Errorf("received an error while trying to get OutboundWebhook CRD from AlertmanagerConfig: %w", err)
				}
			} else {
				if err = r.Update(ctx, outboundWebhook); err != nil {
					return fmt.Errorf("received an error while trying to update OutboundWebhook CRD from AlertmanagerConfig: %w", err)
				}
			}
		}
	}

	return nil
}

func shouldTrackIntegrations(alertmanager *prometheus.AlertmanagerConfig) bool {
	if alertmanagerConfiguration := alertmanager.Labels["coralogix.com/alertmanager-configuration"]; alertmanagerConfiguration == "true" {
		return true
	}
	return false
}

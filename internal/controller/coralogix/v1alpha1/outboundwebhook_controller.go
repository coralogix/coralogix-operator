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

package v1alpha1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	webhooks "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/outgoing_webhooks_service"

	"github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconcile "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
)

// OutboundWebhookReconciler reconciles a OutboundWebhook object
type OutboundWebhookReconciler struct {
	OutboundWebhooksClient *webhooks.OutgoingWebhooksServiceAPIService
	Interval               time.Duration
}

const outboundWebhookSecretIndex = "outboundWebhookSecretNames"

//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *OutboundWebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconcile.ReconcileResource(ctx, req, &v1alpha1.OutboundWebhook{}, r)
}

func (r *OutboundWebhookReconciler) FinalizerName() string {
	return "outbound-webhook.coralogix.com/finalizer"
}

func (r *OutboundWebhookReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *OutboundWebhookReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	createRequest, err := outboundWebhook.ExtractCreateOutboundWebhookRequestWithContext(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting create outbound-webhook request: %w", err)
	}
	log.Info("Creating remote outbound-webhook")
	createResponse, httpResp, err := r.OutboundWebhooksClient.
		OutgoingWebhooksServiceCreateOutgoingWebhook(ctx).
		CreateOutgoingWebhookRequest(*createRequest).
		Execute()
	if err != nil {
		return safeWebhookAPIError("create", httpResp, err)
	}
	log.Info("Remote outbound-webhook created", "id", createResponse.Id)

	log.Info("Getting outbound-webhook from remote", "id", createResponse.Id)
	remoteOutboundWebhook, httpResp, err := r.OutboundWebhooksClient.
		OutgoingWebhooksServiceGetOutgoingWebhook(ctx, createResponse.Id).
		Execute()
	if err != nil {
		return safeWebhookAPIError("get", httpResp, err)
	}

	status, err := getOutboundWebhookStatus(remoteOutboundWebhook.Webhook)
	if err != nil {
		return fmt.Errorf("error on getting outbound-webhook status: %w", err)
	}
	outboundWebhook.Status = *status

	return nil
}

func (r *OutboundWebhookReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	updateRequest, err := outboundWebhook.ExtractUpdateOutboundWebhookRequestWithContext(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting update outbound-webhook request: %w", err)
	}
	log.Info("Updating remote outbound-webhook")
	_, httpResp, err := r.OutboundWebhooksClient.
		OutgoingWebhooksServiceUpdateOutgoingWebhook(ctx).
		UpdateOutgoingWebhookRequest(*updateRequest).
		Execute()
	if err != nil {
		return safeWebhookAPIError("update", httpResp, err)
	}
	log.Info("Remote outbound-webhook updated")
	return nil
}

func (r *OutboundWebhookReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	log.Info("Deleting outbound-webhook from remote system", "id", *outboundWebhook.Status.ID)
	_, httpResp, err := r.OutboundWebhooksClient.OutgoingWebhooksServiceDeleteOutgoingWebhook(ctx, *outboundWebhook.Status.ID).
		Execute()
	if err != nil {
		if apiErr := safeWebhookAPIError("delete", httpResp, err); !cxsdk.IsNotFound(apiErr) {
			return apiErr
		}
	}
	log.Info("outbound-webhook deleted from remote system", "id", *outboundWebhook.Status.ID)
	return nil
}

// The shared reconciler writes errors to logs and status. Keep API response bodies out of both.
func safeWebhookAPIError(operation string, resp *http.Response, err error) error {
	apiErr := cxsdk.NewAPIError(resp, err)
	code := cxsdk.Code(apiErr)
	var body []byte
	if cxsdk.IsNotFound(apiErr) {
		body = []byte("Not Found: outbound webhook")
	}
	return &cxsdk.APIError{
		Err:        fmt.Errorf("%s outbound-webhook failed (HTTP %d)", operation, code),
		StatusCode: code,
		Body:       body,
	}
}

func getOutboundWebhookStatus(webhook webhooks.OutgoingWebhook) (*v1alpha1.OutboundWebhookStatus, error) {
	if webhook.Id == "" {
		return nil, fmt.Errorf("outbound-webhook id is not set")
	}

	status := &v1alpha1.OutboundWebhookStatus{
		ID:         ptr.To(webhook.Id),
		ExternalID: ptr.To(strconv.FormatInt(webhook.ExternalId, 10)),
	}

	return status, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OutboundWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.OutboundWebhook{}, outboundWebhookSecretIndex, func(obj client.Object) []string {
		return outboundWebhookSecretNames(obj.(*v1alpha1.OutboundWebhook))
	}); err != nil {
		return fmt.Errorf("index outbound-webhook Secret references: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OutboundWebhook{}, builder.WithPredicates(config.GetConfig().Selector.Predicate())).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.webhooksForSecret)).
		Complete(r)
}

func (r *OutboundWebhookReconciler) webhooksForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	webhooks := &v1alpha1.OutboundWebhookList{}
	if err := config.GetClient().List(ctx, webhooks,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFields{outboundWebhookSecretIndex: obj.GetName()}); err != nil {
		ctrl.Log.Info("Failed to list outbound-webhooks for Secret", "namespace", obj.GetNamespace(), "secret", obj.GetName())
		return nil
	}

	requests := make([]reconcile.Request, 0, len(webhooks.Items))
	for _, webhook := range webhooks.Items {
		if config.GetConfig().Selector.Matches(webhook.Labels, webhook.Namespace) {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&webhook)})
		}
	}
	return requests
}

func outboundWebhookSecretNames(webhook *v1alpha1.OutboundWebhook) []string {
	refs := make([]*corev1.SecretKeySelector, 0, 3)
	webhookType := webhook.Spec.OutboundWebhookType
	if generic := webhookType.GenericWebhook; generic != nil {
		refs = append(refs, generic.UrlSecretKeyRef)
		for _, ref := range generic.HeadersFromSecret {
			refs = append(refs, &ref)
		}
	}
	if slack := webhookType.Slack; slack != nil {
		refs = append(refs, slack.UrlSecretKeyRef)
	}
	if pagerDuty := webhookType.PagerDuty; pagerDuty != nil {
		refs = append(refs, pagerDuty.ServiceKeySecretKeyRef)
	}
	if sendLog := webhookType.SendLog; sendLog != nil {
		refs = append(refs, sendLog.UrlSecretKeyRef)
	}
	if teams := webhookType.MicrosoftTeams; teams != nil {
		refs = append(refs, teams.UrlSecretKeyRef)
	}
	if jira := webhookType.Jira; jira != nil {
		refs = append(refs, jira.UrlSecretKeyRef, jira.ApiTokenSecretKeyRef)
	}
	if opsgenie := webhookType.Opsgenie; opsgenie != nil {
		refs = append(refs, opsgenie.UrlSecretKeyRef)
	}
	if demisto := webhookType.Demisto; demisto != nil {
		refs = append(refs, demisto.UrlSecretKeyRef)
	}

	names := make([]string, 0, len(refs))
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		if ref != nil && ref.Name != "" && !seen[ref.Name] {
			names = append(names, ref.Name)
			seen[ref.Name] = true
		}
	}
	return names
}

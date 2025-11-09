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
	"strconv"
	"time"

	"github.com/coralogix/coralogix-operator/internal/utils"
	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	webhooks "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/outgoing_webhooks_service"

	"github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconcile "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// OutboundWebhookReconciler reconciles a OutboundWebhook object
type OutboundWebhookReconciler struct {
	OutboundWebhooksClient *webhooks.OutgoingWebhooksServiceAPIService
	Interval               time.Duration
}

//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks/finalizers,verbs=update

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
	createRequest, err := outboundWebhook.ExtractCreateOutboundWebhookRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create outbound-webhook request: %w", err)
	}
	log.Info("Creating remote outbound-webhook", "outbound-webhook", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.OutboundWebhooksClient.
		OutgoingWebhooksServiceCreateOutgoingWebhook(ctx).
		CreateOutgoingWebhookRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote outbound-webhook: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote outbound-webhook created", "response", utils.FormatJSON(createResponse))

	log.Info("Getting outbound-webhook from remote", "id", createResponse.Id)
	remoteOutboundWebhook, httpResp, err := r.OutboundWebhooksClient.
		OutgoingWebhooksServiceGetOutgoingWebhook(ctx, *createResponse.Id).
		Execute()
	if err != nil {
		return fmt.Errorf("error to get outbound-webhook %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info(fmt.Sprintf("outbound-webhook was read\n%s", utils.FormatJSON(remoteOutboundWebhook)))

	status, err := getOutboundWebhookStatus(remoteOutboundWebhook.Webhook)
	if err != nil {
		return fmt.Errorf("error on getting outbound-webhook status: %w", err)
	}
	outboundWebhook.Status = *status

	return nil
}

func (r *OutboundWebhookReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	updateRequest, err := outboundWebhook.ExtractUpdateOutboundWebhookRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update outbound-webhook request: %w", err)
	}
	log.Info("Updating remote outbound-webhook", "outbound-webhook", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.OutboundWebhooksClient.
		OutgoingWebhooksServiceUpdateOutgoingWebhook(ctx).
		UpdateOutgoingWebhookRequest(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote outbound-webhook updated", "outbound-webhook", utils.FormatJSON(updateResponse))
	return nil
}

func (r *OutboundWebhookReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	log.Info("Deleting outbound-webhook from remote system", "id", *outboundWebhook.Status.ID)
	_, httpResp, err := r.OutboundWebhooksClient.OutgoingWebhooksServiceDeleteOutgoingWebhook(ctx, *outboundWebhook.Status.ID).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(apiErr, "Error deleting remote outbound-webhook", "id", *outboundWebhook.Status.ID)
			return fmt.Errorf("error deleting remote outbound-webhook %s: %w", *outboundWebhook.Status.ID, apiErr)
		}
	}
	log.Info("outbound-webhook deleted from remote system", "id", *outboundWebhook.Status.ID)
	return nil
}

func getOutboundWebhookStatus(webhook *webhooks.OutgoingWebhook) (*v1alpha1.OutboundWebhookStatus, error) {
	if webhook == nil {
		return nil, fmt.Errorf("outbound-webhook is nil")
	}

	status := &v1alpha1.OutboundWebhookStatus{}

	extract := func(id *string, externalID *int64) (*string, *string) {
		if id == nil {
			return nil, nil
		}
		var ext *string
		if externalID != nil {
			ext = ptr.To(strconv.Itoa(int(*externalID)))
		}
		return id, ext
	}

	switch {
	case webhook.OutgoingWebhookAwsEventBridge != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookAwsEventBridge.Id, webhook.OutgoingWebhookAwsEventBridge.ExternalId)
	case webhook.OutgoingWebhookDemisto != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookDemisto.Id, webhook.OutgoingWebhookDemisto.ExternalId)
	case webhook.OutgoingWebhookEmailGroup != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookEmailGroup.Id, webhook.OutgoingWebhookEmailGroup.ExternalId)
	case webhook.OutgoingWebhookGenericWebhook != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookGenericWebhook.Id, webhook.OutgoingWebhookGenericWebhook.ExternalId)
	case webhook.OutgoingWebhookIbmEventNotifications != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookIbmEventNotifications.Id, webhook.OutgoingWebhookIbmEventNotifications.ExternalId)
	case webhook.OutgoingWebhookJira != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookJira.Id, webhook.OutgoingWebhookJira.ExternalId)
	case webhook.OutgoingWebhookMicrosoftTeams != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookMicrosoftTeams.Id, webhook.OutgoingWebhookMicrosoftTeams.ExternalId)
	case webhook.OutgoingWebhookMsTeamsWorkflow != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookMsTeamsWorkflow.Id, webhook.OutgoingWebhookMsTeamsWorkflow.ExternalId)
	case webhook.OutgoingWebhookOpsgenie != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookOpsgenie.Id, webhook.OutgoingWebhookOpsgenie.ExternalId)
	case webhook.OutgoingWebhookPagerDuty != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookPagerDuty.Id, webhook.OutgoingWebhookPagerDuty.ExternalId)
	case webhook.OutgoingWebhookSendLog != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookSendLog.Id, webhook.OutgoingWebhookSendLog.ExternalId)
	case webhook.OutgoingWebhookSlack != nil:
		status.ID, status.ExternalID = extract(webhook.OutgoingWebhookSlack.Id, webhook.OutgoingWebhookSlack.ExternalId)
	default:
		return nil, fmt.Errorf("unsupported or unknown outbound-webhook type")
	}

	return status, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OutboundWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OutboundWebhook{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}

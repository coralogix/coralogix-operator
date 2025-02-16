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

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	coralogixreconcile "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	util "github.com/coralogix/coralogix-operator/internal/utils"
)

// OutboundWebhookReconciler reconciles a OutboundWebhook object
type OutboundWebhookReconciler struct {
	OutboundWebhooksClient clientset.OutboundWebhooksClientInterface
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

func (r *OutboundWebhookReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) (client.Object, error) {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	createRequest, err := outboundWebhook.ExtractCreateOutboundWebhookRequest()
	if err != nil {
		return nil, fmt.Errorf("error on extracting create outbound-webhook request: %w", err)
	}
	log.V(1).Info("Creating remote outbound-webhook", "outbound-webhook", protojson.Format(createRequest))
	createResponse, err := r.OutboundWebhooksClient.Create(ctx, createRequest)
	if err != nil {
		return nil, fmt.Errorf("error on creating remote outbound-webhook: %w", err)
	}
	log.V(1).Info("Remote outbound-webhook created", "response", protojson.Format(createResponse))
	monitoring.OutboundWebhookInfoMetric.WithLabelValues(outboundWebhook.Name, outboundWebhook.Namespace, getWebhookType(outboundWebhook)).Set(1)

	log.V(1).Info("Getting outbound-webhook from remote", "id", createResponse.Id.Value)
	remoteOutboundWebhook, err := r.OutboundWebhooksClient.Get(ctx,
		&cxsdk.GetOutgoingWebhookRequest{
			Id: createResponse.Id,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error to get outbound-webhook %w", err)
	}
	log.V(1).Info(fmt.Sprintf("outbound-webhook was read\n%s", protojson.Format(remoteOutboundWebhook)))

	status, err := getOutboundWebhookStatus(remoteOutboundWebhook.Webhook)
	if err != nil {
		return nil, fmt.Errorf("error on getting outbound-webhook status: %w", err)
	}
	outboundWebhook.Status = *status

	return outboundWebhook, nil
}

func (r *OutboundWebhookReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	updateRequest, err := outboundWebhook.ExtractUpdateOutboundWebhookRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update outbound-webhook request: %w", err)
	}
	log.V(1).Info("Updating remote outbound-webhook", "outbound-webhook", protojson.Format(updateRequest))
	updateResponse, err := r.OutboundWebhooksClient.Update(ctx, updateRequest)
	if err != nil {
		return fmt.Errorf("error on updating remote outbound-webhook: %w", err)
	}
	log.V(1).Info("Remote outbound-webhook updated", "outbound-webhook", protojson.Format(updateResponse))
	monitoring.OutboundWebhookInfoMetric.WithLabelValues(outboundWebhook.Name, outboundWebhook.Namespace, getWebhookType(outboundWebhook)).Set(1)
	return nil
}

func (r *OutboundWebhookReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	log.V(1).Info("Deleting outbound-webhook from remote system", "id", *outboundWebhook.Status.ID)
	_, err := r.OutboundWebhooksClient.Delete(ctx, &cxsdk.DeleteOutgoingWebhookRequest{Id: wrapperspb.String(*outboundWebhook.Status.ID)})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error deleting remote outbound-webhook", "id", *outboundWebhook.Status.ID)
		return fmt.Errorf("error deleting remote outbound-webhook %s: %w", *outboundWebhook.Status.ID, err)
	}
	log.V(1).Info("outbound-webhook deleted from remote system", "id", *outboundWebhook.Status.ID)
	monitoring.OutboundWebhookInfoMetric.WithLabelValues(outboundWebhook.Name, outboundWebhook.Namespace, getWebhookType(outboundWebhook)).Set(0)
	return nil
}

func getOutboundWebhookStatus(webhook *cxsdk.OutgoingWebhook) (*v1alpha1.OutboundWebhookStatus, error) {
	if webhook == nil {
		return nil, fmt.Errorf("outbound-webhook is nil")
	}

	status := &v1alpha1.OutboundWebhookStatus{
		ID:         ptr.To(webhook.Id.GetValue()),
		ExternalID: ptr.To(strconv.Itoa(int(webhook.ExternalId.GetValue()))),
	}

	return status, nil
}

func (r *OutboundWebhookReconciler) CheckIDInStatus(obj client.Object) bool {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	return outboundWebhook.Status.ID != nil && *outboundWebhook.Status.ID != ""
}

func getWebhookType(webhook *v1alpha1.OutboundWebhook) string {
	if webhook.Spec.OutboundWebhookType.GenericWebhook != nil {
		return "genericWebhook"
	}

	if webhook.Spec.OutboundWebhookType.Slack != nil {
		return "slack"
	}

	if webhook.Spec.OutboundWebhookType.PagerDuty != nil {
		return "pager_duty"
	}

	if webhook.Spec.OutboundWebhookType.SendLog != nil {
		return "send_log"
	}

	if webhook.Spec.OutboundWebhookType.EmailGroup != nil {
		return "email_group"
	}

	if webhook.Spec.OutboundWebhookType.MicrosoftTeams != nil {
		return "microsoft_teams"
	}

	if webhook.Spec.OutboundWebhookType.Jira != nil {
		return "jira"
	}

	if webhook.Spec.OutboundWebhookType.Opsgenie != nil {
		return "opsgenie"
	}

	if webhook.Spec.OutboundWebhookType.Demisto != nil {
		return "demisto"
	}

	if webhook.Spec.OutboundWebhookType.AwsEventBridge != nil {
		return "aws_event_bridge"
	}

	return "unknown"
}

// SetupWithManager sets up the controller with the Manager.
func (r *OutboundWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OutboundWebhook{}).
		WithEventFilter(util.GetLabelFilter().Predicate()).
		Complete(r)
}

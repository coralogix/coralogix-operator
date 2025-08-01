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

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	coralogixreconcile "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// OutboundWebhookReconciler reconciles a OutboundWebhook object
type OutboundWebhookReconciler struct {
	OutboundWebhooksClient clientset.OutboundWebhooksClientInterface
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
	log.Info("Creating remote outbound-webhook", "outbound-webhook", protojson.Format(createRequest))
	createResponse, err := r.OutboundWebhooksClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote outbound-webhook: %w", err)
	}
	log.Info("Remote outbound-webhook created", "response", protojson.Format(createResponse))

	log.Info("Getting outbound-webhook from remote", "id", createResponse.Id.Value)
	remoteOutboundWebhook, err := r.OutboundWebhooksClient.Get(ctx,
		&cxsdk.GetOutgoingWebhookRequest{
			Id: createResponse.Id,
		},
	)
	if err != nil {
		return fmt.Errorf("error to get outbound-webhook %w", err)
	}
	log.Info(fmt.Sprintf("outbound-webhook was read\n%s", protojson.Format(remoteOutboundWebhook)))

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
	log.Info("Updating remote outbound-webhook", "outbound-webhook", protojson.Format(updateRequest))
	updateResponse, err := r.OutboundWebhooksClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote outbound-webhook updated", "outbound-webhook", protojson.Format(updateResponse))
	return nil
}

func (r *OutboundWebhookReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	outboundWebhook := obj.(*v1alpha1.OutboundWebhook)
	log.Info("Deleting outbound-webhook from remote system", "id", *outboundWebhook.Status.ID)
	_, err := r.OutboundWebhooksClient.Delete(ctx, &cxsdk.DeleteOutgoingWebhookRequest{Id: wrapperspb.String(*outboundWebhook.Status.ID)})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote outbound-webhook", "id", *outboundWebhook.Status.ID)
		return fmt.Errorf("error deleting remote outbound-webhook %s: %w", *outboundWebhook.Status.ID, err)
	}
	log.Info("outbound-webhook deleted from remote system", "id", *outboundWebhook.Status.ID)
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

// SetupWithManager sets up the controller with the Manager.
func (r *OutboundWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OutboundWebhook{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}

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

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/api/coralogix"
	"github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	util "github.com/coralogix/coralogix-operator/internal/utils"
)

// OutboundWebhookReconciler reconciles a OutboundWebhook object
type OutboundWebhookReconciler struct {
	client.Client
	OutboundWebhooksClient clientset.OutboundWebhooksClientInterface
	Scheme                 *runtime.Scheme
}

//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=outboundwebhooks/finalizers,verbs=update

var (
	outboundWebhookFinalizerName = "outbound-webhook.coralogix.com/finalizer"
)

func (r *OutboundWebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		resultError = ctrl.Result{RequeueAfter: 40}
		err         error
	)

	log := log.FromContext(ctx).WithValues(
		"outboundWebhook", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	outboundWebhook := &v1alpha1.OutboundWebhook{}
	if err = r.Client.Get(ctx, req.NamespacedName, outboundWebhook); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return resultError, err
	}

	if ptr.Deref(outboundWebhook.Status.ID, "") == "" {
		err = r.create(ctx, log, outboundWebhook)
		if err != nil {
			log.Error(err, "Error on creating outbound-webhook")
			return resultError, err
		}
		monitoring.OutboundWebhookInfoMetric.WithLabelValues(outboundWebhook.Name, outboundWebhook.Namespace, getWebhookType(outboundWebhook)).Set(1)
		return ctrl.Result{}, nil
	}

	if !outboundWebhook.ObjectMeta.DeletionTimestamp.IsZero() {
		err = r.delete(ctx, log, outboundWebhook)
		if err != nil {
			log.Error(err, "Error on deleting outbound-webhook")
			return resultError, err
		}
		monitoring.OutboundWebhookInfoMetric.DeleteLabelValues(outboundWebhook.Name, outboundWebhook.Namespace, getWebhookType(outboundWebhook))
		return ctrl.Result{}, nil
	}

	if !util.GetLabelFilter().Matches(outboundWebhook.GetLabels()) {
		err = r.deleteRemoteWebhook(ctx, log, outboundWebhook.Status.ID)
		if err != nil {
			log.Error(err, "Error on deleting OutboundWebhook")
			return ctrl.Result{RequeueAfter: util.DefaultErrRequeuePeriod}, err
		}
		monitoring.OutboundWebhookInfoMetric.DeleteLabelValues(outboundWebhook.Name, outboundWebhook.Namespace, getWebhookType(outboundWebhook))
		return ctrl.Result{}, nil
	}

	err = r.update(ctx, log, outboundWebhook)
	if err != nil {
		log.Error(err, "Error on updating outbound-webhook")
		return resultError, err
	}
	monitoring.OutboundWebhookInfoMetric.WithLabelValues(outboundWebhook.Name, outboundWebhook.Namespace, getWebhookType(outboundWebhook)).Set(1)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OutboundWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OutboundWebhook{}).
		WithEventFilter(util.GetLabelFilter().Predicate()).
		Complete(r)
}

func (r *OutboundWebhookReconciler) create(ctx context.Context, log logr.Logger, webhook *v1alpha1.OutboundWebhook) error {
	createRequest, err := webhook.ExtractCreateOutboundWebhookRequest()
	if err != nil {
		return fmt.Errorf("error to extract create-request out of the outbound-webhook -\n%v", webhook)
	}

	log.V(1).Info(fmt.Sprintf("Creating outbound-webhook-\n%s", protojson.Format(createRequest)))
	createResponse, err := r.OutboundWebhooksClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error to create remote outbound-webhook - %s\n%w", protojson.Format(createRequest), err)
	}
	log.V(1).Info(fmt.Sprintf("outbound-webhook was created- %s", protojson.Format(createResponse)))

	webhook.Status = v1alpha1.OutboundWebhookStatus{
		ID: ptr.To(createResponse.Id.GetValue()),
	}
	if err = r.Status().Update(ctx, webhook); err != nil {
		if err := r.deleteRemoteWebhook(ctx, log, webhook.Status.ID); err != nil {
			return fmt.Errorf("error to delete outbound-webhook after status update error -\n%v", webhook)
		}
		return fmt.Errorf("error to update outbound-webhook status -\n%v", webhook)
	}

	readRequest := &cxsdk.GetOutgoingWebhookRequest{Id: createResponse.Id}
	log.V(1).Info(fmt.Sprintf("Getting outbound-webhook -\n%s", protojson.Format(readRequest)))
	readResponse, err := r.OutboundWebhooksClient.Get(ctx, readRequest)
	if err != nil {
		return fmt.Errorf("error to get outbound-webhook -\n%v", webhook)
	}
	log.V(1).Info(fmt.Sprintf("outbound-webhook was read -\n%s", protojson.Format(readResponse)))

	status, err := getOutboundWebhookStatus(readResponse.GetWebhook())
	if err != nil {
		return fmt.Errorf("error to flatten outbound-webhook -\n%v", webhook)
	}

	webhook.Status = *status
	if err = r.Status().Update(ctx, webhook); err != nil {
		return fmt.Errorf("error to update outbound-webhook status -\n%v", webhook)
	}

	if !controllerutil.ContainsFinalizer(webhook, outboundWebhookFinalizerName) {
		controllerutil.AddFinalizer(webhook, outboundWebhookFinalizerName)
	}
	if err = r.Client.Update(ctx, webhook); err != nil {
		return fmt.Errorf("error to update outbound-webhook -\n%v", webhook)
	}

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

func (r *OutboundWebhookReconciler) update(ctx context.Context, log logr.Logger, webhook *v1alpha1.OutboundWebhook) error {
	updateReq, err := webhook.ExtractUpdateOutboundWebhookRequest()
	if err != nil {
		return fmt.Errorf("error to parse update outbound-webhook request -\n%v", webhook)
	}

	log.V(1).Info(fmt.Sprintf("updating outbound-webhook\n%s", protojson.Format(updateReq)))
	_, err = r.OutboundWebhooksClient.Update(ctx, updateReq)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			webhook.Status = v1alpha1.OutboundWebhookStatus{}
			if err = r.Status().Update(ctx, webhook); err != nil {
				return fmt.Errorf("error to update outbound-webhook status -\n%v", webhook)
			}
			return fmt.Errorf("outbound-webhook %s not found on remote, recreating it", *webhook.Status.ID)
		}
		return fmt.Errorf("error to update outbound-webhook -\n%v", webhook)
	}

	log.V(1).Info("Getting outbound-webhook from remote", "id", webhook.Status.ID)
	remoteOutboundWebhook, err := r.OutboundWebhooksClient.Get(ctx,
		&cxsdk.GetOutgoingWebhookRequest{
			Id: coralogix.StringPointerToWrapperspbString(webhook.Status.ID),
		},
	)
	if err != nil {
		return fmt.Errorf("error to get outbound-webhook -\n%v", webhook)
	}
	log.V(1).Info(fmt.Sprintf("outbound-webhook was read\n%s", protojson.Format(remoteOutboundWebhook)))

	status, err := getOutboundWebhookStatus(remoteOutboundWebhook.GetWebhook())
	if err != nil {
		return fmt.Errorf("error to flatten outbound-webhook -\n%v", webhook)
	}
	webhook.Status = *status
	if err = r.Status().Update(ctx, webhook); err != nil {
		return fmt.Errorf("error to update outbound-webhook status -\n%v", webhook)
	}

	return nil
}

func (r *OutboundWebhookReconciler) delete(ctx context.Context, log logr.Logger, webhook *v1alpha1.OutboundWebhook) error {
	if err := r.deleteRemoteWebhook(ctx, log, webhook.Status.ID); err != nil {
		return fmt.Errorf("error to delete outbound-webhook -\n%v", webhook)
	}

	controllerutil.RemoveFinalizer(webhook, outboundWebhookFinalizerName)
	if err := r.Update(ctx, webhook); err != nil {
		return fmt.Errorf("error to update outbound-webhook -\n%v", webhook)
	}

	return nil
}

func (r *OutboundWebhookReconciler) deleteRemoteWebhook(ctx context.Context, log logr.Logger, webhookID *string) error {
	log.V(1).Info("Deleting outbound-webhook from remote", "id", webhookID)
	if _, err := r.OutboundWebhooksClient.Delete(ctx, &cxsdk.DeleteOutgoingWebhookRequest{Id: wrapperspb.String(*webhookID)}); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting outbound-webhook", "id", webhookID)
		return fmt.Errorf("error to delete outbound-webhook -\n%v", webhookID)
	}
	log.V(1).Info("outbound-webhook was deleted from remote", "id", webhookID)

	return nil
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

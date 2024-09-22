/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alphacontrollers

import (
	"context"
	"fmt"
	"strconv"

	utils "github.com/coralogix/coralogix-operator/apis"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	outboundwebhooks "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/outbound-webhooks"
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

	outboundWebhook := &coralogixv1alpha1.OutboundWebhook{}
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
		return ctrl.Result{}, nil
	}

	if !outboundWebhook.ObjectMeta.DeletionTimestamp.IsZero() {
		err = r.delete(ctx, log, outboundWebhook)
		if err != nil {
			log.Error(err, "Error on deleting outbound-webhook")
			return resultError, err
		}
		return ctrl.Result{}, nil
	}

	err = r.update(ctx, log, outboundWebhook)
	if err != nil {
		log.Error(err, "Error on updating outbound-webhook")
		return resultError, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OutboundWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.OutboundWebhook{}).
		Complete(r)
}

func (r *OutboundWebhookReconciler) create(ctx context.Context, log logr.Logger, webhook *coralogixv1alpha1.OutboundWebhook) error {
	createRequest, err := webhook.ExtractCreateOutboundWebhookRequest()
	if err != nil {
		return fmt.Errorf("error to extract create-request out of the outbound-webhook -\n%v", webhook)
	}

	log.V(int(zapcore.DebugLevel)).Info(fmt.Sprintf("Creating outbound-webhook-\n%s", protojson.Format(createRequest)))
	createResponse, err := r.OutboundWebhooksClient.CreateOutboundWebhook(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error to create remote outbound-webhook - %s\n%w", protojson.Format(createRequest), err)
	}
	log.V(int(zapcore.DebugLevel)).Info(fmt.Sprintf("outbound-webhook was created- %s", protojson.Format(createResponse)))

	webhook.Status = coralogixv1alpha1.OutboundWebhookStatus{
		ID:                  ptr.To(createResponse.Id.GetValue()),
		Name:                webhook.Name,
		OutboundWebhookType: &coralogixv1alpha1.OutboundWebhookTypeStatus{},
	}
	if err = r.Status().Update(ctx, webhook); err != nil {
		return fmt.Errorf("error to update outbound-webhook status -\n%v", webhook)
	}

	readRequest := &outboundwebhooks.GetOutgoingWebhookRequest{Id: createResponse.Id}
	log.V(int(zapcore.DebugLevel)).Info(fmt.Sprintf("Getting outbound-webhook -\n%s", protojson.Format(readRequest)))
	readResponse, err := r.OutboundWebhooksClient.GetOutboundWebhook(ctx, readRequest)
	if err != nil {
		return fmt.Errorf("error to get outbound-webhook -\n%v", webhook)
	}
	log.V(int(zapcore.DebugLevel)).Info(fmt.Sprintf("outbound-webhook was read -\n%s", protojson.Format(readResponse)))

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

func getOutboundWebhookStatus(webhook *outboundwebhooks.OutgoingWebhook) (*coralogixv1alpha1.OutboundWebhookStatus, error) {
	if webhook == nil {
		return nil, fmt.Errorf("outbound-webhook is nil")
	}

	outboundWebhookType, err := getOutboundWebhookTypeStatus(webhook)
	if err != nil {
		return nil, err
	}

	status := &coralogixv1alpha1.OutboundWebhookStatus{
		ID:                  ptr.To(webhook.Id.GetValue()),
		ExternalID:          ptr.To(strconv.Itoa(int(webhook.ExternalId.GetValue()))),
		Name:                webhook.Name.GetValue(),
		OutboundWebhookType: outboundWebhookType,
	}

	return status, nil
}

func getOutboundWebhookTypeStatus(webhook *outboundwebhooks.OutgoingWebhook) (*coralogixv1alpha1.OutboundWebhookTypeStatus, error) {
	if webhook == nil {
		return nil, fmt.Errorf("outbound-webhook is nil")
	}

	outboundWebhooks := &coralogixv1alpha1.OutboundWebhookTypeStatus{}
	switch webhookType := webhook.Config.(type) {
	case *outboundwebhooks.OutgoingWebhook_GenericWebhook:
		outboundWebhooks.GenericWebhook = getOutboundWebhookGenericTypeStatus(webhookType.GenericWebhook, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_Slack:
		outboundWebhooks.Slack = getOutgoingWebhookSlackStatus(webhookType.Slack, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_PagerDuty:
		outboundWebhooks.PagerDuty = getOutgoingWebhookPagerDutyStatus(webhookType.PagerDuty)
	case *outboundwebhooks.OutgoingWebhook_SendLog:
		outboundWebhooks.SendLog = getOutgoingWebhookSendLogStatus(webhookType.SendLog, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_EmailGroup:
		outboundWebhooks.EmailGroup = getOutgoingWebhookEmailGroupStatus(webhookType.EmailGroup)
	case *outboundwebhooks.OutgoingWebhook_MicrosoftTeams:
		outboundWebhooks.MicrosoftTeams = getOutgoingWebhookMicrosoftTeamsStatus(webhookType.MicrosoftTeams, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_Jira:
		outboundWebhooks.Jira = getOutboundWebhookJiraStatus(webhookType.Jira, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_Opsgenie:
		outboundWebhooks.Opsgenie = getOutboundWebhookOpsgenieStatus(webhookType.Opsgenie, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_Demisto:
		outboundWebhooks.Demisto = getOutboundWebhookDemistoStatus(webhookType.Demisto, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_AwsEventBridge:
		outboundWebhooks.AwsEventBridge = getOutboundWebhookAwsEventBridgeStatus(webhookType.AwsEventBridge)
	default:
		return nil, fmt.Errorf("unsupported outbound-webhook type %T", webhookType)
	}

	return outboundWebhooks, nil
}

func getOutboundWebhookAwsEventBridgeStatus(awsEventBridge *outboundwebhooks.AwsEventBridgeConfig) *coralogixv1alpha1.AwsEventBridge {
	if awsEventBridge == nil {
		return nil
	}

	return &coralogixv1alpha1.AwsEventBridge{
		EventBusArn: awsEventBridge.EventBusArn.GetValue(),
		Detail:      awsEventBridge.Detail.GetValue(),
		DetailType:  awsEventBridge.DetailType.GetValue(),
		Source:      awsEventBridge.Source.GetValue(),
		RoleName:    awsEventBridge.RoleName.GetValue(),
	}
}

func getOutboundWebhookGenericTypeStatus(generic *outboundwebhooks.GenericWebhookConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.GenericWebhookStatus {
	if generic == nil {
		return nil
	}

	return &coralogixv1alpha1.GenericWebhookStatus{
		Uuid:    generic.Uuid.GetValue(),
		Url:     url.GetValue(),
		Method:  coralogixv1alpha1.GenericWebhookMethodTypeFromProto[generic.Method],
		Headers: generic.Headers,
		Payload: utils.WrapperspbStringToStringPointer(generic.Payload),
	}
}

func getOutgoingWebhookSlackStatus(slack *outboundwebhooks.SlackConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.Slack {
	if slack == nil {
		return nil
	}

	return &coralogixv1alpha1.Slack{
		Url:         url.GetValue(),
		Digests:     flattenSlackDigests(slack.Digests),
		Attachments: flattenSlackConfigAttachments(slack.Attachments),
	}
}

func getOutgoingWebhookPagerDutyStatus(pagerDuty *outboundwebhooks.PagerDutyConfig) *coralogixv1alpha1.PagerDuty {
	if pagerDuty == nil {
		return nil
	}

	return &coralogixv1alpha1.PagerDuty{
		ServiceKey: pagerDuty.ServiceKey.GetValue(),
	}
}

func getOutgoingWebhookMicrosoftTeamsStatus(teams *outboundwebhooks.MicrosoftTeamsConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.MicrosoftTeams {
	if teams == nil {
		return nil
	}

	return &coralogixv1alpha1.MicrosoftTeams{
		Url: url.GetValue(),
	}
}

func getOutboundWebhookJiraStatus(jira *outboundwebhooks.JiraConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.Jira {
	if jira == nil {
		return nil
	}

	return &coralogixv1alpha1.Jira{
		ApiToken:   jira.ApiToken.GetValue(),
		Email:      jira.Email.GetValue(),
		ProjectKey: jira.ProjectKey.GetValue(),
		Url:        url.GetValue(),
	}
}

func getOutboundWebhookOpsgenieStatus(opsgenie *outboundwebhooks.OpsgenieConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.Opsgenie {
	if opsgenie == nil {
		return nil
	}

	return &coralogixv1alpha1.Opsgenie{
		Url: url.GetValue(),
	}
}

func getOutboundWebhookDemistoStatus(demisto *outboundwebhooks.DemistoConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.Demisto {
	if demisto == nil {
		return nil
	}

	return &coralogixv1alpha1.Demisto{
		Uuid:    demisto.Uuid.GetValue(),
		Payload: demisto.Payload.GetValue(),
		Url:     url.GetValue(),
	}
}

func flattenSlackDigests(digests []*outboundwebhooks.SlackConfig_Digest) []coralogixv1alpha1.SlackConfigDigest {
	flattenedSlackDigests := make([]coralogixv1alpha1.SlackConfigDigest, 0, len(digests))
	for _, digest := range digests {
		flattenedSlackDigests = append(flattenedSlackDigests, coralogixv1alpha1.SlackConfigDigest{
			Type:     coralogixv1alpha1.SlackConfigDigestTypeFromProto[digest.Type],
			IsActive: digest.IsActive.GetValue(),
		})
	}
	return flattenedSlackDigests
}

func flattenSlackConfigAttachments(attachments []*outboundwebhooks.SlackConfig_Attachment) []coralogixv1alpha1.SlackConfigAttachment {
	flattenedSlackConfigAttachments := make([]coralogixv1alpha1.SlackConfigAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		flattenedSlackConfigAttachments = append(flattenedSlackConfigAttachments, coralogixv1alpha1.SlackConfigAttachment{
			Type:     coralogixv1alpha1.SlackConfigAttachmentTypeFromProto[attachment.Type],
			IsActive: attachment.IsActive.GetValue(),
		})
	}
	return flattenedSlackConfigAttachments
}

func getOutgoingWebhookSendLogStatus(sendLog *outboundwebhooks.SendLogConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.SendLogStatus {
	if sendLog == nil {
		return nil
	}

	return &coralogixv1alpha1.SendLogStatus{
		Payload: sendLog.Payload.GetValue(),
		Uuid:    sendLog.Payload.GetValue(),
		Url:     url.GetValue(),
	}
}

func getOutgoingWebhookEmailGroupStatus(group *outboundwebhooks.EmailGroupConfig) *coralogixv1alpha1.EmailGroup {
	if group == nil {
		return nil
	}

	return &coralogixv1alpha1.EmailGroup{
		EmailAddresses: utils.WrappedStringSliceToStringSlice(group.EmailAddresses),
	}
}

func (r *OutboundWebhookReconciler) update(ctx context.Context, log logr.Logger, webhook *coralogixv1alpha1.OutboundWebhook) error {
	updateReq, err := webhook.ExtractUpdateOutboundWebhookRequest()
	if err != nil {
		return fmt.Errorf("error to parse update outbound-webhook request -\n%v", webhook)
	}

	log.V(int(zapcore.DebugLevel)).Info(fmt.Sprintf("updating outbound-webhook\n%s", protojson.Format(updateReq)))
	_, err = r.OutboundWebhooksClient.UpdateOutboundWebhook(ctx, updateReq)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			webhook.Status = coralogixv1alpha1.OutboundWebhookStatus{}
			if err = r.Status().Update(ctx, webhook); err != nil {
				return fmt.Errorf("error to update outbound-webhook status -\n%v", webhook)
			}
			return fmt.Errorf("outbound-webhook %s not found on remote, recreating it", *webhook.Status.ID)
		}
		return fmt.Errorf("error to update outbound-webhook -\n%v", webhook)
	}

	log.V(int(zapcore.DebugLevel)).Info("Getting outbound-webhook from remote", "id", webhook.Status.ID)
	remoteOutboundWebhook, err := r.OutboundWebhooksClient.GetOutboundWebhook(ctx,
		&outboundwebhooks.GetOutgoingWebhookRequest{
			Id: utils.StringPointerToWrapperspbString(webhook.Status.ID),
		},
	)
	if err != nil {
		return fmt.Errorf("error to get outbound-webhook -\n%v", webhook)
	}
	log.V(int(zapcore.DebugLevel)).Info(fmt.Sprintf("outbound-webhook was read\n%s", protojson.Format(remoteOutboundWebhook)))

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

func (r *OutboundWebhookReconciler) delete(ctx context.Context, log logr.Logger, webhook *coralogixv1alpha1.OutboundWebhook) error {
	log.V(int(zapcore.DebugLevel)).Info("Deleting outbound-webhook from remote", "id", webhook.Status.ID)
	if _, err := r.OutboundWebhooksClient.DeleteOutboundWebhook(ctx,
		&outboundwebhooks.DeleteOutgoingWebhookRequest{Id: wrapperspb.String(*webhook.Status.ID)}); err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("error to delete outbound-webhook -\n%v", webhook)
	}
	log.V(int(zapcore.DebugLevel)).Info("outbound-webhook was deleted from remote", "id", webhook.Status.ID)

	controllerutil.RemoveFinalizer(webhook, outboundWebhookFinalizerName)
	if err := r.Update(ctx, webhook); err != nil {
		return fmt.Errorf("error to update outbound-webhook -\n%v", webhook)
	}

	return nil
}

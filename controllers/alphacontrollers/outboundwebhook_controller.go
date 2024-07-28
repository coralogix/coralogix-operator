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

	utils "github.com/coralogix/coralogix-operator/apis"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	outboundwebhooks "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/outbound-webhooks"
	"github.com/go-logr/logr"
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

//+kubebuilder:rbac:groups=coralogix.coralogix.com,resources=outboundwebhooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.coralogix.com,resources=outboundwebhooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.coralogix.com,resources=outboundwebhooks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OutboundWebhook object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
var (
	outboundWebhookFinalizerName = "outbound-webhook.coralogix.com/finalizer"
)

func (r *OutboundWebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		resultError = ctrl.Result{RequeueAfter: 40}
		resultOk    = ctrl.Result{RequeueAfter: defaultRequeuePeriod}
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
		return resultOk, nil
	}

	if !outboundWebhook.ObjectMeta.DeletionTimestamp.IsZero() {
		err = r.delete(ctx, log, outboundWebhook)
		if err != nil {
			log.Error(err, "Error on deleting outbound-webhook")
			return resultError, err
		}
		return resultOk, nil
	}

	err = r.update(ctx, log, outboundWebhook)
	if err != nil {
		log.Error(err, "Error on updating outbound-webhook")
		return resultError, err
	}

	return resultOk, nil
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
		log.Error(err, fmt.Sprintf("Error to extrac create-request out of the outbound-webhook -\n%s", webhook))
		return err
	}

	//1 level means debug - for every backend request
	log.V(1).Info(fmt.Sprintf("Creating outbound-webhook-\n%s", protojson.Format(createRequest)))
	createResponse, err := r.OutboundWebhooksClient.CreateOutboundWebhook(ctx, createRequest)
	if err != nil {
		log.Error(err, fmt.Sprintf("Received an error while creating outbound-webhook -\n%s", protojson.Format(createRequest)))
		return err
	}
	log.Info(fmt.Sprintf("outbound-webhook was created-\n%s", protojson.Format(createResponse)))

	webhook.Status = coralogixv1alpha1.OutboundWebhookStatus{
		ID:                  ptr.To(createResponse.Id.GetValue()),
		Name:                webhook.Name,
		OutboundWebhookType: &coralogixv1alpha1.OutboundWebhookType{},
	}
	if err = r.Status().Update(ctx, webhook); err != nil {
		log.Error(err, fmt.Sprintf("Error on updating outbound-webhook status -\n%s", webhook))
		return err
	}

	readRequest := &outboundwebhooks.GetOutgoingWebhookRequest{Id: createResponse.Id}
	readResponse, err := r.OutboundWebhooksClient.GetOutboundWebhook(ctx, readRequest)
	if err != nil {
		log.Error(err, fmt.Sprintf("Received an error while getting outbound-webhook -\n%s", protojson.Format(readRequest)))
		return err
	}

	status, err := getOutboundWebhookStatus(readResponse.GetWebhook())
	if err != nil {
		log.Error(err, "Received an error while getting outbound-webhook status")
		return err
	}

	webhook.Status = *status
	if err = r.Status().Update(ctx, webhook); err != nil {
		log.Error(err, "Error on updating outbound-webhook status")
		return err
	}

	if !controllerutil.ContainsFinalizer(webhook, outboundWebhookFinalizerName) {
		controllerutil.AddFinalizer(webhook, outboundWebhookFinalizerName)
	}

	if err = r.Client.Update(ctx, webhook); err != nil {
		log.Error(err, "Error on updating outbound-webhook")
		return err
	}

	return nil
}

func getOutboundWebhookStatus(webhook *outboundwebhooks.OutgoingWebhook) (*coralogixv1alpha1.OutboundWebhookStatus, error) {
	if webhook == nil {
		return nil, fmt.Errorf("outbound-webhook is nil")
	}

	outboundWebhookType, err := getOutboundWebhookType(webhook)
	if err != nil {
		return nil, err
	}

	status := &coralogixv1alpha1.OutboundWebhookStatus{
		ID:                  ptr.To(webhook.Id.GetValue()),
		Name:                webhook.Name.GetValue(),
		OutboundWebhookType: outboundWebhookType,
	}

	return status, nil
}

func getOutboundWebhookType(webhook *outboundwebhooks.OutgoingWebhook) (*coralogixv1alpha1.OutboundWebhookType, error) {
	if webhook == nil {
		return nil, fmt.Errorf("outbound-webhook is nil")
	}

	outboundWebhooks := &coralogixv1alpha1.OutboundWebhookType{}
	switch webhookType := webhook.Config.(type) {
	case *outboundwebhooks.OutgoingWebhook_GenericWebhook:
		outboundWebhooks.GenericWebhook = getOutboundWebhookGenericType(webhookType.GenericWebhook, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_Slack:
		outboundWebhooks.Slack = getOutgoingWebhookSlackType(webhookType.Slack, webhook.Url)
	case *outboundwebhooks.OutgoingWebhook_PagerDuty:
		outboundWebhooks.PagerDuty = getOutgoingWebhookPagerDutyType(webhookType.PagerDuty)
	case *outboundwebhooks.OutgoingWebhook_SendLog:
		outboundWebhooks.SendLog = getOutgoingWebhookSendLogType(webhookType.SendLog)
	case *outboundwebhooks.OutgoingWebhook_EmailGroup:
		outboundWebhooks.EmailGroup = getOutgoingWebhookEmailGroupType(webhookType.EmailGroup)
	case *outboundwebhooks.OutgoingWebhook_MicrosoftTeams:
		outboundWebhooks.MicrosoftTeams = getOutgoingWebhookMicrosoftTeamsType(webhookType.MicrosoftTeams)
	case *outboundwebhooks.OutgoingWebhook_Jira:
		outboundWebhooks.Jira = getOutboundWebhookJiraType(webhookType.Jira)
	case *outboundwebhooks.OutgoingWebhook_Opsgenie:
		outboundWebhooks.Opsgenie = getOutboundWebhookOpsgenieType(webhookType.Opsgenie)
	case *outboundwebhooks.OutgoingWebhook_Demisto:
		outboundWebhooks.Demisto = getOutboundWebhookDemistoType(webhookType.Demisto)
	case *outboundwebhooks.OutgoingWebhook_AwsEventBridge:
		outboundWebhooks.AwsEventBridge = getOutboundWebhookAwsEventBridgeType(webhookType.AwsEventBridge)
	default:
		return nil, fmt.Errorf("unsupported outbound-webhook type %T", webhookType)
	}

	return outboundWebhooks, nil
}

func getOutboundWebhookAwsEventBridgeType(awsEventBridge *outboundwebhooks.AwsEventBridgeConfig) *coralogixv1alpha1.AwsEventBridge {
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

func getOutboundWebhookGenericType(generic *outboundwebhooks.GenericWebhookConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.GenericWebhook {
	if generic == nil {
		return nil
	}

	return &coralogixv1alpha1.GenericWebhook{
		Uuid:    WrapperspbStringToStringPointer(generic.Uuid),
		Url:     WrapperspbStringToStringPointer(url),
		Method:  coralogixv1alpha1.GenericWebhookMethodTypeFromProto[generic.Method],
		Headers: generic.Headers,
		Payload: WrapperspbStringToStringPointer(generic.Payload),
	}
}

func getOutgoingWebhookSlackType(slack *outboundwebhooks.SlackConfig, url *wrapperspb.StringValue) *coralogixv1alpha1.Slack {
	if slack == nil {
		return nil
	}

	return &coralogixv1alpha1.Slack{
		Url:         WrapperspbStringToStringPointer(url),
		Digests:     flattenSlackDigests(slack.Digests),
		Attachments: flattenSlackConfigAttachments(slack.Attachments),
	}
}

func getOutgoingWebhookPagerDutyType(pagerDuty *outboundwebhooks.PagerDutyConfig) *coralogixv1alpha1.PagerDuty {
	if pagerDuty == nil {
		return nil
	}

	return &coralogixv1alpha1.PagerDuty{
		ServiceKey: pagerDuty.ServiceKey.GetValue(),
	}
}

func getOutgoingWebhookMicrosoftTeamsType(teams *outboundwebhooks.MicrosoftTeamsConfig) *coralogixv1alpha1.MicrosoftTeams {
	if teams == nil {
		return nil
	}

	return &coralogixv1alpha1.MicrosoftTeams{}
}

func getOutboundWebhookJiraType(jira *outboundwebhooks.JiraConfig) *coralogixv1alpha1.Jira {
	if jira == nil {
		return nil
	}

	return &coralogixv1alpha1.Jira{
		ApiToken:   jira.ApiToken.GetValue(),
		Email:      jira.Email.GetValue(),
		ProjectKey: jira.ProjectKey.GetValue(),
	}
}

func getOutboundWebhookOpsgenieType(opsgenie *outboundwebhooks.OpsgenieConfig) *coralogixv1alpha1.Opsgenie {
	if opsgenie == nil {
		return nil
	}

	return &coralogixv1alpha1.Opsgenie{}
}

func getOutboundWebhookDemistoType(demisto *outboundwebhooks.DemistoConfig) *coralogixv1alpha1.Demisto {
	if demisto == nil {
		return nil
	}

	return &coralogixv1alpha1.Demisto{
		Uuid:    demisto.Uuid.GetValue(),
		Payload: demisto.Payload.GetValue(),
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

func getOutgoingWebhookSendLogType(sendLog *outboundwebhooks.SendLogConfig) *coralogixv1alpha1.SendLog {
	if sendLog == nil {
		return nil
	}

	return &coralogixv1alpha1.SendLog{
		Uuid:    sendLog.Uuid.GetValue(),
		Payload: sendLog.Payload.GetValue(),
	}
}

func getOutgoingWebhookEmailGroupType(group *outboundwebhooks.EmailGroupConfig) *coralogixv1alpha1.EmailGroup {
	if group == nil {
		return nil
	}

	return &coralogixv1alpha1.EmailGroup{
		EmailAddresses: utils.WrappedStringSliceToStringSlice(group.EmailAddresses),
	}
}

func (r *OutboundWebhookReconciler) update(ctx context.Context, log logr.Logger, webhook *coralogixv1alpha1.OutboundWebhook) error {
	log.Info("Getting outbound-webhook from remote", "id", webhook.Status.ID)
	remoteOutboundWebhook, err := r.OutboundWebhooksClient.GetOutboundWebhook(ctx,
		&outboundwebhooks.GetOutgoingWebhookRequest{
			Id: utils.StringPointerToWrapperspbString(webhook.Status.ID)},
	)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Info("outbound-webhook not found on remote, recreating it", "id", webhook.Status.ID)
			webhook.Status = coralogixv1alpha1.OutboundWebhookStatus{}
			if err = r.Status().Update(ctx, webhook); err != nil {
				log.Error(err, "Error on updating outbound-webhook status")
				return err
			}
			return err
		}
		log.Error(err, "Error on getting outbound-webhook", "id", webhook.Status.ID)
		return err
	}

	status, err := getOutboundWebhookStatus(remoteOutboundWebhook.GetWebhook())
	if err != nil {
		log.Error(err, "Error on flattening outbound-webhook")
		return err
	}

	if equal, diff := webhook.Spec.DeepEqual(status); equal {
		return nil
	} else {
		log.Info("Outbound-webhook is not equal to remote, updating it", "path", diff.Name, "desired", diff.Desired, "actual", diff.Actual)
	}

	updateReq, err := webhook.ExtractUpdateOutboundWebhookRequest()
	if err != nil {
		log.Error(err, "Error to parse update outbound-webhook request")
		return err
	}

	log.Info(fmt.Sprintf("updating outbound-webhook\n%s", protojson.Format(updateReq)))
	_, err = r.OutboundWebhooksClient.UpdateOutboundWebhook(ctx, updateReq)
	if err != nil {
		log.Error(err, fmt.Sprintf("Error on remote updating outbound-webhook\n%s", protojson.Format(updateReq)))
		return err
	}

	return nil
}

func (r *OutboundWebhookReconciler) delete(ctx context.Context, log logr.Logger, webhook *coralogixv1alpha1.OutboundWebhook) error {
	log.Info("Deleting outbound-webhook from remote", "id", webhook.Status.ID)
	_, err := r.OutboundWebhooksClient.DeleteOutboundWebhook(ctx, &outboundwebhooks.DeleteOutgoingWebhookRequest{
		Id: wrapperspb.String(*webhook.Status.ID),
	})
	if err != nil && status.Code(err) != codes.NotFound {
		log.Error(err, "Error on deleting outbound-webhook from remote")
		return err
	}

	controllerutil.RemoveFinalizer(webhook, outboundWebhookFinalizerName)
	err = r.Update(ctx, webhook)
	if err != nil {
		log.Error(err, "Error on updating outbound-webhook after deletion")
		return err
	}

	return nil
}

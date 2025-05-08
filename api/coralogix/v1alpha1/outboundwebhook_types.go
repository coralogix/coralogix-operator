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
	"fmt"

	gouuid "github.com/google/uuid"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
)

// OutboundWebhookSpec defines the desired state of OutboundWebhook
// See also https://coralogix.com/docs/user-guides/alerting/outbound-webhooks/aws-eventbridge-outbound-webhook/
type OutboundWebhookSpec struct {
	//+kubebuilder:validation:MinLength=0
	// Name of the webhook.
	Name string `json:"name"`

	// Type of webhook.
	OutboundWebhookType OutboundWebhookType `json:"outboundWebhookType"`
}

// Webhook type
// +kubebuilder:validation:XValidation:rule="(self.genericWebhook != null ? 1 : 0) + (self.slack != null ? 1 : 0) + (self.pagerDuty != null ? 1 : 0) + (self.sendLog != null ? 1 : 0) + (self.emailGroup != null ? 1 : 0) + (self.microsoftTeams != null ? 1 : 0) + (self.jira != null ? 1 : 0) + (self.opsgenie != null ? 1 : 0) + (self.demisto != null ? 1 : 0) + (self.awsEventBridge != null ? 1 : 0) == 1",message="Exactly one of genericWebhook, slack, pagerDuty, sendLog, emailGroup, microsoftTeams, jira, opsgenie, demisto or awsEventBridge is required"
type OutboundWebhookType struct {
	// Generic HTTP(s) webhook.
	// +optional
	GenericWebhook *GenericWebhook `json:"genericWebhook,omitempty"`

	// Slack message.
	// +optional
	Slack *Slack `json:"slack,omitempty"`

	// PagerDuty notification.
	// +optional
	PagerDuty *PagerDuty `json:"pagerDuty,omitempty"`

	// SendLog notification.
	// +optional
	SendLog *SendLog `json:"sendLog,omitempty"`

	// Email notification.
	// +optional
	EmailGroup *EmailGroup `json:"emailGroup,omitempty"`

	// Teams message.
	// +optional
	MicrosoftTeams *MicrosoftTeams `json:"microsoftTeams,omitempty"`

	// Jira issue.
	// +optional
	Jira *Jira `json:"jira,omitempty"`

	// Opsgenie notification.
	// +optional
	Opsgenie *Opsgenie `json:"opsgenie,omitempty"`

	// Demisto notification.
	// +optional
	Demisto *Demisto `json:"demisto,omitempty"`

	// AWS eventbridge message.
	// +optional
	AwsEventBridge *AwsEventBridge `json:"awsEventBridge,omitempty"`
}

func (in *OutboundWebhookType) appendOutgoingWebhookConfig(data *cxsdk.OutgoingWebhookInputData) (*cxsdk.OutgoingWebhookInputData, error) {
	if genericWebhook := in.GenericWebhook; genericWebhook != nil {
		data.Config = genericWebhook.extractGenericWebhookConfig()
		data.Type = cxsdk.WebhookTypeGeneric
		data.Url = wrapperspb.String(genericWebhook.Url)
	} else if slack := in.Slack; slack != nil {
		data.Config = slack.extractSlackConfig()
		data.Type = cxsdk.WebhookTypeSlack
		data.Url = wrapperspb.String(slack.Url)
	} else if pagerDuty := in.PagerDuty; pagerDuty != nil {
		data.Config = pagerDuty.extractPagerDutyConfig()
		data.Type = cxsdk.WebhookTypePagerduty
	} else if sendLog := in.SendLog; sendLog != nil {
		data.Config = sendLog.extractSendLogConfig()
		data.Type = cxsdk.WebhookTypeSendLog
		data.Url = wrapperspb.String(sendLog.Url)
	} else if emailGroup := in.EmailGroup; emailGroup != nil {
		data.Config = emailGroup.extractEmailGroupConfig()
		data.Type = cxsdk.WebhookTypeEmailGroup
	} else if microsoftTeams := in.MicrosoftTeams; microsoftTeams != nil {
		data.Config = microsoftTeams.extractMicrosoftTeamsConfig()
		data.Url = wrapperspb.String(microsoftTeams.Url)
		data.Type = cxsdk.WebhookTypeMicrosoftTeams
	} else if jira := in.Jira; jira != nil {
		data.Config = jira.extractJiraConfig()
		data.Url = wrapperspb.String(jira.Url)
		data.Type = cxsdk.WebhookTypeJira
	} else if opsgenie := in.Opsgenie; opsgenie != nil {
		data.Config = opsgenie.extractOpsgenieConfig()
		data.Type = cxsdk.WebhookTypeOpsgenie
		data.Url = wrapperspb.String(opsgenie.Url)
	} else if demisto := in.Demisto; demisto != nil {
		data.Config = demisto.extractDemistoConfig()
		data.Url = wrapperspb.String(demisto.Url)
		data.Type = cxsdk.WebhookTypeDemisto
	} else if in.AwsEventBridge != nil {
		data.Config = in.AwsEventBridge.extractAwsEventBridgeConfig()
		data.Type = cxsdk.WebhookTypeAwsEventBridge
	} else {
		return nil, fmt.Errorf("unsupported outbound-webhook type")
	}

	return data, nil
}

// Generic HTTP(s) webhook.
type GenericWebhook struct {

	// URL to call
	Url string `json:"url"`

	// HTTP Method to use.
	Method GenericWebhookMethodType `json:"method"`

	// Attached HTTP headers.
	// +optional
	Headers map[string]string `json:"headers"`

	// Payload of the webhook call.
	// +optional
	Payload *string `json:"payload"`
}

// Status of the webhook call.
type GenericWebhookStatus struct {
	// ID
	Uuid string `json:"uuid"`

	// Called URL
	Url string `json:"url"`

	// HTTP method.
	Method GenericWebhookMethodType `json:"method"`

	// Headers of the call.
	// +optional
	Headers map[string]string `json:"headers"`

	// Payland of the call.
	// +optional
	Payload *string `json:"payload"`
}

func (in *GenericWebhook) extractGenericWebhookConfig() *cxsdk.GenericWebhookInputData {
	return &cxsdk.GenericWebhookInputData{
		GenericWebhook: &cxsdk.GenericWebhookConfig{
			Uuid:    wrapperspb.String(gouuid.NewString()),
			Method:  GenericWebhookMethodTypeToProto[in.Method],
			Headers: in.Headers,
			Payload: utils.StringPointerToWrapperspbString(in.Payload),
		},
	}
}

// +kubebuilder:validation:Enum=Unkown;Get;Post;Put
type GenericWebhookMethodType string

const (
	GenericWebhookMethodTypeUNKNOWN GenericWebhookMethodType = "Unknown"
	GenericWebhookMethodTypeGet     GenericWebhookMethodType = "Get"
	GenericWebhookMethodTypePost    GenericWebhookMethodType = "Post"
	GenericWebhookMethodTypePut     GenericWebhookMethodType = "Put"
)

var (
	GenericWebhookMethodTypeToProto = map[GenericWebhookMethodType]cxsdk.GenericWebhookConfigMethodType{
		GenericWebhookMethodTypeUNKNOWN: cxsdk.GenericWebhookConfigUnknown,
		GenericWebhookMethodTypeGet:     cxsdk.GenericWebhookConfigGet,
		GenericWebhookMethodTypePost:    cxsdk.GenericWebhookConfigPost,
		GenericWebhookMethodTypePut:     cxsdk.GenericWebhookConfigPut,
	}
	GenericWebhookMethodTypeFromProto = utils.ReverseMap(GenericWebhookMethodTypeToProto)
)

type Slack struct {

	// Digest configuration.
	// +optional
	Digests []SlackConfigDigest `json:"digests"`

	// Attachments of the message.
	// +optional
	Attachments []SlackConfigAttachment `json:"attachments"`
	Url         string                  `json:"url"`
}

func (in *Slack) extractSlackConfig() *cxsdk.SlackWebhookInputData {
	digests := make([]*cxsdk.SlackConfigDigest, 0)
	for _, digest := range in.Digests {
		digests = append(digests, &cxsdk.SlackConfigDigest{
			Type:     SlackConfigDigestTypeToProto[digest.Type],
			IsActive: wrapperspb.Bool(digest.IsActive),
		})
	}

	attachments := make([]*cxsdk.SlackConfigAttachment, 0)
	for _, attachment := range in.Attachments {
		attachments = append(attachments, &cxsdk.SlackConfigAttachment{
			Type:     SlackConfigAttachmentTypeToProto[attachment.Type],
			IsActive: wrapperspb.Bool(attachment.IsActive),
		})
	}

	return &cxsdk.SlackWebhookInputData{
		Slack: &cxsdk.SlackConfig{
			Digests:     digests,
			Attachments: attachments,
		},
	}
}

// Slack config digest type.
type SlackConfigDigestType string

// Slack config digest values.
const (
	SlackConfigDigestTypeUnknown              SlackConfigDigestType = "Unknown"
	SlackConfigDigestTypeErrorAndCriticalLogs SlackConfigDigestType = "ErrorAndCriticalLogs"
	SlackConfigDigestTypeFlowAnomalies        SlackConfigDigestType = "FlowAnomalies"
	SlackConfigSpikeAnomalies                 SlackConfigDigestType = "SpikeAnomalies"
	SlackConfigDigestTypeDataUsage            SlackConfigDigestType = "DataUsage"
)

var (
	SlackConfigDigestTypeToProto = map[SlackConfigDigestType]cxsdk.SlackConfigDigestType{
		SlackConfigDigestTypeUnknown:              cxsdk.SlackConfigUnknown,
		SlackConfigDigestTypeErrorAndCriticalLogs: cxsdk.SlackConfigErrorAndCriticalLogs,
		SlackConfigDigestTypeFlowAnomalies:        cxsdk.SlackConfigFlowAnomalies,
		SlackConfigSpikeAnomalies:                 cxsdk.SlackConfigSpikeAnomalies,
		SlackConfigDigestTypeDataUsage:            cxsdk.SlackConfigDataUsage,
	}
	SlackConfigDigestTypeFromProto = utils.ReverseMap(SlackConfigDigestTypeToProto)
)

// Digest config.
type SlackConfigDigest struct {
	// Type of digest to send
	Type SlackConfigDigestType `json:"type"`

	// Active status.
	IsActive bool `json:"isActive"`
}

// Slack attachment
type SlackConfigAttachment struct {
	// Attachment to the message.
	Type SlackConfigAttachmentType `json:"type"`

	// Active status.
	IsActive bool `json:"isActive"`
}

// Attachment type.
type SlackConfigAttachmentType string

// Attachment type values.
const (
	SlackConfigAttachmentTypeEmpty          SlackConfigAttachmentType = "Empty"
	SlackConfigAttachmentTypeMetricSnapshot SlackConfigAttachmentType = "MetricSnapshot"
	SlackConfigAttachmentTypeLogs           SlackConfigAttachmentType = "Logs"
)

var (
	SlackConfigAttachmentTypeToProto = map[SlackConfigAttachmentType]cxsdk.SlackConfigAttachmentType{
		SlackConfigAttachmentTypeEmpty:          cxsdk.SlackConfigEmpty,
		SlackConfigAttachmentTypeMetricSnapshot: cxsdk.SlackConfigMetricSnapshot,
		SlackConfigAttachmentTypeLogs:           cxsdk.SlackConfigLogs,
	}
	SlackConfigAttachmentTypeFromProto = utils.ReverseMap(SlackConfigAttachmentTypeToProto)
)

// PagerDuty configuration.
type PagerDuty struct {
	// PagerDuty service key.
	ServiceKey string `json:"serviceKey"`
}

func (in *PagerDuty) extractPagerDutyConfig() *cxsdk.PagerDutyWebhookInputData {
	return &cxsdk.PagerDutyWebhookInputData{
		PagerDuty: &cxsdk.PagerDutyConfig{
			ServiceKey: wrapperspb.String(in.ServiceKey),
		},
	}
}

// SendLog configuration.
type SendLog struct {
	// Payload of the notification
	Payload string `json:"payload"`

	// Sendlog URL.
	Url string `json:"url"`
}

// SendLog status.
type SendLogStatus struct {
	// Payload of the SendLog notification
	Payload string `json:"payload"`
	// SendLog URL
	Url string `json:"url"`
	// ID
	Uuid string `json:"uuid"`
}

func (in *SendLog) extractSendLogConfig() *cxsdk.SendLogWebhookInputData {
	return &cxsdk.SendLogWebhookInputData{
		SendLog: &cxsdk.SendLogConfig{
			Payload: wrapperspb.String(in.Payload),
			Uuid:    wrapperspb.String(gouuid.NewString()),
		},
	}
}

// EMail notification configuration
type EmailGroup struct {
	// Recipients
	EmailAddresses []string `json:"emailAddresses"`
}

func (in *EmailGroup) extractEmailGroupConfig() *cxsdk.EmailGroupWebhookInputData {
	return &cxsdk.EmailGroupWebhookInputData{
		EmailGroup: &cxsdk.EmailGroupConfig{
			EmailAddresses: utils.StringSliceToWrappedStringSlice(in.EmailAddresses),
		},
	}
}

// Microsoft Teams configuration.
type MicrosoftTeams struct {
	// Teams URL
	Url string `json:"url"`
}

func (in *MicrosoftTeams) extractMicrosoftTeamsConfig() *cxsdk.MicrosoftTeamsWebhookInputData {
	return &cxsdk.MicrosoftTeamsWebhookInputData{
		MicrosoftTeams: &cxsdk.MicrosoftTeamsConfig{},
	}
}

// Jira configuration
type Jira struct {
	// API token
	ApiToken string `json:"apiToken"`

	// Email address associated with the token
	Email string `json:"email"`

	// Project to add it to.
	ProjectKey string `json:"projectKey"`

	// Jira URL
	Url string `json:"url"`
}

func (in *Jira) extractJiraConfig() *cxsdk.JiraWebhookInputData {
	return &cxsdk.JiraWebhookInputData{
		Jira: &cxsdk.JiraConfig{
			ApiToken:   wrapperspb.String(in.ApiToken),
			Email:      wrapperspb.String(in.Email),
			ProjectKey: wrapperspb.String(in.ProjectKey),
		},
	}
}

type Opsgenie struct {
	Url string `json:"url"`
}

func (in *Opsgenie) extractOpsgenieConfig() *cxsdk.OpsgenieWebhookInputData {
	return &cxsdk.OpsgenieWebhookInputData{
		Opsgenie: &cxsdk.OpsgenieConfig{},
	}
}

type Demisto struct {
	Uuid    string `json:"uuid"`
	Payload string `json:"payload"`
	Url     string `json:"url"`
}

func (in *Demisto) extractDemistoConfig() *cxsdk.DemistoWebhookInputData {
	return &cxsdk.DemistoWebhookInputData{
		Demisto: &cxsdk.DemistoConfig{
			Uuid:    wrapperspb.String(in.Uuid),
			Payload: wrapperspb.String(in.Payload),
		},
	}
}

type AwsEventBridge struct {
	EventBusArn string `json:"eventBusArn"`
	Detail      string `json:"detail"`
	DetailType  string `json:"detailType"`
	Source      string `json:"source"`
	RoleName    string `json:"roleName"`
}

func (in *AwsEventBridge) extractAwsEventBridgeConfig() *cxsdk.AwsEventBridgeWebhookInputData {
	return &cxsdk.AwsEventBridgeWebhookInputData{
		AwsEventBridge: &cxsdk.AwsEventBridgeConfig{
			EventBusArn: wrapperspb.String(in.EventBusArn),
			Detail:      wrapperspb.String(in.Detail),
			DetailType:  wrapperspb.String(in.DetailType),
			Source:      wrapperspb.String(in.Source),
			RoleName:    wrapperspb.String(in.RoleName),
		},
	}
}

// OutboundWebhookStatus defines the observed state of OutboundWebhook
type OutboundWebhookStatus struct {
	// +optional
	ID *string `json:"id"`
	// +optional
	ExternalID *string `json:"externalId"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (in *OutboundWebhook) GetConditions() []metav1.Condition {
	return in.Status.Conditions
}

func (in *OutboundWebhook) SetConditions(conditions []metav1.Condition) {
	in.Status.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OutboundWebhook is the Schema for the API
type OutboundWebhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OutboundWebhookSpec   `json:"spec,omitempty"`
	Status OutboundWebhookStatus `json:"status,omitempty"`
}

func (in *OutboundWebhook) ExtractCreateOutboundWebhookRequest() (*cxsdk.CreateOutgoingWebhookRequest, error) {
	webhookData, err := in.Spec.ExtractOutgoingWebhookInputData()
	if err != nil {
		return nil, err
	}

	return &cxsdk.CreateOutgoingWebhookRequest{
		Data: webhookData,
	}, nil
}

func (in *OutboundWebhook) ExtractUpdateOutboundWebhookRequest() (*cxsdk.UpdateOutgoingWebhookRequest, error) {
	webhookData, err := in.Spec.ExtractOutgoingWebhookInputData()
	if err != nil {
		return nil, err
	}

	if in.Status.ID == nil {
		return nil, fmt.Errorf("outbound-webhook id is not set")
	}

	return &cxsdk.UpdateOutgoingWebhookRequest{
		Id:   *in.Status.ID,
		Data: webhookData,
	}, nil
}

func (in *OutboundWebhookSpec) ExtractOutgoingWebhookInputData() (*cxsdk.OutgoingWebhookInputData, error) {
	webhookData := &cxsdk.OutgoingWebhookInputData{
		Name: wrapperspb.String(in.Name),
	}
	return in.OutboundWebhookType.appendOutgoingWebhookConfig(webhookData)
}

//+kubebuilder:object:root=true

// OutboundWebhookList contains a list of OutboundWebhook
type OutboundWebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OutboundWebhook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OutboundWebhook{}, &OutboundWebhookList{})
}

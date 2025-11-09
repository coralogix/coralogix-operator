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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	webhooks "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/outgoing_webhooks_service"
)

// OutboundWebhookSpec defines the desired state of an outbound webhook.
type OutboundWebhookSpec struct {
	//+kubebuilder:validation:MinLength=0
	// Name of the webhook.
	Name string `json:"name"`

	// Type of webhook.
	OutboundWebhookType OutboundWebhookType `json:"outboundWebhookType"`
}

// Webhook type
// +kubebuilder:validation:XValidation:rule="(has(self.genericWebhook) ? 1 : 0) + (has(self.slack) ? 1 : 0) + (has(self.pagerDuty) ? 1 : 0) + (has(self.sendLog) ? 1 : 0) + (has(self.emailGroup) ? 1 : 0) + (has(self.microsoftTeams) ? 1 : 0) + (has(self.jira) ? 1 : 0) + (has(self.opsgenie) ? 1 : 0) + (has(self.demisto) ? 1 : 0) + (has(self.awsEventBridge) ? 1 : 0) == 1",message="Exactly one of the following fields must be set: genericWebhook, slack, pagerDuty, sendLog, emailGroup, microsoftTeams, jira, opsgenie, demisto, awsEventBridge"
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

func (in *GenericWebhook) extractGenericWebhookConfig() *webhooks.GenericWebhookConfig {
	return &webhooks.GenericWebhookConfig{
		Uuid:    webhooks.PtrString(gouuid.NewString()),
		Method:  GenericWebhookMethodTypeToOpenAPI[in.Method].Ptr(),
		Headers: ptr.To(in.Headers),
		Payload: in.Payload,
	}
}

// +kubebuilder:validation:Enum=Unknown;Get;Post;Put
type GenericWebhookMethodType string

const (
	GenericWebhookMethodTypeUNKNOWN GenericWebhookMethodType = "Unknown"
	GenericWebhookMethodTypeGet     GenericWebhookMethodType = "Get"
	GenericWebhookMethodTypePost    GenericWebhookMethodType = "Post"
	GenericWebhookMethodTypePut     GenericWebhookMethodType = "Put"
)

var (
	GenericWebhookMethodTypeToOpenAPI = map[GenericWebhookMethodType]webhooks.MethodType{
		GenericWebhookMethodTypeUNKNOWN: webhooks.METHODTYPE_UNKNOWN,
		GenericWebhookMethodTypeGet:     webhooks.METHODTYPE_GET,
		GenericWebhookMethodTypePost:    webhooks.METHODTYPE_POST,
		GenericWebhookMethodTypePut:     webhooks.METHODTYPE_PUT,
	}
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

func (in *Slack) extractSlackConfig() *webhooks.SlackConfig {
	digests := make([]webhooks.Digest, 0)
	for _, digest := range in.Digests {
		digests = append(digests, webhooks.Digest{
			Type:     SlackConfigDigestTypeToOpenAPI[digest.Type].Ptr(),
			IsActive: webhooks.PtrBool(digest.IsActive),
		})
	}

	attachments := make([]webhooks.Attachment, 0)
	for _, attachment := range in.Attachments {
		attachments = append(attachments, webhooks.Attachment{
			Type:     SlackConfigAttachmentTypeToOpenAPI[attachment.Type].Ptr(),
			IsActive: webhooks.PtrBool(attachment.IsActive),
		})
	}

	return &webhooks.SlackConfig{
		Digests:     digests,
		Attachments: attachments,
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
	SlackConfigDigestTypeToOpenAPI = map[SlackConfigDigestType]webhooks.DigestType{
		SlackConfigDigestTypeUnknown:              webhooks.DIGESTTYPE_UNKNOWN,
		SlackConfigDigestTypeErrorAndCriticalLogs: webhooks.DIGESTTYPE_ERROR_AND_CRITICAL_LOGS,
		SlackConfigDigestTypeFlowAnomalies:        webhooks.DIGESTTYPE_FLOW_ANOMALIES,
		SlackConfigSpikeAnomalies:                 webhooks.DIGESTTYPE_SPIKE_ANOMALIES,
		SlackConfigDigestTypeDataUsage:            webhooks.DIGESTTYPE_DATA_USAGE,
	}
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
	SlackConfigAttachmentTypeToOpenAPI = map[SlackConfigAttachmentType]webhooks.AttachmentType{
		SlackConfigAttachmentTypeEmpty:          webhooks.ATTACHMENTTYPE_EMPTY,
		SlackConfigAttachmentTypeMetricSnapshot: webhooks.ATTACHMENTTYPE_METRIC_SNAPSHOT,
		SlackConfigAttachmentTypeLogs:           webhooks.ATTACHMENTTYPE_LOGS,
	}
)

// PagerDuty configuration.
type PagerDuty struct {
	// PagerDuty service key.
	ServiceKey string `json:"serviceKey"`
}

func (in *PagerDuty) extractPagerDutyConfig() *webhooks.PagerDutyConfig {
	return &webhooks.PagerDutyConfig{
		ServiceKey: webhooks.PtrString(in.ServiceKey),
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

func (in *SendLog) extractSendLogConfig() *webhooks.SendLogConfig {
	return &webhooks.SendLogConfig{
		Payload: webhooks.PtrString(in.Payload),
		Uuid:    webhooks.PtrString(gouuid.NewString()),
	}
}

// EMail notification configuration
type EmailGroup struct {
	// Recipients
	EmailAddresses []string `json:"emailAddresses"`
}

func (in *EmailGroup) extractEmailGroupConfig() *webhooks.EmailGroupConfig {
	return &webhooks.EmailGroupConfig{
		EmailAddresses: in.EmailAddresses,
	}
}

// Microsoft Teams configuration.
type MicrosoftTeams struct {
	// Teams URL
	Url string `json:"url"`
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

func (in *Jira) extractJiraConfig() *webhooks.JiraConfig {
	return &webhooks.JiraConfig{
		ApiToken:   webhooks.PtrString(in.ApiToken),
		Email:      webhooks.PtrString(in.Email),
		ProjectKey: webhooks.PtrString(in.ProjectKey),
	}
}

type Opsgenie struct {
	Url string `json:"url"`
}

type Demisto struct {
	Uuid    string `json:"uuid"`
	Payload string `json:"payload"`
	Url     string `json:"url"`
}

func (in *Demisto) extractDemistoConfig() *webhooks.DemistoConfig {
	return &webhooks.DemistoConfig{
		Uuid:    webhooks.PtrString(in.Uuid),
		Payload: webhooks.PtrString(in.Payload),
	}
}

type AwsEventBridge struct {
	EventBusArn string `json:"eventBusArn"`
	Detail      string `json:"detail"`
	DetailType  string `json:"detailType"`
	Source      string `json:"source"`
	RoleName    string `json:"roleName"`
}

func (in *AwsEventBridge) extractAwsEventBridgeConfig() *webhooks.AwsEventBridgeConfig {
	return &webhooks.AwsEventBridgeConfig{
		EventBusArn: webhooks.PtrString(in.EventBusArn),
		Detail:      webhooks.PtrString(in.Detail),
		DetailType:  webhooks.PtrString(in.DetailType),
		Source:      webhooks.PtrString(in.Source),
		RoleName:    webhooks.PtrString(in.RoleName),
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

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (in *OutboundWebhook) GetConditions() []metav1.Condition {
	return in.Status.Conditions
}

func (in *OutboundWebhook) SetConditions(conditions []metav1.Condition) {
	in.Status.Conditions = conditions
}

func (in *OutboundWebhook) GetPrintableStatus() string {
	return in.Status.PrintableStatus
}

func (in *OutboundWebhook) SetPrintableStatus(printableStatus string) {
	in.Status.PrintableStatus = printableStatus
}

func (in *OutboundWebhook) HasIDInStatus() bool {
	return in.Status.ID != nil && *in.Status.ID != ""
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// OutboundWebhook is the Schema for the API
// See also https://coralogix.com/docs/user-guides/alerting/outbound-webhooks/aws-eventbridge-outbound-webhook/
//
// **Added in v0.4.0**
type OutboundWebhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OutboundWebhookSpec   `json:"spec,omitempty"`
	Status OutboundWebhookStatus `json:"status,omitempty"`
}

func (in *OutboundWebhook) ExtractCreateOutboundWebhookRequest() (*webhooks.CreateOutgoingWebhookRequest, error) {
	webhookData, err := in.Spec.ExtractOutgoingWebhookInputData()
	if err != nil {
		return nil, err
	}

	return &webhooks.CreateOutgoingWebhookRequest{
		Data: webhookData,
	}, nil
}

func (in *OutboundWebhook) ExtractUpdateOutboundWebhookRequest() (*webhooks.UpdateOutgoingWebhookRequest, error) {
	webhookData, err := in.Spec.ExtractOutgoingWebhookInputData()
	if err != nil {
		return nil, err
	}

	if in.Status.ID == nil {
		return nil, fmt.Errorf("outbound-webhook id is not set")
	}

	return &webhooks.UpdateOutgoingWebhookRequest{
		Id:   in.Status.ID,
		Data: webhookData,
	}, nil
}

func (in *OutboundWebhookSpec) ExtractOutgoingWebhookInputData() (*webhooks.OutgoingWebhookInputData, error) {
	if genericWebhook := in.OutboundWebhookType.GenericWebhook; genericWebhook != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataGenericWebhook: &webhooks.OutgoingWebhookInputDataGenericWebhook{
				Name:           webhooks.PtrString(in.Name),
				Type:           webhooks.WEBHOOKTYPE_GENERIC.Ptr(),
				Url:            webhooks.PtrString(genericWebhook.Url),
				GenericWebhook: genericWebhook.extractGenericWebhookConfig(),
			},
		}, nil
	} else if slack := in.OutboundWebhookType.Slack; slack != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataSlack: &webhooks.OutgoingWebhookInputDataSlack{
				Name:  webhooks.PtrString(in.Name),
				Type:  webhooks.WEBHOOKTYPE_SLACK.Ptr(),
				Url:   webhooks.PtrString(slack.Url),
				Slack: slack.extractSlackConfig(),
			},
		}, nil
	} else if pagerDuty := in.OutboundWebhookType.PagerDuty; pagerDuty != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataPagerDuty: &webhooks.OutgoingWebhookInputDataPagerDuty{
				Name:      webhooks.PtrString(in.Name),
				Type:      webhooks.WEBHOOKTYPE_PAGERDUTY.Ptr(),
				PagerDuty: pagerDuty.extractPagerDutyConfig(),
			},
		}, nil
	} else if sendLog := in.OutboundWebhookType.SendLog; sendLog != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataSendLog: &webhooks.OutgoingWebhookInputDataSendLog{
				Name:    webhooks.PtrString(in.Name),
				Type:    webhooks.WEBHOOKTYPE_SEND_LOG.Ptr(),
				Url:     webhooks.PtrString(sendLog.Url),
				SendLog: sendLog.extractSendLogConfig(),
			},
		}, nil
	} else if emailGroup := in.OutboundWebhookType.EmailGroup; emailGroup != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataEmailGroup: &webhooks.OutgoingWebhookInputDataEmailGroup{
				Name:       webhooks.PtrString(in.Name),
				Type:       webhooks.WEBHOOKTYPE_EMAIL_GROUP.Ptr(),
				EmailGroup: emailGroup.extractEmailGroupConfig(),
			},
		}, nil
	} else if microsoftTeams := in.OutboundWebhookType.MicrosoftTeams; microsoftTeams != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataMicrosoftTeams: &webhooks.OutgoingWebhookInputDataMicrosoftTeams{
				Name:           webhooks.PtrString(in.Name),
				Type:           webhooks.WEBHOOKTYPE_MICROSOFT_TEAMS.Ptr(),
				Url:            webhooks.PtrString(microsoftTeams.Url),
				MicrosoftTeams: map[string]interface{}{},
			},
		}, nil
	} else if jira := in.OutboundWebhookType.Jira; jira != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataJira: &webhooks.OutgoingWebhookInputDataJira{
				Name: webhooks.PtrString(in.Name),
				Type: webhooks.WEBHOOKTYPE_JIRA.Ptr(),
				Url:  webhooks.PtrString(jira.Url),
				Jira: jira.extractJiraConfig(),
			},
		}, nil
	} else if opsgenie := in.OutboundWebhookType.Opsgenie; opsgenie != nil {
		//data.Config = opsgenie.extractOpsgenieConfig()
		//data.Type = cxsdk.WebhookTypeOpsgenie
		//data.Url = opsgenie.Url)
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataOpsgenie: &webhooks.OutgoingWebhookInputDataOpsgenie{
				Name:     webhooks.PtrString(in.Name),
				Type:     webhooks.WEBHOOKTYPE_OPSGENIE.Ptr(),
				Url:      webhooks.PtrString(opsgenie.Url),
				Opsgenie: map[string]interface{}{},
			},
		}, nil
	} else if demisto := in.OutboundWebhookType.Demisto; demisto != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataDemisto: &webhooks.OutgoingWebhookInputDataDemisto{
				Name:    webhooks.PtrString(in.Name),
				Type:    webhooks.WEBHOOKTYPE_DEMISTO.Ptr(),
				Url:     webhooks.PtrString(demisto.Url),
				Demisto: demisto.extractDemistoConfig(),
			},
		}, nil
	} else if in.OutboundWebhookType.AwsEventBridge != nil {
		return &webhooks.OutgoingWebhookInputData{
			OutgoingWebhookInputDataAwsEventBridge: &webhooks.OutgoingWebhookInputDataAwsEventBridge{
				Name:           webhooks.PtrString(in.Name),
				Type:           webhooks.WEBHOOKTYPE_AWS_EVENT_BRIDGE.Ptr(),
				AwsEventBridge: in.OutboundWebhookType.AwsEventBridge.extractAwsEventBridgeConfig(),
			},
		}, nil
	}

	return nil, fmt.Errorf("unsupported outbound-webhook type")
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

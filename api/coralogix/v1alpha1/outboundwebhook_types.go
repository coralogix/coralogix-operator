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
type OutboundWebhookSpec struct {
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	OutboundWebhookType OutboundWebhookType `json:"outboundWebhookType"`
}

type OutboundWebhookType struct {
	// +optional
	GenericWebhook *GenericWebhook `json:"genericWebhook,omitempty"`

	// +optional
	Slack *Slack `json:"slack,omitempty"`

	// +optional
	PagerDuty *PagerDuty `json:"pagerDuty,omitempty"`

	// +optional
	SendLog *SendLog `json:"sendLog,omitempty"`

	// +optional
	EmailGroup *EmailGroup `json:"emailGroup,omitempty"`

	// +optional
	MicrosoftTeams *MicrosoftTeams `json:"microsoftTeams,omitempty"`

	// +optional
	Jira *Jira `json:"jira,omitempty"`

	// +optional
	Opsgenie *Opsgenie `json:"opsgenie,omitempty"`

	// +optional
	Demisto *Demisto `json:"demisto,omitempty"`

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

type GenericWebhook struct {
	Url string `json:"url"`

	Method GenericWebhookMethodType `json:"method"`

	// +optional
	Headers map[string]string `json:"headers"`

	// +optional
	Payload *string `json:"payload"`
}

type GenericWebhookStatus struct {
	Uuid string `json:"uuid"`

	Url string `json:"url"`

	Method GenericWebhookMethodType `json:"method"`

	// +optional
	Headers map[string]string `json:"headers"`

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
	// +optional
	Digests []SlackConfigDigest `json:"digests"`
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

type SlackConfigDigestType string

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

type SlackConfigDigest struct {
	Type     SlackConfigDigestType `json:"type"`
	IsActive bool                  `json:"isActive"`
}

type SlackConfigAttachment struct {
	Type     SlackConfigAttachmentType `json:"type"`
	IsActive bool                      `json:"isActive"`
}

type SlackConfigAttachmentType string

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

type PagerDuty struct {
	ServiceKey string `json:"serviceKey"`
}

func (in *PagerDuty) extractPagerDutyConfig() *cxsdk.PagerDutyWebhookInputData {
	return &cxsdk.PagerDutyWebhookInputData{
		PagerDuty: &cxsdk.PagerDutyConfig{
			ServiceKey: wrapperspb.String(in.ServiceKey),
		},
	}
}

type SendLog struct {
	Payload string `json:"payload"`
	Url     string `json:"url"`
}

type SendLogStatus struct {
	Payload string `json:"payload"`
	Url     string `json:"url"`
	Uuid    string `json:"uuid"`
}

func (in *SendLog) extractSendLogConfig() *cxsdk.SendLogWebhookInputData {
	return &cxsdk.SendLogWebhookInputData{
		SendLog: &cxsdk.SendLogConfig{
			Payload: wrapperspb.String(in.Payload),
			Uuid:    wrapperspb.String(gouuid.NewString()),
		},
	}
}

type EmailGroup struct {
	EmailAddresses []string `json:"emailAddresses"`
}

func (in *EmailGroup) extractEmailGroupConfig() *cxsdk.EmailGroupWebhookInputData {
	return &cxsdk.EmailGroupWebhookInputData{
		EmailGroup: &cxsdk.EmailGroupConfig{
			EmailAddresses: utils.StringSliceToWrappedStringSlice(in.EmailAddresses),
		},
	}
}

type MicrosoftTeams struct {
	Url string `json:"url"`
}

func (in *MicrosoftTeams) extractMicrosoftTeamsConfig() *cxsdk.MicrosoftTeamsWebhookInputData {
	return &cxsdk.MicrosoftTeamsWebhookInputData{
		MicrosoftTeams: &cxsdk.MicrosoftTeamsConfig{},
	}
}

type Jira struct {
	ApiToken   string `json:"apiToken"`
	Email      string `json:"email"`
	ProjectKey string `json:"projectKey"`
	Url        string `json:"url"`
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

func (in *OutboundWebhookStatus) GetConditions() []metav1.Condition {
	return in.Conditions
}

func (in *OutboundWebhookStatus) SetConditions(conditions []metav1.Condition) {
	in.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OutboundWebhook is the Schema for the outboundwebhooks API
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

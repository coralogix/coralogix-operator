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

package v1alpha1

import (
	"fmt"

	gouuid "github.com/google/uuid"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/apis"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

type OutboundWebhookTypeStatus struct {
	GenericWebhook *GenericWebhookStatus `json:"genericWebhook,omitempty"`

	Slack *Slack `json:"slack,omitempty"`

	PagerDuty *PagerDuty `json:"pagerDuty,omitempty"`

	SendLog *SendLogStatus `json:"sendLog,omitempty"`

	EmailGroup *EmailGroup `json:"emailGroup,omitempty"`

	MicrosoftTeams *MicrosoftTeams `json:"microsoftTeams,omitempty"`

	Jira *Jira `json:"jira,omitempty"`

	Opsgenie *Opsgenie `json:"opsgenie,omitempty"`

	Demisto *Demisto `json:"demisto,omitempty"`

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

func (in *OutboundWebhookType) DeepEqual(webhookType *OutboundWebhookTypeStatus) (bool, utils.Diff) {
	if webhookType == nil {
		return false, utils.Diff{
			Name:    "OutboundWebhookType",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	equal, diff := true, utils.Diff{}
	if desiredGenericWebhook, actualGenericWebhook := in.GenericWebhook, webhookType.GenericWebhook; desiredGenericWebhook != nil {
		equal, diff = desiredGenericWebhook.DeepEqual(actualGenericWebhook)
	} else if desiredSlack, actualSlack := in.Slack, webhookType.Slack; desiredSlack != nil {
		equal, diff = desiredSlack.DeepEqual(actualSlack)
	} else if desiredPagerDuty, actualPagerDuty := in.PagerDuty, webhookType.PagerDuty; desiredPagerDuty != nil {
		equal, diff = desiredPagerDuty.DeepEqual(actualPagerDuty)
	} else if desiredSendLog, actualSendLog := in.SendLog, webhookType.SendLog; desiredSendLog != nil {
		equal, diff = desiredSendLog.DeepEqual(actualSendLog)
	} else if desiredEmailGroup, actualEmailGroup := in.EmailGroup, webhookType.EmailGroup; desiredEmailGroup != nil {
		equal, diff = desiredEmailGroup.DeepEqual(actualEmailGroup)
	} else if desiredMicrosoftTeams, actualMicrosoftTeams := in.MicrosoftTeams, webhookType.MicrosoftTeams; desiredMicrosoftTeams != nil {
		equal, diff = desiredMicrosoftTeams.DeepEqual(actualMicrosoftTeams)
	} else if desiredJira, actualJira := in.Jira, webhookType.Jira; desiredJira != nil {
		equal, diff = desiredJira.DeepEqual(actualJira)
	} else if desiredOpsgenie, actualOpsgenie := in.Opsgenie, webhookType.Opsgenie; desiredOpsgenie != nil {
		equal, diff = desiredOpsgenie.DeepEqual(actualOpsgenie)
	} else if desiredDemisto, actualDemisto := in.Demisto, webhookType.Demisto; desiredDemisto != nil {
		equal, diff = desiredDemisto.DeepEqual(actualDemisto)
	} else if desiredAwsEventBridge, actualAwsEventBridge := in.AwsEventBridge, webhookType.AwsEventBridge; desiredAwsEventBridge != nil {
		equal, diff = desiredAwsEventBridge.DeepEqual(actualAwsEventBridge)
	} else {
		return false, utils.Diff{
			Name:    "OutboundWebhookType",
			Desired: utils.PointerToString(in),
			Actual:  utils.PointerToString(webhookType),
		}
	}

	if !equal {
		return false, utils.Diff{
			Name:    fmt.Sprintf("OutboundWebhookType.%s", diff.Name),
			Desired: diff.Desired,
			Actual:  diff.Actual,
		}
	}

	return true, utils.Diff{}
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

func (in *GenericWebhook) DeepEqual(webhook *GenericWebhookStatus) (bool, utils.Diff) {
	if webhook == nil {
		return false, utils.Diff{
			Name:    "GenericWebhook",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.Url != webhook.Url {
		return false, utils.Diff{
			Name:    "GenericWebhook.Url",
			Desired: in.Url,
			Actual:  webhook.Url,
		}
	}

	if in.Method != webhook.Method {
		return false, utils.Diff{
			Name:    "GenericWebhook.Method",
			Desired: in.Method,
			Actual:  webhook.Method,
		}
	}

	if !utils.StringMapEqual(in.Headers, webhook.Headers) {
		return false, utils.Diff{
			Name:    "GenericWebhook.Headers",
			Desired: in.Headers,
			Actual:  webhook.Headers,
		}
	}

	if utils.PointerToString(in.Payload) != utils.PointerToString(webhook.Payload) {
		return false, utils.Diff{
			Name:    "GenericWebhook.Payload",
			Desired: in.Payload,
			Actual:  webhook.Payload,
		}
	}

	return true, utils.Diff{}
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

func (in *Slack) DeepEqual(slack *Slack) (bool, utils.Diff) {
	if slack == nil {
		return false, utils.Diff{
			Name:    "Slack",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if !utils.SlicesWithUniqueValuesEqual(in.Digests, slack.Digests) {
		return false, utils.Diff{
			Name:    "Slack.Digests",
			Desired: in.Digests,
			Actual:  slack.Digests,
		}
	}

	if !utils.SlicesWithUniqueValuesEqual(in.Attachments, slack.Attachments) {
		return false, utils.Diff{
			Name:    "Slack.Attachments",
			Desired: in.Attachments,
			Actual:  slack.Attachments,
		}
	}

	if in.Url != slack.Url {
		return false, utils.Diff{
			Name:    "Slack.Url",
			Desired: in.Url,
			Actual:  slack.Url,
		}
	}

	return true, utils.Diff{}
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

func (in *PagerDuty) DeepEqual(pagerDuty *PagerDuty) (bool, utils.Diff) {
	if pagerDuty == nil {
		return false, utils.Diff{
			Name:    "PagerDuty",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.ServiceKey != pagerDuty.ServiceKey {
		return false, utils.Diff{
			Name:    "PagerDuty.ServiceKey",
			Desired: in.ServiceKey,
			Actual:  pagerDuty.ServiceKey,
		}
	}

	return true, utils.Diff{}
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

func (in *SendLog) DeepEqual(sendLog *SendLogStatus) (bool, utils.Diff) {
	if sendLog == nil {
		return false, utils.Diff{
			Name:    "SendLog",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.Payload != sendLog.Payload {
		return false, utils.Diff{
			Name:    "SendLog.Payload",
			Desired: in.Payload,
			Actual:  sendLog.Payload,
		}
	}

	if in.Url != sendLog.Url {
		return false, utils.Diff{
			Name:    "SendLog.Url",
			Desired: in.Url,
			Actual:  sendLog.Url,
		}
	}

	return true, utils.Diff{}
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

func (in *EmailGroup) DeepEqual(emailGroup *EmailGroup) (bool, utils.Diff) {
	if emailGroup == nil {
		return false, utils.Diff{
			Name:    "EmailGroup",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if !utils.SlicesWithUniqueValuesEqual(in.EmailAddresses, emailGroup.EmailAddresses) {
		return false, utils.Diff{
			Name:    "EmailGroup.EmailAddresses",
			Desired: in.EmailAddresses,
			Actual:  emailGroup.EmailAddresses,
		}
	}

	return true, utils.Diff{}
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

func (in *MicrosoftTeams) DeepEqual(microsoftTeams *MicrosoftTeams) (bool, utils.Diff) {
	if microsoftTeams == nil {
		return false, utils.Diff{
			Name:    "MicrosoftTeams",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.Url != microsoftTeams.Url {
		return false, utils.Diff{
			Name:    "MicrosoftTeams.Url",
			Desired: in.Url,
			Actual:  microsoftTeams.Url,
		}
	}

	return true, utils.Diff{}
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

func (in *Jira) DeepEqual(jira *Jira) (bool, utils.Diff) {
	if jira == nil {
		return false, utils.Diff{
			Name:    "Jira",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.ApiToken != jira.ApiToken {
		return false, utils.Diff{
			Name:    "Jira.ApiToken",
			Desired: in.ApiToken,
			Actual:  jira.ApiToken,
		}
	}

	if in.Email != jira.Email {
		return false, utils.Diff{
			Name:    "Jira.Email",
			Desired: in.Email,
			Actual:  jira.Email,
		}
	}

	if in.ProjectKey != jira.ProjectKey {
		return false, utils.Diff{
			Name:    "Jira.ProjectKey",
			Desired: in.ProjectKey,
			Actual:  jira.ProjectKey,
		}
	}

	if in.Url != jira.Url {
		return false, utils.Diff{
			Name:    "Jira.Url",
			Desired: in.Url,
			Actual:  jira.Url,
		}
	}

	return true, utils.Diff{}
}

type Opsgenie struct {
	Url string `json:"url"`
}

func (in *Opsgenie) extractOpsgenieConfig() *cxsdk.OpsgenieWebhookInputData {
	return &cxsdk.OpsgenieWebhookInputData{
		Opsgenie: &cxsdk.OpsgenieConfig{},
	}
}

func (in *Opsgenie) DeepEqual(opsgenie *Opsgenie) (bool, utils.Diff) {
	if opsgenie == nil {
		return false, utils.Diff{
			Name:    "Opsgenie",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.Url != opsgenie.Url {
		return false, utils.Diff{
			Name:    "Opsgenie.Url",
			Desired: in.Url,
			Actual:  opsgenie.Url,
		}
	}

	return true, utils.Diff{}
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

func (in *Demisto) DeepEqual(demisto *Demisto) (bool, utils.Diff) {
	if demisto == nil {
		return false, utils.Diff{
			Name:    "Demisto",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.Uuid != demisto.Uuid {
		return false, utils.Diff{
			Name:    "Demisto.Uuid",
			Desired: in.Uuid,
			Actual:  demisto.Uuid,
		}
	}

	if in.Payload != demisto.Payload {
		return false, utils.Diff{
			Name:    "Demisto.Payload",
			Desired: in.Payload,
			Actual:  demisto.Payload,
		}
	}

	if in.Url != demisto.Url {
		return false, utils.Diff{
			Name:    "Demisto.Url",
			Desired: in.Url,
			Actual:  demisto.Url,
		}
	}

	return true, utils.Diff{}
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

func (in *AwsEventBridge) DeepEqual(awsEventBridge *AwsEventBridge) (bool, utils.Diff) {
	if awsEventBridge == nil {
		return false, utils.Diff{
			Name:    "AwsEventBridge",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.EventBusArn != awsEventBridge.EventBusArn {
		return false, utils.Diff{
			Name:    "AwsEventBridge.EventBusArn",
			Desired: in.EventBusArn,
			Actual:  awsEventBridge.EventBusArn,
		}
	}

	if in.Detail != awsEventBridge.Detail {
		return false, utils.Diff{
			Name:    "AwsEventBridge.Detail",
			Desired: in.Detail,
			Actual:  awsEventBridge.Detail,
		}
	}

	if in.DetailType != awsEventBridge.DetailType {
		return false, utils.Diff{
			Name:    "AwsEventBridge.DetailType",
			Desired: in.DetailType,
			Actual:  awsEventBridge.DetailType,
		}
	}

	if in.Source != awsEventBridge.Source {
		return false, utils.Diff{
			Name:    "AwsEventBridge.Source",
			Desired: in.Source,
			Actual:  awsEventBridge.Source,
		}
	}

	if in.RoleName != awsEventBridge.RoleName {
		return false, utils.Diff{
			Name:    "AwsEventBridge.RoleName",
			Desired: in.RoleName,
			Actual:  awsEventBridge.RoleName,
		}
	}

	return true, utils.Diff{}
}

// OutboundWebhookStatus defines the observed state of OutboundWebhook
type OutboundWebhookStatus struct {
	ID *string `json:"id"`

	// +optional
	ExternalID *string `json:"externalId"`

	Name string `json:"name"`

	OutboundWebhookType *OutboundWebhookTypeStatus `json:"outboundWebhookType"`
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

func (in *OutboundWebhookSpec) DeepEqual(status *OutboundWebhookStatus) (bool, utils.Diff) {
	if status == nil {
		return false, utils.Diff{
			Name:    "OutboundWebhookStatus",
			Desired: utils.PointerToString(in),
			Actual:  nil,
		}
	}

	if in.Name != status.Name {
		return false, utils.Diff{
			Name:    "OutboundWebhookStatus.Name",
			Desired: in.Name,
			Actual:  status.Name,
		}
	}

	equal, diff := in.OutboundWebhookType.DeepEqual(status.OutboundWebhookType)
	if !equal {
		return false, utils.Diff{
			Name:    fmt.Sprintf("OutboundWebhookStatus.OutboundWebhookType.%s", diff.Name),
			Desired: diff.Desired,
			Actual:  diff.Actual,
		}
	}

	return true, utils.Diff{}
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

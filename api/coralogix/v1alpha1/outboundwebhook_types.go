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

	gouuid "github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
// +kubebuilder:validation:XValidation:rule="has(self.url) != has(self.urlSecretKeyRef)",message="Exactly one of url or urlSecretKeyRef must be set"
// +kubebuilder:validation:XValidation:rule="!has(self.headers) || !has(self.headersFromSecret) || self.headersFromSecret.all(k, !(k in self.headers))",message="A header can have only one source"
type GenericWebhook struct {

	// URL to call. Use UrlSecretKeyRef when the URL contains a credential.
	// +optional
	Url string `json:"url,omitempty"`

	// UrlSecretKeyRef reads the URL from a Secret in the webhook's namespace.
	// +optional
	UrlSecretKeyRef *corev1.SecretKeySelector `json:"urlSecretKeyRef,omitempty"`

	// HTTP Method to use.
	Method GenericWebhookMethodType `json:"method"`

	// Attached HTTP headers. Use HeadersFromSecret for credential values.
	// +optional
	Headers map[string]string `json:"headers"`

	// HeadersFromSecret maps header names to Secret keys in the webhook's namespace.
	// +optional
	HeadersFromSecret map[string]corev1.SecretKeySelector `json:"headersFromSecret,omitempty"`

	// Payload of the webhook call.
	// +optional
	Payload *string `json:"payload"`
}

func (in *GenericWebhook) extractGenericWebhookConfig() *webhooks.GenericWebhookConfig {
	return &webhooks.GenericWebhookConfig{
		Uuid:    webhooks.PtrString(gouuid.NewString()),
		Method:  GenericWebhookMethodTypeToOpenAPI[in.Method],
		Headers: in.Headers,
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

// +kubebuilder:validation:XValidation:rule="has(self.url) != has(self.urlSecretKeyRef)",message="Exactly one of url or urlSecretKeyRef must be set"
type Slack struct {

	// Digest configuration.
	// +optional
	Digests []SlackConfigDigest `json:"digests"`

	// Attachments of the message.
	// +optional
	Attachments []SlackConfigAttachment `json:"attachments"`
	// Use UrlSecretKeyRef when the URL contains a credential.
	// +optional
	Url string `json:"url,omitempty"`

	// UrlSecretKeyRef reads the URL from a Secret in the webhook's namespace.
	// +optional
	UrlSecretKeyRef *corev1.SecretKeySelector `json:"urlSecretKeyRef,omitempty"`
}

func (in *Slack) extractSlackConfig() *webhooks.SlackConfig {
	digests := make([]webhooks.Digest, 0)
	for _, digest := range in.Digests {
		digests = append(digests, webhooks.Digest{
			Type:     SlackConfigDigestTypeToOpenAPI[digest.Type],
			IsActive: webhooks.PtrBool(digest.IsActive),
		})
	}

	attachments := make([]webhooks.Attachment, 0)
	for _, attachment := range in.Attachments {
		attachments = append(attachments, webhooks.Attachment{
			Type:     SlackConfigAttachmentTypeToOpenAPI[attachment.Type],
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
// +kubebuilder:validation:XValidation:rule="has(self.serviceKey) != has(self.serviceKeySecretKeyRef)",message="Exactly one of serviceKey or serviceKeySecretKeyRef must be set"
type PagerDuty struct {
	// Deprecated: use ServiceKeySecretKeyRef for the PagerDuty service key.
	// +optional
	ServiceKey string `json:"serviceKey,omitempty"`

	// ServiceKeySecretKeyRef reads the service key from a Secret in the webhook's namespace.
	// +optional
	ServiceKeySecretKeyRef *corev1.SecretKeySelector `json:"serviceKeySecretKeyRef,omitempty"`
}

func (in *PagerDuty) extractPagerDutyConfig() *webhooks.PagerDutyConfig {
	return &webhooks.PagerDutyConfig{
		ServiceKey: in.ServiceKey,
	}
}

// SendLog configuration.
// +kubebuilder:validation:XValidation:rule="has(self.url) != has(self.urlSecretKeyRef)",message="Exactly one of url or urlSecretKeyRef must be set"
type SendLog struct {
	// Payload of the notification
	Payload string `json:"payload"`

	// Sendlog URL. Use UrlSecretKeyRef when the URL contains a credential.
	// +optional
	Url string `json:"url,omitempty"`

	// UrlSecretKeyRef reads the URL from a Secret in the webhook's namespace.
	// +optional
	UrlSecretKeyRef *corev1.SecretKeySelector `json:"urlSecretKeyRef,omitempty"`
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
		Payload: in.Payload,
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
// +kubebuilder:validation:XValidation:rule="has(self.url) != has(self.urlSecretKeyRef)",message="Exactly one of url or urlSecretKeyRef must be set"
type MicrosoftTeams struct {
	// Teams URL. Use UrlSecretKeyRef when the URL contains a credential.
	// +optional
	Url string `json:"url,omitempty"`

	// UrlSecretKeyRef reads the URL from a Secret in the webhook's namespace.
	// +optional
	UrlSecretKeyRef *corev1.SecretKeySelector `json:"urlSecretKeyRef,omitempty"`
}

// Jira configuration
// +kubebuilder:validation:XValidation:rule="has(self.apiToken) != has(self.apiTokenSecretKeyRef)",message="Exactly one of apiToken or apiTokenSecretKeyRef must be set"
// +kubebuilder:validation:XValidation:rule="has(self.url) != has(self.urlSecretKeyRef)",message="Exactly one of url or urlSecretKeyRef must be set"
type Jira struct {
	// Deprecated: use ApiTokenSecretKeyRef for the Jira API token.
	// +optional
	ApiToken string `json:"apiToken,omitempty"`

	// ApiTokenSecretKeyRef reads the API token from a Secret in the webhook's namespace.
	// +optional
	ApiTokenSecretKeyRef *corev1.SecretKeySelector `json:"apiTokenSecretKeyRef,omitempty"`

	// Email address associated with the token
	Email string `json:"email"`

	// Project to add it to.
	ProjectKey string `json:"projectKey"`

	// Jira URL. Use UrlSecretKeyRef when the URL contains a credential.
	// +optional
	Url string `json:"url,omitempty"`

	// UrlSecretKeyRef reads the URL from a Secret in the webhook's namespace.
	// +optional
	UrlSecretKeyRef *corev1.SecretKeySelector `json:"urlSecretKeyRef,omitempty"`
}

func (in *Jira) extractJiraConfig() *webhooks.JiraConfig {
	return &webhooks.JiraConfig{
		ApiToken:   in.ApiToken,
		Email:      in.Email,
		ProjectKey: in.ProjectKey,
	}
}

// +kubebuilder:validation:XValidation:rule="has(self.url) != has(self.urlSecretKeyRef)",message="Exactly one of url or urlSecretKeyRef must be set"
type Opsgenie struct {
	// Use UrlSecretKeyRef when the URL contains a credential.
	// +optional
	Url string `json:"url,omitempty"`

	// UrlSecretKeyRef reads the URL from a Secret in the webhook's namespace.
	// +optional
	UrlSecretKeyRef *corev1.SecretKeySelector `json:"urlSecretKeyRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.url) != has(self.urlSecretKeyRef)",message="Exactly one of url or urlSecretKeyRef must be set"
type Demisto struct {
	Uuid    string `json:"uuid"`
	Payload string `json:"payload"`
	// Use UrlSecretKeyRef when the URL contains a credential.
	// +optional
	Url string `json:"url,omitempty"`

	// UrlSecretKeyRef reads the URL from a Secret in the webhook's namespace.
	// +optional
	UrlSecretKeyRef *corev1.SecretKeySelector `json:"urlSecretKeyRef,omitempty"`
}

func (in *Demisto) extractDemistoConfig() *webhooks.DemistoConfig {
	return &webhooks.DemistoConfig{
		Uuid:    webhooks.PtrString(in.Uuid),
		Payload: in.Payload,
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
		EventBusArn: in.EventBusArn,
		Detail:      in.Detail,
		DetailType:  in.DetailType,
		Source:      in.Source,
		RoleName:    in.RoleName,
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
	return in.ExtractCreateOutboundWebhookRequestWithContext(context.Background())
}

func (in *OutboundWebhook) ExtractCreateOutboundWebhookRequestWithContext(ctx context.Context) (*webhooks.CreateOutgoingWebhookRequest, error) {
	webhookData, err := in.Spec.ExtractOutgoingWebhookInputDataWithContext(ctx, in.Namespace)
	if err != nil {
		return nil, err
	}

	return &webhooks.CreateOutgoingWebhookRequest{
		Data: *webhookData,
	}, nil
}

func (in *OutboundWebhook) ExtractUpdateOutboundWebhookRequest() (*webhooks.UpdateOutgoingWebhookRequest, error) {
	return in.ExtractUpdateOutboundWebhookRequestWithContext(context.Background())
}

func (in *OutboundWebhook) ExtractUpdateOutboundWebhookRequestWithContext(ctx context.Context) (*webhooks.UpdateOutgoingWebhookRequest, error) {
	webhookData, err := in.Spec.ExtractOutgoingWebhookInputDataWithContext(ctx, in.Namespace)
	if err != nil {
		return nil, err
	}

	if in.Status.ID == nil {
		return nil, fmt.Errorf("outbound-webhook id is not set")
	}

	return &webhooks.UpdateOutgoingWebhookRequest{
		Id:   *in.Status.ID,
		Data: *webhookData,
	}, nil
}

// ExtractOutgoingWebhookInputData keeps the literal-only API. Use
// ExtractOutgoingWebhookInputDataWithContext to resolve Secret references.
func (in *OutboundWebhookSpec) ExtractOutgoingWebhookInputData() (*webhooks.OutgoingWebhookInputData, error) {
	return in.ExtractOutgoingWebhookInputDataWithContext(context.Background(), "")
}

func (in *OutboundWebhookSpec) ExtractOutgoingWebhookInputDataWithContext(ctx context.Context, namespace string) (*webhooks.OutgoingWebhookInputData, error) {
	if genericWebhook := in.OutboundWebhookType.GenericWebhook; genericWebhook != nil {
		url, err := resolveWebhookValue(ctx, namespace, "genericWebhook.url", genericWebhook.Url, genericWebhook.UrlSecretKeyRef)
		if err != nil {
			return nil, err
		}
		headers, err := resolveWebhookHeaders(ctx, namespace, genericWebhook.Headers, genericWebhook.HeadersFromSecret)
		if err != nil {
			return nil, err
		}
		config := genericWebhook.extractGenericWebhookConfig()
		config.Headers = headers
		return &webhooks.OutgoingWebhookInputData{
			Name:           in.Name,
			Type:           webhooks.WEBHOOKTYPE_GENERIC.Ptr(),
			Url:            webhooks.PtrString(url),
			GenericWebhook: config,
		}, nil
	} else if slack := in.OutboundWebhookType.Slack; slack != nil {
		url, err := resolveWebhookValue(ctx, namespace, "slack.url", slack.Url, slack.UrlSecretKeyRef)
		if err != nil {
			return nil, err
		}
		return &webhooks.OutgoingWebhookInputData{
			Name:  in.Name,
			Type:  webhooks.WEBHOOKTYPE_SLACK.Ptr(),
			Url:   webhooks.PtrString(url),
			Slack: slack.extractSlackConfig(),
		}, nil
	} else if pagerDuty := in.OutboundWebhookType.PagerDuty; pagerDuty != nil {
		serviceKey, err := resolveWebhookValue(ctx, namespace, "pagerDuty.serviceKey", pagerDuty.ServiceKey, pagerDuty.ServiceKeySecretKeyRef)
		if err != nil {
			return nil, err
		}
		config := pagerDuty.extractPagerDutyConfig()
		config.ServiceKey = serviceKey
		return &webhooks.OutgoingWebhookInputData{
			Name:      in.Name,
			Type:      webhooks.WEBHOOKTYPE_PAGERDUTY.Ptr(),
			PagerDuty: config,
		}, nil
	} else if sendLog := in.OutboundWebhookType.SendLog; sendLog != nil {
		url, err := resolveWebhookValue(ctx, namespace, "sendLog.url", sendLog.Url, sendLog.UrlSecretKeyRef)
		if err != nil {
			return nil, err
		}
		return &webhooks.OutgoingWebhookInputData{
			Name:    in.Name,
			Type:    webhooks.WEBHOOKTYPE_SEND_LOG.Ptr(),
			Url:     webhooks.PtrString(url),
			SendLog: sendLog.extractSendLogConfig(),
		}, nil
	} else if emailGroup := in.OutboundWebhookType.EmailGroup; emailGroup != nil {
		return &webhooks.OutgoingWebhookInputData{
			Name:       in.Name,
			Type:       webhooks.WEBHOOKTYPE_EMAIL_GROUP.Ptr(),
			EmailGroup: emailGroup.extractEmailGroupConfig(),
		}, nil
	} else if microsoftTeams := in.OutboundWebhookType.MicrosoftTeams; microsoftTeams != nil {
		url, err := resolveWebhookValue(ctx, namespace, "microsoftTeams.url", microsoftTeams.Url, microsoftTeams.UrlSecretKeyRef)
		if err != nil {
			return nil, err
		}
		return &webhooks.OutgoingWebhookInputData{
			Name:           in.Name,
			Type:           webhooks.WEBHOOKTYPE_MICROSOFT_TEAMS.Ptr(),
			Url:            webhooks.PtrString(url),
			MicrosoftTeams: map[string]interface{}{},
		}, nil
	} else if jira := in.OutboundWebhookType.Jira; jira != nil {
		url, err := resolveWebhookValue(ctx, namespace, "jira.url", jira.Url, jira.UrlSecretKeyRef)
		if err != nil {
			return nil, err
		}
		apiToken, err := resolveWebhookValue(ctx, namespace, "jira.apiToken", jira.ApiToken, jira.ApiTokenSecretKeyRef)
		if err != nil {
			return nil, err
		}
		config := jira.extractJiraConfig()
		config.ApiToken = apiToken
		return &webhooks.OutgoingWebhookInputData{
			Name: in.Name,
			Type: webhooks.WEBHOOKTYPE_JIRA.Ptr(),
			Url:  webhooks.PtrString(url),
			Jira: config,
		}, nil
	} else if opsgenie := in.OutboundWebhookType.Opsgenie; opsgenie != nil {
		url, err := resolveWebhookValue(ctx, namespace, "opsgenie.url", opsgenie.Url, opsgenie.UrlSecretKeyRef)
		if err != nil {
			return nil, err
		}
		return &webhooks.OutgoingWebhookInputData{
			Name:     in.Name,
			Type:     webhooks.WEBHOOKTYPE_OPSGENIE.Ptr(),
			Url:      webhooks.PtrString(url),
			Opsgenie: map[string]interface{}{},
		}, nil
	} else if demisto := in.OutboundWebhookType.Demisto; demisto != nil {
		url, err := resolveWebhookValue(ctx, namespace, "demisto.url", demisto.Url, demisto.UrlSecretKeyRef)
		if err != nil {
			return nil, err
		}
		return &webhooks.OutgoingWebhookInputData{
			Name:    in.Name,
			Type:    webhooks.WEBHOOKTYPE_DEMISTO.Ptr(),
			Url:     webhooks.PtrString(url),
			Demisto: demisto.extractDemistoConfig(),
		}, nil
	} else if in.OutboundWebhookType.AwsEventBridge != nil {
		return &webhooks.OutgoingWebhookInputData{
			Name:           in.Name,
			Type:           webhooks.WEBHOOKTYPE_AWS_EVENT_BRIDGE.Ptr(),
			AwsEventBridge: in.OutboundWebhookType.AwsEventBridge.extractAwsEventBridgeConfig(),
		}, nil
	}

	return nil, fmt.Errorf("unsupported outbound-webhook type")
}

func resolveWebhookValue(ctx context.Context, namespace, field, literal string, ref *corev1.SecretKeySelector) (string, error) {
	if ref == nil {
		return literal, nil
	}
	if literal != "" {
		return "", fmt.Errorf("%s has both a literal value and a Secret reference", field)
	}
	value, err := readSecret(ctx, *ref, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to read Secret for %s: %w", field, err)
	}
	return value, nil
}

func resolveWebhookHeaders(ctx context.Context, namespace string, literals map[string]string, refs map[string]corev1.SecretKeySelector) (map[string]string, error) {
	if len(literals) == 0 && len(refs) == 0 {
		return literals, nil
	}

	headers := make(map[string]string, len(literals)+len(refs))
	for name, value := range literals {
		headers[name] = value
	}
	for name, ref := range refs {
		if _, exists := headers[name]; exists {
			return nil, fmt.Errorf("header %q has both a literal value and a Secret reference", name)
		}
		value, err := readSecret(ctx, ref, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read Secret for header %q: %w", name, err)
		}
		headers[name] = value
	}
	return headers, nil
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

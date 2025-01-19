package clientset

import cxsdk "github.com/coralogix/coralogix-management-sdk/go"

//go:generate mockgen -destination=../mock_clientset/mock_clientset.go -package=mock_clientset github.com/coralogix/coralogix-operator/controllers/clientset ClientSetInterface
type ClientSetInterface interface {
	RuleGroups() RuleGroupsClientInterface
	RecordingRuleGroups() RecordingRulesGroupsClientInterface
	OutboundWebhooks() OutboundWebhooksClientInterface
	Alerts() *cxsdk.AlertsClient
	ApiKeys() ApiKeysClientInterface
}

type ClientSet struct {
	ruleGroups          *cxsdk.RuleGroupsClient
	alerts              *AlertsClient
	recordingRuleGroups *cxsdk.RecordingRuleGroupSetsClient
	outboundWebhooks    *cxsdk.WebhooksClient
	alertsV3            *cxsdk.AlertsClient
	apiKeys             *cxsdk.ApikeysClient
}

func (c *ClientSet) RuleGroups() RuleGroupsClientInterface {
	return c.ruleGroups
}

func (c *ClientSet) RecordingRuleGroups() RecordingRulesGroupsClientInterface {
	return c.recordingRuleGroups
}

func (c *ClientSet) OutboundWebhooks() OutboundWebhooksClientInterface {
	return c.outboundWebhooks
}

func (c *ClientSet) Alerts() *cxsdk.AlertsClient {
	return c.alertsV3
}

func (c *ClientSet) ApiKeys() ApiKeysClientInterface {
	return c.apiKeys
}

func NewClientSet(targetUrl, apiKey string) ClientSetInterface {
	apikeyCPC := NewCallPropertiesCreator(targetUrl, apiKey)
	SDKAPIKeyCPC := cxsdk.NewCallPropertiesCreatorOperator(targetUrl, cxsdk.NewAuthContext(apiKey, apiKey), "0.0.1")

	return &ClientSet{
		ruleGroups:          cxsdk.NewRuleGroupsClient(SDKAPIKeyCPC),
		alerts:              NewAlertsClient(apikeyCPC),
		recordingRuleGroups: cxsdk.NewRecordingRuleGroupSetsClient(SDKAPIKeyCPC),
		outboundWebhooks:    cxsdk.NewWebhooksClient(SDKAPIKeyCPC),
		alertsV3:            cxsdk.NewAlertsClient(SDKAPIKeyCPC),
		apiKeys:             cxsdk.NewAPIKeysClient(SDKAPIKeyCPC),
	}
}

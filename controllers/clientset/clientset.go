package clientset

type ClientSetInterface interface {
	RuleGroups() RuleGroupsClientInterface
	Alerts() AlertsClientInterface
	RecordingRuleGroups() RecordingRulesGroupsClientInterface
	Webhooks() WebhooksClientInterface
}

type ClientSet struct {
	ruleGroups          *RuleGroupsClient
	alerts              *AlertsClient
	recordingRuleGroups *RecordingRulesGroupsClient
	webhooks            *WebhooksClient
}

func (c *ClientSet) RuleGroups() RuleGroupsClientInterface {
	return c.ruleGroups
}

func (c *ClientSet) Alerts() AlertsClientInterface {
	return c.alerts
}

func (c *ClientSet) Webhooks() WebhooksClientInterface {
	return c.webhooks
}

func (c *ClientSet) RecordingRuleGroups() RecordingRulesGroupsClientInterface {
	return c.recordingRuleGroups
}

func NewClientSet(targetUrl, apiKey string) ClientSetInterface {
	apikeyCPC := NewCallPropertiesCreator(targetUrl, apiKey)

	return &ClientSet{
		ruleGroups:          NewRuleGroupsClient(apikeyCPC),
		alerts:              NewAlertsClient(apikeyCPC),
		recordingRuleGroups: NewRecordingRuleGroupsClient(apikeyCPC),
		webhooks:            NewWebhooksClient(apikeyCPC),
	}
}

package clientset

//go:generate mockgen -destination=../mock_clientset/mock_clientset.go -package=mock_clientset github.com/coralogix/coralogix-operator/controllers/clientset ClientSetInterface
type ClientSetInterface interface {
	RuleGroups() RuleGroupsClientInterface
	Alerts() AlertsClientInterface
	RecordingRuleGroups() RecordingRulesGroupsClientInterface
	OutboundWebhooks() OutboundWebhooksClientInterface
}

type ClientSet struct {
	ruleGroups          *RuleGroupsClient
	alerts              *AlertsClient
	recordingRuleGroups *RecordingRulesGroupsClient
	outboundWebhooks    *OutboundWebhooksClient
}

func (c *ClientSet) RuleGroups() RuleGroupsClientInterface {
	return c.ruleGroups
}

func (c *ClientSet) Alerts() AlertsClientInterface {
	return c.alerts
}

func (c *ClientSet) RecordingRuleGroups() RecordingRulesGroupsClientInterface {
	return c.recordingRuleGroups
}

func (c *ClientSet) OutboundWebhooks() OutboundWebhooksClientInterface {
	return c.outboundWebhooks
}

func NewClientSet(targetUrl, apiKey string) ClientSetInterface {
	apikeyCPC := NewCallPropertiesCreator(targetUrl, apiKey)

	return &ClientSet{
		ruleGroups:          NewRuleGroupsClient(apikeyCPC),
		alerts:              NewAlertsClient(apikeyCPC),
		recordingRuleGroups: NewRecordingRuleGroupsClient(apikeyCPC),
		outboundWebhooks:    NewOutboundWebhooksClient(apikeyCPC),
	}
}

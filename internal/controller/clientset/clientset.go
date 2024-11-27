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

package clientset

import cxsdk "github.com/coralogix/coralogix-management-sdk/go"

//go:generate mockgen -destination=../mock_clientset/mock_clientset.go -package=mock_clientset -source=clientset.go ClientSetInterface
type ClientSetInterface interface {
	RuleGroups() RuleGroupsClientInterface
	Alerts() AlertsClientInterface
	RecordingRuleGroups() RecordingRulesGroupsClientInterface
	OutboundWebhooks() OutboundWebhooksClientInterface
	ApiKeys() ApiKeysClientInterface
}

type ClientSet struct {
	ruleGroups          *cxsdk.RuleGroupsClient
	alerts              *AlertsClient
	recordingRuleGroups *cxsdk.RecordingRuleGroupSetsClient
	outboundWebhooks    *cxsdk.WebhooksClient
	apiKeys             *cxsdk.ApikeysClient
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

func (c *ClientSet) ApiKeys() ApiKeysClientInterface {
	return c.apiKeys
}

func NewClientSet(targetUrl, apiKey string) ClientSetInterface {
	apikeyCPC := NewCallPropertiesCreator(targetUrl, apiKey)
	SDKAPIKeyCPC := cxsdk.NewCallPropertiesCreator(targetUrl, cxsdk.NewAuthContext(apiKey, apiKey))

	return &ClientSet{
		ruleGroups:          cxsdk.NewRuleGroupsClient(SDKAPIKeyCPC),
		alerts:              NewAlertsClient(apikeyCPC),
		recordingRuleGroups: cxsdk.NewRecordingRuleGroupSetsClient(SDKAPIKeyCPC),
		outboundWebhooks:    cxsdk.NewWebhooksClient(SDKAPIKeyCPC),
		apiKeys:             cxsdk.NewAPIKeysClient(SDKAPIKeyCPC),
	}
}

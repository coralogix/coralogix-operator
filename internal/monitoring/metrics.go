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

package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	AlertInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cx_operator_alert_info",
			Help: "Coralogix Operator Alert information.",
		},
		[]string{"name", "namespace", "alert_type"},
	)
	RuleGroupInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cx_operator_rule_group_info",
			Help: "Coralogix Operator RuleGroup information.",
		},
		[]string{"name", "namespace"},
	)
	RecordingRuleGroupSetInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cx_operator_recording_rule_group_set_info",
			Help: "Coralogix Operator RecordingRuleGroupSet information.",
		},
		[]string{"name", "namespace"},
	)
	OutboundWebhookInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cx_operator_outbound_webhook_info",
			Help: "Coralogix Operator OutboundWebhook information.",
		},
		[]string{"name", "namespace", "webhook_type"},
	)
	PrometheusRuleInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cx_operator_tracked_prometheus_rule_info",
			Help: "Coralogix Operator tracked PrometheusRule information.",
		},
		[]string{"name", "namespace"},
	)
	AlertmanagerConfigInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cx_operator_tracked_alertmanager_config_info",
			Help: "Coralogix Operator tracked AlertmanagerConfig information.",
		},
		[]string{"name", "namespace"},
	)
	TotalRejectedRulesGroupsMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cx_operator_rejected_rule_groups_total",
			Help: "Total number of rejected rule groups.",
		},
	)
	TotalRejectedOutboundWebhooksMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cx_operator_rejected_outbound_webhooks_total",
			Help: "Total number of rejected outbound webhooks.",
		},
	)
)

func RegisterMetrics() error {
	metricsList := []prometheus.Collector{
		AlertInfoMetric,
		RuleGroupInfoMetric,
		RecordingRuleGroupSetInfoMetric,
		OutboundWebhookInfoMetric,
		PrometheusRuleInfoMetric,
		AlertmanagerConfigInfoMetric,
		TotalRejectedRulesGroupsMetric,
		TotalRejectedOutboundWebhooksMetric,
	}

	for _, metric := range metricsList {
		err := metrics.Registry.Register(metric)
		if err != nil {
			return err
		}
	}

	return nil
}

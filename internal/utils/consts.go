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

package utils

const (
	MonitoringAPIGroup = "monitoring.coreos.com"
	CoralogixAPIGroup  = "coralogix.com"

	V1alpha1APIVersion = "v1alpha1"
	V1beta1APIVersion  = "v1beta1"
	V1APIVersion       = "v1"

	RuleGroupKind             = "RuleGroup"
	AlertKind                 = "Alert"
	RecordingRuleGroupSetKind = "RecordingRuleGroupSet"
	OutboundWebhookKind       = "OutboundWebhook"
	ApiKeyKind                = "ApiKey"
	CustomRoleKind            = "CustomRole"
	ScopeKind                 = "Scope"
	GroupKind                 = "Group"
	TCOLogsPoliciesKind       = "TCOLogsPolicies"
	TCOTracesPoliciesKind     = "TCOTracesPolicies"
	IntegrationKind           = "Integration"
	AlertSchedulerKind        = "AlertScheduler"
	PrometheusRuleKind        = "PrometheusRule"
	DashboardKind             = "Dashboard"
	DashboardsFolderKind      = "DashboardsFolder"
	ViewKind                  = "View"
	ViewFolderKind            = "ViewFolder"
	ConnectorKind             = "Connector"
	PresetKind                = "Preset"
	GlobalRouterKind          = "GlobalRouter"
	ArchiveLogsTargetKind     = "ArchiveLogsTarget"
	ArchiveMetricsTargetKind  = "ArchiveMetricsTarget"
	SLOKind                   = "SLO"
	Events2MetricKind         = "Events2Metric"
	IPAccess                  = "IPAccess"
	CustomEnrichmentKind      = "CustomEnrichment"
	EnrichmentKind            = "Enrichment"

	TrackPrometheusRuleAlertsLabelKey         = "app.coralogix.com/track-alerting-rules"
	TrackPrometheusRuleRecordingRulesLabelKey = "app.coralogix.com/track-recording-rules"

	LogVerbosityAnnotationKey = "app.coralogix.com/log-verbosity"
)

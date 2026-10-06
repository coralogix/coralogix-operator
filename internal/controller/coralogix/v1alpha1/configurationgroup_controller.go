// Copyright 2026 Coralogix Ltd.
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
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	cfggroups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/fleet_manager_configuration_groups"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
)

// ConfigurationGroupReconciler reconciles a ConfigurationGroup object.
type ConfigurationGroupReconciler struct {
	ConfigurationGroupsClient *cfggroups.FleetManagerConfigurationGroupsAPIService
	Interval                  time.Duration
}

//+kubebuilder:rbac:groups=coralogix.com,resources=configurationgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=configurationgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=configurationgroups/finalizers,verbs=update

func (r *ConfigurationGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.ConfigurationGroup{}, r)
}

func (r *ConfigurationGroupReconciler) FinalizerName() string {
	return "configurationgroup.coralogix.com/finalizer"
}

func (r *ConfigurationGroupReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *ConfigurationGroupReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	group := obj.(*coralogixv1alpha1.ConfigurationGroup)
	createReq, err := expandCreateRequest(group)
	if err != nil {
		return fmt.Errorf("error on expanding configuration group: %w", err)
	}
	log.Info("Creating remote configuration group", "name", group.Spec.Name)
	createResp, httpResp, err := r.ConfigurationGroupsClient.
		ConfigurationGroupServiceCreateConfigurationGroup(ctx).
		ConfigurationGroupServiceCreateConfigurationGroupRequest(createReq).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote configuration group: %w", cxsdk.NewAPIError(httpResp, err))
	}
	if createResp == nil || createResp.Group == nil || createResp.Group.Id == nil {
		return fmt.Errorf("error on creating remote configuration group: empty response")
	}
	log.Info("Remote configuration group created", "id", createResp.Group.GetId(), "name", group.Spec.Name)
	group.Status = coralogixv1alpha1.ConfigurationGroupStatus{
		ID: ptr.To(createResp.Group.GetId()),
	}
	return nil
}

func (r *ConfigurationGroupReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	group := obj.(*coralogixv1alpha1.ConfigurationGroup)
	replaceReq, err := expandReplaceRequest(group)
	if err != nil {
		return fmt.Errorf("error on expanding configuration group: %w", err)
	}
	log.Info("Updating remote configuration group", "id", *group.Status.ID, "name", group.Spec.Name)
	_, httpResp, err := r.ConfigurationGroupsClient.
		ConfigurationGroupServiceReplaceConfigurationGroup(ctx, *group.Status.ID).
		ConfigurationGroupServiceReplaceConfigurationGroupRequest(replaceReq).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote configuration group updated")
	return nil
}

func (r *ConfigurationGroupReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	group := obj.(*coralogixv1alpha1.ConfigurationGroup)
	id := *group.Status.ID
	log.Info("Archiving configuration group in remote system", "id", id)

	if err := r.deactivateFamilyIfActive(ctx, group); err != nil {
		return err
	}

	_, httpResp, err := r.ConfigurationGroupsClient.
		ConfigurationGroupServiceArchiveConfigurationGroup(ctx, id).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			return fmt.Errorf("error archiving remote configuration group %s: %w", id, apiErr)
		}
	}
	log.Info("Configuration group archived in remote system", "id", id)
	return nil
}

func (r *ConfigurationGroupReconciler) deactivateFamilyIfActive(ctx context.Context, group *coralogixv1alpha1.ConfigurationGroup) error {
	// Send only family.active=false so a rejected desired spec (for example
	// forbidden collector YAML) cannot block archive by failing this replace.
	family := cfggroups.NewConfigurationGroupServiceReplaceConfigurationGroupRequestGroupFamily()
	family.SetActive(false)
	replace := cfggroups.NewConfigurationGroupServiceReplaceConfigurationGroupRequestGroup()
	replace.SetFamily(*family)
	replaceReq := cfggroups.NewConfigurationGroupServiceReplaceConfigurationGroupRequest()
	replaceReq.SetGroup(*replace)
	_, httpResp, err := r.ConfigurationGroupsClient.
		ConfigurationGroupServiceReplaceConfigurationGroup(ctx, *group.Status.ID).
		ConfigurationGroupServiceReplaceConfigurationGroupRequest(*replaceReq).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); cxsdk.IsNotFound(apiErr) {
			return nil
		}
		return fmt.Errorf("deactivating family before archive: %w", cxsdk.NewAPIError(httpResp, err))
	}
	return nil
}

func (r *ConfigurationGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ConfigurationGroup{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}

func expandCreateRequest(group *coralogixv1alpha1.ConfigurationGroup) (cfggroups.ConfigurationGroupServiceCreateConfigurationGroupRequest, error) {
	create := cfggroups.NewConfigurationGroupCreate()
	create.SetName(group.Spec.Name)
	if group.Spec.Description != nil {
		create.SetDescription(*group.Spec.Description)
	}
	if group.Spec.Tags != nil {
		create.SetTags(group.Spec.Tags)
	}
	if group.Spec.PriorityOrder != nil {
		create.SetPriorityOrder(*group.Spec.PriorityOrder)
	}
	family, err := expandFamilyCreate(group.Spec.Family)
	if err != nil {
		return cfggroups.ConfigurationGroupServiceCreateConfigurationGroupRequest{}, err
	}
	create.SetFamily(*family)
	req := cfggroups.NewConfigurationGroupServiceCreateConfigurationGroupRequest()
	req.SetGroup(*create)
	return *req, nil
}

func expandReplaceRequest(group *coralogixv1alpha1.ConfigurationGroup) (cfggroups.ConfigurationGroupServiceReplaceConfigurationGroupRequest, error) {
	replace := cfggroups.NewConfigurationGroupServiceReplaceConfigurationGroupRequestGroup()
	replace.SetName(group.Spec.Name)
	if group.Spec.Description != nil {
		replace.SetDescription(*group.Spec.Description)
	} else {
		replace.SetDescription("")
	}
	tags := group.Spec.Tags
	if tags == nil {
		tags = []string{}
	}
	replace.SetTags(tags)
	priorityOrder := int32(0)
	if group.Spec.PriorityOrder != nil {
		priorityOrder = *group.Spec.PriorityOrder
	}
	replace.SetPriorityOrder(priorityOrder)
	family, err := expandFamilyReplace(group.Spec.Family)
	if err != nil {
		return cfggroups.ConfigurationGroupServiceReplaceConfigurationGroupRequest{}, err
	}
	replace.SetFamily(*family)
	req := cfggroups.NewConfigurationGroupServiceReplaceConfigurationGroupRequest()
	req.SetGroup(*replace)
	return *req, nil
}

var schemaToOpenAPIChartName = map[string]cfggroups.ChartName{
	"otelIntegration":       cfggroups.CHARTNAME_CHART_NAME_OTEL_INTEGRATION,
	"otelLinuxStandalone":   cfggroups.CHARTNAME_CHART_NAME_OTEL_LINUX_STANDALONE,
	"otelWindowsStandalone": cfggroups.CHARTNAME_CHART_NAME_OTEL_WINDOWS_STANDALONE,
	"otelMacosStandalone":   cfggroups.CHARTNAME_CHART_NAME_OTEL_MACOS_STANDALONE,
	"otelEcsEc2":            cfggroups.CHARTNAME_CHART_NAME_OTEL_ECS_EC2,
}

func expandFamilyCreate(family coralogixv1alpha1.ConfigurationFamilySpec) (*cfggroups.ConfigurationFamilyCreate, error) {
	out := cfggroups.NewConfigurationFamilyCreate()
	if family.Active != nil {
		out.SetActive(*family.Active)
	}
	if family.Description != nil {
		out.SetDescription(*family.Description)
	}
	switch {
	case family.Preset != nil:
		chartName, observabilityFeatures, err := expandPresetCommon(family.Preset)
		if err != nil {
			return nil, err
		}
		preset := cfggroups.NewPresetConfigurationFamilyCreate(chartName, family.Preset.ChartVersion, observabilityFeatures)
		if family.Preset.IntegrationVersion != nil {
			preset.SetIntegrationVersion(*family.Preset.IntegrationVersion)
		}
		if family.Preset.Metadata != nil {
			preset.SetMetadata(family.Preset.Metadata)
		}
		out.SetPreset(*preset)
	case family.Raw != nil:
		raw := cfggroups.NewRawConfigurationFamilyCreate(expandRemoteCreates(family.Raw.RemoteConfigurations))
		if family.Raw.CollectorVersion != nil {
			raw.SetCollectorVersion(*family.Raw.CollectorVersion)
		}
		if family.Raw.Metadata != nil {
			raw.SetMetadata(family.Raw.Metadata)
		}
		out.SetRaw(*raw)
	default:
		return nil, fmt.Errorf("exactly one of family.preset or family.raw is required")
	}
	return out, nil
}

func expandFamilyReplace(family coralogixv1alpha1.ConfigurationFamilySpec) (*cfggroups.ConfigurationGroupServiceReplaceConfigurationGroupRequestGroupFamily, error) {
	out := cfggroups.NewConfigurationGroupServiceReplaceConfigurationGroupRequestGroupFamily()
	// Always send active: an omitted value deactivates the family on update.
	// Fall back to the CRD default (true) if the API server did not apply it.
	out.SetActive(ptr.Deref(family.Active, true))
	if family.Description != nil {
		out.SetDescription(*family.Description)
	} else {
		out.SetDescription("")
	}
	switch {
	case family.Preset != nil:
		chartName, observabilityFeatures, err := expandPresetCommon(family.Preset)
		if err != nil {
			return nil, err
		}
		preset := cfggroups.NewPresetConfigurationFamilyReplace(chartName, family.Preset.ChartVersion, observabilityFeatures)
		if family.Preset.IntegrationVersion != nil {
			preset.SetIntegrationVersion(*family.Preset.IntegrationVersion)
		}
		preset.SetMetadata(nonNilStringMap(family.Preset.Metadata))
		out.SetPreset(*preset)
	case family.Raw != nil:
		raw := cfggroups.NewRawConfigurationFamilyReplace(expandRemoteReplaces(family.Raw.RemoteConfigurations))
		if family.Raw.CollectorVersion != nil {
			raw.SetCollectorVersion(*family.Raw.CollectorVersion)
		}
		raw.SetMetadata(nonNilStringMap(family.Raw.Metadata))
		out.SetRaw(*raw)
	default:
		return nil, fmt.Errorf("exactly one of family.preset or family.raw is required")
	}
	return out, nil
}

func expandPresetCommon(preset *coralogixv1alpha1.PresetConfigurationFamilySpec) (cfggroups.ChartName, string, error) {
	chartName, ok := schemaToOpenAPIChartName[preset.ChartName]
	if !ok {
		return "", "", fmt.Errorf("unsupported family.preset.chartName %q", preset.ChartName)
	}
	observabilityFeatures, err := expandObservabilityFeatures(preset.ObservabilityFeatures)
	if err != nil {
		return "", "", err
	}
	return chartName, observabilityFeatures, nil
}

// expandObservabilityFeatures serializes the structured CRD object into the JSON object string the API expects.
func expandObservabilityFeatures(features runtime.RawExtension) (string, error) {
	if len(features.Raw) == 0 {
		return "{}", nil
	}
	var object map[string]interface{}
	if err := json.Unmarshal(features.Raw, &object); err != nil {
		return "", fmt.Errorf("family.preset.observabilityFeatures must be a JSON object: %w", err)
	}
	if object == nil {
		return "{}", nil
	}
	out, err := json.Marshal(object)
	if err != nil {
		return "", fmt.Errorf("marshaling family.preset.observabilityFeatures: %w", err)
	}
	return string(out), nil
}

func nonNilStringMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

func expandRemoteCreates(remotes []coralogixv1alpha1.RemoteConfigurationSpec) []cfggroups.RemoteConfigurationCreate {
	out := make([]cfggroups.RemoteConfigurationCreate, 0, len(remotes))
	for _, remote := range remotes {
		item := cfggroups.NewRemoteConfigurationCreate()
		item.SetName(remote.Name)
		item.SetRawConfiguration(remote.RawConfiguration)
		if len(remote.AgentSelector) > 0 {
			selector := cfggroups.NewAgentSelectorRequest()
			selector.SetAttributes(remote.AgentSelector)
			item.SetAgentSelector(*selector)
		}
		out = append(out, *item)
	}
	return out
}

func expandRemoteReplaces(remotes []coralogixv1alpha1.RemoteConfigurationSpec) []cfggroups.RemoteConfigurationReplace {
	out := make([]cfggroups.RemoteConfigurationReplace, 0, len(remotes))
	for _, remote := range remotes {
		item := cfggroups.NewRemoteConfigurationReplace()
		item.SetName(remote.Name)
		item.SetRawConfiguration(remote.RawConfiguration)
		if len(remote.AgentSelector) > 0 {
			selector := cfggroups.NewAgentSelectorRequest()
			selector.SetAttributes(remote.AgentSelector)
			item.SetAgentSelector(*selector)
		}
		out = append(out, *item)
	}
	return out
}

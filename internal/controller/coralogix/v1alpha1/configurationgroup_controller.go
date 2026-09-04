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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	cfggroups "github.com/coralogix/coralogix-operator/v2/internal/openapi/configuration_group_service"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
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
	createReq := expandCreateRequest(group)
	log.Info("Creating remote configuration group", "configurationGroup", utils.FormatJSON(createReq))
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
	log.Info("Remote configuration group created", "response", utils.FormatJSON(createResp))
	group.Status = coralogixv1alpha1.ConfigurationGroupStatus{
		ID: ptr.To(createResp.Group.GetId()),
	}
	return nil
}

func (r *ConfigurationGroupReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	group := obj.(*coralogixv1alpha1.ConfigurationGroup)
	replaceReq := expandReplaceRequest(group)
	log.Info("Updating remote configuration group", "configurationGroup", utils.FormatJSON(replaceReq))
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
			log.Error(err, "Error archiving remote configuration group", "id", id)
			return fmt.Errorf("error archiving remote configuration group %s: %w", id, apiErr)
		}
	}
	log.Info("Configuration group archived in remote system", "id", id)
	return nil
}

func (r *ConfigurationGroupReconciler) deactivateFamilyIfActive(ctx context.Context, group *coralogixv1alpha1.ConfigurationGroup) error {
	if group.Spec.Family.Active != nil && !*group.Spec.Family.Active {
		return nil
	}
	inactive := group.DeepCopy()
	inactive.Spec.Family.Active = ptr.To(false)
	replaceReq := expandReplaceRequest(inactive)
	_, httpResp, err := r.ConfigurationGroupsClient.
		ConfigurationGroupServiceReplaceConfigurationGroup(ctx, *group.Status.ID).
		ConfigurationGroupServiceReplaceConfigurationGroupRequest(replaceReq).
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

func expandCreateRequest(group *coralogixv1alpha1.ConfigurationGroup) cfggroups.ConfigurationGroupServiceCreateConfigurationGroupRequest {
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
	create.SetFamily(*expandFamilyCreate(group.Spec.Family))
	req := cfggroups.NewConfigurationGroupServiceCreateConfigurationGroupRequest()
	req.SetGroup(*create)
	return *req
}

func expandReplaceRequest(group *coralogixv1alpha1.ConfigurationGroup) cfggroups.ConfigurationGroupServiceReplaceConfigurationGroupRequest {
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
	if group.Spec.PriorityOrder != nil {
		replace.SetPriorityOrder(*group.Spec.PriorityOrder)
	}
	replace.SetFamily(*expandFamilyReplace(group.Spec.Family))
	req := cfggroups.NewConfigurationGroupServiceReplaceConfigurationGroupRequest()
	req.SetGroup(*replace)
	return *req
}

func expandFamilyCreate(family coralogixv1alpha1.ConfigurationFamilySpec) *cfggroups.ConfigurationFamilyCreate {
	out := cfggroups.NewConfigurationFamilyCreate()
	if family.Active != nil {
		out.SetActive(*family.Active)
	}
	if family.Description != nil {
		out.SetDescription(*family.Description)
	}
	if family.CollectorVersion != nil {
		out.SetCollectorVersion(*family.CollectorVersion)
	}
	if family.Metadata != nil {
		out.SetMetadata(family.Metadata)
	}
	out.SetRemoteConfigurations(expandRemoteCreates(family.RemoteConfigurations))
	return out
}

func expandFamilyReplace(family coralogixv1alpha1.ConfigurationFamilySpec) *cfggroups.ConfigurationGroupServiceReplaceConfigurationGroupRequestGroupFamily {
	out := cfggroups.NewConfigurationGroupServiceReplaceConfigurationGroupRequestGroupFamily()
	if family.Active != nil {
		out.SetActive(*family.Active)
	}
	if family.Description != nil {
		out.SetDescription(*family.Description)
	} else {
		out.SetDescription("")
	}
	if family.CollectorVersion != nil {
		out.SetCollectorVersion(*family.CollectorVersion)
	}
	metadata := family.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	out.SetMetadata(metadata)
	out.SetRemoteConfigurations(expandRemoteReplaces(family.RemoteConfigurations))
	return out
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

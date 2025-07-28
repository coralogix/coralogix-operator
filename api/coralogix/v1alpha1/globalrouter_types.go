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
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/internal/config"
)

// GlobalRouterSpec defines the desired state of the Global Router.
type GlobalRouterSpec struct {
	// Name is the name of the global router.
	Name string `json:"name"`

	// Description is the description of the global router.
	Description string `json:"description"`

	// EntityType is the entity type for the global router. Should equal "alerts".
	// +kubebuilder:validation:Enum=alerts
	EntityType string `json:"entityType"`

	// EntityLabels are optional labels to attach to the global router.
	// +optional
	EntityLabels map[string]string `json:"entityLabels,omitempty"`

	// Fallback is the fallback routing target for the global router.
	// +optional
	Fallback []RoutingTarget `json:"fallback,omitempty"`

	// Rules are the routing rules for the global router.
	// +optional
	Rules []RoutingRule `json:"rules,omitempty"`
}

type RoutingRule struct {
	// Name is the name of the routing rule.
	Name string `json:"name"`

	// CustomDetails are optional custom details to attach to the routing rule.
	// +optional
	CustomDetails map[string]string `json:"customDetails,omitempty"`

	// Condition is the condition for the routing rule.
	Condition string `json:"condition"`

	// Targets are the routing targets for the routing rule.
	Targets []RoutingTarget `json:"targets"`
}

type RoutingTarget struct {
	// CustomDetails are optional custom details to attach to the routing target.
	// +optional
	CustomDetails map[string]string `json:"customDetails,omitempty"`

	// Connector is the connector for the routing target. Should be one of backendRef or resourceRef.
	Connector NCRef `json:"connector"`

	// Preset is the preset for the routing target. Should be one of backendRef or resourceRef.
	// +optional
	Preset *NCRef `json:"preset,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type NCRef struct {
	// BackendRef is a reference to a backend resource.
	// +optional
	BackendRef *NCBackendRef `json:"backendRef,omitempty"`

	// ResourceRef is a reference to a Kubernetes resource.
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

func (g *GlobalRouter) ExtractCreateOrReplaceGlobalRouterRequest(ctx context.Context) (*cxsdk.CreateOrReplaceGlobalRouterRequest, error) {
	router, err := g.ExtractGlobalRouter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract global-router: %w", err)
	}

	router.Id = ptr.To("router_default")
	return &cxsdk.CreateOrReplaceGlobalRouterRequest{Router: router}, nil
}

func (g *GlobalRouter) ExtractGlobalRouter(ctx context.Context) (*cxsdk.GlobalRouter, error) {
	fallback, err := extractRoutingTargets(ctx, g.Namespace, g.Spec.Fallback)
	if err != nil {
		return nil, err
	}

	rules, err := extractRoutingRules(ctx, g.Namespace, g.Spec.Rules)
	if err != nil {
		return nil, err
	}

	entityType, ok := schemaToProtoEntityType[g.Spec.EntityType]
	if !ok {
		return nil, fmt.Errorf("invalid entity type %s", g.Spec.EntityType)
	}

	return &cxsdk.GlobalRouter{
		Name:        g.Spec.Name,
		Description: g.Spec.Description,
		EntityType:  entityType,
		Fallback:    fallback,
		Rules:       rules,
	}, nil
}

func extractRoutingRules(ctx context.Context, namespace string, rules []RoutingRule) ([]*cxsdk.RoutingRule, error) {
	var result []*cxsdk.RoutingRule
	var errs error
	for _, rule := range rules {
		routingRule, err := extractRoutingRule(ctx, namespace, rule)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		result = append(result, routingRule)
	}

	if errs != nil {
		return nil, errs
	}

	return result, nil
}

func extractRoutingRule(ctx context.Context, namespace string, rule RoutingRule) (*cxsdk.RoutingRule, error) {
	targets, err := extractRoutingTargets(ctx, namespace, rule.Targets)
	if err != nil {
		return nil, err
	}

	return &cxsdk.RoutingRule{
		Name:      ptr.To(rule.Name),
		Condition: rule.Condition,
		Targets:   targets,
	}, nil
}

func extractRoutingTargets(ctx context.Context, namespace string, targets []RoutingTarget) ([]*cxsdk.RoutingTarget, error) {
	var result []*cxsdk.RoutingTarget
	var errs error
	for _, target := range targets {
		routingTarget, err := extractRoutingTarget(ctx, namespace, target)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		result = append(result, routingTarget)
	}

	if errs != nil {
		return nil, errs
	}

	return result, nil
}

func extractRoutingTarget(ctx context.Context, namespace string, target RoutingTarget) (*cxsdk.RoutingTarget, error) {
	connectorID, err := extractConnectorID(ctx, namespace, target.Connector)
	if err != nil {
		return nil, err
	}

	var presetID string
	if target.Preset != nil {
		presetID, err = extractPresetID(ctx, namespace, target.Preset)
		if err != nil {
			return nil, err
		}
	}

	return &cxsdk.RoutingTarget{
		ConnectorId: connectorID,
		PresetId:    ptr.To(presetID),
	}, nil
}

func extractConnectorID(ctx context.Context, namespace string, connector NCRef) (string, error) {
	if connector.BackendRef != nil {
		return connector.BackendRef.ID, nil
	}

	if connector.ResourceRef != nil && connector.ResourceRef.Namespace != nil {
		namespace = *connector.ResourceRef.Namespace
	}

	c := &Connector{}
	err := config.GetClient().Get(ctx, client.ObjectKey{Name: connector.ResourceRef.Name, Namespace: namespace}, c)
	if err != nil {
		return "", err
	}

	if !config.GetConfig().Selector.Matches(c.Labels, c.Namespace) {
		return "", fmt.Errorf("connector %s does not match selector", c.Name)
	}

	if c.Status.Id == nil {
		return "", fmt.Errorf("ID is not populated for Connector %s", c.Name)
	}

	return *c.Status.Id, nil
}

func extractPresetID(ctx context.Context, namespace string, preset *NCRef) (string, error) {
	if preset.BackendRef != nil {
		return preset.BackendRef.ID, nil
	}

	if preset.ResourceRef != nil && preset.ResourceRef.Namespace != nil {
		namespace = *preset.ResourceRef.Namespace
	}

	p := &Preset{}
	err := config.GetClient().Get(ctx, client.ObjectKey{Name: preset.ResourceRef.Name, Namespace: namespace}, p)
	if err != nil {
		return "", err
	}

	if !config.GetConfig().Selector.Matches(p.Labels, p.Namespace) {
		return "", fmt.Errorf("preset %s does not match selector", p.Name)
	}

	if p.Status.Id == nil {
		return "", fmt.Errorf("ID is not populated for Preset %s", p.Name)
	}

	return *p.Status.Id, nil
}

type NCBackendRef struct {
	ID string `json:"id"`
}

// GlobalRouterStatus defines the observed state of GlobalRouter.
type GlobalRouterStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (g *GlobalRouter) GetConditions() []metav1.Condition {
	return g.Status.Conditions
}

func (g *GlobalRouter) SetConditions(conditions []metav1.Condition) {
	g.Status.Conditions = conditions
}

func (g *GlobalRouter) HasIDInStatus() bool {
	return g.Status.Id != nil && *g.Status.Id != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// GlobalRouter is the Schema for the GlobalRouters API.
// NOTE: This CRD exposes a new feature and may have breaking changes in future releases.
//
// See also https://coralogix.com/docs/user-guides/notification-center/routing/
//
// **Added in v0.4.0**
type GlobalRouter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GlobalRouterSpec   `json:"spec,omitempty"`
	Status GlobalRouterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GlobalRouterList contains a list of GlobalRouter.
type GlobalRouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalRouter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GlobalRouter{}, &GlobalRouterList{})
}

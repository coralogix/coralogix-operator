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

// GlobalRouterSpec defines the desired state of GlobalRouter.
type GlobalRouterSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	// +kubebuilder:validation:Enum=alerts
	EntityType string `json:"entityType"`

	EntityLabels map[string]string `json:"entityLabels"`

	Fallback []RoutingTarget `json:"fallback"`

	Rules []RoutingRule `json:"rules"`
}

type RoutingRule struct {
	// +optional
	Name *string `json:"name,omitempty"`

	CustomDetails map[string]string `json:"customDetails"`

	Condition string `json:"condition"`

	Targets []RoutingTarget `json:"targets"`
}

type RoutingTarget struct {
	CustomDetails map[string]string `json:"customDetails"`

	Connector *NCRef `json:"connector"`

	Preset *NCRef `json:"preset"`
}

type NCRef struct {
	// +optional
	BackendRef *NCBackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

func (g *GlobalRouter) ExtractCreateGlobalRouterRequest(ctx context.Context) (*cxsdk.CreateGlobalRouterRequest, error) {
	router, err := g.ExtractGlobalRouter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract global-router: %w", err)
	}

	return &cxsdk.CreateGlobalRouterRequest{Router: router}, nil
}

func (g *GlobalRouter) ExtractUpdateGlobalRouterRequest(ctx context.Context) (*cxsdk.ReplaceGlobalRouterRequest, error) {
	router, err := g.ExtractGlobalRouter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract global-router: %w", err)
	}

	router.Id = g.Status.Id
	return &cxsdk.ReplaceGlobalRouterRequest{Router: router}, nil
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

	var entityType cxsdk.EntityType
	if et, ok := schemaToProtoEntityType[g.Spec.EntityType]; ok {
		entityType = et
	} else {
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

	presetID, err := extractPresetID(ctx, namespace, target.Preset)
	if err != nil {
		return nil, err
	}

	return &cxsdk.RoutingTarget{
		ConnectorId: connectorID,
		PresetId:    presetID,
	}, nil
}

func extractConnectorID(ctx context.Context, namespace string, connector *NCRef) (string, error) {
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

func extractPresetID(ctx context.Context, namespace string, preset *NCRef) (*string, error) {
	if preset.BackendRef != nil {
		return ptr.To(preset.BackendRef.ID), nil
	}

	if preset.ResourceRef != nil && preset.ResourceRef.Namespace != nil {
		namespace = *preset.ResourceRef.Namespace
	}

	p := &Preset{}
	err := config.GetClient().Get(ctx, client.ObjectKey{Name: preset.ResourceRef.Name, Namespace: namespace}, p)
	if err != nil {
		return nil, err
	}

	if !config.GetConfig().Selector.Matches(p.Labels, p.Namespace) {
		return nil, fmt.Errorf("preset %s does not match selector", p.Name)
	}

	if p.Status.Id == nil {
		return nil, fmt.Errorf("ID is not populated for Preset %s", p.Name)
	}

	return p.Status.Id, nil
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GlobalRouter is the Schema for the globalrouters API.
// NOTE: This CRD exposes a new feature and may have breaking changes in future releases.
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

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

	"github.com/coralogix/coralogix-operator/internal/utils"
)

// GlobalRouterSpec defines the desired state of GlobalRouter.
type GlobalRouterSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	EntityType string `json:"entityType"`

	Fallback []RoutingTarget `json:"fallback"`

	// +optional
	Rules []RoutingRule `json:"rules,omitempty"`
}

type RoutingRule struct {
	Name string `json:"name"`

	Condition string `json:"condition"`

	Targets []RoutingTarget `json:"targets"`
}

type RoutingTarget struct {
	Connector *NCRef `json:"connector"`

	Preset *NCRef `json:"preset"`
}

type NCRef struct {
	// +optional
	BackendRef *NCBackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

func (g *GlobalRouter) ExtractCreateGlobalRouterRequest(refProperties *ResourceRefProperties) (*cxsdk.CreateGlobalRouterRequest, error) {
	router, err := g.ExtractGlobalRouter(refProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to extract global-router: %w", err)
	}

	return &cxsdk.CreateGlobalRouterRequest{Router: router}, nil
}

func (g *GlobalRouter) ExtractUpdateGlobalRouterRequest(refProperties *ResourceRefProperties) (*cxsdk.ReplaceGlobalRouterRequest, error) {
	router, err := g.ExtractGlobalRouter(refProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to extract global-router: %w", err)
	}

	router.Id = g.Status.ID
	return &cxsdk.ReplaceGlobalRouterRequest{Router: router}, nil
}

func (g *GlobalRouter) ExtractGlobalRouter(refProperties *ResourceRefProperties) (*cxsdk.GlobalRouter, error) {
	fallback, err := extractRoutingTargets(refProperties, g.Spec.Fallback)
	if err != nil {
		return nil, err
	}

	rules, err := extractRoutingRules(refProperties, g.Spec.Rules)
	if err != nil {
		return nil, err
	}

	return &cxsdk.GlobalRouter{
		Name:        g.Spec.Name,
		Description: g.Spec.Description,
		EntityType:  g.Spec.EntityType,
		Fallback:    fallback,
		Rules:       rules,
	}, nil
}

func extractRoutingRules(refProperties *ResourceRefProperties, rules []RoutingRule) ([]*cxsdk.RoutingRule, error) {
	var result []*cxsdk.RoutingRule
	var errs error
	for _, rule := range rules {
		routingRule, err := extractRoutingRule(refProperties, rule)
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

func extractRoutingRule(refProperties *ResourceRefProperties, rule RoutingRule) (*cxsdk.RoutingRule, error) {
	targets, err := extractRoutingTargets(refProperties, rule.Targets)
	if err != nil {
		return nil, err
	}

	return &cxsdk.RoutingRule{
		Name:      ptr.To(rule.Name),
		Condition: rule.Condition,
		Targets:   targets,
	}, nil
}

func extractRoutingTargets(refProperties *ResourceRefProperties, targets []RoutingTarget) ([]*cxsdk.RoutingTarget, error) {
	var result []*cxsdk.RoutingTarget
	var errs error
	for _, target := range targets {
		routingTarget, err := extractRoutingTarget(refProperties, target)
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

func extractRoutingTarget(refProperties *ResourceRefProperties, target RoutingTarget) (*cxsdk.RoutingTarget, error) {
	connectorID, err := extractConnectorID(refProperties, target.Connector)
	if err != nil {
		return nil, err
	}

	presetID, err := extractPresetID(refProperties, target.Preset)
	if err != nil {
		return nil, err
	}

	return &cxsdk.RoutingTarget{
		ConnectorId: connectorID,
		PresetId:    presetID,
	}, nil
}

func extractConnectorID(refProperties *ResourceRefProperties, connector *NCRef) (string, error) {
	if connector.BackendRef != nil {
		return connector.BackendRef.ID, nil
	}

	var namespace string
	if connector.ResourceRef != nil && connector.ResourceRef.Namespace != nil {
		namespace = *connector.ResourceRef.Namespace
	} else {
		namespace = refProperties.Namespace
	}

	c := &Connector{}
	err := refProperties.Client.Get(context.Background(), client.ObjectKey{Name: connector.ResourceRef.Name, Namespace: namespace}, c)
	if err != nil {
		return "", err
	}

	if !utils.GetLabelFilter().Matches(c.Labels) {
		return "", fmt.Errorf("connector %s does not match label selector", c.Name)
	}

	if c.Status.Id == nil {
		return "", fmt.Errorf("ID is not populated for Connector %s", c.Name)
	}

	return *c.Status.Id, nil
}

func extractPresetID(refProperties *ResourceRefProperties, preset *NCRef) (*string, error) {
	if preset.BackendRef != nil {
		return ptr.To(preset.BackendRef.ID), nil
	}

	var namespace string
	if preset.ResourceRef != nil && preset.ResourceRef.Namespace != nil {
		namespace = *preset.ResourceRef.Namespace
	} else {
		namespace = refProperties.Namespace
	}

	p := &Preset{}
	err := refProperties.Client.Get(context.Background(), client.ObjectKey{Name: preset.ResourceRef.Name, Namespace: namespace}, p)
	if err != nil {
		return nil, err
	}

	if !utils.GetLabelFilter().Matches(p.Labels) {
		return nil, fmt.Errorf("Preset %s does not match label selector", p.Name)
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
	ID *string `json:"id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GlobalRouter is the Schema for the globalrouters API.
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

// +k8s:deepcopy-gen=false
type ResourceRefProperties struct {
	Client    client.Client
	Namespace string
}

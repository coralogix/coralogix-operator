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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var presetlog = logf.Log.WithName("preset-resource")

// SetupPresetWebhookWithManager registers the webhook for Preset in the manager.
func SetupPresetWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralogixv1alpha1.Preset{}).
		WithValidator(&PresetCustomValidator{}).
		Complete()
}

var entityTypesToSubtypes = map[string][]string{
	"alerts": {
		"metricThresholdMoreThanTriggered",
		"metricThresholdMoreThanResolved",
		"metricThresholdMoreThanOrEqualsTriggered",
		"metricThresholdMoreThanOrEqualsResolved",
		"metricThresholdLessThanTriggered",
		"metricThresholdLessThanResolved",
		"metricThresholdLessThanOrEqualsTriggered",
		"metricThresholdLessThanOrEqualsResolved",
		"logsImmediateTriggered",
		"logsImmediateResolved",
		"logsThresholdLessThanTriggered",
		"logsThresholdLessThanResolved",
		"logsThresholdMoreThanTriggered",
		"logsThresholdMoreThanResolved",
	},
}

// +kubebuilder:webhook:path=/validate-coralogix-com-v1alpha1-preset,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=presets,verbs=create;update,versions=v1alpha1,name=vpreset-v1alpha1.kb.io,admissionReviewVersions=v1

// PresetCustomValidator struct is responsible for validating the Preset resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type PresetCustomValidator struct{}

var _ webhook.CustomValidator = &PresetCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Preset.
func (v *PresetCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	preset, ok := obj.(*coralogixv1alpha1.Preset)
	if !ok {
		return nil, fmt.Errorf("expected a Preset object but got %T", obj)
	}
	presetlog.Info("Validation for Preset upon creation", "name", preset.GetName())

	typeErr := validatePresetConnectorType(preset.Spec.ConnectorType)
	if typeErr != nil {
		return admission.Warnings{typeErr.Error()}, fmt.Errorf("validation failed: %v", typeErr)
	}

	if preset.Spec.ConnectorType.GenericHttps != nil {
		genericHttpsErr := validateGenericHttps(preset.Spec)
		if genericHttpsErr != nil {
			return admission.Warnings{genericHttpsErr.Error()}, fmt.Errorf("validation failed: %v", genericHttpsErr)
		}
	}

	if preset.Spec.ConnectorType.Slack != nil {
		slackErr := validateSlack(preset.Spec)
		if slackErr != nil {
			return admission.Warnings{slackErr.Error()}, fmt.Errorf("validation failed: %v", slackErr)
		}
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Preset.
func (v *PresetCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	preset, ok := newObj.(*coralogixv1alpha1.Preset)
	if !ok {
		return nil, fmt.Errorf("expected a Preset object for the newObj but got %T", newObj)
	}
	presetlog.Info("Validation for Preset upon update", "name", preset.GetName())

	typeErr := validatePresetConnectorType(preset.Spec.ConnectorType)
	if typeErr != nil {
		return admission.Warnings{typeErr.Error()}, fmt.Errorf("validation failed: %v", typeErr)
	}

	if preset.Spec.ConnectorType.GenericHttps != nil {
		genericHttpsErr := validateGenericHttps(preset.Spec)
		if genericHttpsErr != nil {
			return admission.Warnings{genericHttpsErr.Error()}, fmt.Errorf("validation failed: %v", genericHttpsErr)
		}
	}

	if preset.Spec.ConnectorType.Slack != nil {
		slackErr := validateSlack(preset.Spec)
		if slackErr != nil {
			return admission.Warnings{slackErr.Error()}, fmt.Errorf("validation failed: %v", slackErr)
		}
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Preset.
func (v *PresetCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validatePresetConnectorType(presetConnectorType *coralogixv1alpha1.PresetConnectorType) error {
	if presetConnectorType == nil {
		return fmt.Errorf("connector type should be set")
	}

	var typesSet []string
	if presetConnectorType.GenericHttps != nil {
		typesSet = append(typesSet, "GenericHttps")
	}
	if presetConnectorType.Slack != nil {
		typesSet = append(typesSet, "Slack")
	}

	if len(typesSet) > 1 {
		return fmt.Errorf("only one connector type should be set, got: %v", typesSet)
	}

	return nil
}

func validateGenericHttps(spec coralogixv1alpha1.PresetSpec) error {
	return validateGenericHttpsSubtypes(spec.EntityType, spec.ConnectorType.GenericHttps)
}

func validateGenericHttpsSubtypes(entityType string, genericHttps *coralogixv1alpha1.PresetConnectorTypeGenericHttps) error {
	var invalidSubtypes []string

	for _, override := range genericHttps.Overrides {
		valid := false
		for _, validSubType := range entityTypesToSubtypes[entityType] {
			if override.EntitySubType == validSubType {
				valid = true
				break
			}
		}
		if !valid {
			invalidSubtypes = append(invalidSubtypes, override.EntitySubType)
		}
	}

	if len(invalidSubtypes) > 0 {
		return fmt.Errorf("invalid entity subtypes for entity type %s: %v. Can be one of: %v", entityType,
			invalidSubtypes, entityTypesToSubtypes[entityType])
	}

	return nil
}

func validateSlack(spec coralogixv1alpha1.PresetSpec) error {
	return validateSlackEntitySubtypes(spec)
}

func validateSlackEntitySubtypes(presetSpec coralogixv1alpha1.PresetSpec) error {
	invalidSubtypes := getInvalidSlackSubtypes(presetSpec.EntityType, presetSpec.ConnectorType.Slack)
	if len(invalidSubtypes) > 0 {
		return fmt.Errorf("invalid entity subtypes for entity type %s: %v. Can be one of: %v", presetSpec.EntityType,
			invalidSubtypes, entityTypesToSubtypes[presetSpec.EntityType])
	}

	return nil
}

func getInvalidSlackSubtypes(entityType string, slack *coralogixv1alpha1.PresetConnectorTypeSlack) []string {
	var invalidSubtypes []string

	for _, override := range slack.Overrides {
		valid := false
		for _, validSubType := range entityTypesToSubtypes[entityType] {
			if override.EntitySubType == validSubType {
				valid = true
				break
			}
		}
		if !valid {
			invalidSubtypes = append(invalidSubtypes, override.EntitySubType)
		}
	}

	return invalidSubtypes
}

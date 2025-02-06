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

	err := validatePresetConnectorType(preset.Spec.ConnectorType)
	if err != nil {
		return admission.Warnings{err.Error()}, fmt.Errorf("validation failed: %v", err)
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

	err := validatePresetConnectorType(preset.Spec.ConnectorType)
	if err != nil {
		return admission.Warnings{err.Error()}, fmt.Errorf("validation failed: %v", err)
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Preset.
func (v *PresetCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validatePresetConnectorType(presetConnectorType *coralogixv1alpha1.PresetConnectorType) error {
	if presetConnectorType == nil {
		return fmt.Errorf("connector type should be set for the Preset")
	}

	var typesSet []string
	if presetConnectorType.GenericHttps != nil {
		typesSet = append(typesSet, "GenericHttps")
	}
	if presetConnectorType.Slack != nil {
		typesSet = append(typesSet, "Slack")
	}

	if len(typesSet) > 1 {
		return fmt.Errorf("only one connector type should be set for the Preset, got: %v", typesSet)
	}

	return nil
}

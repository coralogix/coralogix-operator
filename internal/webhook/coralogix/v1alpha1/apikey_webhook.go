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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
)

var apikeylog = logf.Log.WithName("apikey-resource")

// SetupApiKeyWebhookWithManager registers the webhook for ApiKey in the manager.
func SetupApiKeyWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralogixv1alpha1.ApiKey{}).
		WithValidator(&ApiKeyCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-coralogix-com-v1alpha1-apikey,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=apikeys,verbs=create;update,versions=v1alpha1,name=vapikey-v1alpha1.kb.io,admissionReviewVersions=v1

// ApiKeyCustomValidator struct is responsible for validating the ApiKey resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ApiKeyCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &ApiKeyCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type ApiKey.
func (v *ApiKeyCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	apikey, ok := obj.(*coralogixv1alpha1.ApiKey)

	if !ok {
		return nil, fmt.Errorf("expected a ApiKey object but got %T", obj)
	}
	apikeylog.Info("Validation for ApiKey upon creation", "name", apikey.GetName())

	var warnings admission.Warnings
	var errs error
	err := validateOwner(apikey.Spec.Owner)
	if err != nil {
		warnings = append(warnings, err.Error())
		errs = errors.Join(errs, err)
	}

	err = validatePresetsAndPermissions(apikey.Spec.Presets, apikey.Spec.Permissions)
	if err != nil {
		warnings = append(warnings, err.Error())
		errs = errors.Join(errs, err)
	}

	err = validateActive(apikey.Spec.Active)
	if err != nil {
		warnings = append(warnings, err.Error())
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		monitoring.IncResourceRejectionsTotalMetric(apikey.Kind, apikey.Name, apikey.Namespace)
		return warnings, errs
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type ApiKey.
func (v *ApiKeyCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	apikey, ok := newObj.(*coralogixv1alpha1.ApiKey)
	if !ok {
		return nil, fmt.Errorf("expected a ApiKey object for the newObj but got %T", newObj)
	}
	apikeylog.Info("Validation for ApiKey upon update", "name", apikey.GetName())

	var warnings admission.Warnings
	var errs error
	err := validateOwner(apikey.Spec.Owner)
	if err != nil {
		warnings = append(warnings, err.Error())
		errs = errors.Join(errs, err)
	}

	err = validatePresetsAndPermissions(apikey.Spec.Presets, apikey.Spec.Permissions)
	if err != nil {
		warnings = append(warnings, err.Error())
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		monitoring.IncResourceRejectionsTotalMetric(apikey.Kind, apikey.Name, apikey.Namespace)
		return warnings, errs
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type ApiKey.
func (v *ApiKeyCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateOwner(owner coralogixv1alpha1.ApiKeyOwner) error {
	if owner.UserId != nil && owner.TeamId != nil {
		return fmt.Errorf("only one of the owner user ID or owner team ID can be set")
	}
	return nil
}

func validatePresetsAndPermissions(presets, permissions []string) error {
	if presets == nil && permissions == nil {
		return fmt.Errorf("at least one of the presets or permissions fields must be set")
	}
	return nil
}

func validateActive(active bool) error {
	if !active {
		return fmt.Errorf("ApiKey must be activated on creation")
	}
	return nil
}

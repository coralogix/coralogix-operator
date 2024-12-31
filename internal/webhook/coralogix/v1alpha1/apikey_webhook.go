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
	var errorsMessages []string
	errorMsg := validateOwner(apikey.Spec.Owner)
	if errorMsg != "" {
		warnings = append(warnings, errorMsg)
		errorsMessages = append(errorsMessages, errorMsg)
	}

	errorMsg = validatePresetsAndPermissions(apikey.Spec.Presets, apikey.Spec.Permissions)
	if errorMsg != "" {
		warnings = append(warnings, errorMsg)
		errorsMessages = append(errorsMessages, errorMsg)
	}

	errorMsg = validateActive(apikey.Spec.Active)
	if errorMsg != "" {
		warnings = append(warnings, errorMsg)
		errorsMessages = append(errorsMessages, errorMsg)
	}

	if len(errorsMessages) > 0 {
		monitoring.TotalRejectedApiKeysMetric.Inc()
		return warnings, fmt.Errorf("validation failed: %v", errorsMessages)
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
	var errorsMessages []string
	errorMsg := validateOwner(apikey.Spec.Owner)
	if errorMsg != "" {
		warnings = append(warnings, errorMsg)
		errorsMessages = append(errorsMessages, errorMsg)
	}

	errorMsg = validatePresetsAndPermissions(apikey.Spec.Presets, apikey.Spec.Permissions)
	if errorMsg != "" {
		warnings = append(warnings, errorMsg)
		errorsMessages = append(errorsMessages, errorMsg)
	}

	if len(errorsMessages) > 0 {
		monitoring.TotalRejectedApiKeysMetric.Inc()
		return warnings, fmt.Errorf("validation failed: %v", errorsMessages)
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type ApiKey.
func (v *ApiKeyCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateOwner(owner coralogixv1alpha1.ApiKeyOwner) string {
	if owner.UserId != nil && owner.TeamId != nil {
		return "Only one of the owner user ID or owner team ID can be set"
	}
	return ""
}

func validatePresetsAndPermissions(presets, permissions []string) string {
	if (presets == nil || len(presets) == 0) && (permissions == nil || len(permissions) == 0) {
		return "At least one of the presets or permissions fields must be set"
	}
	return ""
}

func validateActive(active bool) string {
	if !active {
		return "ApiKey must be activated on creation"
	}
	return ""
}

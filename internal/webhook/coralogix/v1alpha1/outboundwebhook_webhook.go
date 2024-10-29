/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

// nolint:unused
// log is for logging in this package.
var outboundwebhooklog = logf.Log.WithName("outboundwebhook-resource")

// SetupOutboundWebhookWebhookWithManager registers the webhook for OutboundWebhook in the manager.
func SetupOutboundWebhookWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralogixv1alpha1.OutboundWebhook{}).
		WithValidator(&OutboundWebhookCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-coralogix-com-v1alpha1-outboundwebhook,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=outboundwebhooks,verbs=create;update,versions=v1alpha1,name=voutboundwebhook-v1alpha1.kb.io,admissionReviewVersions=v1

// OutboundWebhookCustomValidator struct is responsible for validating the OutboundWebhook resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type OutboundWebhookCustomValidator struct{}

var _ webhook.CustomValidator = &OutboundWebhookCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type OutboundWebhook.
func (v *OutboundWebhookCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	outboundWebhook, ok := obj.(*coralogixv1alpha1.OutboundWebhook)
	if !ok {
		return nil, fmt.Errorf("expected a OutboundWebhook object but got %T", obj)
	}
	outboundwebhooklog.Info("Validation for OutboundWebhook upon creation", "name", outboundWebhook.GetName())

	return validateWebhookType(outboundWebhook.Spec.OutboundWebhookType)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type OutboundWebhook.
func (v *OutboundWebhookCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	outboundWebhook, ok := newObj.(*coralogixv1alpha1.OutboundWebhook)
	if !ok {
		return nil, fmt.Errorf("expected a OutboundWebhook object for the newObj but got %T", newObj)
	}
	outboundwebhooklog.Info("Validation for OutboundWebhook upon update", "name", outboundWebhook.GetName())

	return validateWebhookType(outboundWebhook.Spec.OutboundWebhookType)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type OutboundWebhook.
func (v *OutboundWebhookCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateWebhookType(webhookType coralogixv1alpha1.OutboundWebhookType) (admission.Warnings, error) {
	webhookTypes := webhookTypesBeingSet(webhookType)
	if len(webhookTypes) == 0 {
		return admission.Warnings{"at least one webhook type should be set"}, fmt.Errorf("at least one webhook type should be set")
	}

	if len(webhookTypes) > 1 {
		return admission.Warnings{"only one webhook type should be set"}, fmt.Errorf("only one webhook type should be set, but got: %v", webhookTypes)
	}

	return nil, nil
}

func webhookTypesBeingSet(webhookType coralogixv1alpha1.OutboundWebhookType) []string {
	var typesSet []string
	if webhookType.GenericWebhook != nil {
		typesSet = append(typesSet, "GenericWebhook")
	}
	if webhookType.Opsgenie != nil {
		typesSet = append(typesSet, "Opsgenie")
	}
	if webhookType.Slack != nil {
		typesSet = append(typesSet, "Slack")
	}
	if webhookType.SendLog != nil {
		typesSet = append(typesSet, "SendLog")
	}
	if webhookType.EmailGroup != nil {
		typesSet = append(typesSet, "EmailGroup")
	}
	if webhookType.MicrosoftTeams != nil {
		typesSet = append(typesSet, "MicrosoftTeams")
	}
	if webhookType.PagerDuty != nil {
		typesSet = append(typesSet, "PagerDuty")
	}
	if webhookType.Jira != nil {
		typesSet = append(typesSet, "Jira")
	}
	if webhookType.Demisto != nil {
		typesSet = append(typesSet, "Demisto")
	}
	if webhookType.AwsEventBridge != nil {
		typesSet = append(typesSet, "AwsEventBridge")
	}

	return typesSet
}

/*
Copyright 2023.

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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var outboundwebhooklog = logf.Log.WithName("outboundwebhook-resource")

func (r *OutboundWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-coralogix-com-v1alpha1-outboundwebhook,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=outboundwebhooks,verbs=create;update,versions=v1alpha1,name=outboundwebhook.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OutboundWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OutboundWebhook) ValidateCreate() (warnings admission.Warnings, err error) {
	if !atLeastOneWebhookTypeIsSet(r.Spec.OutboundWebhookType) {
		return admission.Warnings{"at least one webhook type should be set"}, fmt.Errorf("at least one webhook type should be set")
	}

	if valid, WebhookTypes := onlyOneWebhookTypeIsSet(r.Spec.OutboundWebhookType); !valid {
		return admission.Warnings{"only one webhook type should be set"}, fmt.Errorf("only one webhook type should be set, but got: %v", WebhookTypes)
	}

	return nil, nil
}

func onlyOneWebhookTypeIsSet(webhookType OutboundWebhookType) (bool, []string) {
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

	if len(typesSet) > 1 {
		return false, typesSet
	}

	return true, typesSet
}

func atLeastOneWebhookTypeIsSet(webhookType OutboundWebhookType) bool {
	if webhookType.GenericWebhook != nil ||
		webhookType.Opsgenie != nil ||
		webhookType.Slack != nil ||
		webhookType.SendLog != nil ||
		webhookType.EmailGroup != nil ||
		webhookType.MicrosoftTeams != nil ||
		webhookType.PagerDuty != nil ||
		webhookType.Jira != nil ||
		webhookType.Demisto != nil ||
		webhookType.AwsEventBridge != nil {
		return true
	}

	return false
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OutboundWebhook) ValidateUpdate(old runtime.Object) (warnings admission.Warnings, err error) {
	if !atLeastOneWebhookTypeIsSet(r.Spec.OutboundWebhookType) {
		return admission.Warnings{"at least one webhook type should be set"}, fmt.Errorf("at least one webhook type should be set")
	}

	if valid, WebhookTypes := onlyOneWebhookTypeIsSet(r.Spec.OutboundWebhookType); !valid {
		return admission.Warnings{"only one webhook type should be set"}, fmt.Errorf("only one webhook type should be set, but got: %v", WebhookTypes)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OutboundWebhook) ValidateDelete() (warnings admission.Warnings, err error) {
	return nil, nil
}
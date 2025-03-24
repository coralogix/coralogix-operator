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

package v1beta1

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
)

// nolint:unused
// log is for logging in this package.
var alertlog = logf.Log.WithName("alert-resource")

// SetupAlertWebhookWithManager registers the webhook for Alert in the manager.
func SetupAlertWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralogixv1beta1.Alert{}).
		WithValidator(&AlertCustomValidator{}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-coralogix-com-v1beta1-alert,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=alerts,verbs=create;update,versions=v1beta1,name=valert-v1beta1.kb.io,admissionReviewVersions=v1

// AlertCustomValidator struct is responsible for validating the Alert resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AlertCustomValidator struct {
}

var _ webhook.CustomValidator = &AlertCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Alert.
func (v *AlertCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	alert, ok := obj.(*coralogixv1beta1.Alert)
	if !ok {
		return nil, fmt.Errorf("expected a Alert object but got %T", obj)
	}
	alertlog.Info("Validation for Alert upon creation", "name", alert.GetName())

	return validateAlert(alert)

}

func validateAlert(alert *coralogixv1beta1.Alert) (admission.Warnings, error) {
	var warnings admission.Warnings
	var errs error

	warns, err := validateAlertType(alert.Spec.TypeDefinition, alert.Spec.GroupByKeys)
	warns = append(warnings, warns...)
	errs = errors.Join(errs, err)

	if typeDef := alert.Spec.TypeDefinition; typeDef.MetricAnomaly == nil && typeDef.MetricThreshold == nil {
		warns, err = validateAlertNotificationGroup(alert.Spec.NotificationGroup, alert.Spec.GroupByKeys)
		warns = append(warnings, warns...)
		errs = errors.Join(errs, err)
	}

	if notificationGroup := alert.Spec.NotificationGroup; notificationGroup != nil {
		if destinations := notificationGroup.Destinations; len(destinations) > 0 {
			warns, err = validateDestinations(destinations)
			warns = append(warnings, warns...)
			errs = errors.Join(errs, err)
		}
	}

	if errs != nil {
		monitoring.IncResourceRejectionsTotalMetric(alert.Kind, alert.Name, alert.Namespace)
	}
	return warns, errs
}

func validateAlertNotificationGroup(group *coralogixv1beta1.NotificationGroup, groupBy []string) (admission.Warnings, error) {
	if group == nil {
		return nil, nil
	}

	if !isSubset(groupBy, group.GroupByKeys) {
		return admission.Warnings{"group by keys must be a subset of the group by keys in the alert type"}, fmt.Errorf("group by keys must be a subset of the group by keys in the alert type")
	}

	var warnings admission.Warnings
	var errs error
	for i, webhook := range group.Webhooks {
		warns, err := validateWebhook(webhook)
		warnings = append(warnings, warns...)
		if err != nil {
			errs = errors.Join(fmt.Errorf("error in webhook %d: %v", i, err))
		}
	}

	return warnings, errs
}

func validateWebhook(webhookSetting coralogixv1beta1.WebhookSettings) (admission.Warnings, error) {
	return validateIntegration(webhookSetting.Integration)
}

func validateIntegration(integration coralogixv1beta1.IntegrationType) (admission.Warnings, error) {
	if integration.IntegrationRef == nil && len(integration.Recipients) == 0 {
		return admission.Warnings{"integration reference or recipients must be set"}, fmt.Errorf("integration reference or recipients must be set")
	}

	if integration.IntegrationRef != nil && len(integration.Recipients) > 0 {
		return admission.Warnings{"only one of integration reference or recipients should be set"}, fmt.Errorf("only one of integration reference or recipients should be set")
	}

	if integration.IntegrationRef != nil {
		return validateIntegrationRef(integration.IntegrationRef)
	}

	return nil, nil
}

func validateIntegrationRef(integrationRef *coralogixv1beta1.IntegrationRef) (admission.Warnings, error) {
	if integrationRef.BackendRef == nil && integrationRef.ResourceRef == nil {
		return admission.Warnings{"backend reference or resource reference must be set"}, fmt.Errorf("backend reference or resource reference must be set")
	}

	if integrationRef.BackendRef != nil && integrationRef.ResourceRef != nil {
		return admission.Warnings{"only one of backend reference or resource reference should be set"}, fmt.Errorf("only one of backend reference or resource reference should be set")
	}

	if integrationRef.BackendRef != nil {
		return validateWebhookBackendRef(integrationRef.BackendRef)
	}

	return nil, nil
}

func validateWebhookBackendRef(ref *coralogixv1beta1.OutboundWebhookBackendRef) (admission.Warnings, error) {
	if ref.Name == nil && ref.ID == nil {
		return admission.Warnings{"name or ID must be set"}, fmt.Errorf("name or ID must be set")
	}

	if ref.Name != nil && ref.ID != nil {
		return admission.Warnings{"only one of name or ID should be set"}, fmt.Errorf("only one of name or ID should be set")
	}

	return nil, nil
}

func validateDestinations(destinations []coralogixv1beta1.Destination) (admission.Warnings, error) {
	var warnings admission.Warnings
	var errs error
	for i, destination := range destinations {
		warns, err := validateDestination(destination)
		if err != nil {
			warnings = append(warnings, warns...)
			errs = errors.Join(fmt.Errorf("error in destination %d: %v", i, err.Error()))
		}
	}

	return warnings, errs
}

func validateDestination(destination coralogixv1beta1.Destination) (admission.Warnings, error) {
	var warnings admission.Warnings
	var errs error
	if destination.DestinationType == nil {
		warnings = append(warnings, "destination type must be set")
		errs = errors.Join(errs, fmt.Errorf("destination type must be set"))
	}

	if destination.DestinationType.Slack == nil && destination.DestinationType.GenericHttps == nil {
		warnings = append(warnings, "destination type must be set")
		errs = errors.Join(errs, fmt.Errorf("destination type must be set"))
	}

	if destination.DestinationType.Slack != nil && destination.DestinationType.GenericHttps != nil {
		warnings = append(warnings, "only one destination type should be set")
		errs = errors.Join(errs, fmt.Errorf("only one destination type should be set"))
	}

	if slack := destination.DestinationType.Slack; slack != nil {
		if slack.ConnectorRef != nil {
			if slack.ConnectorRef.ResourceRef != nil && slack.ConnectorRef.BackendRef != nil {
				warnings = append(warnings, "only one of resource reference or backend reference should be set")
				errs = errors.Join(errs, fmt.Errorf("only one of resource reference or backend reference should be set"))
			}
		}

		if slack.PresetRef != nil {
			if slack.PresetRef.ResourceRef != nil && slack.PresetRef.BackendRef != nil {
				warnings = append(warnings, "only one of resource reference or backend reference should be set")
				errs = errors.Join(errs, fmt.Errorf("only one of resource reference or backend reference should be set"))
			}
		}
	}
	if genericHttps := destination.DestinationType.GenericHttps; genericHttps != nil {
		if genericHttps.ConnectorRef != nil {
			if genericHttps.ConnectorRef.ResourceRef != nil && genericHttps.ConnectorRef.BackendRef != nil {
				warnings = append(warnings, "only one of resource reference or backend reference should be set")
				errs = errors.Join(errs, fmt.Errorf("only one of resource reference or backend reference should be set"))
			}
		}

		if genericHttps.PresetRef != nil {
			if genericHttps.PresetRef.ResourceRef != nil && genericHttps.PresetRef.BackendRef != nil {
				warnings = append(warnings, "only one of resource reference or backend reference should be set")
				errs = errors.Join(errs, fmt.Errorf("only one of resource reference or backend reference should be set"))
			}
		}
	}

	return warnings, errs
}

func isSubset(mainArray, subArray []string) bool {
	// Create a map to store elements of mainArray
	elementMap := make(map[string]bool)
	for _, elem := range mainArray {
		elementMap[elem] = true
	}

	// Check if every element in subArray exists in the map
	for _, elem := range subArray {
		if !elementMap[elem] {
			return false
		}
	}

	return true
}

func validateAlertType(alertType coralogixv1beta1.AlertTypeDefinition, groupBy []string) (admission.Warnings, error) {
	alertTypes := alertTypesBeingSet(alertType)
	if len(alertTypes) == 0 {
		return admission.Warnings{"no alert type is set"}, fmt.Errorf("no alert type is set")
	}

	if len(alertTypes) > 1 {
		return admission.Warnings{"only one alert type should be set"}, fmt.Errorf("only one alert type should be set, but got: %v", alertTypes)
	}

	switch {
	case alertType.LogsImmediate != nil:
		if len(groupBy) > 0 {
			return admission.Warnings{"group by is not supported for LogsImmediate alert type"}, fmt.Errorf("group by is not supported for LogsImmediate alert type")
		}
	case alertType.TracingImmediate != nil:
		if len(groupBy) > 0 {
			return admission.Warnings{"group by is not supported for TracingImmediate alert type"}, fmt.Errorf("group by is not supported for TracingImmediate alert type")
		}
	case alertType.MetricThreshold != nil:
		return validateMetricThreshold(alertType.MetricThreshold)
	case alertType.Flow != nil:
		return validateFlow(alertType.Flow)
	}

	return nil, nil
}

func validateMetricThreshold(metricThreshold *coralogixv1beta1.MetricThreshold) (admission.Warnings, error) {
	return validateMissingValues(metricThreshold.MissingValues)
}

func validateMissingValues(missingValues coralogixv1beta1.MetricMissingValues) (admission.Warnings, error) {
	if !missingValues.ReplaceWithZero && missingValues.MinNonNullValuesPct == nil {
		return admission.Warnings{"missingValues.minNonNullValuesPct is required when missingValues.replaceWithZero is false"}, fmt.Errorf("missingValues.minNonNullValuesPct is required when missingValues.replaceWithZero is false")
	} else if missingValues.ReplaceWithZero && missingValues.MinNonNullValuesPct != nil {
		return admission.Warnings{"missingValues.minNonNullValuesPct should not be set when missingValues.replaceWithZero is true"}, fmt.Errorf("missingValues.minNonNullValuesPct should not be set when missingValues.replaceWithZero is true")
	}
	return nil, nil
}

func validateFlow(flow *coralogixv1beta1.Flow) (admission.Warnings, error) {
	var warnings admission.Warnings
	var errs error
	for i, stage := range flow.Stages {
		warns, err := validateFlowStage(stage)
		warnings = append(warnings, warns...)
		if err != nil {
			errs = errors.Join(fmt.Errorf("error in stage %d: %v", i, err))
		}
	}

	return warnings, errs
}

func validateFlowStage(stage coralogixv1beta1.FlowStage) (admission.Warnings, error) {
	var warnings admission.Warnings
	var errs error
	for i, group := range stage.FlowStagesType.Groups {
		warns, err := validateFlowGroup(group)
		warnings = append(warnings, warns...)
		if err != nil {
			errs = errors.Join(fmt.Errorf("error in group %d: %v", i, err))
		}
	}

	return warnings, errs
}

func validateFlowGroup(group coralogixv1beta1.FlowStageGroup) (admission.Warnings, error) {
	var warnings admission.Warnings
	var errs error
	for i, alertDef := range group.AlertDefs {
		warns, err := validateAlertDefinition(alertDef)
		warnings = append(warnings, warns...)
		if err != nil {
			errs = errors.Join(fmt.Errorf("error in alert definition %d: %v", i, err))
		}
	}

	return warnings, errs
}

func validateAlertDefinition(alertDef coralogixv1beta1.FlowStagesGroupsAlertDefs) (admission.Warnings, error) {
	return validateAlertRefType(alertDef.AlertRef)
}

func validateAlertRefType(ref coralogixv1beta1.AlertRef) (admission.Warnings, error) {
	if ref.ResourceRef == nil && ref.BackendRef == nil {
		return admission.Warnings{"resource reference or backend reference must be set"}, fmt.Errorf("resource reference or backend reference must be set")
	}

	if ref.ResourceRef != nil && ref.BackendRef != nil {
		return admission.Warnings{"only one of resource reference or backend reference should be set"}, fmt.Errorf("only one of resource reference or backend reference should be set")
	}

	if ref.BackendRef != nil {
		return validateAlertBackendRef(ref.BackendRef)
	}

	return nil, nil
}

func validateAlertBackendRef(ref *coralogixv1beta1.AlertBackendRef) (admission.Warnings, error) {
	if ref.Name == nil && ref.ID == nil {
		return admission.Warnings{"name or ID must be set"}, fmt.Errorf("name or ID must be set")
	}

	if ref.Name != nil && ref.ID != nil {
		return admission.Warnings{"only one of name or ID should be set"}, fmt.Errorf("only one of name or ID should be set")
	}

	return nil, nil
}

func alertTypesBeingSet(alertType coralogixv1beta1.AlertTypeDefinition) []string {
	var typesSet []string
	if alertType.LogsThreshold != nil {
		typesSet = append(typesSet, "LogsThreshold")
	}
	if alertType.LogsImmediate != nil {
		typesSet = append(typesSet, "LogsImmediate")
	}
	if alertType.LogsNewValue != nil {
		typesSet = append(typesSet, "LogsNewValue")
	}
	if alertType.LogsAnomaly != nil {
		typesSet = append(typesSet, "LogsAnomaly")
	}
	if alertType.LogsTimeRelativeThreshold != nil {
		typesSet = append(typesSet, "LogsTimeRelativeThreshold")
	}
	if alertType.LogsRatioThreshold != nil {
		typesSet = append(typesSet, "LogsRatioThreshold")
	}
	if alertType.LogsUniqueCount != nil {
		typesSet = append(typesSet, "LogsUniqueCount")
	}
	if alertType.TracingImmediate != nil {
		typesSet = append(typesSet, "TracingImmediate")
	}
	if alertType.TracingThreshold != nil {
		typesSet = append(typesSet, "TracingThreshold")
	}
	if alertType.Flow != nil {
		typesSet = append(typesSet, "Flow")
	}
	if alertType.MetricThreshold != nil {
		typesSet = append(typesSet, "MetricThreshold")
	}
	if alertType.MetricAnomaly != nil {
		typesSet = append(typesSet, "MetricAnomaly")
	}

	return typesSet
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Alert.
func (v *AlertCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	alert, ok := newObj.(*coralogixv1beta1.Alert)
	if !ok {
		return nil, fmt.Errorf("expected a Alert object for the newObj but got %T", newObj)
	}
	alertlog.Info("Validation for Alert upon update", "name", alert.GetName())

	return validateAlert(alert)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Alert.
func (v *AlertCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

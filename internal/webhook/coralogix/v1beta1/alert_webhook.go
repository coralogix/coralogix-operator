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

package v1beta1

import (
	"context"
	"errors"
	"fmt"

	"github.com/coralogix/coralogix-operator/internal/monitoring"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
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

	var warnings admission.Warnings
	var errs error

	warns, err := validateAlertNotificationGroup(alert.Spec.NotificationGroup, alert.Spec.GroupByKeys)
	warns = append(warnings, warns...)
	errs = errors.Join(errs, err)

	warns, err = validateAlertType(alert.Spec.TypeDefinition, alert.Spec.GroupByKeys)
	warns = append(warnings, warns...)
	errs = errors.Join(errs, err)

	if errs != nil {
		monitoring.TotalRejectedAlertsMetric.Inc()
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
		return validateLogsImmediate(alertType.LogsImmediate)
	case alertType.LogsThreshold != nil:
		return validateLogsThreshold(alertType.LogsThreshold)
	case alertType.LogsNewValue != nil:
		return validateLogsNewValue(alertType.LogsNewValue)
	case alertType.LogsAnomaly != nil:
		return validateLogsAnomaly(alertType.LogsAnomaly)
	case alertType.LogsTimeRelativeThreshold != nil:
		return validateLogsTimeRelativeThreshold(alertType.LogsTimeRelativeThreshold)
	case alertType.LogsRatioThreshold != nil:
		return validateLogsRatioThreshold(alertType.LogsRatioThreshold)
	case alertType.LogsUniqueCount != nil:
		return validateLogsUniqueCount(alertType.LogsUniqueCount)
	case alertType.TracingImmediate != nil:
		if len(groupBy) > 0 {
			return admission.Warnings{"group by is not supported for TracingImmediate alert type"}, fmt.Errorf("group by is not supported for TracingImmediate alert type")
		}
	case alertType.TracingThreshold != nil:
		return validateTracingThreshold(alertType.TracingThreshold)
	case alertType.Flow != nil:
		return validateFlow(alertType.Flow)
	case alertType.MetricThreshold != nil:
		return validateMetricThreshold(alertType.MetricThreshold)
	case alertType.MetricAnomaly != nil:
		return validateMetricAnomaly(alertType.MetricAnomaly)
	}

	return nil, nil
}

func validateMetricAnomaly(anomaly *coralogixv1beta1.MetricAnomaly) (admission.Warnings, error) {
	return nil, nil
}

func validateMetricThreshold(threshold *coralogixv1beta1.MetricThreshold) (admission.Warnings, error) {
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

func validateTracingThreshold(threshold *coralogixv1beta1.TracingThreshold) (admission.Warnings, error) {
	return nil, nil
}

func validateTracingImmediate(immediate *coralogixv1beta1.TracingImmediate) (admission.Warnings, error) {
	return nil, nil
}

func validateLogsUniqueCount(count *coralogixv1beta1.LogsUniqueCount) (admission.Warnings, error) {
	return nil, nil
}

func validateLogsRatioThreshold(threshold *coralogixv1beta1.LogsRatioThreshold) (admission.Warnings, error) {
	return nil, nil
}

func validateLogsTimeRelativeThreshold(threshold *coralogixv1beta1.LogsTimeRelativeThreshold) (admission.Warnings, error) {
	return nil, nil
}

func validateLogsAnomaly(anomaly *coralogixv1beta1.LogsAnomaly) (admission.Warnings, error) {
	return nil, nil
}

func validateLogsNewValue(value *coralogixv1beta1.LogsNewValue) (admission.Warnings, error) {
	return nil, nil
}

func validateLogsThreshold(threshold *coralogixv1beta1.LogsThreshold) (admission.Warnings, error) {
	return nil, nil
}

func validateLogsImmediate(logsImmediate *coralogixv1beta1.LogsImmediate) (admission.Warnings, error) {
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

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Alert.
func (v *AlertCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	alert, ok := obj.(*coralogixv1beta1.Alert)
	if !ok {
		return nil, fmt.Errorf("expected a Alert object but got %T", obj)
	}
	alertlog.Info("Validation for Alert upon deletion", "name", alert.GetName())

	return nil, nil
}

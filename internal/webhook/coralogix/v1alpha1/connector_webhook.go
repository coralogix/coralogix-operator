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

var connectorlog = logf.Log.WithName("connector-resource")

// SetupConnectorWebhookWithManager registers the webhook for Connector in the manager.
func SetupConnectorWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralogixv1alpha1.Connector{}).
		WithValidator(&ConnectorCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-coralogix-com-v1alpha1-connector,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=connectors,verbs=create;update,versions=v1alpha1,name=vconnector-v1alpha1.kb.io,admissionReviewVersions=v1

// ConnectorCustomValidator struct is responsible for validating the Connector resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ConnectorCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &ConnectorCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Connector.
func (v *ConnectorCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	connector, ok := obj.(*coralogixv1alpha1.Connector)
	if !ok {
		return nil, fmt.Errorf("expected a Connector object but got %T", obj)
	}
	connectorlog.Info("Validation for Connector upon creation", "name", connector.GetName())

	err := validateConnectorType(connector.Spec.ConnectorType)
	if err != nil {
		return admission.Warnings{err.Error()}, fmt.Errorf("validation failed: %v", err)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Connector.
func (v *ConnectorCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	connector, ok := newObj.(*coralogixv1alpha1.Connector)
	if !ok {
		return nil, fmt.Errorf("expected a Connector object for the newObj but got %T", newObj)
	}
	connectorlog.Info("Validation for Connector upon update", "name", connector.GetName())

	err := validateConnectorType(connector.Spec.ConnectorType)
	if err != nil {
		return admission.Warnings{err.Error()}, fmt.Errorf("validation failed: %v", err)
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Connector.
func (v *ConnectorCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateConnectorType(connectorType *coralogixv1alpha1.ConnectorType) error {
	if connectorType == nil {
		return fmt.Errorf("connector type should be set")
	}

	var typesSet []string
	if connectorType.GenericHttps != nil {
		typesSet = append(typesSet, "GenericHttps")
	}
	if connectorType.Slack != nil {
		typesSet = append(typesSet, "Slack")
	}

	if len(typesSet) > 1 {
		return fmt.Errorf("only one connector type should be set, got: %v", typesSet)
	}

	return nil
}

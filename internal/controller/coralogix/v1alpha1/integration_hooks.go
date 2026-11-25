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

	"github.com/go-logr/logr"

	integrations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/integration_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The hooks read spec.parameters and spec.parametersFromSecret, resolve the Secret references,
// and write the parameter list of the request. Each run builds the whole list, so a hook that
// runs again sends the same body.

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get

func (r *IntegrationReconciler) beforeCreate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.Integration, body *integrations.SaveIntegrationRequest) error {
	parameters, err := cr.Spec.ExtractParameters(ctx, cr.Namespace)
	if err != nil {
		return fmt.Errorf("failed to extract parameters: %w", err)
	}
	body.Metadata.IntegrationParameters.Parameters = parameters
	return nil
}

func (r *IntegrationReconciler) beforeUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.Integration, body *integrations.UpdateIntegrationRequest) error {
	parameters, err := cr.Spec.ExtractParameters(ctx, cr.Namespace)
	if err != nil {
		return fmt.Errorf("failed to extract parameters: %w", err)
	}
	body.Metadata.IntegrationParameters.Parameters = parameters
	return nil
}

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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	connectors "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/connectors_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ConnectorReconciler reconciles a Connector object
type ConnectorReconciler struct {
	ConnectorsClient *connectors.ConnectorsServiceAPIService
	Interval         time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=connectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=connectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=connectors/finalizers,verbs=update

func (r *ConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Connector{}, r)
}

func (r *ConnectorReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *ConnectorReconciler) FinalizerName() string {
	return "connector.coralogix.com/finalizer"
}

func (r *ConnectorReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	connector := obj.(*coralogixv1alpha1.Connector)
	createRequest, err := connector.ExtractConnector()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}

	log.Info("Creating remote Connector", "Connector", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.ConnectorsClient.
		ConnectorsServiceCreateConnector(ctx).
		Connector1(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote Connector: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote connector created", "response", utils.FormatJSON(createResponse))

	connector.Status = coralogixv1alpha1.ConnectorStatus{
		Id: createResponse.Connector.Id,
	}

	return nil
}

func (r *ConnectorReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	connector := obj.(*coralogixv1alpha1.Connector)
	updateRequest, err := connector.ExtractConnector()
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	updateRequest.Id = connector.Status.Id
	log.Info("Updating remote Connector", "Connector", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.ConnectorsClient.
		ConnectorsServiceReplaceConnector(ctx).
		Connector1(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote Connector updated", "Connector", utils.FormatJSON(updateResponse))

	return nil
}

func (r *ConnectorReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	connector := obj.(*coralogixv1alpha1.Connector)
	log.Info("Deleting Connector from remote system", "id", *connector.Status.Id)
	_, httpResp, err := r.ConnectorsClient.
		ConnectorsServiceDeleteConnector(ctx, ptr.Deref(connector.Status.Id, "")).
		Execute()

	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(apiErr, "Error deleting remote Connector", "id", *connector.Status.Id)
			return fmt.Errorf("error deleting remote Connector %s: %w", *connector.Status.Id, apiErr)
		}
	}

	log.Info("Connector deleted from remote system", "id", *connector.Status.Id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Connector{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}

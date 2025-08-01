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
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// ConnectorReconciler reconciles a Connector object
type ConnectorReconciler struct {
	NotificationsClient *cxsdk.NotificationsClient
	Interval            time.Duration
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
	createRequest, err := connector.ExtractCreateConnectorRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}

	log.Info("Creating remote Connector", "Connector", protojson.Format(createRequest))
	createResponse, err := r.NotificationsClient.CreateConnector(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote Connector: %w", err)
	}
	log.Info("Remote connector created", "response", protojson.Format(createResponse))

	connector.Status = coralogixv1alpha1.ConnectorStatus{
		Id: createResponse.Connector.Id,
	}

	return nil
}

func (r *ConnectorReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	connector := obj.(*coralogixv1alpha1.Connector)
	updateRequest, err := connector.ExtractUpdateConnectorRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote Connector", "Connector", protojson.Format(updateRequest))
	updateResponse, err := r.NotificationsClient.ReplaceConnector(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote Connector updated", "Connector", protojson.Format(updateResponse))

	return nil
}

func (r *ConnectorReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	connector := obj.(*coralogixv1alpha1.Connector)
	log.Info("Deleting Connector from remote system", "id", *connector.Status.Id)
	_, err := r.NotificationsClient.DeleteConnector(ctx,
		&cxsdk.DeleteConnectorRequest{
			Id: ptr.Deref(connector.Status.Id, ""),
		})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote Connector", "id", *connector.Status.Id)
		return fmt.Errorf("error deleting remote Connector %s: %w", *connector.Status.Id, err)
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

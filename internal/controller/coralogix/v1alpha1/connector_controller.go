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
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ConnectorReconciler reconciles a Connector object
type ConnectorReconciler struct {
	client.Client
	NotificationsClient *cxsdk.NotificationsClient
	Scheme              *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=connectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=connectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=connectors/finalizers,verbs=update

var (
	connectorFinalizerName = "connector.coralogix.com/finalizer"
)

func (r *ConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"connector", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	connector := &coralogixv1alpha1.Connector{}
	if err := r.Client.Get(ctx, req.NamespacedName, connector); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(connector.Status.Id, "") == "" {
		err := r.create(ctx, log, connector)
		if err != nil {
			log.Error(err, "Error on creating Connector")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !connector.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, connector)
		if err != nil {
			log.Error(err, "Error on deleting Connector")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(connector.GetLabels()) {
		err := r.deleteRemoteConnector(ctx, log, connector.Status.Id)
		if err != nil {
			log.Error(err, "Error on deleting Connector")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, connector)
	if err != nil {
		log.Error(err, "Error on updating Connector")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *ConnectorReconciler) create(ctx context.Context, log logr.Logger, connector *coralogixv1alpha1.Connector) error {
	createReq := connector.Spec.ExtractCreateConnectorRequest()
	log.V(1).Info("Creating remote connector", "connector", protojson.Format(createReq))
	createRes, err := r.NotificationsClient.CreateConnector(ctx, createReq)
	if err != nil {
		return fmt.Errorf("error on creating remote connector: %w", err)
	}
	log.V(1).Info("Remote connector created", "response", protojson.Format(createRes))

	connector.Status = coralogixv1alpha1.ConnectorStatus{
		Id: createRes.Connector.Id,
	}

	log.V(1).Info("Updating Connector status", "id", createRes.Connector.Id)
	if err = r.Status().Update(ctx, connector); err != nil {
		if err := r.deleteRemoteConnector(ctx, log, connector.Status.Id); err != nil {
			return fmt.Errorf("error to delete connector after status update error -\n%v", connector)
		}
		return fmt.Errorf("error to update connector status -\n%v", connector)
	}

	if !controllerutil.ContainsFinalizer(connector, connectorFinalizerName) {
		log.V(1).Info("Updating Connector to add finalizer", "id", createRes.Connector.Id)
		controllerutil.AddFinalizer(connector, connectorFinalizerName)
		if err := r.Update(ctx, connector); err != nil {
			return fmt.Errorf("error on updating Connector: %w", err)
		}
	}

	return nil
}

func (r *ConnectorReconciler) update(ctx context.Context, log logr.Logger, connector *coralogixv1alpha1.Connector) error {
	replaceReq := connector.Spec.ExtractReplaceConnectorRequest(connector.Status.Id)
	log.V(1).Info("Updating remote connector", "connector", protojson.Format(replaceReq))
	replaceRes, err := r.NotificationsClient.ReplaceConnector(ctx, replaceReq)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("connector not found on remote, removing id from status")
			connector.Status = coralogixv1alpha1.ConnectorStatus{
				Id: ptr.To(""),
			}
			if err = r.Status().Update(ctx, connector); err != nil {
				return fmt.Errorf("error on updating Connector status: %w", err)
			}
			return fmt.Errorf("connector not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating connector: %w", err)
	}
	log.V(1).Info("Remote connector updated", "connector", protojson.Format(replaceRes))

	return nil
}

func (r *ConnectorReconciler) delete(ctx context.Context, log logr.Logger, connector *coralogixv1alpha1.Connector) error {
	if err := r.deleteRemoteConnector(ctx, log, connector.Status.Id); err != nil {
		return fmt.Errorf("error on deleting remote connector: %w", err)
	}

	log.V(1).Info("Removing finalizer from Connector")
	controllerutil.RemoveFinalizer(connector, connectorFinalizerName)
	if err := r.Update(ctx, connector); err != nil {
		return fmt.Errorf("error on updating Connector: %w", err)
	}

	return nil
}

func (r *ConnectorReconciler) deleteRemoteConnector(ctx context.Context, log logr.Logger, id *string) error {
	log.V(1).Info("Deleting remote connector", "id", *id)
	_, err := r.NotificationsClient.DeleteConnector(ctx, &cxsdk.DeleteConnectorRequest{
		Id: *id,
	})
	if err != nil {
		return fmt.Errorf("error on deleting remote connector: %w", err)
	}
	log.V(1).Info("Remote connector deleted", "id", *id)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Connector{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}

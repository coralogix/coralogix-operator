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

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ipaccess "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ip_access_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// IPAccessReconciler reconciles a IPAccess object
type IPAccessReconciler struct {
	IPAccesssClient *ipaccess.IPAccessServiceAPIService
	Interval        time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=ipaccesses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=ipaccesses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=ipaccesses/finalizers,verbs=update

func (r *IPAccessReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.IPAccess{}, r)
}

func (r *IPAccessReconciler) FinalizerName() string {
	return "ip-access.coralogix.com/finalizer"
}

func (r *IPAccessReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *IPAccessReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	ipAccess := obj.(*coralogixv1alpha1.IPAccess)
	createReq, err := ipAccess.ExtractCreateIPAccessRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	log.Info("Creating remote ipAccess", "ipAccess", utils.FormatJSON(createReq))
	createRes, httpResp, err := r.IPAccesssClient.
		IpAccessServiceCreateCompanyIpAccessSettings(ctx).
		CreateCompanyIPAccessSettingsRequest(*createReq).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote ipAccess: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote ipAccess created", "response", utils.FormatJSON(createRes))
	ipAccess.Status = coralogixv1alpha1.IPAccessStatus{
		ID: createRes.Settings.Id,
	}

	return nil
}

func (r *IPAccessReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	ipAccess := obj.(*coralogixv1alpha1.IPAccess)
	replaceReq, err := ipAccess.ExtractReplaceIPAccessRequest()
	if err != nil {
		return fmt.Errorf("error on extracting replace request: %w", err)
	}
	log.Info("Replacing remote ipAccess", "ipAccess", utils.FormatJSON(replaceReq))
	replaceRes, httpResp, err := r.IPAccesssClient.
		IpAccessServiceReplaceCompanyIpAccessSettings(ctx).
		ReplaceCompanyIPAccessSettingsRequest(*replaceReq).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote ipAccess replaces", "response", utils.FormatJSON(replaceRes))

	return nil
}

func (r *IPAccessReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	ipAccess := obj.(*coralogixv1alpha1.IPAccess)
	id := ipAccess.Status.ID
	if id == nil {
		log.Info("IPAccess ID is nil, skipping deletion in remote system")
		return nil
	}

	log.Info("Deleting ipAccess from remote system")
	_, httpResp, err := r.IPAccesssClient.
		IpAccessServiceDeleteCompanyIpAccessSettings(ctx).
		Id(*id).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote IpAccess", "id", *ipAccess.Status.ID)
			return fmt.Errorf("error deleting remote IpAccess %s: %w", *ipAccess.Status.ID, apiErr)
		}
	}
	log.Info("IPAccess deleted from remote system")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPAccessReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.IPAccess{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}

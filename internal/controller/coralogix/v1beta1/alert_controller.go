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
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// AlertReconciler reconciles a Alert object
type AlertReconciler struct {
	ClientSet *cxsdk.ClientSet
	Interval  time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=alerts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=alerts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=alerts/finalizers,verbs=update

func (r *AlertReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1beta1.Alert{}, r)
}

func (r *AlertReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *AlertReconciler) FinalizerName() string {
	return "alert.coralogix.com/finalizer"
}

func (r *AlertReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	alert := obj.(*coralogixv1beta1.Alert)
	props, err := alert.Spec.ExtractAlertDefProperties(
		&coralogixv1beta1.GetResourceRefProperties{
			Ctx:       ctx,
			Log:       log,
			ClientSet: r.ClientSet,
			Namespace: alert.Namespace,
		},
	)
	if err != nil {
		return fmt.Errorf("error on extracting alert properties: %w", err)
	}

	createRequest := &alerts.CreateAlertDefinitionRequest{
		AlertDefProperties: props,
		AccessPolicy:       alert.Spec.ExtractCreateAccessPolicy(),
	}

	log.Info("Creating remote alert", "alert", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.ClientSet.Alerts().
		AlertDefsServiceCreateAlertDef(ctx).
		CreateAlertDefinitionRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote alert: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote alert created", "response", utils.FormatJSON(createResponse))
	alert.Status = coralogixv1beta1.AlertStatus{ID: createResponse.AlertDef.Id, AccessPolicyManaged: createRequest.AccessPolicy != nil}
	if createRequest.AccessPolicy != nil {
		setAccessPolicyManagedAnnotation(ctx, alert, createRequest.AccessPolicy)
	}
	return nil
}

func (r *AlertReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	alert := obj.(*coralogixv1beta1.Alert)
	props, err := alert.Spec.ExtractAlertDefProperties(
		&coralogixv1beta1.GetResourceRefProperties{
			Ctx:       ctx,
			Log:       log,
			ClientSet: r.ClientSet,
			Namespace: alert.Namespace,
		},
	)
	if err != nil {
		return fmt.Errorf("error on extracting alert properties: %w", err)
	}

	if alert.Status.ID == nil {
		return fmt.Errorf("alert ID is missing")
	}

	policyAction := alert.Spec.ExtractAccessPolicyAction()
	wroteAhead := false
	if policyAction != nil && policyAction.Upsert != nil {
		// Record the policy before sending it: once applied, a policy that denies the operator's API key
		// makes the alert look deleted, and the record must already exist to stop a recreate.
		if !alert.Status.AccessPolicyManaged {
			wroteAhead = true
			alert.Status.AccessPolicyManaged = true
			if err := config.GetClient().Status().Update(ctx, alert); err != nil {
				return fmt.Errorf("error on recording the alert access policy in status: %w", err)
			}
		}
		// On every upsert, not only the first: a failed annotation write is retried here, and the
		// call is a no-op once the annotation holds the desired policy.
		setAccessPolicyManagedAnnotation(ctx, alert, policyAction.Upsert)
	}

	updateRequest := alerts.ReplaceAlertDefinitionRequest{
		AlertDefProperties: props,
		Id:                 alert.Status.ID,
		AccessPolicy:       policyAction,
	}

	log.Info("Updating remote alert", "alert", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.ClientSet.Alerts().
		AlertDefsServiceReplaceAlertDef(ctx).
		ReplaceAlertDefinitionRequest(updateRequest).
		Execute()
	if err != nil {
		apiErr := cxsdk.NewAPIError(httpResp, err)
		if cxsdk.IsNotFound(apiErr) && alert.Spec.AccessPolicyMayDenyOperator(alert.Status.AccessPolicyManaged, alert.Annotations) {
			// Not wrapped on purpose: the shared reconciler drops status.id on a wrapped not-found error
			// and recreates the alert, which would duplicate an alert whose policy locks the operator out.
			return fmt.Errorf("remote alert %s was not found, or its access policy denies the operator's API key; "+
				"not recreating it. If it was deleted, remove status.id to recreate it; if it is locked, "+
				"recreating leaves it orphaned: %s", *alert.Status.ID, apiErr.Error())
		}
		if wroteAhead && rejectedBeforeApplying(httpResp) {
			// The request was refused, so the policy recorded ahead of it was never applied. Keeping
			// the record would make a later genuine deletion look like a lockout.
			// Persisted here: the shared reconciler writes status after an error only when the
			// condition changes.
			alert.Status.AccessPolicyManaged = false
			if clearErr := clearAccessPolicyManagedAnnotation(ctx, alert); clearErr != nil {
				return errors.Join(apiErr, clearErr)
			}
			if statusErr := config.GetClient().Status().Update(ctx, alert); statusErr != nil {
				return errors.Join(apiErr, fmt.Errorf("error on rolling back the alert access policy record: %w", statusErr))
			}
		}
		return apiErr
	}
	log.Info("Remote alert updated", "alert", utils.FormatJSON(updateResponse))
	if policyAction != nil && policyAction.Delete != nil {
		alert.Status.AccessPolicyManaged = false
		if err := clearAccessPolicyManagedAnnotation(ctx, alert); err != nil {
			return err
		}
	}
	return nil
}

// rejectedBeforeApplying reports whether a failed replace provably changed nothing remotely: a
// client error other than not-found. A not-found may hide a policy lockout, and a server error
// may follow a partial apply (the API also answers some invalid policies with 500).
func rejectedBeforeApplying(resp *http.Response) bool {
	return resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusNotFound
}

// clearAccessPolicyManagedAnnotation removes the policy record from the CR metadata through
// updateAlertMetadata, so the reconciler's following writes do not conflict.
func clearAccessPolicyManagedAnnotation(ctx context.Context, alert *coralogixv1beta1.Alert) error {
	delete(alert.Annotations, coralogixv1beta1.AccessPolicyManagedAnnotation)
	err := updateAlertMetadata(ctx, alert, func(fresh *coralogixv1beta1.Alert) bool {
		if _, recorded := fresh.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation]; !recorded {
			return false
		}
		delete(fresh.Annotations, coralogixv1beta1.AccessPolicyManagedAnnotation)
		return true
	})
	if err != nil {
		return fmt.Errorf("error on clearing the alert access policy record: %w", err)
	}
	return nil
}

// setAccessPolicyManagedAnnotation records the applied policy durably on the CR metadata.
// Unlike status.accessPolicyManaged, the annotation survives a rolling upgrade's old-binary
// writes (their typed client preserves annotations but drops the unknown status field), so the
// not-found guard keeps protecting the alert across the upgrade window. A failed annotation
// write is logged but does not fail the reconcile: the status tombstone still covers the
// current version, and the annotation is re-set on the next policy-applying reconcile.
func setAccessPolicyManagedAnnotation(ctx context.Context, alert *coralogixv1beta1.Alert, policy *string) {
	value := ptr.Deref(policy, "")
	err := updateAlertMetadata(ctx, alert, func(fresh *coralogixv1beta1.Alert) bool {
		if current, recorded := fresh.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation]; recorded && current == value {
			return false
		}
		if fresh.Annotations == nil {
			fresh.Annotations = map[string]string{}
		}
		fresh.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation] = value
		return true
	})
	if err != nil {
		log.FromContext(ctx).Error(err, "Error recording the alert access policy annotation")
		return
	}
	if alert.Annotations == nil {
		alert.Annotations = map[string]string{}
	}
	alert.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation] = value
}

// updateAlertMetadata applies mutate to a fresh copy of the alert and writes it, retrying on
// conflict; mutate returns false when no write is needed. The write runs on a fresh copy because
// a full-object Update never carries status (a subresource), and alert may hold status the
// reconciler has not persisted yet (on create, the new remote ID). After a write, alert adopts the
// written metadata and spec but keeps its in-memory status: the reconciler persists that status
// next, and a stale resource version would make the write conflict and lose it, while taking
// only the new resource version would let its later full-object writes revert concurrent edits.
func updateAlertMetadata(ctx context.Context, alert *coralogixv1beta1.Alert, mutate func(*coralogixv1beta1.Alert) bool) error {
	var written *coralogixv1beta1.Alert
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		fresh := &coralogixv1beta1.Alert{}
		if err := config.GetClient().Get(ctx, client.ObjectKeyFromObject(alert), fresh); err != nil {
			return err
		}
		if !mutate(fresh) {
			return nil
		}
		if err := config.GetClient().Update(ctx, fresh); err != nil {
			return err
		}
		written = fresh
		return nil
	})
	if err != nil || written == nil {
		return err
	}
	status := alert.Status
	written.ObjectMeta.DeepCopyInto(&alert.ObjectMeta)
	written.Spec.DeepCopyInto(&alert.Spec)
	alert.Status = status
	return nil
}

func (r *AlertReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	alert := obj.(*coralogixv1beta1.Alert)
	if alert.Status.ID == nil {
		return fmt.Errorf("alert ID is missing")
	}
	log.Info("Deleting alert from remote system", "id", *alert.Status.ID)
	_, httpResp, err := r.ClientSet.Alerts().
		AlertDefsServiceDeleteAlertDef(ctx, *alert.Status.ID).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote alert", "id", *alert.Status.ID)
			return fmt.Errorf("error deleting remote alert %s: %w", *alert.Status.ID, err)
		}
		if alert.Spec.AccessPolicyMayDenyOperator(alert.Status.AccessPolicyManaged, alert.Annotations) {
			log.Info("Remote alert not found; if its access policy denied the operator's API key, it still exists remotely and is now orphaned", "id", *alert.Status.ID)
		}
	}
	log.Info("Alert deleted from remote system", "id", *alert.Status.ID)
	// The remote alert is gone, so its policy record goes too. This matters when the CR stops
	// matching the selector and stays: a later recreate must not inherit a stale record.
	return clearAccessPolicyManagedAnnotation(ctx, alert)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1beta1.Alert{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}

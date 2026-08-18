// Copyright 2026 Coralogix Ltd.
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
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/monitoring"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

const alertSetFinalizer = "alertset.coralogix.com/finalizer"

// AlertSetReconciler reconciles an AlertSet object.
type AlertSetReconciler struct {
	ClientSet *cxsdk.ClientSet
	Interval  time.Duration
	api       alertSetAPI
}

type alertSetAPI interface {
	BulkCreate(context.Context, alerts.BulkCreateAlertDefinitionsRequest) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error)
	BulkReplace(context.Context, alerts.BulkReplaceAlertDefinitionsRequest) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error)
	BulkDelete(context.Context, alerts.BulkDeleteAlertDefinitionsRequest) (*alerts.BulkDeleteAlertDefsResponse, *http.Response, error)
	Delete(context.Context, string) (*http.Response, error)
}

type sdkAlertSetAPI struct {
	client *alerts.AlertDefinitionsServiceAPIService
}

func (a sdkAlertSetAPI) BulkCreate(
	ctx context.Context,
	request alerts.BulkCreateAlertDefinitionsRequest,
) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error) {
	return a.client.AlertDefsServiceBulkCreateAlertDefs(ctx).
		BulkCreateAlertDefinitionsRequest(request).
		Execute()
}

func (a sdkAlertSetAPI) BulkReplace(
	ctx context.Context,
	request alerts.BulkReplaceAlertDefinitionsRequest,
) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
	return a.client.AlertDefsServiceBulkReplaceAlertDefs(ctx).
		BulkReplaceAlertDefinitionsRequest(request).
		Execute()
}

func (a sdkAlertSetAPI) BulkDelete(
	ctx context.Context,
	request alerts.BulkDeleteAlertDefinitionsRequest,
) (*alerts.BulkDeleteAlertDefsResponse, *http.Response, error) {
	return a.client.AlertDefsServiceBulkDeleteAlertDefs(ctx).
		BulkDeleteAlertDefinitionsRequest(request).
		Execute()
}

func (a sdkAlertSetAPI) Delete(ctx context.Context, id string) (*http.Response, error) {
	_, response, err := a.client.AlertDefsServiceDeleteAlertDef(ctx, id).Execute()
	return response, err
}

func (r *AlertSetReconciler) alertsAPI() alertSetAPI {
	if r.api != nil {
		return r.api
	}
	return sdkAlertSetAPI{client: r.ClientSet.Alerts()}
}

// +kubebuilder:rbac:groups=coralogix.com,resources=alertsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=alertsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=alertsets/finalizers,verbs=update

func (r *AlertSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	alertSet := &coralogixv1beta1.AlertSet{}
	if err := config.GetClient().Get(ctx, req.NamespacedName, alertSet); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get AlertSet: %w", err)
	}

	reconcileLog := log.FromContext(ctx).WithValues(
		"gvk", coralogixv1beta1.GroupVersion.WithKind(utils.AlertSetKind).String(),
		"name", req.Name,
		"namespace", req.Namespace,
	)
	originalStatus := deepCopyAlertSetStatus(alertSet)

	if !alertSet.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, reconcileLog, alertSet, originalStatus)
	}

	if !config.GetConfig().Selector.Matches(alertSet.Labels, alertSet.Namespace) {
		return r.reconcileSelectorMismatch(ctx, reconcileLog, alertSet, originalStatus)
	}

	if !controllerutil.ContainsFinalizer(alertSet, alertSetFinalizer) {
		controllerutil.AddFinalizer(alertSet, alertSetFinalizer)
		if err := config.GetClient().Update(ctx, alertSet); err != nil {
			if k8serrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("add AlertSet finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if err := validateAlertSetItems(alertSet.Spec.Alerts); err != nil {
		return r.finish(ctx, alertSet, originalStatus, utils.ReasonPartialFailure, []error{err})
	}

	statusByKey, err := alertSetStatusByKey(alertSet.Status.Alerts)
	if err != nil {
		return r.finish(ctx, alertSet, originalStatus, utils.ReasonPartialFailure, []error{err})
	}
	desiredByKey := desiredAlertSetItemsByKey(alertSet.Spec.Alerts)
	for key := range desiredByKey {
		if _, found := statusByKey[key]; !found {
			statusByKey[key] = coralogixv1beta1.AlertSetItemStatus{
				Key:   key,
				State: coralogixv1beta1.AlertSetItemStatePending,
			}
		}
	}

	removedKeys := make([]string, 0)
	for key, status := range statusByKey {
		if _, found := desiredByKey[key]; found {
			continue
		}
		if hasAlertSetStatusID(status) {
			removedKeys = append(removedKeys, key)
		} else {
			delete(statusByKey, key)
		}
	}
	sort.Strings(removedKeys)
	if len(removedKeys) > 0 {
		if cleanupErrs := r.deleteAlerts(ctx, reconcileLog, statusByKey, removedKeys); len(cleanupErrs) > 0 {
			alertSet.Status.Alerts = sortedAlertSetStatuses(statusByKey)
			return r.finish(ctx, alertSet, originalStatus, utils.ReasonRemoteDeletionFailed, cleanupErrs)
		}
	}

	var reconcileErrs []error
	createdKeys, createErrs, createRequestErr := r.createAlerts(ctx, reconcileLog, alertSet, desiredByKey, statusByKey)
	reconcileErrs = append(reconcileErrs, createErrs...)
	if len(createdKeys) > 0 {
		statusPersistedAfterConflict, err := r.persistCreatedAlertSetStatus(
			ctx,
			alertSet,
			statusByKey,
			createdKeys,
		)
		if err != nil {
			reconcileErrs = append(reconcileErrs, err)
			alertSet.Status.Alerts = sortedAlertSetStatuses(statusByKey)
			return r.finish(ctx, alertSet, originalStatus, utils.ReasonRemoteCreationFailed, reconcileErrs)
		}
		if statusPersistedAfterConflict {
			return ctrl.Result{Requeue: true}, nil
		}
	}
	if createRequestErr != nil {
		reconcileErrs = append(reconcileErrs, createRequestErr)
		alertSet.Status.Alerts = sortedAlertSetStatuses(statusByKey)
		return r.finish(ctx, alertSet, originalStatus, utils.ReasonRemoteCreationFailed, reconcileErrs)
	}

	replaceErrs, replaceRequestErr := r.replaceAlerts(
		ctx,
		reconcileLog,
		alertSet,
		desiredByKey,
		statusByKey,
		createdKeys,
	)
	reconcileErrs = append(reconcileErrs, replaceErrs...)
	if replaceRequestErr != nil {
		reconcileErrs = append(reconcileErrs, replaceRequestErr)
		alertSet.Status.Alerts = sortedAlertSetStatuses(statusByKey)
		return r.finish(ctx, alertSet, originalStatus, utils.ReasonRemoteUpdateFailed, reconcileErrs)
	}

	alertSet.Status.Alerts = sortedAlertSetStatuses(statusByKey)
	if len(reconcileErrs) == 0 {
		if err := validateSynchronizedAlertSet(desiredByKey, statusByKey); err != nil {
			reconcileErrs = append(reconcileErrs, err)
		}
	}
	return r.finish(ctx, alertSet, originalStatus, utils.ReasonPartialFailure, reconcileErrs)
}

func (r *AlertSetReconciler) reconcileDeletion(
	ctx context.Context,
	reconcileLog logr.Logger,
	alertSet *coralogixv1beta1.AlertSet,
	originalStatus coralogixv1beta1.AlertSetStatus,
) (ctrl.Result, error) {
	statusByKey, err := alertSetStatusByKey(alertSet.Status.Alerts)
	if err != nil {
		return r.finish(ctx, alertSet, originalStatus, utils.ReasonRemoteDeletionFailed, []error{err})
	}
	keys := alertSetStatusKeysWithIDs(statusByKey)
	if cleanupErrs := r.deleteAlerts(ctx, reconcileLog, statusByKey, keys); len(cleanupErrs) > 0 {
		alertSet.Status.Alerts = sortedAlertSetStatuses(statusByKey)
		return r.finish(ctx, alertSet, originalStatus, utils.ReasonRemoteDeletionFailed, cleanupErrs)
	}

	alertSet.Status.Alerts = nil
	if err := updateAlertSetStatusIfChanged(ctx, alertSet, originalStatus); err != nil {
		if k8serrors.IsNotFound(err) {
			monitoring.DeleteResourceInfoMetric(utils.AlertSetKind, alertSet.Name, alertSet.Namespace)
			return ctrl.Result{}, nil
		}
		if k8serrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, fmt.Errorf("update AlertSet deletion progress: %w", err)
	}
	if controllerutil.ContainsFinalizer(alertSet, alertSetFinalizer) {
		controllerutil.RemoveFinalizer(alertSet, alertSetFinalizer)
		if err := config.GetClient().Update(ctx, alertSet); err != nil {
			if k8serrors.IsNotFound(err) {
				monitoring.DeleteResourceInfoMetric(utils.AlertSetKind, alertSet.Name, alertSet.Namespace)
				return ctrl.Result{}, nil
			}
			if k8serrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("remove AlertSet finalizer: %w", err)
		}
	}
	monitoring.DeleteResourceInfoMetric(utils.AlertSetKind, alertSet.Name, alertSet.Namespace)
	return ctrl.Result{}, nil
}

func (r *AlertSetReconciler) reconcileSelectorMismatch(
	ctx context.Context,
	reconcileLog logr.Logger,
	alertSet *coralogixv1beta1.AlertSet,
	originalStatus coralogixv1beta1.AlertSetStatus,
) (ctrl.Result, error) {
	statusByKey, err := alertSetStatusByKey(alertSet.Status.Alerts)
	if err != nil {
		return r.finish(ctx, alertSet, originalStatus, utils.ReasonRemoteDeletionFailed, []error{err})
	}
	if cleanupErrs := r.deleteAlerts(ctx, reconcileLog, statusByKey, alertSetStatusKeysWithIDs(statusByKey)); len(cleanupErrs) > 0 {
		alertSet.Status.Alerts = sortedAlertSetStatuses(statusByKey)
		return r.finish(ctx, alertSet, originalStatus, utils.ReasonRemoteDeletionFailed, cleanupErrs)
	}

	alertSet.Status = coralogixv1beta1.AlertSetStatus{}
	if err := updateAlertSetStatusIfChanged(ctx, alertSet, originalStatus); err != nil {
		if k8serrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, fmt.Errorf("clear AlertSet status after selector mismatch: %w", err)
	}
	if controllerutil.ContainsFinalizer(alertSet, alertSetFinalizer) {
		controllerutil.RemoveFinalizer(alertSet, alertSetFinalizer)
		if err := config.GetClient().Update(ctx, alertSet); err != nil {
			if k8serrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("remove AlertSet finalizer after selector mismatch: %w", err)
		}
	}
	monitoring.DeleteResourceInfoMetric(utils.AlertSetKind, alertSet.Name, alertSet.Namespace)
	return ctrl.Result{}, nil
}

func (r *AlertSetReconciler) createAlerts(
	ctx context.Context,
	reconcileLog logr.Logger,
	alertSet *coralogixv1beta1.AlertSet,
	desiredByKey map[string]coralogixv1beta1.AlertSetItem,
	statusByKey map[string]coralogixv1beta1.AlertSetItemStatus,
) (map[string]struct{}, []error, error) {
	keys := sortedDesiredAlertSetKeys(desiredByKey)
	requestItems := make([]alerts.AlertDefToCreate, 0, len(keys))
	requestKeys := make([]string, 0, len(keys))
	var itemErrs []error
	for _, key := range keys {
		status := statusByKey[key]
		if hasAlertSetStatusID(status) {
			continue
		}
		desired := desiredByKey[key]
		props, err := desired.Spec.ExtractAlertDefProperties(&coralogixv1beta1.GetResourceRefProperties{
			Ctx:       ctx,
			Log:       reconcileLog,
			ClientSet: r.ClientSet,
			Namespace: alertSet.Namespace,
		})
		if err != nil {
			itemErr := fmt.Errorf("convert alert %q for create: %w", key, err)
			setAlertSetStatusFailure(statusByKey, key, itemErr.Error())
			itemErrs = append(itemErrs, itemErr)
			continue
		}
		requestItems = append(requestItems, alerts.AlertDefToCreate{AlertDefProperties: *props})
		requestKeys = append(requestKeys, key)
	}
	createdKeys := make(map[string]struct{}, len(requestKeys))
	if len(requestItems) == 0 {
		return createdKeys, itemErrs, nil
	}

	request := alerts.BulkCreateAlertDefinitionsRequest{AlertDefsToCreate: requestItems}
	reconcileLog.Info("Creating remote alerts", "count", len(requestItems))
	response, httpResponse, err := r.alertsAPI().BulkCreate(ctx, request)
	if err != nil {
		return createdKeys, itemErrs, fmt.Errorf("bulk create remote alerts: %w", cxsdk.NewAPIError(httpResponse, err))
	}

	responseErrs := applyBulkCreateResponse(requestKeys, response, statusByKey, createdKeys)
	return createdKeys, append(itemErrs, responseErrs...), nil
}

// persistCreatedAlertSetStatus writes remote IDs before the reconcile continues. If the
// resource changed while the remote request was in flight, retry against the latest object
// and keep the returned IDs under their stable keys.
func (r *AlertSetReconciler) persistCreatedAlertSetStatus(
	ctx context.Context,
	alertSet *coralogixv1beta1.AlertSet,
	statusByKey map[string]coralogixv1beta1.AlertSetItemStatus,
	createdKeys map[string]struct{},
) (bool, error) {
	alertSet.Status.Alerts = sortedAlertSetStatuses(statusByKey)
	if err := config.GetClient().Status().Update(ctx, alertSet); err == nil {
		return false, nil
	} else if !k8serrors.IsConflict(err) {
		return false, fmt.Errorf("persist created AlertSet IDs: %w", err)
	}

	latest := &coralogixv1beta1.AlertSet{}
	if err := config.GetClient().Get(ctx, client.ObjectKeyFromObject(alertSet), latest); err != nil {
		return true, fmt.Errorf("get latest AlertSet after status conflict: %w", err)
	}
	latestStatuses, err := alertSetStatusByKey(latest.Status.Alerts)
	if err != nil {
		return true, fmt.Errorf("read latest AlertSet status after conflict: %w", err)
	}
	for key := range createdKeys {
		status, found := statusByKey[key]
		if found && hasAlertSetStatusID(status) {
			latestStatuses[key] = status
		}
	}
	latest.Status.Alerts = sortedAlertSetStatuses(latestStatuses)
	if err := config.GetClient().Status().Update(ctx, latest); err != nil {
		return true, fmt.Errorf("persist created AlertSet IDs after conflict: %w", err)
	}
	*alertSet = *latest
	return true, nil
}

func (r *AlertSetReconciler) replaceAlerts(
	ctx context.Context,
	reconcileLog logr.Logger,
	alertSet *coralogixv1beta1.AlertSet,
	desiredByKey map[string]coralogixv1beta1.AlertSetItem,
	statusByKey map[string]coralogixv1beta1.AlertSetItemStatus,
	createdKeys map[string]struct{},
) ([]error, error) {
	keys := sortedDesiredAlertSetKeys(desiredByKey)
	requestItems := make([]alerts.AlertDefToReplace, 0, len(keys))
	requestIDs := make(map[string]string, len(keys))
	var itemErrs []error
	for _, key := range keys {
		if _, created := createdKeys[key]; created {
			continue
		}
		status := statusByKey[key]
		if !hasAlertSetStatusID(status) {
			continue
		}
		desired := desiredByKey[key]
		props, err := desired.Spec.ExtractAlertDefProperties(&coralogixv1beta1.GetResourceRefProperties{
			Ctx:       ctx,
			Log:       reconcileLog,
			ClientSet: r.ClientSet,
			Namespace: alertSet.Namespace,
		})
		if err != nil {
			itemErr := fmt.Errorf("convert alert %q for replace: %w", key, err)
			setAlertSetStatusFailure(statusByKey, key, itemErr.Error())
			itemErrs = append(itemErrs, itemErr)
			continue
		}
		id := *status.ID
		requestItems = append(requestItems, alerts.AlertDefToReplace{Id: &id, AlertDefProperties: props})
		requestIDs[id] = key
	}
	if len(requestItems) == 0 {
		return itemErrs, nil
	}

	request := alerts.BulkReplaceAlertDefinitionsRequest{AlertDefsToReplace: requestItems}
	reconcileLog.Info("Replacing remote alerts", "count", len(requestItems))
	response, httpResponse, err := r.alertsAPI().BulkReplace(ctx, request)
	if err != nil {
		return itemErrs, fmt.Errorf("bulk replace remote alerts: %w", cxsdk.NewAPIError(httpResponse, err))
	}

	responseErrs := applyBulkReplaceResponse(requestIDs, response, statusByKey)
	return append(itemErrs, responseErrs...), nil
}

func (r *AlertSetReconciler) deleteAlerts(
	ctx context.Context,
	reconcileLog logr.Logger,
	statusByKey map[string]coralogixv1beta1.AlertSetItemStatus,
	keys []string,
) []error {
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	ids := make([]string, 0, len(keys))
	idToKey := make(map[string]string, len(keys))
	for _, key := range keys {
		status := statusByKey[key]
		if !hasAlertSetStatusID(status) {
			delete(statusByKey, key)
			continue
		}
		status.State = coralogixv1beta1.AlertSetItemStateDeleting
		status.Message = ""
		statusByKey[key] = status
		id := *status.ID
		ids = append(ids, id)
		idToKey[id] = key
	}
	if len(ids) == 0 {
		return nil
	}

	reconcileLog.Info("Deleting remote alerts", "count", len(ids))
	request := alerts.BulkDeleteAlertDefinitionsRequest{Ids: ids}
	response, httpResponse, err := r.alertsAPI().BulkDelete(ctx, request)
	if err != nil {
		if httpResponse != nil && httpResponse.StatusCode == http.StatusBadRequest {
			return r.deleteAlertsIndividually(ctx, statusByKey, ids, idToKey)
		}
		apiErr := cxsdk.NewAPIError(httpResponse, err)
		message := fmt.Sprintf("bulk delete remote alerts: %v", apiErr)
		for _, key := range keys {
			status := statusByKey[key]
			status.Message = message
			statusByKey[key] = status
		}
		return []error{errors.New(message)}
	}
	if response == nil {
		return []error{errors.New("bulk delete remote alerts: response is empty")}
	}

	completedIDs := make(map[string]struct{}, len(response.DeletedIds)+len(response.NotFoundIds))
	for _, id := range response.DeletedIds {
		completedIDs[id] = struct{}{}
	}
	for _, id := range response.NotFoundIds {
		completedIDs[id] = struct{}{}
	}
	var resultErrs []error
	for _, id := range ids {
		key := idToKey[id]
		if _, completed := completedIDs[id]; completed {
			delete(statusByKey, key)
			continue
		}
		message := fmt.Sprintf("bulk delete response omitted alert %q with ID %q", key, id)
		status := statusByKey[key]
		status.Message = message
		statusByKey[key] = status
		resultErrs = append(resultErrs, errors.New(message))
	}
	for id := range completedIDs {
		if _, requested := idToKey[id]; !requested {
			resultErrs = append(resultErrs, fmt.Errorf("bulk delete response returned unknown ID %q", id))
		}
	}
	return resultErrs
}

func (r *AlertSetReconciler) deleteAlertsIndividually(
	ctx context.Context,
	statusByKey map[string]coralogixv1beta1.AlertSetItemStatus,
	ids []string,
	idToKey map[string]string,
) []error {
	var resultErrs []error
	for _, id := range ids {
		httpResponse, err := r.alertsAPI().Delete(ctx, id)
		if err == nil {
			delete(statusByKey, idToKey[id])
			continue
		}
		apiErr := cxsdk.NewAPIError(httpResponse, err)
		if httpResponse != nil && httpResponse.StatusCode == http.StatusNotFound {
			delete(statusByKey, idToKey[id])
			continue
		}
		key := idToKey[id]
		itemErr := fmt.Errorf("delete alert %q with ID %q: %w", key, id, apiErr)
		status := statusByKey[key]
		status.Message = itemErr.Error()
		statusByKey[key] = status
		resultErrs = append(resultErrs, itemErr)
	}
	return resultErrs
}

func (r *AlertSetReconciler) finish(
	ctx context.Context,
	alertSet *coralogixv1beta1.AlertSet,
	originalStatus coralogixv1beta1.AlertSetStatus,
	reason string,
	reconcileErrs []error,
) (ctrl.Result, error) {
	if len(reconcileErrs) > 0 {
		joinedErr := errors.Join(reconcileErrs...)
		utils.SetSyncedConditionFalse(&alertSet.Status.Conditions, alertSet.Generation, reason, joinedErr.Error())
		alertSet.Status.PrintableStatus = "RemoteUnsynced"
		if err := updateAlertSetStatusIfChanged(ctx, alertSet, originalStatus); err != nil {
			if k8serrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("update failed AlertSet status: %w", err)
		}
		monitoring.SetResourceInfoMetricUnsynced(utils.AlertSetKind, alertSet.Name, alertSet.Namespace)
		return ctrl.Result{}, joinedErr
	}

	utils.SetSyncedConditionTrue(
		&alertSet.Status.Conditions,
		alertSet.Generation,
		utils.ReasonRemoteSyncedSuccessfully,
	)
	alertSet.Status.PrintableStatus = "RemoteSynced"
	if err := updateAlertSetStatusIfChanged(ctx, alertSet, originalStatus); err != nil {
		if k8serrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, fmt.Errorf("update synchronized AlertSet status: %w", err)
	}
	monitoring.SetResourceInfoMetricSynced(utils.AlertSetKind, alertSet.Name, alertSet.Namespace)
	return ctrl.Result{RequeueAfter: r.Interval}, nil
}

func applyBulkCreateResponse(
	requestKeys []string,
	response *alerts.BulkCreateAlertDefsResponse,
	statusByKey map[string]coralogixv1beta1.AlertSetItemStatus,
	createdKeys map[string]struct{},
) []error {
	if response == nil {
		return []error{errors.New("bulk create response is empty")}
	}
	failedByIndex := make(map[int]string, len(response.FailedToCreateAlertDefs))
	var resultErrs []error
	for _, failure := range response.FailedToCreateAlertDefs {
		index := int(failure.Index)
		if index < 0 || index >= len(requestKeys) {
			resultErrs = append(resultErrs, fmt.Errorf("bulk create response returned invalid failed index %d", index))
			continue
		}
		if _, duplicate := failedByIndex[index]; duplicate {
			resultErrs = append(resultErrs, fmt.Errorf("bulk create response returned failed index %d more than once", index))
			continue
		}
		failedByIndex[index] = failure.Reason
	}

	successIndex := 0
	for requestIndex, key := range requestKeys {
		if reason, failed := failedByIndex[requestIndex]; failed {
			itemErr := fmt.Errorf("create alert %q: %s", key, reason)
			setAlertSetStatusFailure(statusByKey, key, itemErr.Error())
			resultErrs = append(resultErrs, itemErr)
			continue
		}
		if successIndex >= len(response.AlertDefs) {
			itemErr := fmt.Errorf("bulk create response omitted result for alert %q", key)
			setAlertSetStatusFailure(statusByKey, key, itemErr.Error())
			resultErrs = append(resultErrs, itemErr)
			continue
		}
		created := response.AlertDefs[successIndex]
		successIndex++
		if created.Id == nil || *created.Id == "" {
			itemErr := fmt.Errorf("bulk create response returned no ID for alert %q", key)
			setAlertSetStatusFailure(statusByKey, key, itemErr.Error())
			resultErrs = append(resultErrs, itemErr)
			continue
		}
		id := *created.Id
		statusByKey[key] = coralogixv1beta1.AlertSetItemStatus{
			Key:   key,
			ID:    &id,
			State: coralogixv1beta1.AlertSetItemStateSynced,
		}
		createdKeys[key] = struct{}{}
	}
	if successIndex < len(response.AlertDefs) {
		resultErrs = append(resultErrs, fmt.Errorf(
			"bulk create response returned %d unexpected successful results",
			len(response.AlertDefs)-successIndex,
		))
	}
	return resultErrs
}

func applyBulkReplaceResponse(
	requestIDs map[string]string,
	response *alerts.BulkReplaceAlertDefsResponse,
	statusByKey map[string]coralogixv1beta1.AlertSetItemStatus,
) []error {
	if response == nil {
		return []error{errors.New("bulk replace response is empty")}
	}
	seen := make(map[string]struct{}, len(requestIDs))
	var resultErrs []error
	markSynced := func(id, result string) {
		key, found := requestIDs[id]
		if !found {
			resultErrs = append(resultErrs, fmt.Errorf("bulk replace %s result returned unknown ID %q", result, id))
			return
		}
		if _, duplicate := seen[id]; duplicate {
			resultErrs = append(resultErrs, fmt.Errorf("bulk replace response returned ID %q more than once", id))
			return
		}
		seen[id] = struct{}{}
		status := statusByKey[key]
		status.State = coralogixv1beta1.AlertSetItemStateSynced
		status.Message = ""
		statusByKey[key] = status
	}
	for _, replaced := range response.AlertDefs {
		if replaced.Id == nil || *replaced.Id == "" {
			resultErrs = append(resultErrs, errors.New("bulk replace response returned a successful result without an ID"))
			continue
		}
		markSynced(*replaced.Id, "successful")
	}
	for _, id := range response.SkippedIds {
		markSynced(id, "skipped")
	}
	for _, failure := range response.FailedToReplaceAlertDefs {
		if failure.Id == nil || *failure.Id == "" {
			resultErrs = append(resultErrs, errors.New("bulk replace response returned a failure without an ID"))
			continue
		}
		id := *failure.Id
		key, found := requestIDs[id]
		if !found {
			resultErrs = append(resultErrs, fmt.Errorf("bulk replace failure returned unknown ID %q", id))
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			resultErrs = append(resultErrs, fmt.Errorf("bulk replace response returned ID %q more than once", id))
			continue
		}
		seen[id] = struct{}{}
		reason := "no reason returned"
		if failure.Reason != nil && *failure.Reason != "" {
			reason = *failure.Reason
		}
		itemErr := fmt.Errorf("replace alert %q: %s", key, reason)
		setAlertSetStatusFailure(statusByKey, key, itemErr.Error())
		resultErrs = append(resultErrs, itemErr)
	}
	for _, id := range response.NotFoundIds {
		key, found := requestIDs[id]
		if !found {
			resultErrs = append(resultErrs, fmt.Errorf("bulk replace not-found result returned unknown ID %q", id))
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			resultErrs = append(resultErrs, fmt.Errorf("bulk replace response returned ID %q more than once", id))
			continue
		}
		seen[id] = struct{}{}
		message := fmt.Sprintf("remote alert %q with ID %q was not found", key, id)
		statusByKey[key] = coralogixv1beta1.AlertSetItemStatus{
			Key:     key,
			State:   coralogixv1beta1.AlertSetItemStatePending,
			Message: message,
		}
		resultErrs = append(resultErrs, errors.New(message))
	}
	for id, key := range requestIDs {
		if _, found := seen[id]; found {
			continue
		}
		itemErr := fmt.Errorf("bulk replace response omitted alert %q with ID %q", key, id)
		setAlertSetStatusFailure(statusByKey, key, itemErr.Error())
		resultErrs = append(resultErrs, itemErr)
	}
	return resultErrs
}

func validateAlertSetItems(items []coralogixv1beta1.AlertSetItem) error {
	if len(items) < 1 || len(items) > 100 {
		return fmt.Errorf("AlertSet must contain between 1 and 100 alerts, got %d", len(items))
	}
	seen := make(map[string]struct{}, len(items))
	var validationErrs []error
	for _, item := range items {
		if messages := validation.IsDNS1123Label(item.Key); len(messages) > 0 {
			validationErrs = append(validationErrs, fmt.Errorf("invalid alert key %q: %s", item.Key, strings.Join(messages, ", ")))
		}
		if _, duplicate := seen[item.Key]; duplicate {
			validationErrs = append(validationErrs, fmt.Errorf("duplicate alert key %q", item.Key))
		}
		seen[item.Key] = struct{}{}
	}
	return errors.Join(validationErrs...)
}

func alertSetStatusByKey(statuses []coralogixv1beta1.AlertSetItemStatus) (map[string]coralogixv1beta1.AlertSetItemStatus, error) {
	result := make(map[string]coralogixv1beta1.AlertSetItemStatus, len(statuses))
	var resultErrs []error
	for _, status := range statuses {
		if _, duplicate := result[status.Key]; duplicate {
			resultErrs = append(resultErrs, fmt.Errorf("duplicate status key %q", status.Key))
			continue
		}
		if status.ID != nil && *status.ID == "" {
			status.ID = nil
		}
		result[status.Key] = status
	}
	return result, errors.Join(resultErrs...)
}

func desiredAlertSetItemsByKey(items []coralogixv1beta1.AlertSetItem) map[string]coralogixv1beta1.AlertSetItem {
	result := make(map[string]coralogixv1beta1.AlertSetItem, len(items))
	for _, item := range items {
		result[item.Key] = item
	}
	return result
}

func sortedDesiredAlertSetKeys(items map[string]coralogixv1beta1.AlertSetItem) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedAlertSetStatuses(statuses map[string]coralogixv1beta1.AlertSetItemStatus) []coralogixv1beta1.AlertSetItemStatus {
	keys := make([]string, 0, len(statuses))
	for key := range statuses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]coralogixv1beta1.AlertSetItemStatus, 0, len(keys))
	for _, key := range keys {
		result = append(result, statuses[key])
	}
	return result
}

func alertSetStatusKeysWithIDs(statuses map[string]coralogixv1beta1.AlertSetItemStatus) []string {
	keys := make([]string, 0, len(statuses))
	for key, status := range statuses {
		if hasAlertSetStatusID(status) {
			keys = append(keys, key)
		} else {
			delete(statuses, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func hasAlertSetStatusID(status coralogixv1beta1.AlertSetItemStatus) bool {
	return status.ID != nil && *status.ID != ""
}

func deepCopyAlertSetStatus(alertSet *coralogixv1beta1.AlertSet) coralogixv1beta1.AlertSetStatus {
	return alertSet.DeepCopy().Status
}

func setAlertSetStatusFailure(
	statuses map[string]coralogixv1beta1.AlertSetItemStatus,
	key string,
	message string,
) {
	status := statuses[key]
	status.Key = key
	status.State = coralogixv1beta1.AlertSetItemStateFailed
	status.Message = message
	statuses[key] = status
}

func validateSynchronizedAlertSet(
	desired map[string]coralogixv1beta1.AlertSetItem,
	statuses map[string]coralogixv1beta1.AlertSetItemStatus,
) error {
	if len(desired) != len(statuses) {
		return fmt.Errorf("AlertSet has %d desired alerts and %d status entries", len(desired), len(statuses))
	}
	for key := range desired {
		status, found := statuses[key]
		if !found || !hasAlertSetStatusID(status) || status.State != coralogixv1beta1.AlertSetItemStateSynced {
			return fmt.Errorf("alert %q is not synchronized", key)
		}
	}
	return nil
}

func updateAlertSetStatusIfChanged(
	ctx context.Context,
	alertSet *coralogixv1beta1.AlertSet,
	originalStatus coralogixv1beta1.AlertSetStatus,
) error {
	if reflect.DeepEqual(originalStatus, alertSet.Status) {
		return nil
	}
	return config.GetClient().Status().Update(ctx, alertSet)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1beta1.AlertSet{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}

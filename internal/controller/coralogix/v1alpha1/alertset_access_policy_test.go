// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

const testAlertSetAccessPolicy = `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

func accessPolicyAlertSetItem(key string, policy *string) coralogixv1alpha1.AlertSetItem {
	item := minimalAlertSetItem(key)
	item.Spec.AccessPolicy = policy
	return item
}

func initAlertSetClient(t *testing.T, alertSet *coralogixv1alpha1.AlertSet, wrap func(client.Client) client.Client) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))
	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&coralogixv1alpha1.AlertSet{}).
		WithObjects(alertSet.DeepCopy()).
		Build()
	originalClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(originalClient) })
	if wrap != nil {
		config.InitClient(wrap(baseClient))
	} else {
		config.InitClient(baseClient)
	}
	return baseClient
}

func findCreateItemByName(t *testing.T, request alerts.BulkCreateAlertDefinitionsRequest, name string) alerts.AlertDefToCreate {
	t.Helper()
	for _, item := range request.AlertDefsToCreate {
		require.NotNil(t, item.AlertDefProperties.Name)
		if *item.AlertDefProperties.Name == name {
			return item
		}
	}
	t.Fatalf("no create item with name %q", name)
	return alerts.AlertDefToCreate{}
}

func findReplaceItemByID(t *testing.T, request alerts.BulkReplaceAlertDefinitionsRequest, id string) alerts.AlertDefToReplace {
	t.Helper()
	for _, item := range request.AlertDefsToReplace {
		require.NotNil(t, item.Id)
		if *item.Id == id {
			return item
		}
	}
	t.Fatalf("no replace item with ID %q", id)
	return alerts.AlertDefToReplace{}
}

func TestCreateAlertsSendsAccessPolicy(t *testing.T) {
	originalClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(originalClient) })
	scheme := runtime.NewScheme()
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).Build())

	items := []coralogixv1alpha1.AlertSetItem{
		accessPolicyAlertSetItem("nil-policy", nil),
		accessPolicyAlertSetItem("empty-policy", ptr.To("")),
		accessPolicyAlertSetItem("set-policy", ptr.To(testAlertSetAccessPolicy)),
	}
	var captured alerts.BulkCreateAlertDefinitionsRequest
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkCreate: func(
			_ context.Context,
			request alerts.BulkCreateAlertDefinitionsRequest,
		) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error) {
			captured = request
			created := make([]alerts.AlertDef, len(request.AlertDefsToCreate))
			for i := range request.AlertDefsToCreate {
				id := "id-" + *request.AlertDefsToCreate[i].AlertDefProperties.Name
				created[i].Id = &id
			}
			return &alerts.BulkCreateAlertDefsResponse{AlertDefs: created}, nil, nil
		},
	}}
	alertSet := &coralogixv1alpha1.AlertSet{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}}
	statuses := map[string]coralogixv1alpha1.AlertSetItemStatus{
		"nil-policy":   {Key: "nil-policy", State: coralogixv1alpha1.AlertSetItemStatePending},
		"empty-policy": {Key: "empty-policy", State: coralogixv1alpha1.AlertSetItemStatePending},
		"set-policy":   {Key: "set-policy", State: coralogixv1alpha1.AlertSetItemStatePending},
	}

	created, itemErrs, requestErr := reconciler.createAlerts(
		context.Background(),
		logr.Discard(),
		alertSet,
		desiredAlertSetItemsByKey(items),
		statuses,
	)

	require.NoError(t, requestErr)
	require.Empty(t, itemErrs)
	require.Len(t, created, 3)
	require.Nil(t, findCreateItemByName(t, captured, "nil-policy").AccessPolicy)
	require.Nil(t, findCreateItemByName(t, captured, "empty-policy").AccessPolicy)
	setItem := findCreateItemByName(t, captured, "set-policy")
	require.NotNil(t, setItem.AccessPolicy)
	require.Equal(t, testAlertSetAccessPolicy, *setItem.AccessPolicy)
}

func TestReplaceAlertsSendsAccessPolicyAction(t *testing.T) {
	items := []coralogixv1alpha1.AlertSetItem{
		accessPolicyAlertSetItem("nil-policy", nil),
		accessPolicyAlertSetItem("empty-policy", ptr.To("")),
		accessPolicyAlertSetItem("set-policy", ptr.To(testAlertSetAccessPolicy)),
	}
	alertSet := &coralogixv1alpha1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-alert-set", Namespace: "default"},
		Spec:       coralogixv1alpha1.AlertSetSpec{Alerts: items},
	}
	baseClient := initAlertSetClient(t, alertSet, nil)
	stored := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), stored))
	var captured alerts.BulkReplaceAlertDefinitionsRequest
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkReplace: func(
			_ context.Context,
			request alerts.BulkReplaceAlertDefinitionsRequest,
		) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
			captured = request
			replaced := make([]alerts.AlertDef, len(request.AlertDefsToReplace))
			for i := range request.AlertDefsToReplace {
				replaced[i].Id = request.AlertDefsToReplace[i].Id
			}
			return &alerts.BulkReplaceAlertDefsResponse{AlertDefs: replaced}, nil, nil
		},
	}}
	statuses := map[string]coralogixv1alpha1.AlertSetItemStatus{}
	for _, key := range []string{"nil-policy", "empty-policy", "set-policy"} {
		id := "id-" + key
		statuses[key] = coralogixv1alpha1.AlertSetItemStatus{
			Key:   key,
			ID:    &id,
			State: coralogixv1alpha1.AlertSetItemStateSynced,
		}
	}

	itemErrs, requestErr := reconciler.replaceAlerts(
		context.Background(),
		logr.Discard(),
		stored,
		desiredAlertSetItemsByKey(items),
		statuses,
		map[string]struct{}{},
	)

	require.NoError(t, requestErr)
	require.Empty(t, itemErrs)
	require.Len(t, captured.AlertDefsToReplace, 3)
	require.Nil(t, findReplaceItemByID(t, captured, "id-nil-policy").AccessPolicy)
	emptyItem := findReplaceItemByID(t, captured, "id-empty-policy")
	require.NotNil(t, emptyItem.AccessPolicy)
	require.Nil(t, emptyItem.AccessPolicy.Upsert)
	require.NotNil(t, emptyItem.AccessPolicy.Delete)
	setItem := findReplaceItemByID(t, captured, "id-set-policy")
	require.NotNil(t, setItem.AccessPolicy)
	require.Nil(t, setItem.AccessPolicy.Delete)
	require.NotNil(t, setItem.AccessPolicy.Upsert)
	require.Equal(t, testAlertSetAccessPolicy, *setItem.AccessPolicy.Upsert)

	// requestIDs maps remote IDs to item keys; the record must name the item, not its ID.
	persisted := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), persisted))
	require.Equal(t, `["set-policy"]`, persisted.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation],
		"an upserted policy must be recorded durably under its item key")
}

func TestCreateAlertsRecordsAccessPolicyManaged(t *testing.T) {
	originalClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(originalClient) })
	scheme := runtime.NewScheme()
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).Build())

	items := []coralogixv1alpha1.AlertSetItem{
		accessPolicyAlertSetItem("nil-policy", nil),
		accessPolicyAlertSetItem("empty-policy", ptr.To("")),
		accessPolicyAlertSetItem("set-policy", ptr.To(testAlertSetAccessPolicy)),
	}
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkCreate: func(
			_ context.Context,
			request alerts.BulkCreateAlertDefinitionsRequest,
		) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error) {
			created := make([]alerts.AlertDef, len(request.AlertDefsToCreate))
			for i := range request.AlertDefsToCreate {
				id := "id-" + *request.AlertDefsToCreate[i].AlertDefProperties.Name
				created[i].Id = &id
			}
			return &alerts.BulkCreateAlertDefsResponse{AlertDefs: created}, nil, nil
		},
	}}
	alertSet := &coralogixv1alpha1.AlertSet{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}}
	statuses := map[string]coralogixv1alpha1.AlertSetItemStatus{
		"nil-policy":   {Key: "nil-policy", State: coralogixv1alpha1.AlertSetItemStatePending},
		"empty-policy": {Key: "empty-policy", State: coralogixv1alpha1.AlertSetItemStatePending},
		"set-policy":   {Key: "set-policy", State: coralogixv1alpha1.AlertSetItemStatePending},
	}

	_, _, requestErr := reconciler.createAlerts(
		context.Background(),
		logr.Discard(),
		alertSet,
		desiredAlertSetItemsByKey(items),
		statuses,
	)

	require.NoError(t, requestErr)
	require.False(t, statuses["nil-policy"].AccessPolicyManaged)
	require.False(t, statuses["empty-policy"].AccessPolicyManaged)
	require.True(t, statuses["set-policy"].AccessPolicyManaged)
}

func TestReplaceAlertsRecordsAccessPolicyBeforeSending(t *testing.T) {
	newAlertSet := func() *coralogixv1alpha1.AlertSet {
		id := "id-set-policy"
		return &coralogixv1alpha1.AlertSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-alert-set", Namespace: "default"},
			Spec: coralogixv1alpha1.AlertSetSpec{
				Alerts: []coralogixv1alpha1.AlertSetItem{
					accessPolicyAlertSetItem("set-policy", ptr.To(testAlertSetAccessPolicy)),
				},
			},
			Status: coralogixv1alpha1.AlertSetStatus{
				Alerts: []coralogixv1alpha1.AlertSetItemStatus{{
					Key:   "set-policy",
					ID:    &id,
					State: coralogixv1alpha1.AlertSetItemStateSynced,
				}},
			},
		}
	}

	t.Run("tombstone is persisted before the request", func(t *testing.T) {
		var baseClient client.Client
		var stored *coralogixv1alpha1.AlertSet
		requestSent := false
		baseClient = initAlertSetClient(t, newAlertSet(), nil)
		stored = &coralogixv1alpha1.AlertSet{}
		require.NoError(t, baseClient.Get(context.Background(), client.ObjectKey{Name: "test-alert-set", Namespace: "default"}, stored))
		reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
			bulkReplace: func(
				_ context.Context,
				request alerts.BulkReplaceAlertDefinitionsRequest,
			) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
				requestSent = true
				remote := &coralogixv1alpha1.AlertSet{}
				require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(stored), remote))
				require.Len(t, remote.Status.Alerts, 1)
				require.True(t, remote.Status.Alerts[0].AccessPolicyManaged, "tombstone must be persisted before the bulk replace is sent")
				require.Equal(t, `["set-policy"]`, remote.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation],
					"the durable record must be persisted before the bulk replace is sent")
				replaced := make([]alerts.AlertDef, len(request.AlertDefsToReplace))
				for i := range request.AlertDefsToReplace {
					replaced[i].Id = request.AlertDefsToReplace[i].Id
				}
				return &alerts.BulkReplaceAlertDefsResponse{AlertDefs: replaced}, nil, nil
			},
		}}
		statuses, err := alertSetStatusByKey(stored.Status.Alerts)
		require.NoError(t, err)

		itemErrs, requestErr := reconciler.replaceAlerts(
			context.Background(),
			logr.Discard(),
			stored,
			desiredAlertSetItemsByKey(stored.Spec.Alerts),
			statuses,
			map[string]struct{}{},
		)

		require.NoError(t, requestErr)
		require.Empty(t, itemErrs)
		require.True(t, requestSent)
		require.True(t, statuses["set-policy"].AccessPolicyManaged)
	})

	t.Run("status update error sends no request", func(t *testing.T) {
		alertSet := newAlertSet()
		initAlertSetClient(t, alertSet, func(base client.Client) client.Client {
			return &errorOnceStatusClient{Client: base, err: context.DeadlineExceeded}
		})
		replaceCalled := false
		reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
			bulkReplace: func(
				context.Context,
				alerts.BulkReplaceAlertDefinitionsRequest,
			) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
				replaceCalled = true
				return &alerts.BulkReplaceAlertDefsResponse{}, nil, nil
			},
		}}
		statuses, err := alertSetStatusByKey(alertSet.Status.Alerts)
		require.NoError(t, err)

		_, requestErr := reconciler.replaceAlerts(
			context.Background(),
			logr.Discard(),
			alertSet,
			desiredAlertSetItemsByKey(alertSet.Spec.Alerts),
			statuses,
			map[string]struct{}{},
		)

		require.ErrorContains(t, requestErr, "recording the alertset access policies")
		require.False(t, replaceCalled)
	})
}

func TestApplyBulkReplaceResponseKeepsIDForNotFoundWhenPolicyManaged(t *testing.T) {
	for _, tc := range []struct {
		name    string
		policy  *string
		managed bool
		state   coralogixv1alpha1.AlertSetItemState
		keepID  bool
		keepTag bool
	}{
		{name: "absent with tombstone keeps ID", policy: nil, managed: true, state: coralogixv1alpha1.AlertSetItemStateFailed, keepID: true, keepTag: true},
		{name: "set without tombstone keeps ID", policy: ptr.To(testAlertSetAccessPolicy), managed: false, state: coralogixv1alpha1.AlertSetItemStateFailed, keepID: true, keepTag: false},
		{name: "absent without tombstone resets", policy: nil, managed: false, state: coralogixv1alpha1.AlertSetItemStatePending, keepID: false, keepTag: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := "missing-id"
			statuses := map[string]coralogixv1alpha1.AlertSetItemStatus{
				"alpha": {Key: "alpha", ID: &id, State: coralogixv1alpha1.AlertSetItemStateSynced, AccessPolicyManaged: tc.managed},
			}
			desired := desiredAlertSetItemsByKey([]coralogixv1alpha1.AlertSetItem{
				accessPolicyAlertSetItem("alpha", tc.policy),
			})
			response := &alerts.BulkReplaceAlertDefsResponse{NotFoundIds: []string{id}}

			resultErrs := applyBulkReplaceResponse(map[string]string{id: "alpha"}, response, statuses, desired, nil)

			require.Len(t, resultErrs, 1)
			require.Equal(t, tc.state, statuses["alpha"].State)
			if tc.keepID {
				require.NotNil(t, statuses["alpha"].ID)
				require.Equal(t, id, *statuses["alpha"].ID)
			} else {
				require.Nil(t, statuses["alpha"].ID)
			}
			require.Equal(t, tc.keepTag, statuses["alpha"].AccessPolicyManaged)
			if tc.state == coralogixv1alpha1.AlertSetItemStateFailed {
				require.Contains(t, resultErrs[0].Error(), "not recreating it")
			}
		})
	}
}

func TestApplyBulkReplaceResponseClearsAccessPolicyManagedAfterDelete(t *testing.T) {
	newStatuses := func() map[string]coralogixv1alpha1.AlertSetItemStatus {
		id := "alpha-id"
		return map[string]coralogixv1alpha1.AlertSetItemStatus{
			"alpha": {Key: "alpha", ID: &id, State: coralogixv1alpha1.AlertSetItemStateSynced, AccessPolicyManaged: true},
		}
	}
	newDesired := func() map[string]coralogixv1alpha1.AlertSetItem {
		return desiredAlertSetItemsByKey([]coralogixv1alpha1.AlertSetItem{
			accessPolicyAlertSetItem("alpha", ptr.To("")),
		})
	}

	t.Run("successful delete clears the tombstone", func(t *testing.T) {
		id := "alpha-id"
		statuses := newStatuses()
		response := &alerts.BulkReplaceAlertDefsResponse{AlertDefs: []alerts.AlertDef{{Id: &id}}}

		resultErrs := applyBulkReplaceResponse(map[string]string{id: "alpha"}, response, statuses, newDesired(), nil)

		require.Empty(t, resultErrs)
		require.Equal(t, coralogixv1alpha1.AlertSetItemStateSynced, statuses["alpha"].State)
		require.False(t, statuses["alpha"].AccessPolicyManaged)
	})

	t.Run("skipped delete clears the tombstone", func(t *testing.T) {
		// The API reports a replace with unchanged properties as skipped even when it applied
		// the policy delete, and also when the policy was already absent.
		statuses := newStatuses()
		response := &alerts.BulkReplaceAlertDefsResponse{SkippedIds: []string{"alpha-id"}}

		resultErrs := applyBulkReplaceResponse(map[string]string{"alpha-id": "alpha"}, response, statuses, newDesired(), nil)

		require.Empty(t, resultErrs)
		require.Equal(t, coralogixv1alpha1.AlertSetItemStateSynced, statuses["alpha"].State)
		require.False(t, statuses["alpha"].AccessPolicyManaged)
	})
}

func TestApplyBulkReplaceResponseScopesPolicyRecordToItem(t *testing.T) {
	alphaID, betaID := "alpha-id", "beta-id"
	statuses := map[string]coralogixv1alpha1.AlertSetItemStatus{
		"alpha": {Key: "alpha", ID: &alphaID, State: coralogixv1alpha1.AlertSetItemStateSynced},
		"beta":  {Key: "beta", ID: &betaID, State: coralogixv1alpha1.AlertSetItemStateSynced},
	}
	desired := desiredAlertSetItemsByKey([]coralogixv1alpha1.AlertSetItem{
		accessPolicyAlertSetItem("alpha", nil),
		accessPolicyAlertSetItem("beta", nil),
	})
	response := &alerts.BulkReplaceAlertDefsResponse{NotFoundIds: []string{alphaID, betaID}}

	resultErrs := applyBulkReplaceResponse(
		map[string]string{alphaID: "alpha", betaID: "beta"}, response, statuses, desired,
		map[string]struct{}{"alpha": {}},
	)

	require.Len(t, resultErrs, 2)
	require.Equal(t, coralogixv1alpha1.AlertSetItemStateFailed, statuses["alpha"].State, "the recorded item may be locked out")
	require.NotNil(t, statuses["alpha"].ID)
	require.Equal(t, coralogixv1alpha1.AlertSetItemStatePending, statuses["beta"].State,
		"a policy recorded on another item must not stop a deleted policy-free item from being recreated")
	require.Nil(t, statuses["beta"].ID)
}

func TestAlertSetPolicyRecord(t *testing.T) {
	alertSet := &coralogixv1alpha1.AlertSet{Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{
		minimalAlertSetItem("alpha"), minimalAlertSetItem("beta"),
	}}}
	require.Empty(t, alertSetPolicyRecord(alertSet))

	alertSet.Annotations = map[string]string{coralogixv1beta1.AccessPolicyManagedAnnotation: `["alpha"]`}
	require.Equal(t, map[string]struct{}{"alpha": {}}, alertSetPolicyRecord(alertSet))

	alertSet.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation] = `{"version":"2025-01-01"}`
	require.Equal(t, map[string]struct{}{"alpha": {}, "beta": {}}, alertSetPolicyRecord(alertSet),
		"an unreadable record must cover every item")

	require.Nil(t, alertSetItemAnnotations(map[string]struct{}{"alpha": {}}, "beta"))
	require.Contains(t, alertSetItemAnnotations(map[string]struct{}{"alpha": {}}, "alpha"), coralogixv1beta1.AccessPolicyManagedAnnotation)
}

func TestWriteAlertSetPolicyRecord(t *testing.T) {
	alertSet := &coralogixv1alpha1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{Name: "set", Namespace: "default"},
		Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{
			minimalAlertSetItem("alpha"), minimalAlertSetItem("beta"),
		}},
	}
	initAlertSetClient(t, alertSet, nil)
	require.NoError(t, config.GetClient().Get(context.Background(), client.ObjectKeyFromObject(alertSet), alertSet))

	require.NoError(t, writeAlertSetPolicyRecord(context.Background(), alertSet, map[string]struct{}{"beta": {}, "alpha": {}, "removed": {}}))
	persisted := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, config.GetClient().Get(context.Background(), client.ObjectKeyFromObject(alertSet), persisted))
	require.Equal(t, `["alpha","beta"]`, persisted.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation],
		"keys of items no longer in the spec are dropped")

	version := alertSet.GetResourceVersion()
	require.NoError(t, writeAlertSetPolicyRecord(context.Background(), alertSet, map[string]struct{}{"alpha": {}, "beta": {}}))
	require.Equal(t, version, alertSet.GetResourceVersion(), "an unchanged record must not be rewritten")

	require.NoError(t, writeAlertSetPolicyRecord(context.Background(), alertSet, map[string]struct{}{}))
	require.NoError(t, config.GetClient().Get(context.Background(), client.ObjectKeyFromObject(alertSet), persisted))
	require.NotContains(t, persisted.Annotations, coralogixv1beta1.AccessPolicyManagedAnnotation)
}

func TestReconcileSelectorMismatchClearsPolicyRecord(t *testing.T) {
	id := "alpha-id"
	alertSet := &coralogixv1alpha1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "set",
			Namespace:   "default",
			Finalizers:  []string{alertSetFinalizer},
			Annotations: map[string]string{coralogixv1beta1.AccessPolicyManagedAnnotation: `["alpha"]`},
		},
		Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{
			accessPolicyAlertSetItem("alpha", ptr.To(testAlertSetAccessPolicy)),
		}},
		Status: coralogixv1alpha1.AlertSetStatus{Alerts: []coralogixv1alpha1.AlertSetItemStatus{
			{Key: "alpha", ID: &id, State: coralogixv1alpha1.AlertSetItemStateSynced, AccessPolicyManaged: true},
		}},
	}
	initAlertSetClient(t, alertSet, nil)
	stored := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, config.GetClient().Get(context.Background(), client.ObjectKeyFromObject(alertSet), stored))
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkDelete: func(_ context.Context, request alerts.BulkDeleteAlertDefinitionsRequest) (*alerts.BulkDeleteAlertDefsResponse, *http.Response, error) {
			return &alerts.BulkDeleteAlertDefsResponse{DeletedIds: request.Ids}, nil, nil
		},
	}}

	_, err := reconciler.reconcileSelectorMismatch(context.Background(), logr.Discard(), stored, deepCopyAlertSetStatus(stored))
	require.NoError(t, err)

	persisted := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, config.GetClient().Get(context.Background(), client.ObjectKeyFromObject(alertSet), persisted))
	require.NotContains(t, persisted.Annotations, coralogixv1beta1.AccessPolicyManagedAnnotation,
		"selector cleanup must not leave policy records for items that may be recreated later")
	require.Empty(t, persisted.Finalizers)
}

func TestReplaceAlertsRollsBackTombstonesWhenRequestIsRejected(t *testing.T) {
	for _, tc := range []struct {
		name       string
		statusCode int
		wantKept   bool
	}{
		{name: "client error rolls back", statusCode: http.StatusBadRequest, wantKept: false},
		{name: "server error keeps the record", statusCode: http.StatusInternalServerError, wantKept: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := "id-set-policy"
			alertSet := &coralogixv1alpha1.AlertSet{
				ObjectMeta: metav1.ObjectMeta{Name: "test-alert-set", Namespace: "default"},
				Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{
					accessPolicyAlertSetItem("set-policy", ptr.To(testAlertSetAccessPolicy)),
				}},
				Status: coralogixv1alpha1.AlertSetStatus{Alerts: []coralogixv1alpha1.AlertSetItemStatus{
					{Key: "set-policy", ID: &id, State: coralogixv1alpha1.AlertSetItemStateSynced},
				}},
			}
			baseClient := initAlertSetClient(t, alertSet, nil)
			stored := &coralogixv1alpha1.AlertSet{}
			require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), stored))
			reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
				bulkReplace: func(context.Context, alerts.BulkReplaceAlertDefinitionsRequest) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
					return nil, &http.Response{StatusCode: tc.statusCode, Body: http.NoBody}, errors.New("rejected")
				},
			}}
			statuses, err := alertSetStatusByKey(stored.Status.Alerts)
			require.NoError(t, err)

			_, requestErr := reconciler.replaceAlerts(context.Background(), logr.Discard(), stored,
				desiredAlertSetItemsByKey(stored.Spec.Alerts), statuses, map[string]struct{}{})

			require.Error(t, requestErr)
			require.Equal(t, tc.wantKept, statuses["set-policy"].AccessPolicyManaged)
			persisted := &coralogixv1alpha1.AlertSet{}
			require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), persisted))
			require.Equal(t, tc.wantKept, persisted.Status.Alerts[0].AccessPolicyManaged,
				"the persisted tombstone must match: a refused request applied nothing, a server error may have")
			_, recorded := persisted.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation]
			require.Equal(t, tc.wantKept, recorded, "the durable record written ahead must follow the tombstone")
		})
	}
}

func TestReplaceAlertsDoesNotRecordPerItemFailures(t *testing.T) {
	failedID, okID := "id-failed", "id-ok"
	alertSet := &coralogixv1alpha1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-alert-set", Namespace: "default"},
		Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{
			accessPolicyAlertSetItem("failed", ptr.To(testAlertSetAccessPolicy)),
			accessPolicyAlertSetItem("ok", ptr.To(testAlertSetAccessPolicy)),
		}},
		Status: coralogixv1alpha1.AlertSetStatus{Alerts: []coralogixv1alpha1.AlertSetItemStatus{
			{Key: "failed", ID: &failedID, State: coralogixv1alpha1.AlertSetItemStateSynced},
			{Key: "ok", ID: &okID, State: coralogixv1alpha1.AlertSetItemStateSynced},
		}},
	}
	baseClient := initAlertSetClient(t, alertSet, nil)
	stored := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), stored))
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkReplace: func(context.Context, alerts.BulkReplaceAlertDefinitionsRequest) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
			return &alerts.BulkReplaceAlertDefsResponse{
				AlertDefs:                []alerts.AlertDef{{Id: &okID}},
				FailedToReplaceAlertDefs: []alerts.FailedToReplaceAlertDef{{Id: &failedID, Reason: ptr.To("rejected")}},
			}, nil, nil
		},
	}}
	statuses, err := alertSetStatusByKey(stored.Status.Alerts)
	require.NoError(t, err)

	itemErrs, requestErr := reconciler.replaceAlerts(context.Background(), logr.Discard(), stored,
		desiredAlertSetItemsByKey(stored.Spec.Alerts), statuses, map[string]struct{}{})

	require.NoError(t, requestErr)
	require.Len(t, itemErrs, 1)
	require.False(t, statuses["failed"].AccessPolicyManaged, "a failed item's policy was not applied")
	require.True(t, statuses["ok"].AccessPolicyManaged)
	persisted := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), persisted))
	byKey, err := alertSetStatusByKey(persisted.Status.Alerts)
	require.NoError(t, err)
	require.False(t, byKey["failed"].AccessPolicyManaged, "the rollback must be persisted")
	require.Equal(t, `["ok"]`, persisted.Annotations[coralogixv1beta1.AccessPolicyManagedAnnotation])
}

func TestCreateAlertsDropsStaleRecordForPolicyFreeItem(t *testing.T) {
	alertSet := &coralogixv1alpha1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-alert-set",
			Namespace: "default",
			// Left by an earlier item with the same key whose cleanup never pruned it.
			Annotations: map[string]string{coralogixv1beta1.AccessPolicyManagedAnnotation: `["readded"]`},
		},
		Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{
			accessPolicyAlertSetItem("readded", nil),
		}},
	}
	baseClient := initAlertSetClient(t, alertSet, nil)
	stored := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), stored))
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkCreate: func(context.Context, alerts.BulkCreateAlertDefinitionsRequest) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error) {
			return &alerts.BulkCreateAlertDefsResponse{AlertDefs: []alerts.AlertDef{{Id: ptr.To("new-id")}}}, nil, nil
		},
	}}
	statuses := map[string]coralogixv1alpha1.AlertSetItemStatus{
		"readded": {Key: "readded", State: coralogixv1alpha1.AlertSetItemStatePending},
	}

	createdKeys, _, requestErr := reconciler.createAlerts(context.Background(), logr.Discard(), stored,
		desiredAlertSetItemsByKey(stored.Spec.Alerts), statuses)

	require.NoError(t, requestErr)
	require.Contains(t, createdKeys, "readded")
	persisted := &coralogixv1alpha1.AlertSet{}
	require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), persisted))
	require.NotContains(t, persisted.Annotations, coralogixv1beta1.AccessPolicyManagedAnnotation,
		"a policy-free item created under a recorded key must not inherit the stale record")
}

// failingAlertSetUpdateClient rejects every full-object Update without committing it.
type failingAlertSetUpdateClient struct {
	client.Client
}

func (c failingAlertSetUpdateClient) Update(context.Context, client.Object, ...client.UpdateOption) error {
	return errors.New("unavailable")
}

func TestAlertSetPolicyIsNotSentWithoutConfirmedRecord(t *testing.T) {
	newAlertSet := func(withID bool) *coralogixv1alpha1.AlertSet {
		alertSet := &coralogixv1alpha1.AlertSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-alert-set", Namespace: "default"},
			Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{
				accessPolicyAlertSetItem("set-policy", ptr.To(testAlertSetAccessPolicy)),
			}},
		}
		if withID {
			alertSet.Status.Alerts = []coralogixv1alpha1.AlertSetItemStatus{
				{Key: "set-policy", ID: ptr.To("id-set-policy"), State: coralogixv1alpha1.AlertSetItemStateSynced},
			}
		}
		return alertSet
	}
	wrap := func(c client.Client) client.Client { return failingAlertSetUpdateClient{Client: c} }

	t.Run("create", func(t *testing.T) {
		alertSet := newAlertSet(false)
		baseClient := initAlertSetClient(t, alertSet, wrap)
		stored := &coralogixv1alpha1.AlertSet{}
		require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), stored))
		called := false
		reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
			bulkCreate: func(context.Context, alerts.BulkCreateAlertDefinitionsRequest) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error) {
				called = true
				return &alerts.BulkCreateAlertDefsResponse{}, nil, nil
			},
		}}
		statuses := map[string]coralogixv1alpha1.AlertSetItemStatus{
			"set-policy": {Key: "set-policy", State: coralogixv1alpha1.AlertSetItemStatePending},
		}
		_, _, requestErr := reconciler.createAlerts(context.Background(), logr.Discard(), stored,
			desiredAlertSetItemsByKey(stored.Spec.Alerts), statuses)
		require.Error(t, requestErr)
		require.False(t, called, "no alert may be created with a policy before its durable record is confirmed")
	})

	t.Run("replace", func(t *testing.T) {
		alertSet := newAlertSet(true)
		baseClient := initAlertSetClient(t, alertSet, wrap)
		stored := &coralogixv1alpha1.AlertSet{}
		require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), stored))
		called := false
		reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
			bulkReplace: func(context.Context, alerts.BulkReplaceAlertDefinitionsRequest) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
				called = true
				return &alerts.BulkReplaceAlertDefsResponse{}, nil, nil
			},
		}}
		statuses, err := alertSetStatusByKey(stored.Status.Alerts)
		require.NoError(t, err)
		_, requestErr := reconciler.replaceAlerts(context.Background(), logr.Discard(), stored,
			desiredAlertSetItemsByKey(stored.Spec.Alerts), statuses, map[string]struct{}{})
		require.Error(t, requestErr)
		require.False(t, called, "no policy may be sent before its durable record is confirmed")
	})
}

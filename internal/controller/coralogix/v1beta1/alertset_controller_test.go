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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

type fakeAlertSetAPI struct {
	bulkCreate  func(context.Context, alerts.BulkCreateAlertDefinitionsRequest) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error)
	bulkReplace func(context.Context, alerts.BulkReplaceAlertDefinitionsRequest) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error)
	bulkDelete  func(context.Context, alerts.BulkDeleteAlertDefinitionsRequest) (*alerts.BulkDeleteAlertDefsResponse, *http.Response, error)
	delete      func(context.Context, string) (*http.Response, error)
}

type notFoundUpdateClient struct {
	client.Client
}

type conflictOnceStatusClient struct {
	client.Client
	conflict bool
}

type conflictOnceStatusWriter struct {
	client.SubResourceWriter
	parent *conflictOnceStatusClient
}

func (c *conflictOnceStatusClient) Status() client.SubResourceWriter {
	return conflictOnceStatusWriter{
		SubResourceWriter: c.Client.Status(),
		parent:            c,
	}
}

func (w conflictOnceStatusWriter) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.SubResourceUpdateOption,
) error {
	if w.parent.conflict {
		w.parent.conflict = false
		return k8serrors.NewConflict(schema.GroupResource{
			Group:    coralogixv1beta1.GroupVersion.Group,
			Resource: "alertsets",
		}, obj.GetName(), nil)
	}
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

func (c notFoundUpdateClient) Update(
	_ context.Context,
	obj client.Object,
	_ ...client.UpdateOption,
) error {
	return k8serrors.NewNotFound(schema.GroupResource{
		Group:    coralogixv1beta1.GroupVersion.Group,
		Resource: "alertsets",
	}, obj.GetName())
}

func (f fakeAlertSetAPI) BulkCreate(
	ctx context.Context,
	request alerts.BulkCreateAlertDefinitionsRequest,
) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error) {
	return f.bulkCreate(ctx, request)
}

func (f fakeAlertSetAPI) BulkReplace(
	ctx context.Context,
	request alerts.BulkReplaceAlertDefinitionsRequest,
) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
	return f.bulkReplace(ctx, request)
}

func (f fakeAlertSetAPI) BulkDelete(
	ctx context.Context,
	request alerts.BulkDeleteAlertDefinitionsRequest,
) (*alerts.BulkDeleteAlertDefsResponse, *http.Response, error) {
	return f.bulkDelete(ctx, request)
}

func (f fakeAlertSetAPI) Delete(ctx context.Context, id string) (*http.Response, error) {
	return f.delete(ctx, id)
}

func TestApplyBulkCreateResponseMapsFailuresByRequestIndex(t *testing.T) {
	statuses := map[string]coralogixv1beta1.AlertSetItemStatus{
		"alpha":   {Key: "alpha", State: coralogixv1beta1.AlertSetItemStatePending},
		"bravo":   {Key: "bravo", State: coralogixv1beta1.AlertSetItemStatePending},
		"charlie": {Key: "charlie", State: coralogixv1beta1.AlertSetItemStatePending},
	}
	createdKeys := map[string]struct{}{}
	firstID, thirdID := "first-id", "third-id"
	response := &alerts.BulkCreateAlertDefsResponse{
		AlertDefs: []alerts.AlertDef{{Id: &firstID}, {Id: &thirdID}},
		FailedToCreateAlertDefs: []alerts.FailedToCreateAlertDef{
			{Index: 1, Reason: "invalid alert"},
		},
	}

	resultErrs := applyBulkCreateResponse(
		[]string{"alpha", "bravo", "charlie"},
		response,
		statuses,
		createdKeys,
	)

	require.Len(t, resultErrs, 1)
	require.Equal(t, firstID, *statuses["alpha"].ID)
	require.Equal(t, coralogixv1beta1.AlertSetItemStateFailed, statuses["bravo"].State)
	require.Contains(t, statuses["bravo"].Message, "invalid alert")
	require.Equal(t, thirdID, *statuses["charlie"].ID)
	require.Contains(t, createdKeys, "alpha")
	require.NotContains(t, createdKeys, "bravo")
	require.Contains(t, createdKeys, "charlie")
}

func TestAlertSetStatusSnapshotDoesNotChangeWithLiveStatus(t *testing.T) {
	alertSet := &coralogixv1beta1.AlertSet{
		Status: coralogixv1beta1.AlertSetStatus{
			Conditions: []metav1.Condition{{
				Type:               "RemoteSynced",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 1,
			}},
		},
	}
	originalStatus := deepCopyAlertSetStatus(alertSet)

	alertSet.Status.Conditions[0].ObservedGeneration = 2

	require.Equal(t, int64(1), originalStatus.Conditions[0].ObservedGeneration)
	require.Equal(t, int64(2), alertSet.Status.Conditions[0].ObservedGeneration)
}

func TestPersistCreatedAlertSetStatusRecoversFromStatusConflict(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1beta1.AddToScheme(scheme))

	existingID := "existing-id"
	alertSet := &coralogixv1beta1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-alert-set",
			Namespace: "default",
		},
		Status: coralogixv1beta1.AlertSetStatus{
			Alerts: []coralogixv1beta1.AlertSetItemStatus{{
				Key: "existing",
				ID:  &existingID,
			}},
		},
	}
	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&coralogixv1beta1.AlertSet{}).
		WithObjects(alertSet.DeepCopy()).
		Build()
	conflictClient := &conflictOnceStatusClient{Client: baseClient, conflict: true}

	originalClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(originalClient) })
	config.InitClient(conflictClient)

	createdID := "created-id"
	statusByKey := map[string]coralogixv1beta1.AlertSetItemStatus{
		"created": {
			Key:   "created",
			ID:    &createdID,
			State: coralogixv1beta1.AlertSetItemStateSynced,
		},
	}

	reconciler := &AlertSetReconciler{}
	recovered, err := reconciler.persistCreatedAlertSetStatus(
		context.Background(),
		alertSet,
		statusByKey,
		map[string]struct{}{"created": {}},
	)

	require.NoError(t, err)
	require.True(t, recovered)

	stored := &coralogixv1beta1.AlertSet{}
	require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), stored))
	require.Len(t, stored.Status.Alerts, 2)
	require.Equal(t, "created", stored.Status.Alerts[0].Key)
	require.Equal(t, createdID, *stored.Status.Alerts[0].ID)
	require.Equal(t, "existing", stored.Status.Alerts[1].Key)
	require.Equal(t, existingID, *stored.Status.Alerts[1].ID)
}

func TestApplyBulkReplaceResponseHandlesAllResultTypes(t *testing.T) {
	ids := map[string]string{
		"success-id": "alpha",
		"skipped-id": "bravo",
		"failed-id":  "charlie",
		"missing-id": "delta",
	}
	statuses := make(map[string]coralogixv1beta1.AlertSetItemStatus, len(ids))
	for id, key := range ids {
		itemID := id
		statuses[key] = coralogixv1beta1.AlertSetItemStatus{
			Key:   key,
			ID:    &itemID,
			State: coralogixv1beta1.AlertSetItemStateSynced,
		}
	}
	failureReason := "invalid replacement"
	failedID := "failed-id"
	successID := "success-id"
	response := &alerts.BulkReplaceAlertDefsResponse{
		AlertDefs:  []alerts.AlertDef{{Id: &successID}},
		SkippedIds: []string{"skipped-id"},
		FailedToReplaceAlertDefs: []alerts.FailedToReplaceAlertDef{
			{Id: &failedID, Reason: &failureReason},
		},
		NotFoundIds: []string{"missing-id"},
	}

	resultErrs := applyBulkReplaceResponse(ids, response, statuses)

	require.Len(t, resultErrs, 2)
	require.Equal(t, coralogixv1beta1.AlertSetItemStateSynced, statuses["alpha"].State)
	require.Equal(t, coralogixv1beta1.AlertSetItemStateSynced, statuses["bravo"].State)
	require.Equal(t, coralogixv1beta1.AlertSetItemStateFailed, statuses["charlie"].State)
	require.Equal(t, failedID, *statuses["charlie"].ID)
	require.Nil(t, statuses["delta"].ID)
	require.Equal(t, coralogixv1beta1.AlertSetItemStatePending, statuses["delta"].State)
}

func TestCreateAlertsContinuesAfterLocalConversionFailure(t *testing.T) {
	originalClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(originalClient) })
	scheme := runtime.NewScheme()
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).Build())

	valid := minimalAlertSetItem("valid")
	invalid := minimalAlertSetItem("invalid")
	invalid.Spec.NotificationGroup = &coralogixv1beta1.NotificationGroup{
		Webhooks: []coralogixv1beta1.WebhookSettings{
			{
				Integration: coralogixv1beta1.IntegrationType{
					IntegrationRef: &coralogixv1beta1.IntegrationRef{
						ResourceRef: &coralogixv1beta1.ResourceRef{Name: "missing-webhook"},
					},
				},
			},
		},
	}
	createdID := "created-id"
	requestCount := 0
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkCreate: func(
			_ context.Context,
			request alerts.BulkCreateAlertDefinitionsRequest,
		) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error) {
			requestCount++
			require.Len(t, request.AlertDefsToCreate, 1)
			return &alerts.BulkCreateAlertDefsResponse{
				AlertDefs: []alerts.AlertDef{{Id: &createdID}},
			}, nil, nil
		},
	}}
	alertSet := &coralogixv1beta1.AlertSet{}
	alertSet.Namespace = "default"
	desired := desiredAlertSetItemsByKey([]coralogixv1beta1.AlertSetItem{valid, invalid})
	statuses := map[string]coralogixv1beta1.AlertSetItemStatus{
		"valid":   {Key: "valid", State: coralogixv1beta1.AlertSetItemStatePending},
		"invalid": {Key: "invalid", State: coralogixv1beta1.AlertSetItemStatePending},
	}

	created, itemErrs, requestErr := reconciler.createAlerts(
		context.Background(),
		logr.Discard(),
		alertSet,
		desired,
		statuses,
	)

	require.NoError(t, requestErr)
	require.Len(t, itemErrs, 1)
	require.Equal(t, 1, requestCount)
	require.Contains(t, created, "valid")
	require.Equal(t, createdID, *statuses["valid"].ID)
	require.Equal(t, coralogixv1beta1.AlertSetItemStateFailed, statuses["invalid"].State)
	require.Contains(t, statuses["invalid"].Message, "missing-webhook")
}

func TestCreateAlertsUsesOneRequestForOneHundredItems(t *testing.T) {
	items := make([]coralogixv1beta1.AlertSetItem, 100)
	statuses := make(map[string]coralogixv1beta1.AlertSetItemStatus, len(items))
	for i := range items {
		key := fmt.Sprintf("alert-%03d", i)
		items[i] = minimalAlertSetItem(key)
		statuses[key] = coralogixv1beta1.AlertSetItemStatus{
			Key:   key,
			State: coralogixv1beta1.AlertSetItemStatePending,
		}
	}

	requestCount := 0
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkCreate: func(
			_ context.Context,
			request alerts.BulkCreateAlertDefinitionsRequest,
		) (*alerts.BulkCreateAlertDefsResponse, *http.Response, error) {
			requestCount++
			require.Len(t, request.AlertDefsToCreate, 100)
			created := make([]alerts.AlertDef, len(request.AlertDefsToCreate))
			for i := range created {
				id := fmt.Sprintf("id-%03d", i)
				created[i].Id = &id
			}
			return &alerts.BulkCreateAlertDefsResponse{AlertDefs: created}, nil, nil
		},
	}}

	created, itemErrs, requestErr := reconciler.createAlerts(
		context.Background(),
		logr.Discard(),
		&coralogixv1beta1.AlertSet{},
		desiredAlertSetItemsByKey(items),
		statuses,
	)

	require.NoError(t, requestErr)
	require.Empty(t, itemErrs)
	require.Equal(t, 1, requestCount)
	require.Len(t, created, 100)
	for key, status := range statuses {
		require.NotNil(t, status.ID, key)
		require.Equal(t, coralogixv1beta1.AlertSetItemStateSynced, status.State, key)
	}
}

func TestReplaceAlertsContinuesAfterLocalConversionFailure(t *testing.T) {
	originalClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(originalClient) })
	scheme := runtime.NewScheme()
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).Build())

	valid := minimalAlertSetItem("valid")
	invalid := minimalAlertSetItem("invalid")
	invalid.Spec.NotificationGroup = &coralogixv1beta1.NotificationGroup{
		Webhooks: []coralogixv1beta1.WebhookSettings{
			{
				Integration: coralogixv1beta1.IntegrationType{
					IntegrationRef: &coralogixv1beta1.IntegrationRef{
						ResourceRef: &coralogixv1beta1.ResourceRef{Name: "missing-webhook"},
					},
				},
			},
		},
	}
	validID, invalidID := "valid-id", "invalid-id"
	requestCount := 0
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkReplace: func(
			_ context.Context,
			request alerts.BulkReplaceAlertDefinitionsRequest,
		) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
			requestCount++
			require.Len(t, request.AlertDefsToReplace, 1)
			require.Equal(t, validID, *request.AlertDefsToReplace[0].Id)
			return &alerts.BulkReplaceAlertDefsResponse{
				AlertDefs: []alerts.AlertDef{{Id: &validID}},
			}, nil, nil
		},
	}}
	statuses := map[string]coralogixv1beta1.AlertSetItemStatus{
		"valid":   {Key: "valid", ID: &validID, State: coralogixv1beta1.AlertSetItemStateSynced},
		"invalid": {Key: "invalid", ID: &invalidID, State: coralogixv1beta1.AlertSetItemStateSynced},
	}

	itemErrs, requestErr := reconciler.replaceAlerts(
		context.Background(),
		logr.Discard(),
		&coralogixv1beta1.AlertSet{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
		desiredAlertSetItemsByKey([]coralogixv1beta1.AlertSetItem{valid, invalid}),
		statuses,
		map[string]struct{}{},
	)

	require.NoError(t, requestErr)
	require.Len(t, itemErrs, 1)
	require.Equal(t, 1, requestCount)
	require.Equal(t, coralogixv1beta1.AlertSetItemStateSynced, statuses["valid"].State)
	require.Equal(t, coralogixv1beta1.AlertSetItemStateFailed, statuses["invalid"].State)
	require.Equal(t, invalidID, *statuses["invalid"].ID)
	require.Contains(t, statuses["invalid"].Message, "missing-webhook")
}

func TestReplaceAlertsUsesOneRequestForOneHundredItems(t *testing.T) {
	items := make([]coralogixv1beta1.AlertSetItem, 100)
	statuses := make(map[string]coralogixv1beta1.AlertSetItemStatus, len(items))
	for i := range items {
		key := fmt.Sprintf("alert-%03d", i)
		id := fmt.Sprintf("id-%03d", i)
		items[i] = minimalAlertSetItem(key)
		statuses[key] = coralogixv1beta1.AlertSetItemStatus{
			Key:   key,
			ID:    &id,
			State: coralogixv1beta1.AlertSetItemStateSynced,
		}
	}

	requestCount := 0
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkReplace: func(
			_ context.Context,
			request alerts.BulkReplaceAlertDefinitionsRequest,
		) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
			requestCount++
			require.Len(t, request.AlertDefsToReplace, 100)
			replaced := make([]alerts.AlertDef, len(request.AlertDefsToReplace))
			for i := range request.AlertDefsToReplace {
				replaced[i].Id = request.AlertDefsToReplace[i].Id
			}
			return &alerts.BulkReplaceAlertDefsResponse{AlertDefs: replaced}, nil, nil
		},
	}}

	itemErrs, requestErr := reconciler.replaceAlerts(
		context.Background(),
		logr.Discard(),
		&coralogixv1beta1.AlertSet{},
		desiredAlertSetItemsByKey(items),
		statuses,
		map[string]struct{}{},
	)

	require.NoError(t, requestErr)
	require.Empty(t, itemErrs)
	require.Equal(t, 1, requestCount)
	for key, status := range statuses {
		require.NotNil(t, status.ID, key)
		require.Equal(t, coralogixv1beta1.AlertSetItemStateSynced, status.State, key)
	}
}

func TestDeleteAlertsFallsBackAndPersistsPartialProgress(t *testing.T) {
	statuses := map[string]coralogixv1beta1.AlertSetItemStatus{}
	for key, id := range map[string]string{"alpha": "one", "bravo": "two", "charlie": "three"} {
		itemID := id
		statuses[key] = coralogixv1beta1.AlertSetItemStatus{Key: key, ID: &itemID}
	}
	deleteCalls := make(map[string]int)
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkDelete: func(
			context.Context,
			alerts.BulkDeleteAlertDefinitionsRequest,
		) (*alerts.BulkDeleteAlertDefsResponse, *http.Response, error) {
			return nil, &http.Response{StatusCode: http.StatusBadRequest}, errors.New("bad request")
		},
		delete: func(_ context.Context, id string) (*http.Response, error) {
			deleteCalls[id]++
			switch id {
			case "one":
				return &http.Response{StatusCode: http.StatusOK}, nil
			case "two":
				return &http.Response{StatusCode: http.StatusNotFound}, errors.New("not found")
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError}, errors.New("server error")
			}
		},
	}}

	resultErrs := reconciler.deleteAlerts(
		context.Background(),
		logr.Discard(),
		statuses,
		[]string{"alpha", "bravo", "charlie"},
	)

	require.Len(t, resultErrs, 1)
	require.NotContains(t, statuses, "alpha")
	require.NotContains(t, statuses, "bravo")
	require.Contains(t, statuses, "charlie")
	require.Equal(t, coralogixv1beta1.AlertSetItemStateDeleting, statuses["charlie"].State)
	require.Equal(t, map[string]int{"one": 1, "two": 1, "three": 1}, deleteCalls)
}

func TestDeleteAlertsDoesNotFallBackForOtherErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{name: "transport", statusCode: 0},
		{name: "unauthorized", statusCode: http.StatusUnauthorized},
		{name: "forbidden", statusCode: http.StatusForbidden},
		{name: "rate limited", statusCode: http.StatusTooManyRequests},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			id := "one"
			statuses := map[string]coralogixv1beta1.AlertSetItemStatus{
				"alpha": {Key: "alpha", ID: &id},
			}
			fallbackCalls := 0
			reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
				bulkDelete: func(
					context.Context,
					alerts.BulkDeleteAlertDefinitionsRequest,
				) (*alerts.BulkDeleteAlertDefsResponse, *http.Response, error) {
					if testCase.statusCode == 0 {
						return nil, nil, errors.New("transport error")
					}
					return nil, &http.Response{StatusCode: testCase.statusCode}, errors.New("request failed")
				},
				delete: func(context.Context, string) (*http.Response, error) {
					fallbackCalls++
					return nil, nil
				},
			}}

			resultErrs := reconciler.deleteAlerts(
				context.Background(),
				logr.Discard(),
				statuses,
				[]string{"alpha"},
			)

			require.Len(t, resultErrs, 1)
			require.Zero(t, fallbackCalls)
			require.Contains(t, statuses, "alpha")
			require.Equal(t, coralogixv1beta1.AlertSetItemStateDeleting, statuses["alpha"].State)
		})
	}
}

func TestReconcileDeletionTreatsMissingResourceAsComplete(t *testing.T) {
	originalClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(originalClient) })

	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1beta1.AddToScheme(scheme))
	now := metav1.Now()
	id := "remote-id"
	alertSet := &coralogixv1beta1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-alert-set",
			Namespace:         "default",
			Finalizers:        []string{alertSetFinalizer},
			DeletionTimestamp: &now,
		},
		Status: coralogixv1beta1.AlertSetStatus{
			Alerts: []coralogixv1beta1.AlertSetItemStatus{{
				Key:   "alpha",
				ID:    &id,
				State: coralogixv1beta1.AlertSetItemStateSynced,
			}},
		},
	}
	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&coralogixv1beta1.AlertSet{}).
		WithObjects(alertSet.DeepCopy()).
		Build()
	config.InitClient(notFoundUpdateClient{Client: baseClient})
	storedAlertSet := &coralogixv1beta1.AlertSet{}
	require.NoError(t, baseClient.Get(context.Background(), client.ObjectKeyFromObject(alertSet), storedAlertSet))

	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkDelete: func(
			_ context.Context,
			request alerts.BulkDeleteAlertDefinitionsRequest,
		) (*alerts.BulkDeleteAlertDefsResponse, *http.Response, error) {
			require.Equal(t, []string{id}, request.Ids)
			return &alerts.BulkDeleteAlertDefsResponse{DeletedIds: []string{id}}, nil, nil
		},
	}}

	result, err := reconciler.reconcileDeletion(
		context.Background(),
		logr.Discard(),
		storedAlertSet,
		deepCopyAlertSetStatus(storedAlertSet),
	)

	require.NoError(t, err)
	require.Zero(t, result)
}

func minimalAlertSetItem(key string) coralogixv1beta1.AlertSetItem {
	return coralogixv1beta1.AlertSetItem{
		Key: key,
		Spec: coralogixv1beta1.AlertSpec{
			Name:     key,
			Priority: coralogixv1beta1.AlertPriorityP5,
			TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
				LogsImmediate: &coralogixv1beta1.LogsImmediate{},
			},
		},
	}
}

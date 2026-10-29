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
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

const testAlertSetAccessPolicy = `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

func accessPolicyAlertSetItem(key string, policy *string) coralogixv1alpha1.AlertSetItem {
	item := minimalAlertSetItem(key)
	item.Spec.AccessPolicy = policy
	return item
}

func initAlertSetClient(t *testing.T, alertSet *coralogixv1alpha1.AlertSet) client.Client {
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
	config.InitClient(baseClient)
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
	baseClient := initAlertSetClient(t, alertSet)
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
}

func TestReplaceAlertsResetsNotFoundItemWithAccessPolicy(t *testing.T) {
	// The API answers not-found only for an absent alert; a denying policy answers 403. An item
	// with a policy is therefore reset to Pending like any other, so the next pass recreates it.
	items := []coralogixv1alpha1.AlertSetItem{accessPolicyAlertSetItem("alpha", ptr.To(testAlertSetAccessPolicy))}
	alertSet := &coralogixv1alpha1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-alert-set", Namespace: "default"},
		Spec:       coralogixv1alpha1.AlertSetSpec{Alerts: items},
	}
	initAlertSetClient(t, alertSet)
	reconciler := &AlertSetReconciler{api: fakeAlertSetAPI{
		bulkReplace: func(
			_ context.Context,
			request alerts.BulkReplaceAlertDefinitionsRequest,
		) (*alerts.BulkReplaceAlertDefsResponse, *http.Response, error) {
			return &alerts.BulkReplaceAlertDefsResponse{NotFoundIds: []string{*request.AlertDefsToReplace[0].Id}}, nil, nil
		},
	}}
	id := "missing-id"
	statuses := map[string]coralogixv1alpha1.AlertSetItemStatus{
		"alpha": {Key: "alpha", ID: &id, State: coralogixv1alpha1.AlertSetItemStateSynced},
	}

	itemErrs, requestErr := reconciler.replaceAlerts(
		context.Background(),
		logr.Discard(),
		alertSet,
		desiredAlertSetItemsByKey(items),
		statuses,
		map[string]struct{}{},
	)

	require.NoError(t, requestErr)
	require.Len(t, itemErrs, 1)
	require.Equal(t, coralogixv1alpha1.AlertSetItemStatePending, statuses["alpha"].State)
	require.Nil(t, statuses["alpha"].ID, "a not-found item must drop its ID so it is recreated")
}

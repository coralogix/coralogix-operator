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

package coralogixreconciler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	legacycxsdk "github.com/coralogix/coralogix-management-sdk/go"
	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

type remoteNotFoundReconciler struct {
	updateErr error
	created   *int
	deleted   *int
}

type openAPIErrorWithBody struct {
	body []byte
}

func (e openAPIErrorWithBody) Error() string {
	return string(e.body)
}

func (e openAPIErrorWithBody) Body() []byte {
	return e.body
}

func (e openAPIErrorWithBody) Model() interface{} {
	return nil
}

func (r remoteNotFoundReconciler) HandleCreation(_ context.Context, _ logr.Logger, _ client.Object) error {
	if r.created != nil {
		(*r.created)++
	}
	return nil
}

func (r remoteNotFoundReconciler) HandleUpdate(_ context.Context, _ logr.Logger, _ client.Object) error {
	return r.updateErr
}

func (r remoteNotFoundReconciler) HandleDeletion(_ context.Context, _ logr.Logger, _ client.Object) error {
	if r.deleted != nil {
		(*r.deleted)++
	}
	return nil
}

func (remoteNotFoundReconciler) FinalizerName() string {
	return "test.coralogix.com/finalizer"
}

func (remoteNotFoundReconciler) RequeueInterval() time.Duration {
	return time.Minute
}

func TestReconcileResourceClearsStatusIDAndRequeuesOnRemoteNotFound(t *testing.T) {
	tests := []struct {
		name      string
		updateErr error
	}{
		{
			name:      "grpc not found",
			updateErr: legacycxsdk.NewSdkAPIError(status.Error(codes.NotFound, "missing remote resource"), "test", "test"),
		},
		{
			name: "openapi http not found",
			updateErr: oapicxsdk.NewAPIError(
				&http.Response{StatusCode: http.StatusNotFound},
				errors.New("missing remote resource"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			scheme := runtime.NewScheme()
			require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

			oldID := "old-id"
			connector := &coralogixv1alpha1.Connector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "connector",
					Namespace: "default",
				},
				Status: coralogixv1alpha1.ConnectorStatus{
					Id: &oldID,
				},
			}

			originalClient := config.GetClient()
			originalScheme := config.GetScheme()
			originalSelector := config.GetConfig().Selector
			t.Cleanup(func() {
				config.InitClient(originalClient)
				config.InitScheme(originalScheme)
				config.GetConfig().Selector = originalSelector
			})

			config.InitScheme(scheme)
			config.InitClient(fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(connector).
				WithStatusSubresource(connector).
				Build())
			config.GetConfig().Selector = config.Selector{}
			require.NoError(t, config.GetClient().Status().Update(ctx, connector))

			result, err := ReconcileResource(
				ctx,
				ctrl.Request{NamespacedName: client.ObjectKeyFromObject(connector)},
				&coralogixv1alpha1.Connector{},
				remoteNotFoundReconciler{updateErr: tt.updateErr},
			)
			require.NoError(t, err)
			require.Equal(t, time.Second, result.RequeueAfter)

			updated := &coralogixv1alpha1.Connector{}
			require.NoError(t, config.GetClient().Get(ctx, client.ObjectKeyFromObject(connector), updated))
			require.Nil(t, updated.Status.Id)
		})
	}
}

func TestReconcileResourceFallsBackWhenRemoteNotFoundDoesNotClearID(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

	ruleSet := &coralogixv1alpha1.QuotaAllocationRuleSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "quota-allocation-rule-set",
			Namespace: "default",
		},
	}

	originalClient := config.GetClient()
	originalScheme := config.GetScheme()
	originalSelector := config.GetConfig().Selector
	t.Cleanup(func() {
		config.InitClient(originalClient)
		config.InitScheme(originalScheme)
		config.GetConfig().Selector = originalSelector
	})

	config.InitScheme(scheme)
	config.InitClient(fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ruleSet).
		WithStatusSubresource(ruleSet).
		Build())
	config.GetConfig().Selector = config.Selector{}

	result, err := ReconcileResource(
		ctx,
		ctrl.Request{NamespacedName: client.ObjectKeyFromObject(ruleSet)},
		&coralogixv1alpha1.QuotaAllocationRuleSet{},
		remoteNotFoundReconciler{
			updateErr: oapicxsdk.NewAPIError(
				&http.Response{StatusCode: http.StatusNotFound},
				errors.New("missing remote resource"),
			),
		},
	)
	require.Error(t, err)
	require.Empty(t, result)

	updated := &coralogixv1alpha1.QuotaAllocationRuleSet{}
	require.NoError(t, config.GetClient().Get(ctx, client.ObjectKeyFromObject(ruleSet), updated))
	require.Equal(t, "RemoteUnsynced", updated.Status.PrintableStatus)
}

func TestReconcileResourceDoesNotRecoverGenericOpenAPINotFound(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

	oldID := "old-id"
	connector := &coralogixv1alpha1.Connector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "connector",
			Namespace: "default",
		},
		Status: coralogixv1alpha1.ConnectorStatus{
			Id: &oldID,
		},
	}

	originalClient := config.GetClient()
	originalScheme := config.GetScheme()
	originalSelector := config.GetConfig().Selector
	t.Cleanup(func() {
		config.InitClient(originalClient)
		config.InitScheme(originalScheme)
		config.GetConfig().Selector = originalSelector
	})

	config.InitScheme(scheme)
	config.InitClient(fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(connector).
		WithStatusSubresource(connector).
		Build())
	config.GetConfig().Selector = config.Selector{}
	require.NoError(t, config.GetClient().Status().Update(ctx, connector))

	result, err := ReconcileResource(
		ctx,
		ctrl.Request{NamespacedName: client.ObjectKeyFromObject(connector)},
		&coralogixv1alpha1.Connector{},
		remoteNotFoundReconciler{
			updateErr: oapicxsdk.NewAPIError(
				&http.Response{StatusCode: http.StatusNotFound},
				openAPIErrorWithBody{body: []byte("Not Found: Not Found")},
			),
		},
	)
	require.Error(t, err)
	require.Empty(t, result)

	updated := &coralogixv1alpha1.Connector{}
	require.NoError(t, config.GetClient().Get(ctx, client.ObjectKeyFromObject(connector), updated))
	require.NotNil(t, updated.Status.Id)
	require.Equal(t, oldID, *updated.Status.Id)
}

func TestReconcileResourceDoesNotCreateMissingIDObjectThatDoesNotMatchSelector(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

	connector := &coralogixv1alpha1.Connector{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "connector",
			Namespace:  "default",
			Labels:     map[string]string{"managed": "false"},
			Finalizers: []string{"test.coralogix.com/finalizer"},
		},
	}

	originalClient := config.GetClient()
	originalScheme := config.GetScheme()
	originalSelector := config.GetConfig().Selector
	t.Cleanup(func() {
		config.InitClient(originalClient)
		config.InitScheme(originalScheme)
		config.GetConfig().Selector = originalSelector
	})

	config.InitScheme(scheme)
	config.InitClient(fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(connector).
		WithStatusSubresource(connector).
		Build())
	config.GetConfig().Selector = config.Selector{
		LabelSelector: labels.SelectorFromSet(labels.Set{"managed": "true"}),
	}

	created := 0
	deleted := 0
	result, err := ReconcileResource(
		ctx,
		ctrl.Request{NamespacedName: client.ObjectKeyFromObject(connector)},
		&coralogixv1alpha1.Connector{},
		remoteNotFoundReconciler{
			created: &created,
			deleted: &deleted,
		},
	)
	require.NoError(t, err)
	require.Empty(t, result)
	require.Zero(t, created)
	require.Zero(t, deleted)

	updated := &coralogixv1alpha1.Connector{}
	require.NoError(t, config.GetClient().Get(ctx, client.ObjectKeyFromObject(connector), updated))
	require.Empty(t, updated.Finalizers)
	require.Nil(t, updated.Status.Id)
}

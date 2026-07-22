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
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// noopReconciler is a stub CoralogixReconciler used to drive ReconcileResource in tests
// without talking to a real Coralogix backend.
type noopReconciler struct {
	deletionCalls int
	creationCalls int
}

func (n *noopReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	n.creationCalls++
	return nil
}

func (n *noopReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	return nil
}

func (n *noopReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	n.deletionCalls++
	return nil
}

func (n *noopReconciler) FinalizerName() string {
	return "dashboard.coralogix.com/finalizer"
}

func (n *noopReconciler) RequeueInterval() time.Duration {
	return time.Minute
}

func TestReconcileResourceSelectorMismatchPreservesDashboardImported(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

	dashboardID := "some-remote-id"
	dashboard := &coralogixv1alpha1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dashboard-import",
			Namespace: "default",
			Annotations: map[string]string{
				coralogixv1alpha1.ImportDashboardIDAnnotationKey: dashboardID,
			},
		},
		Status: coralogixv1alpha1.DashboardStatus{
			ID:       &dashboardID,
			Imported: true,
			Conditions: []metav1.Condition{
				{
					Type:               "RemoteSynced",
					Status:             metav1.ConditionTrue,
					Reason:             "RemoteSyncedSuccessfully",
					Message:            "synced",
					LastTransitionTime: metav1.Now(),
				},
			},
			PrintableStatus: "RemoteSynced",
		},
	}
	controllerutil.AddFinalizer(dashboard, (&noopReconciler{}).FinalizerName())

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(dashboard).
		WithStatusSubresource(dashboard).
		Build()

	originalClient := config.GetClient()
	originalScheme := config.GetScheme()
	originalSelector := config.GetConfig().Selector
	t.Cleanup(func() {
		config.InitClient(originalClient)
		config.InitScheme(originalScheme)
		config.GetConfig().Selector = originalSelector
	})

	config.InitClient(fakeClient)
	config.InitScheme(scheme)
	// The Dashboard carries no labels, so this selector never matches it -
	// simulating the CR having just fallen out of the operator's scope.
	config.GetConfig().Selector.LabelSelector = labels.SelectorFromSet(labels.Set{"team": "alpha"})

	reconciler := &noopReconciler{}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: dashboard.Name, Namespace: dashboard.Namespace}}

	_, err := ReconcileResource(context.Background(), req, &coralogixv1alpha1.Dashboard{}, reconciler)
	require.NoError(t, err)
	require.Equal(t, 1, reconciler.deletionCalls)

	fetched := &coralogixv1alpha1.Dashboard{}
	require.NoError(t, fakeClient.Get(context.Background(), req.NamespacedName, fetched))

	require.Nil(t, fetched.Status.ID)
	require.Empty(t, fetched.Status.Conditions)
	require.Empty(t, fetched.Status.PrintableStatus)
	require.True(t, fetched.Status.Imported, "status.imported must survive a selector-mismatch status clear")
}

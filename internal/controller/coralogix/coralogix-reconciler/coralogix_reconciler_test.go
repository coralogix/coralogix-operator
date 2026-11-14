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
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// noopReconciler is a stub CoralogixReconciler used to drive ReconcileResource in tests
// without talking to a real Coralogix backend.
type noopReconciler struct {
	deletionCalls int
	creationCalls int
	// createID, when set, is written to the object's status by HandleCreation to
	// mimic the remote backend assigning an ID on a successful create.
	createID string
}

func (n *noopReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	n.creationCalls++
	if n.createID != "" {
		if d, ok := obj.(*coralogixv1alpha1.Dashboard); ok {
			id := n.createID
			d.Status.ID = &id
		}
	}
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

// skippingReconciler is a noopReconciler that opts out of the creation rollback,
// mirroring reconcilers like the imported Dashboard or the archive-target
// singletons.
type skippingReconciler struct {
	*noopReconciler
}

func (s *skippingReconciler) SkipCreationRollback(client.Object) bool { return true }

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

// TestReconcileResourceCreationPersistsID covers the happy path of the creation
// branch: a successful remote create must have its ID persisted to the status so
// the next reconcile takes the update path instead of creating a duplicate.
func TestReconcileResourceCreationPersistsID(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

	dashboard := &coralogixv1alpha1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{Name: "dashboard-create", Namespace: "default"},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(dashboard).
		WithStatusSubresource(dashboard).
		Build()

	restore := swapConfig(t, fakeClient, scheme)
	defer restore()

	reconciler := &noopReconciler{createID: "new-remote-id"}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: dashboard.Name, Namespace: dashboard.Namespace}}

	_, err := ReconcileResource(context.Background(), req, &coralogixv1alpha1.Dashboard{}, reconciler)
	require.NoError(t, err)
	require.Equal(t, 1, reconciler.creationCalls)
	require.Equal(t, 0, reconciler.deletionCalls)

	fetched := &coralogixv1alpha1.Dashboard{}
	require.NoError(t, fakeClient.Get(context.Background(), req.NamespacedName, fetched))
	require.NotNil(t, fetched.Status.ID)
	require.Equal(t, "new-remote-id", *fetched.Status.ID)
	require.True(t, controllerutil.ContainsFinalizer(fetched, reconciler.FinalizerName()))
}

// TestReconcileResourceStatusUpdateFailureRollsBackRemoteResource covers #595:
// if persisting the remote ID to the status fails after a successful create, the
// remote resource must be deleted (rolled back) so the next reconcile doesn't see
// a missing ID and create a duplicate remote resource.
func TestReconcileResourceStatusUpdateFailureRollsBackRemoteResource(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

	dashboard := &coralogixv1alpha1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{Name: "dashboard-create-fail", Namespace: "default"},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(dashboard).
		WithStatusSubresource(dashboard).
		WithInterceptorFuncs(interceptor.Funcs{
			// Simulate the apiserver being unable to write the status subresource,
			// which is the real #595 scenario: the ID is never persisted.
			SubResourceUpdate: func(_ context.Context, _ client.Client, subResourceName string, _ client.Object, _ ...client.SubResourceUpdateOption) error {
				if subResourceName == "status" {
					return apierrors.NewInternalError(fmt.Errorf("simulated status update failure"))
				}
				return nil
			},
		}).
		Build()

	restore := swapConfig(t, fakeClient, scheme)
	defer restore()

	reconciler := &noopReconciler{createID: "orphan-remote-id"}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: dashboard.Name, Namespace: dashboard.Namespace}}

	_, err := ReconcileResource(context.Background(), req, &coralogixv1alpha1.Dashboard{}, reconciler)
	require.Error(t, err)
	require.Equal(t, 1, reconciler.creationCalls)
	require.Equal(t, 1, reconciler.deletionCalls, "remote resource must be rolled back when status persist fails")

	// The ID was never persisted, so the next reconcile would re-enter the
	// creation branch - which is safe only because the remote resource was deleted.
	fetched := &coralogixv1alpha1.Dashboard{}
	require.NoError(t, fakeClient.Get(context.Background(), req.NamespacedName, fetched))
	require.Nil(t, fetched.Status.ID)
}

// TestReconcileResourceStatusUpdateFailureSkipsRollbackWhenOptedOut covers the
// per-object opt-out: a reconciler that implements CreationRollbackSkipper (e.g.
// an imported Dashboard or an archive-target singleton) must NOT have its remote
// resource deleted when the post-creation status write fails.
func TestReconcileResourceStatusUpdateFailureSkipsRollbackWhenOptedOut(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

	dashboard := &coralogixv1alpha1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{Name: "dashboard-skip-rollback", Namespace: "default"},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(dashboard).
		WithStatusSubresource(dashboard).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(_ context.Context, _ client.Client, subResourceName string, _ client.Object, _ ...client.SubResourceUpdateOption) error {
				if subResourceName == "status" {
					return apierrors.NewInternalError(fmt.Errorf("simulated status update failure"))
				}
				return nil
			},
		}).
		Build()

	restore := swapConfig(t, fakeClient, scheme)
	defer restore()

	reconciler := &skippingReconciler{noopReconciler: &noopReconciler{createID: "adopted-remote-id"}}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: dashboard.Name, Namespace: dashboard.Namespace}}

	_, err := ReconcileResource(context.Background(), req, &coralogixv1alpha1.Dashboard{}, reconciler)
	require.Error(t, err)
	require.Equal(t, 1, reconciler.creationCalls)
	require.Equal(t, 0, reconciler.deletionCalls, "opted-out resource must not be rolled back on status persist failure")
}

// swapConfig points the package-level config at the given fake client/scheme for
// the duration of a test and returns a function that restores the originals.
func swapConfig(t *testing.T, c client.Client, scheme *runtime.Scheme) func() {
	t.Helper()
	originalClient := config.GetClient()
	originalScheme := config.GetScheme()
	config.InitClient(c)
	config.InitScheme(scheme)
	return func() {
		config.InitClient(originalClient)
		config.InitScheme(originalScheme)
	}
}

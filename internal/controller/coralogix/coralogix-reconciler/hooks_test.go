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

package coralogixreconciler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	oapisdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// These tests drive ReconcileResource with a fake reconciler that implements the optional
// interfaces of the hooks of generated kinds. A Dashboard is the vehicle: its status has a spare
// field, imported, that a hook can change.

const remoteID = "remote-id"

// hookReconciler is a fake generated kind with hooks. Each function is one step of the lifecycle.
type hookReconciler struct {
	// create sets status.id and returns the follow-up step.
	create   func(obj *coralogixv1alpha1.Dashboard) (func(context.Context) error, error)
	update   func(obj *coralogixv1alpha1.Dashboard) error
	deletion func(obj *coralogixv1alpha1.Dashboard) error
	// saves makes the reconciler a HookStatusSaver.
	saves bool
	// statusFields are the status fields of the reconciler that a selector mismatch removes.
	statusFields [][]string

	creations, followUps, updates, deletions int
}

func (h *hookReconciler) HandleCreation(context.Context, logr.Logger, client.Object) error {
	return errors.New("the shared reconciler must call HandleCreationWithFollowUp")
}

func (h *hookReconciler) HandleCreationWithFollowUp(_ context.Context, _ logr.Logger, obj client.Object) (func(context.Context) error, error) {
	h.creations++
	followUp, err := h.create(obj.(*coralogixv1alpha1.Dashboard))
	if err != nil || followUp == nil {
		return followUp, err
	}
	return func(ctx context.Context) error {
		h.followUps++
		return followUp(ctx)
	}, nil
}

func (h *hookReconciler) HandleUpdate(_ context.Context, _ logr.Logger, obj client.Object) error {
	h.updates++
	if h.update == nil {
		return nil
	}
	return h.update(obj.(*coralogixv1alpha1.Dashboard))
}

func (h *hookReconciler) HandleDeletion(_ context.Context, _ logr.Logger, obj client.Object) error {
	h.deletions++
	if h.deletion == nil {
		return nil
	}
	return h.deletion(obj.(*coralogixv1alpha1.Dashboard))
}

func (h *hookReconciler) FinalizerName() string          { return "dashboard.coralogix.com/finalizer" }
func (h *hookReconciler) RequeueInterval() time.Duration { return time.Minute }
func (h *hookReconciler) SavesHookStatus() bool          { return h.saves }
func (h *hookReconciler) RemoteStatusFields() [][]string { return h.statusFields }

// createWithID is a Create that sets status.id and has the given follow-up step.
func createWithID(followUp func(*coralogixv1alpha1.Dashboard) error) func(*coralogixv1alpha1.Dashboard) (func(context.Context) error, error) {
	return func(obj *coralogixv1alpha1.Dashboard) (func(context.Context) error, error) {
		obj.Status = coralogixv1alpha1.DashboardStatus{ID: ptr(remoteID)}
		if followUp == nil {
			return nil, nil
		}
		return func(context.Context) error { return followUp(obj) }, nil
	}
}

func ptr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

func isTrue(b *bool) bool { return b != nil && *b }

// newDashboard is a Dashboard in the state that a successful run leaves: the RemoteSynced
// condition is True, so the shared reconciler saves no status for the condition alone. A test then
// sees only the saves of the hook status.
func newDashboard(withID bool) *coralogixv1alpha1.Dashboard {
	dashboard := &coralogixv1alpha1.Dashboard{ObjectMeta: metav1.ObjectMeta{Name: "dashboard", Namespace: "default"}}
	if !withID {
		return dashboard
	}
	dashboard.Status = coralogixv1alpha1.DashboardStatus{
		ID:              ptr(remoteID),
		PrintableStatus: "RemoteSynced",
		Conditions: []metav1.Condition{{
			Type:               utils.ConditionTypeRemoteSynced,
			Status:             metav1.ConditionTrue,
			Reason:             utils.ReasonRemoteSyncedSuccessfully,
			Message:            "Remote resource synced",
			LastTransitionTime: metav1.Now(),
		}},
	}
	controllerutil.AddFinalizer(dashboard, (&hookReconciler{}).FinalizerName())
	return dashboard
}

// hookTest is a fake cluster with one Dashboard.
type hookTest struct {
	t      *testing.T
	client client.Client
	req    ctrl.Request
	// statusUpdates counts the status writes of the shared reconciler.
	statusUpdates int
}

func newHookTest(t *testing.T, dashboard *coralogixv1alpha1.Dashboard) *hookTest {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))
	test := &hookTest{t: t, req: ctrl.Request{NamespacedName: types.NamespacedName{Name: dashboard.Name, Namespace: dashboard.Namespace}}}
	test.client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(dashboard).WithStatusSubresource(dashboard).
		WithInterceptorFuncs(interceptor.Funcs{SubResourceUpdate: func(ctx context.Context, c client.Client, subResource string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
			test.statusUpdates++
			return c.SubResource(subResource).Update(ctx, obj, opts...)
		}}).Build()

	originalClient, originalScheme := config.GetClient(), config.GetScheme()
	t.Cleanup(func() {
		config.InitClient(originalClient)
		config.InitScheme(originalScheme)
	})
	config.InitClient(test.client)
	config.InitScheme(scheme)
	return test
}

func (h *hookTest) reconcile(r CoralogixReconciler) error {
	h.t.Helper()
	_, err := ReconcileResource(context.Background(), h.req, &coralogixv1alpha1.Dashboard{}, r)
	return err
}

func (h *hookTest) get() *coralogixv1alpha1.Dashboard {
	h.t.Helper()
	dashboard := &coralogixv1alpha1.Dashboard{}
	require.NoError(h.t, h.client.Get(context.Background(), h.req.NamespacedName, dashboard))
	return dashboard
}

func syncedReason(dashboard *coralogixv1alpha1.Dashboard) string {
	return utils.GetReasonForRemoteSyncedCondition(dashboard.Status.Conditions)
}

// A failed afterCreate keeps status.id, so the next run does not create a second remote resource.
// The next run takes the Update path, which is where afterUpdate repairs what afterCreate missed.
func TestFailedCreateFollowUpKeepsStatusIDAndTheNextRunUpdates(t *testing.T) {
	test := newHookTest(t, newDashboard(false))
	afterCreateFails := true
	r := &hookReconciler{saves: true}
	r.create = createWithID(func(*coralogixv1alpha1.Dashboard) error {
		return NewHookError(errors.New("afterCreate failed"))
	})
	r.update = func(*coralogixv1alpha1.Dashboard) error {
		afterCreateFails = false // afterUpdate repairs it
		return nil
	}

	require.ErrorContains(t, test.reconcile(r), "afterCreate failed")
	got := test.get()
	require.Equal(t, ptr(remoteID), got.Status.ID, "the id of the created resource must stay")
	require.Contains(t, got.Finalizers, r.FinalizerName())
	require.Equal(t, utils.ReasonHookFailed, syncedReason(got))
	require.Equal(t, 1, r.creations)
	require.Equal(t, 1, r.followUps)

	require.NoError(t, test.reconcile(r))
	require.Equal(t, 1, r.creations, "the second run must not create again")
	require.Equal(t, 1, r.updates)
	require.False(t, afterCreateFails)
	require.Equal(t, utils.ReasonRemoteSyncedSuccessfully, syncedReason(test.get()))
}

// The follow-up step runs after status.id and the finalizer are saved.
func TestCreateFollowUpRunsAfterTheIDAndTheFinalizerAreSaved(t *testing.T) {
	test := newHookTest(t, newDashboard(false))
	r := &hookReconciler{saves: true}
	var seen *coralogixv1alpha1.Dashboard
	r.create = createWithID(func(*coralogixv1alpha1.Dashboard) error {
		seen = test.get() // what the cluster holds while the step runs
		return nil
	})

	require.NoError(t, test.reconcile(r))
	require.NotNil(t, seen)
	require.Equal(t, ptr(remoteID), seen.Status.ID)
	require.Contains(t, seen.Finalizers, r.FinalizerName())
}

// A Create run writes the status for status.id and for the condition. A reconciler that opts in
// has one more write: the status that its follow-up step changed.
func TestCreateFollowUpStatusIsSavedWhenTheReconcilerOptsIn(t *testing.T) {
	for _, tt := range []struct {
		saves            bool
		wantStatusWrites int
	}{{saves: true, wantStatusWrites: 3}, {saves: false, wantStatusWrites: 2}} {
		t.Run(fmt.Sprintf("saves=%v", tt.saves), func(t *testing.T) {
			test := newHookTest(t, newDashboard(false))
			r := &hookReconciler{saves: tt.saves}
			r.create = createWithID(func(obj *coralogixv1alpha1.Dashboard) error {
				obj.Status.Imported = boolPtr(true)
				return nil
			})
			require.NoError(t, test.reconcile(r))
			require.Equal(t, tt.wantStatusWrites, test.statusUpdates)
		})
	}
}

// A follow-up step that changes no status causes no extra write.
func TestCreateFollowUpWithoutStatusChangeSavesNothingExtra(t *testing.T) {
	test := newHookTest(t, newDashboard(false))
	r := &hookReconciler{saves: true}
	r.create = createWithID(func(*coralogixv1alpha1.Dashboard) error { return nil })
	require.NoError(t, test.reconcile(r))
	require.Equal(t, 2, test.statusUpdates)
}

// A status change of HandleUpdate is saved only for a reconciler that opts in. A handwritten kind
// relies on the opposite: SLO changes status.revision in HandleUpdate and does not save it.
func TestUpdateStatusIsSavedOnlyWhenTheReconcilerOptsIn(t *testing.T) {
	for _, tt := range []struct {
		saves     bool
		wantSaved bool
	}{{saves: true, wantSaved: true}, {saves: false, wantSaved: false}} {
		t.Run(fmt.Sprintf("saves=%v", tt.saves), func(t *testing.T) {
			test := newHookTest(t, newDashboard(true))
			r := &hookReconciler{saves: tt.saves}
			r.update = func(obj *coralogixv1alpha1.Dashboard) error {
				obj.Status.Imported = boolPtr(true)
				return nil
			}
			require.NoError(t, test.reconcile(r))
			require.Equal(t, tt.wantSaved, isTrue(test.get().Status.Imported))
		})
	}
}

// The conditions are a slice. A change in place is a change, and an unchanged status is not saved.
func TestUpdateSavesAConditionThatAHookChangedInPlace(t *testing.T) {
	for _, tt := range []struct {
		name       string
		change     func(*coralogixv1alpha1.Dashboard)
		wantWrites int
	}{
		{name: "changed in place", change: func(obj *coralogixv1alpha1.Dashboard) { obj.Status.Conditions[1].Message = "changed by a hook" }, wantWrites: 1},
		{name: "unchanged", change: func(*coralogixv1alpha1.Dashboard) {}, wantWrites: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dashboard := newDashboard(true)
			dashboard.Status.Conditions = append(dashboard.Status.Conditions, metav1.Condition{
				Type: "Extra", Status: metav1.ConditionTrue, Reason: "Extra", Message: "set before", LastTransitionTime: metav1.Now(),
			})
			test := newHookTest(t, dashboard)
			r := &hookReconciler{saves: true, update: func(obj *coralogixv1alpha1.Dashboard) error {
				tt.change(obj)
				return nil
			}}
			require.NoError(t, test.reconcile(r))
			require.Equal(t, tt.wantWrites, test.statusUpdates)
			if tt.wantWrites > 0 {
				require.Equal(t, "changed by a hook", test.get().Status.Conditions[1].Message)
			}
		})
	}
}

// A hook error is not a remote NotFound. A hook can call another API and get a 404 while the
// resource itself is still there. The shared reconciler must keep status.id, or the next run
// creates a duplicate.
func TestHookErrorThatWrapsNotFoundKeepsStatusID(t *testing.T) {
	// A resource that the API confirms as missing: 404 with a "Not Found: <message>" body.
	notFound := &oapisdk.APIError{StatusCode: http.StatusNotFound, Body: []byte("Not Found: no such object"), Err: errors.New("404 Not Found")}
	for _, tt := range []struct {
		name       string
		err        error
		wantID     bool
		wantReason string
	}{
		{name: "remote NotFound removes the id", err: notFound, wantID: false},
		{name: "hook error keeps the id", err: NewHookError(fmt.Errorf("afterUpdate: %w", notFound)), wantID: true, wantReason: utils.ReasonHookFailed},
		{name: "validation error keeps the id", err: NewValidationError(fmt.Errorf("validate: %w", notFound)), wantID: true, wantReason: utils.ReasonValidationFailed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			test := newHookTest(t, newDashboard(true))
			r := &hookReconciler{saves: true, update: func(*coralogixv1alpha1.Dashboard) error { return tt.err }}
			err := test.reconcile(r)
			if tt.wantID {
				// A real NotFound ends in a requeue after the status conflict, so it has no error.
				require.Error(t, err)
			}
			got := test.get()
			require.Equal(t, tt.wantID, got.Status.ID != nil, "status.id")
			if tt.wantReason != "" {
				require.Equal(t, tt.wantReason, syncedReason(got))
			}
		})
	}
}

// A validate error stops the Create before a remote resource exists: no id, no finalizer.
func TestValidationErrorOnCreateLeavesNoIDAndNoFinalizer(t *testing.T) {
	test := newHookTest(t, newDashboard(false))
	r := &hookReconciler{saves: true}
	r.create = func(*coralogixv1alpha1.Dashboard) (func(context.Context) error, error) {
		return nil, NewValidationError(errors.New("only one resource is allowed"))
	}
	require.ErrorContains(t, test.reconcile(r), "only one resource is allowed")
	got := test.get()
	require.Nil(t, got.Status.ID)
	require.Empty(t, got.Finalizers)
	require.Equal(t, utils.ReasonValidationFailed, syncedReason(got))
}

// A failed beforeDelete keeps the finalizer, so the next run runs the hook again.
func TestFailedDeleteHookKeepsTheFinalizer(t *testing.T) {
	dashboard := newDashboard(true)
	now := metav1.Now()
	dashboard.DeletionTimestamp = &now
	test := newHookTest(t, dashboard)
	r := &hookReconciler{saves: true, deletion: func(*coralogixv1alpha1.Dashboard) error {
		return NewHookError(errors.New("beforeDelete failed"))
	}}

	require.ErrorContains(t, test.reconcile(r), "beforeDelete failed")
	got := test.get()
	require.Contains(t, got.Finalizers, r.FinalizerName(), "the finalizer must stay")
	require.Equal(t, utils.ReasonHookFailed, syncedReason(got))

	r.deletion = nil
	require.NoError(t, test.reconcile(r))
	require.Equal(t, 2, r.deletions, "the hook runs again after a failure")
	var gone coralogixv1alpha1.Dashboard
	require.True(t, client.IgnoreNotFound(test.client.Get(context.Background(), test.req.NamespacedName, &gone)) == nil)
}

// A selector mismatch removes the status fields that the reconciler lists, and the fixed fields.
// A field that the reconciler does not list survives, as Dashboard status.imported does.
func TestSelectorMismatchRemovesTheStatusFieldsOfTheReconciler(t *testing.T) {
	for _, tt := range []struct {
		name         string
		statusFields [][]string
		wantImported bool
	}{
		{name: "listed field is removed", statusFields: [][]string{{"status", "imported"}}, wantImported: false},
		{name: "unlisted field survives", statusFields: nil, wantImported: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dashboard := newDashboard(true)
			dashboard.Status.Imported = boolPtr(true)
			test := newHookTest(t, dashboard)
			originalSelector := config.GetConfig().Selector
			t.Cleanup(func() { config.GetConfig().Selector = originalSelector })
			config.GetConfig().Selector.LabelSelector = mustSelector(t, "team=alpha")

			r := &hookReconciler{saves: true, statusFields: tt.statusFields}
			require.NoError(t, test.reconcile(r))
			require.Equal(t, 1, r.deletions)
			got := test.get()
			require.Nil(t, got.Status.ID)
			require.Equal(t, tt.wantImported, isTrue(got.Status.Imported))
		})
	}
}

// A reconciler with status fields of its own owns its cleanup list. A field that it leaves out
// survives, even when a handwritten kind has the same name in the older fixed list.
func TestRemoteStatusFieldsHonorKeepOnCleanupForProviders(t *testing.T) {
	fixed := [][]string{{"status", "id"}, {"status", "conditions"}, {"status", "printableStatus"}}
	provider := &hookReconciler{statusFields: [][]string{{"status", "revision"}}} // externalId is kept
	require.Equal(t, append(fixed, []string{"status", "revision"}), remoteStatusFields(provider))
	require.NotContains(t, remoteStatusFields(provider), []string{"status", "externalId"})

	empty := &hookReconciler{} // all fields are kept
	require.Equal(t, fixed, remoteStatusFields(empty))

	// A handwritten kind has no provider and keeps the older list.
	legacy := remoteStatusFields(&noopReconciler{})
	for _, path := range [][]string{{"status", "accessPolicyManaged"}, {"status", "externalId"}, {"status", "revision"}} {
		require.Contains(t, legacy, path)
	}
}

func TestHookErrorKeepsItsReasonAndTheCauseOfTheError(t *testing.T) {
	cause := errors.New("cause")
	hookErr := NewHookError(fmt.Errorf("wrapped: %w", cause))
	reason, ok := hookReason(fmt.Errorf("outer: %w", hookErr))
	require.True(t, ok)
	require.Equal(t, utils.ReasonHookFailed, reason)
	require.ErrorIs(t, hookErr, cause)
	reason, ok = hookReason(NewValidationError(cause))
	require.True(t, ok)
	require.Equal(t, utils.ReasonValidationFailed, reason)
	_, ok = hookReason(cause)
	require.False(t, ok)
}

func mustSelector(t *testing.T, expression string) labels.Selector {
	t.Helper()
	selector, err := labels.Parse(expression)
	require.NoError(t, err)
	return selector
}

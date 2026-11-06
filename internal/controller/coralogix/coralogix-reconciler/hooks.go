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
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// The types in this file support the hooks of generated kinds (tools/opgen). A kind that has no
// hook implements none of the optional interfaces, so the shared reconciler treats it as before.

// CreationFollowUp is a reconciler whose Create has a step that must run after status.id is saved.
//
// HandleCreationWithFollowUp does the remote Create and sets status.id, as HandleCreation does. It
// also returns the step. The shared reconciler saves the status, adds the finalizer, and then runs
// the step. When the step fails, status.id stays, so the next run takes the Update path and does
// not create a second remote resource.
type CreationFollowUp interface {
	HandleCreationWithFollowUp(ctx context.Context, log logr.Logger, obj client.Object) (followUp func(context.Context) error, err error)
}

// HookStatusSaver is a reconciler whose hooks can change the status. For such a reconciler the
// shared reconciler compares the status before and after HandleUpdate (and before and after the
// Create follow-up) and saves it when it changed.
//
// A reconciler that does not implement it keeps its status changes of HandleUpdate unsaved. Some
// handwritten kinds rely on that: SLO changes status.revision in HandleUpdate on purpose.
type HookStatusSaver interface {
	SavesHookStatus() bool
}

// RemoteStatusFieldsProvider is a reconciler with status fields of its own, beyond the fixed
// fields that the selector-mismatch cleanup removes. Each item is a path, such as
// {"status", "externalId"}. A field that must survive the cleanup (such as Dashboard
// status.imported) is not listed.
type RemoteStatusFieldsProvider interface {
	RemoteStatusFields() [][]string
}

// HookError is an error of a hook. Reason is the reason of the RemoteSynced condition.
//
// A hook error is never a remote NotFound. A hook can call another API and get a 404 from it. The
// resource itself is still there, so the shared reconciler must not remove status.id and create it
// again.
type HookError struct {
	Reason string
	Err    error
}

func (e *HookError) Error() string { return e.Err.Error() }

func (e *HookError) Unwrap() error { return e.Err }

// NewValidationError marks err as the error of a validate hook: reason ValidationFailed.
func NewValidationError(err error) error {
	return &HookError{Reason: utils.ReasonValidationFailed, Err: err}
}

// NewHookError marks err as the error of any other hook: reason HookFailed.
func NewHookError(err error) error {
	return &HookError{Reason: utils.ReasonHookFailed, Err: err}
}

// hookReason returns the reason of a hook error in the chain of err.
func hookReason(err error) (string, bool) {
	var hookErr *HookError
	if errors.As(err, &hookErr) {
		return hookErr.Reason, true
	}
	return "", false
}

// reasonOrDefault returns the reason of a hook error, or fallback for any other error.
func reasonOrDefault(err error, fallback string) string {
	if reason, ok := hookReason(err); ok {
		return reason
	}
	return fallback
}

// handleCreation runs the Create of r. followUp is nil when r has no follow-up step.
func handleCreation(ctx context.Context, log logr.Logger, obj client.Object, r CoralogixReconciler) (followUp func(context.Context) error, err error) {
	if withFollowUp, ok := r.(CreationFollowUp); ok {
		return withFollowUp.HandleCreationWithFollowUp(ctx, log, obj)
	}
	return nil, r.HandleCreation(ctx, log, obj)
}

// statusWatch notices a status change that a hook made. The zero value watches nothing.
type statusWatch struct {
	active bool
	before map[string]any
}

// watchStatus starts to watch the status of obj when r saves hook status. The snapshot is a deep
// copy (converted to unstructured), because the conditions are a slice that a hook can change in
// place.
func watchStatus(obj client.Object, r CoralogixReconciler) (statusWatch, error) {
	if saver, ok := r.(HookStatusSaver); !ok || !saver.SavesHookStatus() {
		return statusWatch{}, nil
	}
	before, err := statusOf(obj)
	if err != nil {
		return statusWatch{}, err
	}
	return statusWatch{active: true, before: before}, nil
}

// changed reports whether the status of obj differs from the snapshot.
func (w statusWatch) changed(obj client.Object) (bool, error) {
	if !w.active {
		return false, nil
	}
	after, err := statusOf(obj)
	if err != nil {
		return false, err
	}
	return !reflect.DeepEqual(w.before, after), nil
}

func statusOf(obj client.Object) (map[string]any, error) {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert object to unstructured: %w", err)
	}
	status, _ := u["status"].(map[string]any)
	return status, nil
}

// remoteStatusFields returns the status paths that selector-mismatch cleanup removes: the fixed
// paths, and the paths of the reconciler.
func remoteStatusFields(r CoralogixReconciler) [][]string {
	fields := [][]string{
		{"status", "id"},
		{"status", "accessPolicyManaged"}, // Alert, deprecated
		{"status", "conditions"},
		{"status", "printableStatus"},
		{"status", "externalId"}, // OutboundWebhook
		{"status", "revision"},   // SLO
	}
	if provider, ok := r.(RemoteStatusFieldsProvider); ok {
		fields = append(fields, provider.RemoteStatusFields()...)
	}
	return fields
}

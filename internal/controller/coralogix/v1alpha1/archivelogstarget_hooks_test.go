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

package v1alpha1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// archiveCall is one request that a hook sent.
type archiveCall struct {
	Method, Path string
	Body         any
}

// archiveAPI answers the archive target and retentions calls. target is the answer of GET; status
// is the status of every answer.
type archiveAPI struct {
	mu     sync.Mutex
	calls  []archiveCall
	target map[string]any
	status int
}

func (a *archiveAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body any
	if len(raw) != 0 {
		_ = json.Unmarshal(raw, &body)
	}
	a.mu.Lock()
	a.calls = append(a.calls, archiveCall{Method: r.Method, Path: strings.TrimPrefix(r.URL.Path, "/mgmt/openapi/5"), Body: body})
	status, target := a.status, a.target
	a.mu.Unlock()
	if status == 0 {
		status = http.StatusOK
	}
	var answer any
	switch {
	case status != http.StatusOK:
		answer = map[string]any{"code": status, "message": http.StatusText(status) + ": target not found"}
	case strings.HasSuffix(r.URL.Path, "/activate"):
		answer = map[string]any{"activateRetentions": true}
	default:
		answer = map[string]any{"target": target}
	}
	out, err := json.Marshal(answer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(out)
}

func (a *archiveAPI) takeCalls() []archiveCall {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.calls
	a.calls = nil
	return out
}

func newArchiveHooks(t *testing.T, api *archiveAPI) *ArchiveLogsTargetReconciler {
	t.Helper()
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)
	cs := cxsdk.NewClientSet(cxsdk.NewConfigBuilder().WithURL(server.URL + "/mgmt/openapi/5").Build())
	return &ArchiveLogsTargetReconciler{Client: cs.ArchiveLogs(), ClientSet: cs}
}

func archiveTarget(s3 *coralogixv1alpha1.ArchiveLogsTargetS3TargetSpec, conditions ...metav1.Condition) *coralogixv1alpha1.ArchiveLogsTarget {
	return &coralogixv1alpha1.ArchiveLogsTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default", Generation: 1},
		Spec:       coralogixv1alpha1.ArchiveLogsTargetSpec{S3Target: s3, IsActive: ptr.To(true)},
		Status:     coralogixv1alpha1.ArchiveLogsTargetStatus{Conditions: conditions},
	}
}

var archiveS3 = &coralogixv1alpha1.ArchiveLogsTargetS3TargetSpec{BucketName: ptr.To("bucket"), Region: ptr.To("eu-north-1")}

func TestArchiveLogsTargetValidateNeedsAnS3Target(t *testing.T) {
	r := &ArchiveLogsTargetReconciler{}
	ibm := archiveTarget(nil)
	ibm.Spec.IbmCosTarget = &coralogixv1alpha1.IbmCosTarget{BucketCrn: "crn", Endpoint: "e"}
	require.ErrorContains(t, r.validate(context.Background(), logr.Discard(), ibm), "spec.s3Target is required")
	require.NoError(t, r.validate(context.Background(), logr.Discard(), archiveTarget(archiveS3)))
}

// afterUpdate activates the retentions only when the previous run did not sync, and a second run
// sends the same one call.
func TestArchiveLogsTargetAfterUpdateActivatesOnlyAfterAFailure(t *testing.T) {
	api := &archiveAPI{}
	r := newArchiveHooks(t, api)
	synced := metav1.Condition{Type: utils.ConditionTypeRemoteSynced, Status: metav1.ConditionTrue, Reason: utils.ReasonRemoteSyncedSuccessfully}
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), archiveTarget(archiveS3, synced), nil))
	require.Empty(t, api.takeCalls(), "a synced resync makes no call")

	failed := metav1.Condition{Type: utils.ConditionTypeRemoteSynced, Status: metav1.ConditionFalse, Reason: "HookFailed"}
	for range 2 {
		require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), archiveTarget(archiveS3, failed), nil))
		require.Equal(t, []archiveCall{{Method: http.MethodPost, Path: "/dataengine/retention-tags/v1/activate"}}, api.takeCalls())
	}

	// An inactive target activates nothing, on Create or after a failure.
	inactive := archiveTarget(archiveS3, failed)
	inactive.Spec.IsActive = ptr.To(false)
	require.NoError(t, r.afterCreate(context.Background(), logr.Discard(), inactive, nil))
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), inactive, nil))
	require.Empty(t, api.takeCalls())
}

// delete sets the target of the spec inactive. A second run sends the same request, and a target
// that the API does not find is deleted.
func TestArchiveLogsTargetDeleteIsSafeToRepeat(t *testing.T) {
	api := &archiveAPI{target: map[string]any{"s3": map[string]any{"bucket": "bucket"}, "archiveSpec": map[string]any{"isActive": false}}}
	r := newArchiveHooks(t, api)
	want := []archiveCall{{Method: http.MethodPost, Path: "/logs/data-setup/v2", Body: map[string]any{
		"isActive": false, "s3": map[string]any{"bucket": "bucket", "region": "eu-north-1"},
	}}}
	for range 2 {
		require.NoError(t, r.delete(context.Background(), logr.Discard(), archiveTarget(archiveS3)))
		require.Equal(t, want, api.takeCalls())
	}
	api.status = http.StatusNotFound
	require.NoError(t, r.delete(context.Background(), logr.Discard(), archiveTarget(archiveS3)))
}

// A CR without an S3 target (an IBM COS target) sets the S3 target of the team inactive, so it can
// be deleted. A team without an S3 target needs no call.
func TestArchiveLogsTargetDeleteWithoutAnS3Target(t *testing.T) {
	api := &archiveAPI{target: map[string]any{"s3": map[string]any{"bucket": "team-bucket", "region": "eu-north-1"},
		"archiveSpec": map[string]any{"isActive": true, "format": "TARGET_FORMAT_CX_DATA_V1"}}}
	r := newArchiveHooks(t, api)
	require.NoError(t, r.delete(context.Background(), logr.Discard(), archiveTarget(nil)))
	require.Equal(t, []archiveCall{
		{Method: http.MethodGet, Path: "/logs/data-setup/v2"},
		{Method: http.MethodPost, Path: "/logs/data-setup/v2", Body: map[string]any{
			"isActive": false, "format": "TARGET_FORMAT_CX_DATA_V1", "s3": map[string]any{"bucket": "team-bucket", "region": "eu-north-1"},
		}},
	}, api.takeCalls())

	// An inactive target, a team without an S3 target, and a 404 need no POST.
	for _, target := range []map[string]any{
		{"s3": map[string]any{"bucket": "team-bucket"}, "archiveSpec": map[string]any{"isActive": false}},
		{"archiveSpec": map[string]any{"isActive": true}},
	} {
		api.target = target
		require.NoError(t, r.delete(context.Background(), logr.Discard(), archiveTarget(nil)))
		require.Equal(t, []archiveCall{{Method: http.MethodGet, Path: "/logs/data-setup/v2"}}, api.takeCalls())
	}
	api.status = http.StatusNotFound
	require.NoError(t, r.delete(context.Background(), logr.Discard(), archiveTarget(nil)))
	require.Equal(t, []archiveCall{{Method: http.MethodGet, Path: "/logs/data-setup/v2"}}, api.takeCalls())
}

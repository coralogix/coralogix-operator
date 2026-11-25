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
	"errors"
	"net/http"
	"strings"
	"testing"

	oapisdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

func TestSafeReconcileErrorOmitsRawError(t *testing.T) {
	const fakeValue = "fake-sensitive-value"
	for _, test := range []struct {
		name      string
		operation string
		reason    string
		err       error
		want      string
	}{
		{name: "API error", operation: "create", reason: utils.ReasonRemoteCreationFailed, err: &oapisdk.APIError{Err: errors.New(fakeValue), StatusCode: http.StatusBadRequest, Body: []byte(fakeValue)}, want: "RemoteCreationFailed (operation=create, category=remote-api, HTTP 400)"},
		{name: "transport error", operation: "delete", reason: utils.ReasonRemoteDeletionFailed, err: &oapisdk.APIError{Err: errors.New(fakeValue)}, want: "RemoteDeletionFailed (operation=delete, category=transport)"},
		{name: "Kubernetes error", operation: "update-status", reason: utils.ReasonInternalK8sError, err: apierrors.NewNotFound(schema.GroupResource{Resource: "widgets"}, fakeValue), want: "InternalK8sError (operation=update-status, category=kubernetes)"},
		{name: "validation error", operation: "create", reason: utils.ReasonValidationFailed, err: NewValidationError(errors.New(fakeValue)), want: "ValidationFailed (operation=create, category=validation)"},
		{name: "other error", operation: "update", reason: utils.ReasonRemoteUpdateFailed, err: errors.New(fakeValue), want: "RemoteUpdateFailed (operation=update, category=unknown)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := SafeReconcileError(test.operation, test.reason, test.err).Error()
			if got != test.want || strings.Contains(got, fakeValue) {
				t.Fatalf("safe error = %q, want %q", got, test.want)
			}
		})
	}
}

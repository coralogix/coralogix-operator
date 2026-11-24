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
)

func TestSafeReconcileErrorOmitsRawError(t *testing.T) {
	const fakeValue = "fake-sensitive-value"
	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{name: "API error", err: &oapisdk.APIError{Err: errors.New(fakeValue), StatusCode: http.StatusBadRequest, Body: []byte(fakeValue)}, want: "RemoteCreationFailed (HTTP 400)"},
		{name: "other error", err: errors.New(fakeValue), want: "reconciliation failed: RemoteCreationFailed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := SafeReconcileError("RemoteCreationFailed", test.err).Error()
			if got != test.want || strings.Contains(got, fakeValue) {
				t.Fatalf("safe error = %q, want %q", got, test.want)
			}
		})
	}
}

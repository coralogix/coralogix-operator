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

package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

func TestValidateNoEmbeddedIDWithImportRejectsNonEmptyIDWithImportAnnotation(t *testing.T) {
	dashboard := &dashboards.Dashboard{Id: ptr.To("3d7f1c2a-9e4b-4a11-8f2d-1a2b3c4d5e6f")}

	err := validateNoEmbeddedIDWithImport("some-import-id", dashboard)

	require.ErrorContains(t, err, "app.coralogix.com/import-id")
}

func TestValidateNoEmbeddedIDWithImportAllowsMissingID(t *testing.T) {
	require.NoError(t, validateNoEmbeddedIDWithImport("some-import-id", &dashboards.Dashboard{}))
	require.NoError(t, validateNoEmbeddedIDWithImport("some-import-id", &dashboards.Dashboard{Id: ptr.To("")}))
}

func TestValidateNoEmbeddedIDWithImportAllowsNonEmptyIDWithoutImportAnnotation(t *testing.T) {
	dashboard := &dashboards.Dashboard{Id: ptr.To("3d7f1c2a-9e4b-4a11-8f2d-1a2b3c4d5e6f")}

	require.NoError(t, validateNoEmbeddedIDWithImport("", dashboard))
}

func TestDashboardRequestsMapAccessPolicy(t *testing.T) {
	const configuredPolicy = `{"version":"2025-01-01","rules":[],"default":{"permissions":{"team-dashboards:Read":"grant"}}}`

	for _, tt := range []struct {
		name         string
		accessPolicy *string
	}{
		{name: "omitted"},
		{name: "configured", accessPolicy: ptr.To(configuredPolicy)},
		{name: "explicit empty", accessPolicy: ptr.To("")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dashboard := &coralogixv1alpha1.Dashboard{Spec: coralogixv1alpha1.DashboardSpec{AccessPolicy: tt.accessPolicy}}

			createRequest := newCreateDashboardRequest(dashboard, dashboards.Dashboard{})
			replaceRequest := newReplaceDashboardRequest(dashboard, dashboards.Dashboard{})

			require.Equal(t, tt.accessPolicy, createRequest.AccessPolicy)
			require.Equal(t, tt.accessPolicy, replaceRequest.AccessPolicy)

			createPayload, err := json.Marshal(createRequest)
			require.NoError(t, err)
			replacePayload, err := json.Marshal(replaceRequest)
			require.NoError(t, err)
			if tt.accessPolicy == nil {
				require.NotContains(t, string(createPayload), `"accessPolicy"`)
				require.NotContains(t, string(replacePayload), `"accessPolicy"`)
			} else {
				require.Contains(t, string(createPayload), `"accessPolicy"`)
				require.Contains(t, string(replacePayload), `"accessPolicy"`)
			}
		})
	}
}

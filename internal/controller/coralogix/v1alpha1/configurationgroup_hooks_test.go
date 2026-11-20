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
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	cfggroups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/fleet_manager_configuration_groups"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

const testConfigurationGroupID = "0b6f9a52-6b1f-4c8e-9a43-2f1d7c5e8a10"

type configurationGroupRequest struct {
	method, path, updateMask, body string
}

// configurationGroupHookServer answers each request with status and a group with an inactive
// family, and records the requests.
func configurationGroupHookServer(t *testing.T, status *int, requests *[]configurationGroupRequest) *cfggroups.FleetManagerConfigurationGroupsAPIService {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		*requests = append(*requests, configurationGroupRequest{r.Method, r.URL.Path, r.URL.Query().Get("update_mask"), strings.TrimSpace(string(body))})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(*status)
		answer := `{"id":"` + testConfigurationGroupID + `","name":"cg","priorityOrder":0,"archived":false,"family":{"id":"` + testConfigurationGroupID + `","version":"1","active":false}}`
		if *status != http.StatusOK {
			answer = `{"code":` + strconv.Itoa(*status) + `,"message":"` + http.StatusText(*status) + `: configuration group not found or archived"}`
		}
		_, err = w.Write([]byte(answer))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	cfg := cfggroups.NewConfiguration()
	cfg.Servers = cfggroups.ServerConfigurations{{URL: server.URL}}
	return cfggroups.NewAPIClient(cfg).FleetManagerConfigurationGroupsAPI
}

// beforeDelete sends one update of family.active and nothing else. A second run sends the same
// request, and a group that is already gone is no error.
func TestConfigurationGroupBeforeDeleteIsSafeToRepeat(t *testing.T) {
	status := http.StatusOK
	var requests []configurationGroupRequest
	r := &ConfigurationGroupReconciler{Client: configurationGroupHookServer(t, &status, &requests)}
	cr := &coralogixv1alpha1.ConfigurationGroup{Status: coralogixv1alpha1.ConfigurationGroupStatus{ID: ptr.To(testConfigurationGroupID)}}

	require.NoError(t, r.beforeDelete(context.Background(), logr.Discard(), cr))
	require.Len(t, requests, 1)
	want := configurationGroupRequest{http.MethodPatch, "/fleet-management/configuration-groups/v1/" + testConfigurationGroupID, "family.active", `{"family":{"active":false}}`}
	require.Equal(t, want, requests[0])

	require.NoError(t, r.beforeDelete(context.Background(), logr.Discard(), cr), "a second run")
	require.Equal(t, []configurationGroupRequest{want, want}, requests)

	status = http.StatusNotFound
	require.NoError(t, r.beforeDelete(context.Background(), logr.Discard(), cr), "the group is already gone")

	status = http.StatusBadRequest
	require.ErrorContains(t, r.beforeDelete(context.Background(), logr.Discard(), cr), "deactivating the family before the archive")
}

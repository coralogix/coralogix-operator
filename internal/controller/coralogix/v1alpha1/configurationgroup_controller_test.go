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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cfggroups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/fleet_manager_configuration_groups"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

const testConfigurationGroupID = "0b6f9a52-6b1f-4c8e-9a43-2f1d7c5e8a10"

type recordedRequest struct {
	method     string
	path       string
	updateMask string
	body       string
}

func configurationGroupsTestClient(t *testing.T, requests *[]recordedRequest) *cfggroups.FleetManagerConfigurationGroupsAPIService {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		*requests = append(*requests, recordedRequest{
			method:     r.Method,
			path:       r.URL.Path,
			updateMask: r.URL.Query().Get("update_mask"),
			body:       string(body),
		})
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"id":"` + testConfigurationGroupID + `","priorityOrder":0}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	cfg := cfggroups.NewConfiguration()
	cfg.Servers = cfggroups.ServerConfigurations{{URL: server.URL}}
	return cfggroups.NewAPIClient(cfg).FleetManagerConfigurationGroupsAPI
}

func syncedConfigurationGroup() *coralogixv1alpha1.ConfigurationGroup {
	return &coralogixv1alpha1.ConfigurationGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "cg", Namespace: "default"},
		Spec: coralogixv1alpha1.ConfigurationGroupSpec{
			Name: "cg",
			Family: coralogixv1alpha1.ConfigurationFamilySpec{
				Active: ptr.To(true),
				Raw: &coralogixv1alpha1.RawConfigurationFamilySpec{
					RemoteConfigurations: []coralogixv1alpha1.RemoteConfigurationSpec{{
						Name:             "default",
						RawConfiguration: "receivers: {}",
					}},
				},
			},
		},
		Status: coralogixv1alpha1.ConfigurationGroupStatus{ID: ptr.To(testConfigurationGroupID)},
	}
}

func TestConfigurationGroupCreateReadsUnwrappedResponse(t *testing.T) {
	var requests []recordedRequest
	r := &ConfigurationGroupReconciler{ConfigurationGroupsClient: configurationGroupsTestClient(t, &requests)}
	group := syncedConfigurationGroup()
	group.Status = coralogixv1alpha1.ConfigurationGroupStatus{}

	require.NoError(t, r.HandleCreation(context.Background(), logr.Discard(), group))

	require.Len(t, requests, 1)
	require.Equal(t, http.MethodPost, requests[0].method)
	require.Equal(t, "/fleet-management/configuration-groups/v1", requests[0].path)
	require.JSONEq(t, `{
		"name": "cg",
		"family": {
			"active": true,
			"raw": {
				"remoteConfigurations": [{"name": "default", "rawConfiguration": "receivers: {}"}]
			}
		}
	}`, requests[0].body)
	require.Equal(t, ptr.To(testConfigurationGroupID), group.Status.ID)
}

func TestConfigurationGroupUpdatePatchesFullMask(t *testing.T) {
	var requests []recordedRequest
	r := &ConfigurationGroupReconciler{ConfigurationGroupsClient: configurationGroupsTestClient(t, &requests)}

	require.NoError(t, r.HandleUpdate(context.Background(), logr.Discard(), syncedConfigurationGroup()))

	require.Len(t, requests, 1)
	require.Equal(t, http.MethodPatch, requests[0].method)
	require.Equal(t, "/fleet-management/configuration-groups/v1/"+testConfigurationGroupID, requests[0].path)
	require.Equal(t, "name,description,tags,priorityOrder,family.description,family.active,family.raw", requests[0].updateMask)
	require.JSONEq(t, `{
		"name": "cg",
		"description": "",
		"tags": [],
		"priorityOrder": 0,
		"family": {
			"active": true,
			"description": "",
			"raw": {
				"metadata": {},
				"remoteConfigurations": [{"name": "default", "rawConfiguration": "receivers: {}"}]
			}
		}
	}`, requests[0].body)
}

func TestConfigurationGroupDeletionDeactivatesThenArchives(t *testing.T) {
	var requests []recordedRequest
	r := &ConfigurationGroupReconciler{ConfigurationGroupsClient: configurationGroupsTestClient(t, &requests)}

	require.NoError(t, r.HandleDeletion(context.Background(), logr.Discard(), syncedConfigurationGroup()))

	require.Len(t, requests, 2)
	require.Equal(t, http.MethodPatch, requests[0].method)
	require.Equal(t, "/fleet-management/configuration-groups/v1/"+testConfigurationGroupID, requests[0].path)
	require.Equal(t, "family.active", requests[0].updateMask)
	require.JSONEq(t, `{"family": {"active": false}}`, requests[0].body)
	require.Equal(t, http.MethodPost, requests[1].method)
	require.Equal(t, "/fleet-management/configuration-groups/v1/"+testConfigurationGroupID+"/archive", requests[1].path)
}

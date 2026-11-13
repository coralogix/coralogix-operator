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
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

const hookTestContentID = "3d7f1c2a-9e4b-4a11-8f2d-1a2b3c4d5e6f"

func dashboardWithContent(json string, annotations map[string]string) *coralogixv1alpha1.Dashboard {
	return &coralogixv1alpha1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{Name: "dashboard", Namespace: "default", Annotations: annotations},
		Spec:       coralogixv1alpha1.DashboardSpec{Json: ptr.To(json)},
	}
}

func TestDashboardValidateRejectsAContentIDWithTheImportAnnotation(t *testing.T) {
	cr := dashboardWithContent(`{"name":"test","layout":{},"id":"`+hookTestContentID+`"}`,
		map[string]string{coralogixv1alpha1.ImportDashboardIDAnnotationKey: "some-import-id"})

	err := (&DashboardReconciler{}).validate(context.Background(), logr.Discard(), cr)

	require.ErrorContains(t, err, "app.coralogix.com/import-id")
}

func TestDashboardValidateAllowsAMissingOrEmptyContentID(t *testing.T) {
	annotations := map[string]string{coralogixv1alpha1.ImportDashboardIDAnnotationKey: "some-import-id"}
	r := &DashboardReconciler{}

	require.NoError(t, r.validate(context.Background(), logr.Discard(), dashboardWithContent(`{"name":"test","layout":{}}`, annotations)))
	require.NoError(t, r.validate(context.Background(), logr.Discard(), dashboardWithContent(`{"name":"test","layout":{},"id":""}`, annotations)))
}

func TestDashboardValidateAllowsAContentIDWithoutTheImportAnnotation(t *testing.T) {
	cr := dashboardWithContent(`{"name":"test","layout":{},"id":"`+hookTestContentID+`"}`, nil)

	require.NoError(t, (&DashboardReconciler{}).validate(context.Background(), logr.Discard(), cr))
}

// The folder hooks run on every reconcile. Running one twice leaves the same body.
func TestDashboardFolderHooksAreSafeToRepeat(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ref      *coralogixv1alpha1.DashboardFolderRef
		content  dashboards.Dashboard
		wantID   *string
		wantPath []string
	}{
		{
			name:    "an id replaces a path of the content",
			ref:     &coralogixv1alpha1.DashboardFolderRef{BackendRef: &coralogixv1alpha1.DashboardFolderRefBackendRef{ID: ptr.To(hookTestContentID)}},
			content: dashboards.Dashboard{FolderPath: &dashboards.FolderPath{Segments: []string{"from", "content"}}},
			wantID:  ptr.To(hookTestContentID),
		},
		{
			name:     "a path replaces an id of the content",
			ref:      &coralogixv1alpha1.DashboardFolderRef{BackendRef: &coralogixv1alpha1.DashboardFolderRefBackendRef{Path: ptr.To("team/infra/observability")}},
			content:  dashboards.Dashboard{FolderId: &dashboards.UUID{Value: ptr.To(hookTestContentID)}},
			wantPath: []string{"team", "infra", "observability"},
		},
		{
			name:     "no folderRef keeps the folder of the content",
			content:  dashboards.Dashboard{FolderPath: &dashboards.FolderPath{Segments: []string{"from", "content"}}},
			wantPath: []string{"from", "content"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cr := dashboardWithContent(`{}`, nil)
			cr.Spec.FolderRef = tt.ref
			r := &DashboardReconciler{}

			create := &dashboards.CreateDashboardRequestDataStructure{Dashboard: tt.content}
			update := &dashboards.ReplaceDashboardRequestDataStructure{Dashboard: ptr.To(tt.content)}
			for range 2 {
				require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), cr, create))
				require.NoError(t, r.beforeUpdate(context.Background(), logr.Discard(), cr, update))
			}
			for _, got := range []dashboards.Dashboard{create.Dashboard, *update.Dashboard} {
				if tt.wantID != nil {
					require.Equal(t, *tt.wantID, *got.FolderId.Value)
					require.Nil(t, got.FolderPath)
				} else {
					require.Equal(t, tt.wantPath, got.FolderPath.Segments)
					require.Nil(t, got.FolderId)
				}
			}
		})
	}
}

func TestDashboardAdoptDoesNothingWithoutTheAnnotationOrAfterAnImport(t *testing.T) {
	r := &DashboardReconciler{} // no client: adopt must not call the API

	id, err := r.adopt(context.Background(), logr.Discard(), dashboardWithContent(`{}`, nil), nil)
	require.NoError(t, err)
	require.Empty(t, id)

	imported := dashboardWithContent(`{}`, map[string]string{coralogixv1alpha1.ImportDashboardIDAnnotationKey: "some-import-id"})
	imported.Status.Imported = ptr.To(true)
	id, err = r.adopt(context.Background(), logr.Discard(), imported, nil)
	require.NoError(t, err)
	require.Empty(t, id)
}

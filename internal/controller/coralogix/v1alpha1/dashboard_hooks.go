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
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The handwritten steps of the generated Dashboard lifecycle: the folder reference, and the
// one-time import of an existing dashboard by the app.coralogix.com/import-id annotation.

// validate rejects an id in the content together with the import annotation: the two would name
// different dashboards. Without the annotation an id in the content is harmless: Create sends it
// as it is, and Update replaces it with status.id.
func (r *DashboardReconciler) validate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.Dashboard) error {
	if importDashboardID(cr) == "" {
		return nil
	}
	// A content that cannot be read is reported when the request is built.
	if content, err := cr.Spec.DecodeContent(ctx, cr.Namespace); err == nil && ptr.Deref(content.Id, "") != "" {
		return fmt.Errorf("spec content must not contain an %q field; use the %q annotation to import an existing dashboard", "id", coralogixv1alpha1.ImportDashboardIDAnnotationKey)
	}
	return nil
}

// beforeCreate places the dashboard in the folder of spec.folderRef, and asks for the id of
// dashboardCreateID when the content has no id.
func (r *DashboardReconciler) beforeCreate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.Dashboard, body *dashboards.CreateDashboardRequestDataStructure) error {
	if ptr.Deref(body.Dashboard.Id, "") == "" {
		body.Dashboard.Id = ptr.To(dashboardCreateID(cr))
	}
	return setDashboardFolder(ctx, cr, &body.Dashboard)
}

// beforeUpdate places the dashboard in the folder of spec.folderRef.
func (r *DashboardReconciler) beforeUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.Dashboard, body *dashboards.ReplaceDashboardRequestDataStructure) error {
	if body.Dashboard == nil {
		return nil
	}
	return setDashboardFolder(ctx, cr, body.Dashboard)
}

// adopt imports the dashboard that the import annotation names, once. status.imported records
// that the import happened. It outlives status.id, so a dashboard that is later deleted outside
// the operator is created from the spec, instead of the import retrying a Get of a missing id.
// Adoption reads the dashboard and changes nothing remote: the next reconcile replaces it with
// the spec.
//
// Without an import, adopt finds the dashboard of an earlier Create attempt: one whose response
// was lost, or had no id. That attempt asked for the same id of dashboardCreateID, so the
// dashboard is adopted instead of created twice. The API does not deduplicate by requestId.
func (r *DashboardReconciler) adopt(ctx context.Context, log logr.Logger, cr *coralogixv1alpha1.Dashboard, body *dashboards.CreateDashboardRequestDataStructure) (string, error) {
	if id := importDashboardID(cr); id != "" && !ptr.Deref(cr.Status.Imported, false) {
		log.Info("Import annotation present, adopting existing remote dashboard", "id", id)
		_, httpResp, err := r.ClientSet.Dashboards().DashboardsServiceGetDashboard(ctx, id).Execute()
		if err != nil {
			return "", fmt.Errorf("error on getting remote dashboard %q for import: %w", id, cxsdk.NewAPIError(httpResp, err))
		}
		cr.Status.Imported = ptr.To(true)
		return id, nil
	}
	// An id of the content is not adopted: it can name a dashboard that the CR does not own. The
	// API rejects its Create while that dashboard exists.
	id := ptr.Deref(body.Dashboard.Id, "")
	if id == "" || id != dashboardCreateID(cr) {
		return "", nil
	}
	_, httpResp, err := r.ClientSet.Dashboards().DashboardsServiceGetDashboard(ctx, id).Execute()
	switch {
	case err == nil:
		log.Info("Found the remote dashboard of an earlier Create attempt", "id", id)
		return id, nil
	case httpResp != nil && httpResp.StatusCode == http.StatusNotFound:
		return "", nil
	default:
		return "", fmt.Errorf("error on getting remote dashboard %q before Create: %w", id, cxsdk.NewAPIError(httpResp, err))
	}
}

// dashboardCreateID is the id that Create asks for when the content has none. The API accepts a
// client id of 21 characters and rejects a second dashboard with the same id. The id comes from
// the UID of the CR, so every Create attempt of one CR asks for the same id, and a CR that is
// created again gets a new one.
func dashboardCreateID(cr *coralogixv1alpha1.Dashboard) string {
	sum := sha256.Sum256([]byte(cr.UID))
	// A SHA-256 sum has 42 or 43 base62 digits.
	return new(big.Int).SetBytes(sum[:]).Text(62)[:21]
}

func importDashboardID(cr *coralogixv1alpha1.Dashboard) string {
	return cr.GetAnnotations()[coralogixv1alpha1.ImportDashboardIDAnnotationKey]
}

// setDashboardFolder sets the folder of spec.folderRef. folderId and folderPath are not a oneOf,
// so setting one clears the other, or a folder of the content would be sent beside it. Without a
// folderRef the folder of the content is kept.
func setDashboardFolder(ctx context.Context, cr *coralogixv1alpha1.Dashboard, dashboard *dashboards.Dashboard) error {
	folderRef := cr.Spec.FolderRef
	if folderRef == nil {
		return nil
	}
	switch {
	case folderRef.BackendRef != nil:
		if id := folderRef.BackendRef.ID; id != nil {
			dashboard.FolderId = &dashboards.UUID{Value: ptr.To(*id)}
			dashboard.FolderPath = nil
		} else if path := folderRef.BackendRef.Path; path != nil {
			dashboard.FolderPath = &dashboards.FolderPath{Segments: strings.Split(*path, "/")}
			dashboard.FolderId = nil
		}
	case folderRef.ResourceRef != nil:
		folderID, err := coralogixv1alpha1.GetFolderIdFromFolderCR(ctx, cr.Namespace, *folderRef.ResourceRef)
		if err != nil {
			return err
		}
		dashboard.FolderId = &dashboards.UUID{Value: ptr.To(*folderID)}
		dashboard.FolderPath = nil
	default:
		return fmt.Errorf("folderRef.BackendRef or folderRef.ResourceRef is required")
	}
	return nil
}

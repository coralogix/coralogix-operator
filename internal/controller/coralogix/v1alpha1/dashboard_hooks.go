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
	"fmt"
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

// beforeCreate places the dashboard in the folder of spec.folderRef.
func (r *DashboardReconciler) beforeCreate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.Dashboard, body *dashboards.CreateDashboardRequestDataStructure) error {
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
func (r *DashboardReconciler) adopt(ctx context.Context, log logr.Logger, cr *coralogixv1alpha1.Dashboard, _ *dashboards.CreateDashboardRequestDataStructure) (string, error) {
	id := importDashboardID(cr)
	if id == "" || ptr.Deref(cr.Status.Imported, false) {
		return "", nil
	}
	log.Info("Import annotation present, adopting existing remote dashboard", "id", id)
	_, httpResp, err := r.ClientSet.Dashboards().DashboardsServiceGetDashboard(ctx, id).Execute()
	if err != nil {
		return "", fmt.Errorf("error on getting remote dashboard %q for import: %w", id, cxsdk.NewAPIError(httpResp, err))
	}
	cr.Status.Imported = ptr.To(true)
	return id, nil
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

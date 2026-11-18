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

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"

	contentthingsservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/content_things_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// These are the handwritten hooks of the synthetic ContentThing kind, with the content key, the
// spec key, and the adopt hook. The generator tests copy this file into the controller package of
// a copy of the operator and build it there.

func (r *ContentThingReconciler) validate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.ContentThing) error {
	_, err := cr.Spec.DecodeContent(ctx, cr.Namespace)
	return err
}

func (r *ContentThingReconciler) beforeCreate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.ContentThing, body *contentthingsservice.CreateContentThingRequest) error {
	if cr.Spec.Place != nil {
		body.Item.Mode = cr.Spec.Place.Mode
	}
	return nil
}

func (r *ContentThingReconciler) adopt(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.ContentThing, _ *contentthingsservice.CreateContentThingRequest) (string, error) {
	id := cr.GetAnnotations()["example.com/import-id"]
	if id == "" || ptr.Deref(cr.Status.Imported, false) {
		return "", nil
	}
	cr.Status.Imported = ptr.To(true)
	return id, nil
}

func (r *ContentThingReconciler) beforeUpdate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.ContentThing, body *contentthingsservice.ReplaceContentThingRequest) error {
	if cr.Spec.Place != nil && body.Item != nil {
		body.Item.Mode = cr.Spec.Place.Mode
	}
	return nil
}

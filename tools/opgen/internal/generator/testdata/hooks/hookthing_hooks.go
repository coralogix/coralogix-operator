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
	"errors"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"

	hookthingsservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/hook_things_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// These are the handwritten hooks of the synthetic HookThing kind. The generator tests copy this
// file into the controller package of a copy of the operator and build it there. A hook missing
// from this file, or a hook with a wrong signature, fails that build. Each hook is safe to run
// many times: it sets a value, and it never appends.

func (r *HookThingReconciler) validate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.HookThing) error {
	if cr.Spec.Name == "forbidden" {
		return errors.New("the name is forbidden")
	}
	return nil
}

func (r *HookThingReconciler) beforeCreate(_ context.Context, _ logr.Logger, _ *coralogixv1alpha1.HookThing, body *hookthingsservice.CreateHookThingRequest) error {
	if body.Limit == nil {
		body.Limit = ptr.To(int32(1))
	}
	return nil
}

func (r *HookThingReconciler) beforeUpdate(_ context.Context, _ logr.Logger, _ *coralogixv1alpha1.HookThing, body *hookthingsservice.ReplaceHookThingRequest) error {
	if body.Limit == nil {
		body.Limit = ptr.To(int32(1))
	}
	return nil
}

func (r *HookThingReconciler) afterCreate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.HookThing, resp *hookthingsservice.HookThing) error {
	cr.Status.ExternalId = ptr.To(resp.Id)
	return nil
}

// The Replace response of HookThing has no fields, so the SDK returns a map.
func (r *HookThingReconciler) afterUpdate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.HookThing, _ map[string]interface{}) error {
	cr.Status.Revision = ptr.To(int64(1))
	return nil
}

func (r *HookThingReconciler) beforeDelete(_ context.Context, _ logr.Logger, _ *coralogixv1alpha1.HookThing) error {
	return nil
}

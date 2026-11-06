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

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The delete hook of the synthetic HookDeleteThing kind. It replaces the generated DELETE. The
// generator tests build it as hookthing_hooks.go.

func (r *HookDeleteThingReconciler) delete(_ context.Context, _ logr.Logger, _ *coralogixv1alpha1.HookDeleteThing) error {
	return nil
}

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

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	cfggroups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/fleet_manager_configuration_groups"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// beforeDelete deactivates the family: the API refuses to archive a group whose family is active.
// The update names only family.active, so a desired spec that the API rejects, such as forbidden
// collector YAML, cannot block the archive. Deactivating an inactive family changes nothing, and a
// group that is already gone answers 404, so the hook is safe to run again.
func (r *ConfigurationGroupReconciler) beforeDelete(ctx context.Context, log logr.Logger, cr *coralogixv1alpha1.ConfigurationGroup) error {
	id := ptr.Deref(cr.Status.ID, "")
	family := cfggroups.NewConfigurationFamilyUpdate()
	family.SetActive(false)
	body := cfggroups.NewConfigurationGroupServiceUpdateConfigurationGroupRequest()
	body.SetFamily(*family)
	log.Info("Deactivating the family of the remote ConfigurationGroup before the archive", "id", id)
	_, httpResp, err := r.Client.ConfigurationGroupServiceUpdateConfigurationGroup(ctx, id).
		UpdateMask("family.active").
		ConfigurationGroupServiceUpdateConfigurationGroupRequest(*body).
		Execute()
	if err != nil {
		apiErr := cxsdk.NewAPIError(httpResp, err)
		if cxsdk.IsNotFound(apiErr) {
			return nil // the archive answers 404 too, which the generated delete accepts
		}
		return fmt.Errorf("deactivating the family before the archive: %w", apiErr)
	}
	return nil
}

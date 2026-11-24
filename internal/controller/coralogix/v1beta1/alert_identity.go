// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/go-logr/logr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// withIdentityLabels returns a copy of labels with the remote identity labels of alert. The
// identity labels replace spec labels with the same keys. The spec map is not changed: the
// operator writes the Alert back (for the finalizer), and the labels must not reach the spec.
func withIdentityLabels(labels map[string]string, alert *coralogixv1beta1.Alert) map[string]string {
	out := make(map[string]string, len(labels)+3)
	maps.Copy(out, labels)
	if alert.UID != "" {
		out[utils.RemoteIdentityUIDLabelKey] = string(alert.UID)
	}
	out[utils.RemoteIdentityNamespaceLabelKey] = alert.Namespace
	out[utils.RemoteIdentityNameLabelKey] = alert.Name
	return out
}

// findAdoptableAlert returns the id of a remote alert that carries the UID label of alert, or ""
// when there is none.
//
// It lists alerts with no query parameters, as the alert name lookups do, and matches the label
// itself. The live API rejects the parameters as the SDK encodes them: pagination[pageSize]
// answers 400 ("too many values for field pagination"), and the query_filter form is not applied.
// So it reads only the page that a plain list returns. When that page has a next page token and
// no match, the alert can be on a later page; it logs that and returns "", and Create runs as it
// did before identity labels.
//
// When more than one remote alert carries the label, it returns the lowest id, so each run picks
// the same alert, and logs the others. It never deletes them.
func (r *AlertReconciler) findAdoptableAlert(ctx context.Context, log logr.Logger, alert *coralogixv1beta1.Alert) (string, error) {
	if alert.UID == "" {
		return "", nil
	}
	uid := string(alert.UID)
	resp, httpResp, err := r.ClientSet.Alerts().AlertDefsServiceListAlertDefs(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("error on listing remote alerts to find one with %s=%s: %w",
			utils.RemoteIdentityUIDLabelKey, uid, cxsdk.NewAPIError(httpResp, err))
	}

	var matches []string
	for _, def := range resp.GetAlertDefs() {
		if def.GetId() == "" || def.AlertDefProperties == nil {
			continue
		}
		if def.AlertDefProperties.GetEntityLabels()[utils.RemoteIdentityUIDLabelKey] == uid {
			matches = append(matches, def.GetId())
		}
	}

	if len(matches) == 0 {
		if page, ok := resp.GetPaginationOk(); ok && page.GetNextPageToken() != "" {
			log.Info("No remote alert on the first page carries the identity of this Alert; later pages are not read",
				"alertsOnPage", len(resp.GetAlertDefs()), "totalSize", page.GetTotalSize())
		}
		return "", nil
	}
	slices.Sort(matches)
	if len(matches) > 1 {
		log.Info("More than one remote alert carries the identity of this Alert; adopting the first, leaving the others",
			"adopted", matches[0], "others", matches[1:])
	}
	return matches[0], nil
}

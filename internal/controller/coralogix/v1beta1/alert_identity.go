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
	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// alertListPageSize is the largest page that ListAlertDefs accepts.
const alertListPageSize int64 = 1000

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
// when there is none. It reads every page and matches the label itself: the API does not apply
// the entity label filter as the SDK encodes it (query_filter[...] form keys), and can return
// every alert of the team. The filter is still sent, so a server that applies it returns less.
//
// When more than one remote alert carries the label, it returns the lowest id, so each run picks
// the same alert, and logs the others. It never deletes them.
func (r *AlertReconciler) findAdoptableAlert(ctx context.Context, log logr.Logger, alert *coralogixv1beta1.Alert) (string, error) {
	if alert.UID == "" {
		return "", nil
	}
	uid := string(alert.UID)
	operator := alerts.FILTERVALUESOPERATOR_FILTER_VALUES_OPERATOR_AND
	filter := alerts.AlertDefQueryFilter{
		EntityLabelsFilter: &alerts.AlertDefEntityLabelsFilter{
			EntityLabels:   map[string]string{utils.RemoteIdentityUIDLabelKey: uid},
			ValuesOperator: &operator,
		},
	}

	var matches []string
	var pageToken *string
	for {
		pagination := alerts.AlertDefsServiceListAlertDefsPaginationParameter{
			PageSize:  alerts.PtrInt64(alertListPageSize),
			PageToken: pageToken,
		}
		resp, httpResp, err := r.ClientSet.Alerts().
			AlertDefsServiceListAlertDefs(ctx).
			QueryFilter(filter).
			Pagination(pagination).
			Execute()
		if err != nil {
			return "", fmt.Errorf("error on listing remote alerts to find one with %s=%s: %w",
				utils.RemoteIdentityUIDLabelKey, uid, cxsdk.NewAPIError(httpResp, err))
		}
		for _, def := range resp.GetAlertDefs() {
			if def.GetId() == "" || def.AlertDefProperties == nil {
				continue
			}
			if def.AlertDefProperties.GetEntityLabels()[utils.RemoteIdentityUIDLabelKey] == uid {
				matches = append(matches, def.GetId())
			}
		}
		page, ok := resp.GetPaginationOk()
		if !ok {
			break
		}
		next := page.GetNextPageToken()
		if next == "" || (pageToken != nil && next == *pageToken) {
			break
		}
		pageToken = &next
	}

	if len(matches) == 0 {
		return "", nil
	}
	slices.Sort(matches)
	if len(matches) > 1 {
		log.Info("More than one remote alert carries the identity of this Alert; adopting the first, leaving the others",
			"adopted", matches[0], "others", matches[1:])
	}
	return matches[0], nil
}

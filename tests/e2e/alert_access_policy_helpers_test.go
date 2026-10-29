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

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// Alert policies use the alerts:*Config permission keys; there is no
// alerts:Read, alerts:Update or alerts:Delete. The API stores every alerts
// permission and denies the ones a policy leaves out, so the operator's API
// key needs ReadConfig and UpdateConfig plus the type-specific pair for the
// alert's type (Logs*Config for the logs alerts under test). Without them the
// API denies the key with 403 Permission Denied.
const (
	alertAccessPolicyP1 = `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant","alerts:UpdateConfig":"grant","alerts:LogsReadConfig":"grant","alerts:LogsUpdateConfig":"grant","alerts:ReadAccessPolicy":"grant","alerts:UpdateAccessPolicy":"grant"}},"rules":[]}`
	// P2 differs from P1 only by also granting alerts:MetricsReadConfig, which
	// does not apply to the logs alerts under test, so reconciles still apply
	// while P2 is active.
	alertAccessPolicyP2 = `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant","alerts:UpdateConfig":"grant","alerts:LogsReadConfig":"grant","alerts:LogsUpdateConfig":"grant","alerts:ReadAccessPolicy":"grant","alerts:UpdateAccessPolicy":"grant","alerts:MetricsReadConfig":"grant"}},"rules":[]}`
	// Unknown permission name: the API rejects it (an unknown version string is accepted and
	// rewritten, so it would not exercise the invalid-policy paths).
	alertAccessPolicyInvalid = `{"version":"2025-01-01","default":{"permissions":{"alerts:Read":"grant"}},"rules":[]}`
)

// expectAlertPolicy asserts the remote alert's access policy. It first asserts
// CanReadAccessPolicy with a message naming the missing scope, then matches
// the policy. A nil want expects no remote policy.
func expectAlertPolicy(ctx context.Context, g Gomega, svc *alerts.AlertDefinitionsServiceAPIService, id string, want *string) {
	resp, httpResp, err := svc.AlertDefsServiceGetAlertDef(ctx, id).IncludeAccessPermissions(true).Execute()
	g.Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
	access, ok := resp.GetAccessOk()
	g.Expect(ok).To(BeTrue(), "expected an access decision in the GetAlertDef response")
	canRead := ptr.Deref(access.GetPermissions().CanReadAccessPolicy, false)
	g.Expect(canRead).To(BeTrue(), "CORALOGIX_API_KEY is missing the alerts:ReadAccessPolicy scope, so the alert access policy cannot be verified")
	policy, hasPolicy := access.GetAccessPolicyOk()
	if want == nil {
		g.Expect(policy).To(BeNil(), "expected no remote access policy")
		return
	}
	g.Expect(hasPolicy).To(BeTrue(), "expected a remote access policy")
	g.Expect(policy).ToNot(BeNil(), "expected a remote access policy")
	g.Expect(expandAlertPolicy(g, *policy)).To(Equal(expandAlertPolicy(g, *want)))
}

// expandAlertPolicy decodes a policy and drops "deny" default permissions.
// The API stores every alerts permission and denies the ones a policy leaves
// out, so an explicit deny and an omitted permission are equivalent.
func expandAlertPolicy(g Gomega, raw string) map[string]any {
	var policy map[string]any
	g.Expect(json.Unmarshal([]byte(raw), &policy)).To(Succeed())
	if def, ok := policy["default"].(map[string]any); ok {
		if perms, ok := def["permissions"].(map[string]any); ok {
			for name, effect := range perms {
				if effect == "deny" {
					delete(perms, name)
				}
			}
		}
	}
	return policy
}

// waitAlertReconciled waits for RemoteSynced=True at the given generation and
// returns the fetched Alert.
func waitAlertReconciled(ctx context.Context, crClient client.Client, name string, generation int64) *coralogixv1beta1.Alert {
	fetched := &coralogixv1beta1.Alert{}
	Eventually(func(g Gomega) {
		g.Expect(crClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNamespace}, fetched)).To(Succeed())
		cond := meta.FindStatusCondition(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)
		g.Expect(cond).ToNot(BeNil())
		g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		g.Expect(cond.ObservedGeneration).To(Equal(generation))
	}, time.Minute, time.Second).Should(Succeed())
	return fetched
}

// waitAlertSetReconciled waits for RemoteSynced=True at the given generation
// and returns the fetched AlertSet.
func waitAlertSetReconciled(ctx context.Context, crClient client.Client, name string, generation int64) *coralogixv1alpha1.AlertSet {
	fetched := &coralogixv1alpha1.AlertSet{}
	Eventually(func(g Gomega) {
		g.Expect(crClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNamespace}, fetched)).To(Succeed())
		cond := meta.FindStatusCondition(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)
		g.Expect(cond).ToNot(BeNil())
		g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		g.Expect(cond.ObservedGeneration).To(Equal(generation))
	}, time.Minute, time.Second).Should(Succeed())
	return fetched
}

// setPolicyExternally applies an access policy action directly, bypassing the
// operator, by reading the remote properties and replacing with the action.
func setPolicyExternally(ctx context.Context, svc *alerts.AlertDefinitionsServiceAPIService, id string, action *alerts.AccessPolicyAction) {
	getResp, httpResp, err := svc.AlertDefsServiceGetAlertDef(ctx, id).Execute()
	Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
	alertDef, ok := getResp.GetAlertDefOk()
	Expect(ok).To(BeTrue())
	props, ok := alertDef.GetAlertDefPropertiesOk()
	Expect(ok).To(BeTrue())
	_, httpResp, err = svc.AlertDefsServiceReplaceAlertDef(ctx).ReplaceAlertDefinitionRequest(alerts.ReplaceAlertDefinitionRequest{
		AlertDefProperties: props,
		Id:                 &id,
		AccessPolicy:       action,
	}).Execute()
	Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
}

// newAccessPolicyTestAlert returns a minimal Alert CR carrying the given
// access policy.
func newAccessPolicyTestAlert(name, namespace string, policy *string) *coralogixv1beta1.Alert {
	return &coralogixv1beta1.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1beta1.AlertSpec{
			Name:         name,
			Description:  "alert access policy e2e",
			Priority:     coralogixv1beta1.AlertPriorityP3,
			AccessPolicy: policy,
			TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
				LogsImmediate: &coralogixv1beta1.LogsImmediate{},
			},
		},
	}
}

// listAlertsByNamePage lists one page of remote alerts, asking the server to
// filter by the given name. The filter is not honored as the SDK encodes it, so
// the page may hold every alert in the team; callers must match names
// themselves. It returns the page and the next page token, or a nil token when
// there are no more pages.
func listAlertsByNamePage(ctx context.Context, svc *alerts.AlertDefinitionsServiceAPIService, name string, pageToken *string) (*alerts.ListAlertDefsResponse, *string, error) {
	matcher := alerts.ALERTSV3FILTERMATCHER_FILTER_MATCHER_EQUALS
	req := svc.AlertDefsServiceListAlertDefs(ctx).QueryFilter(alerts.AlertDefQueryFilter{
		NameFilter: &alerts.AlertDefNameFilter{
			Matcher: &matcher,
			Name:    []string{name},
		},
	}).IncludeAccessPermissions(true)
	if pageToken != nil {
		req = req.Pagination(alerts.AlertDefsServiceListAlertDefsPaginationParameter{PageToken: pageToken})
	}
	resp, httpResp, err := req.Execute()
	if err != nil {
		return nil, nil, oapicxsdk.NewAPIError(httpResp, err)
	}
	pagination, ok := resp.GetPaginationOk()
	if !ok || pagination.GetNextPageToken() == "" {
		return resp, nil, nil
	}
	return resp, ptr.To(pagination.GetNextPageToken()), nil
}

// sweepAlertsByName deletes every remote alert with exactly the given name,
// consuming all ListAlertDefs pages behind the server-side name filter.
// Genuine not-found responses are tolerated; other errors are collected and
// returned so cleanup failures surface instead of leaking fixtures.
func sweepAlertsByName(ctx context.Context, svc *alerts.AlertDefinitionsServiceAPIService, name string) error {
	var errs []error
	var pageToken *string
	for {
		resp, next, err := listAlertsByNamePage(ctx, svc, name, pageToken)
		if err != nil {
			errs = append(errs, err)
			return errors.Join(errs...)
		}
		for i := range resp.GetAlertDefs() {
			alertDef := resp.GetAlertDefs()[i]
			id := alertDef.GetId()
			// Match the name here too: the API ignores the name filter as the SDK encodes it
			// (query_filter[nameFilter][...]) and returns every alert in the team, so trusting
			// it would delete alerts that belong to other specs.
			if id == "" || alertDef.AlertDefProperties.GetName() != name {
				continue
			}
			_, httpResp, err := svc.AlertDefsServiceDeleteAlertDef(ctx, id).Execute()
			if apiErr := oapicxsdk.NewAPIError(httpResp, err); apiErr != nil && !oapicxsdk.IsNotFound(apiErr) {
				errs = append(errs, apiErr)
			}
		}
		if next == nil {
			break
		}
		pageToken = next
	}
	return errors.Join(errs...)
}

// verifyRemoteAlertGone returns nil once the remote alert is really deleted.
// A genuine not-found response is success; any other outcome is an error so
// orphaned fixtures surface instead of passing silently.
func verifyRemoteAlertGone(ctx context.Context, svc *alerts.AlertDefinitionsServiceAPIService, id string) error {
	_, httpResp, err := svc.AlertDefsServiceGetAlertDef(ctx, id).Execute()
	if err == nil {
		return fmt.Errorf("remote alert %s still exists", id)
	}
	if apiErr := oapicxsdk.NewAPIError(httpResp, err); oapicxsdk.IsNotFound(apiErr) {
		return nil
	}
	return oapicxsdk.NewAPIError(httpResp, err)
}

// waitRemoteAlertGone polls until the remote alert is deleted, tolerating the
// API's eventual consistency. A genuine not-found is success.
func waitRemoteAlertGone(ctx context.Context, svc *alerts.AlertDefinitionsServiceAPIService, id string) error {
	var lastErr error
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if lastErr = verifyRemoteAlertGone(ctx, svc, id); lastErr == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return lastErr
}

// waitObjectGone polls until the Kubernetes object is deleted. Not-found is
// success; any other error aborts the wait.
func waitObjectGone(ctx context.Context, crClient client.Client, key types.NamespacedName, obj client.Object) error {
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := crClient.Get(ctx, key, obj); err != nil {
			if client.IgnoreNotFound(err) == nil {
				return nil
			}
			return err
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timed out waiting for %s to be deleted", key.Name)
}

// reportCleanupErrors surfaces collected cleanup failures: the errors are
// attached to the spec report and fail the spec, so a leaking fixture cannot
// pass silently. Genuine not-found responses must already have been filtered
// out by the callers.
func reportCleanupErrors(errs []error) {
	if len(errs) == 0 {
		return
	}
	ginkgo.AddReportEntry("e2e cleanup errors", fmt.Sprintf("%v", errors.Join(errs...)))
	Expect(errs).To(BeEmpty(), "e2e cleanup encountered errors; see the report entry")
}

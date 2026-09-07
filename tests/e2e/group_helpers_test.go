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

package e2e

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	users "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/users_management_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

const (
	groupFixtureUserA = "example@coralogix.com"
	groupFixtureUserB = "example2@coralogix.com"
)

func waitForGroupRemoteSynced(ctx context.Context, crClient client.Client, name types.NamespacedName) *coralogixv1alpha1.Group {
	return waitForGroupRemoteSyncedWithPrintableStatus(ctx, crClient, name, true)
}

// waitForReleasedGroupRemoteSynced does not require status.printableStatus.
// Helm chart 1.0 can set RemoteSynced and status.id without that print column.
func waitForReleasedGroupRemoteSynced(ctx context.Context, crClient client.Client, name types.NamespacedName) *coralogixv1alpha1.Group {
	return waitForGroupRemoteSyncedWithPrintableStatus(ctx, crClient, name, false)
}

func waitForGroupRemoteSyncedWithPrintableStatus(
	ctx context.Context,
	crClient client.Client,
	name types.NamespacedName,
	requirePrintableStatus bool,
) *coralogixv1alpha1.Group {
	fetched := &coralogixv1alpha1.Group{}
	Eventually(func(g Gomega) {
		g.Expect(crClient.Get(ctx, name, fetched)).To(Succeed())
		g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(
			BeTrue(),
			fmt.Sprintf("status=%+v", fetched.Status),
		)
		if requirePrintableStatus {
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
		}
		g.Expect(fetched.Status.ID).ToNot(BeNil())
		g.Expect(*fetched.Status.ID).ToNot(BeEmpty())
	}, time.Minute, time.Second).Should(Succeed())
	return fetched
}

func parseGroupID(id string) int64 {
	parsed, err := strconv.ParseInt(id, 10, 64)
	Expect(err).ToNot(HaveOccurred())
	return parsed
}

func expectRemoteGroupName(ctx context.Context, groupID int64, want string) {
	Eventually(func(g Gomega) {
		resp, httpResp, err := newOpenAPIClientSet().Groups().
			GroupsMgmtServiceGetTeamGroup(ctx, groupID).
			Execute()
		g.Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
		g.Expect(resp.Group).ToNot(BeNil())
		g.Expect(resp.Group.GetName()).To(Equal(want))
	}, time.Minute, time.Second).Should(Succeed())
}

func expectRemoteGroupGone(ctx context.Context, groupID int64) {
	Eventually(func(g Gomega) {
		_, httpResp, err := newOpenAPIClientSet().Groups().
			GroupsMgmtServiceGetTeamGroup(ctx, groupID).
			Execute()
		apiErr := oapicxsdk.NewAPIError(httpResp, err)
		g.Expect(apiErr).To(HaveOccurred())
		g.Expect(oapicxsdk.IsNotFound(apiErr)).To(BeTrue())
	}, time.Minute, time.Second).Should(Succeed())
}

func expectGroupMembers(ctx context.Context, groupID int64, usernames ...string) {
	Eventually(func(g Gomega) {
		wantIDs := make([]string, 0, len(usernames))
		for _, username := range usernames {
			wantIDs = append(wantIDs, userIDByUsername(ctx, g, username))
		}
		gotIDs := remoteGroupUserIDs(ctx, g, groupID)
		g.Expect(gotIDs).To(ConsistOf(wantIDs))
	}, time.Minute, time.Second).Should(Succeed())
}

func userIDByUsername(ctx context.Context, g Gomega, username string) string {
	teamID := whoAmITeamID(ctx, g)
	resp, httpResp, err := newOpenAPIClientSet().Users().
		UsersMgmtServiceSearchUsers(ctx, teamID).
		Username(username).
		PageSize(100).
		Execute()
	g.Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
	g.Expect(resp).ToNot(BeNil())

	var matches []users.RbacV2User
	for _, candidate := range resp.GetUsers() {
		if strings.EqualFold(candidate.GetUsername(), username) && candidate.GetUserId() != "" {
			matches = append(matches, candidate)
		}
	}
	g.Expect(matches).ToNot(BeEmpty(), fmt.Sprintf("no Users OpenAPI match for %s", username))
	return matches[0].GetUserId()
}

func remoteGroupUserIDs(ctx context.Context, g Gomega, groupID int64) []string {
	var ids []string
	var pageToken string
	for {
		req := newOpenAPIClientSet().Groups().
			GroupsMgmtServiceGetGroupUsers(ctx, groupID).
			PageSize(100)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, httpResp, err := req.Execute()
		g.Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
		g.Expect(resp).ToNot(BeNil())
		for _, member := range resp.GetUsers() {
			if id := member.GetUserId(); id != "" {
				ids = append(ids, id)
			}
		}
		next := resp.GetNextPageToken()
		if next == "" || next == pageToken {
			return ids
		}
		pageToken = next
	}
}

func whoAmITeamID(ctx context.Context, g Gomega) int64 {
	resp, httpResp, err := newOpenAPIClientSet().Identity().IdentityServiceWhoAmI(ctx).Execute()
	g.Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
	g.Expect(resp).ToNot(BeNil())
	g.Expect(resp.GetTeamId()).ToNot(BeZero())
	return resp.GetTeamId()
}

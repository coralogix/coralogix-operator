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

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	identity "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/identity_service"
	users "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/users_management_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

const userSearchPageSize = int64(100)

// teamIDCache stores the WhoAmI team id for the process lifetime. The operator
// reads one API key at start; a new key requires a restart.
type teamIDCache struct {
	mu sync.Mutex
	id *int64
}

func (c *teamIDCache) get(ctx context.Context, identityClient *identity.IdentityServiceAPIService) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.id != nil {
		return *c.id, nil
	}

	resp, httpResp, err := identityClient.IdentityServiceWhoAmI(ctx).Execute()
	if err != nil {
		return 0, fmt.Errorf("resolve team id: %w", oapicxsdk.NewAPIError(httpResp, err))
	}
	if resp == nil || resp.GetTeamId() == 0 {
		return 0, fmt.Errorf("resolve team id: WhoAmI returned no team id")
	}

	id := resp.GetTeamId()
	c.id = &id
	return id, nil
}

func resolveMemberUserIDs(
	ctx context.Context,
	usersClient *users.UsersManagementServiceAPIService,
	teamID int64,
	members []coralogixv1alpha1.Member,
) ([]string, error) {
	if len(members) == 0 {
		return nil, nil
	}

	candidates, err := searchUsers(ctx, usersClient, teamID)
	if err != nil {
		return nil, err
	}

	firstID := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		key := strings.ToLower(candidate.GetUsername())
		if key == "" {
			continue
		}
		if _, exists := firstID[key]; exists {
			continue
		}
		if id := candidate.GetUserId(); id != "" {
			firstID[key] = id
		}
	}

	var userIDs []string
	var errs error
	for _, member := range members {
		id, ok := firstID[strings.ToLower(member.UserName)]
		if !ok {
			errs = errors.Join(errs, fmt.Errorf("user %s not found", member.UserName))
			continue
		}
		userIDs = append(userIDs, id)
	}
	if errs != nil {
		return nil, errs
	}
	return userIDs, nil
}

// searchUsers pages SearchUsers for the whole team. A missing next token, or
// one that does not move forward, ends the loop.
func searchUsers(
	ctx context.Context,
	usersClient *users.UsersManagementServiceAPIService,
	teamID int64,
) ([]users.RbacV2User, error) {
	var found []users.RbacV2User
	var pageToken int64

	for {
		req := usersClient.UsersMgmtServiceSearchUsers(ctx, teamID).PageSize(userSearchPageSize)
		if pageToken != 0 {
			req = req.PageToken(pageToken)
		}

		resp, httpResp, err := req.Execute()
		if err != nil {
			return nil, oapicxsdk.NewAPIError(httpResp, err)
		}
		if resp == nil || len(resp.Users) == 0 {
			return found, nil
		}
		found = append(found, resp.Users...)

		next := resp.GetNextPageToken()
		if next <= pageToken {
			return found, nil
		}
		pageToken = next
	}
}

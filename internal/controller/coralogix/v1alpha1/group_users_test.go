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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	users "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/users_management_service"
	"github.com/stretchr/testify/require"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

func TestResolveMemberUserIDsEmptyMembers(t *testing.T) {
	ids, err := resolveMemberUserIDs(context.Background(), nil, 1, nil)
	require.NoError(t, err)
	require.Nil(t, ids)
}

func TestResolveMemberUserIDsSinglePage(t *testing.T) {
	client := usersClientForSearch(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/aaa/teams/v2/7/search", r.URL.Path)
		require.Equal(t, "alice@example.com", r.URL.Query().Get("username"))
		writeSearchUsers(t, w, 0, userJSON("id-1", "alice@example.com"))
	})

	ids, err := resolveMemberUserIDs(context.Background(), client, 7, []coralogixv1alpha1.Member{
		{UserName: "alice@example.com"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"id-1"}, ids)
}

func TestResolveMemberUserIDsPaginates(t *testing.T) {
	pages := 0
	client := usersClientForSearch(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		token := r.URL.Query().Get("page_token")
		switch token {
		case "":
			writeSearchUsers(t, w, 100, userJSON("other", "alice.other@example.com"))
		case "100":
			writeSearchUsers(t, w, 0, userJSON("id-2", "alice@example.com"))
		default:
			t.Fatalf("unexpected page_token %q", token)
		}
	})

	ids, err := resolveMemberUserIDs(context.Background(), client, 7, []coralogixv1alpha1.Member{
		{UserName: "alice@example.com"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"id-2"}, ids)
	require.Equal(t, 2, pages)
}

func TestResolveMemberUserIDsStuckPageToken(t *testing.T) {
	pages := 0
	client := usersClientForSearch(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		writeSearchUsers(t, w, 1, userJSON("id-1", "alice@example.com"))
	})

	ids, err := resolveMemberUserIDs(context.Background(), client, 7, []coralogixv1alpha1.Member{
		{UserName: "alice@example.com"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"id-1"}, ids)
	require.Equal(t, 2, pages)
}

func TestResolveMemberUserIDsMissingUser(t *testing.T) {
	client := usersClientForSearch(t, func(w http.ResponseWriter, r *http.Request) {
		writeSearchUsers(t, w, 0, userJSON("id-1", "bob@example.com"))
	})

	_, err := resolveMemberUserIDs(context.Background(), client, 7, []coralogixv1alpha1.Member{
		{UserName: "alice@example.com"},
	})
	require.ErrorContains(t, err, "user alice@example.com not found")
}

func TestResolveMemberUserIDsTakesFirstCaseInsensitiveMatch(t *testing.T) {
	client := usersClientForSearch(t, func(w http.ResponseWriter, r *http.Request) {
		writeSearchUsers(t, w,
			0,
			userJSON("id-partial", "alice@example.com.br"),
			userJSON("id-first", "ALICE@example.com"),
			userJSON("id-second", "alice@example.com"),
		)
	})

	ids, err := resolveMemberUserIDs(context.Background(), client, 7, []coralogixv1alpha1.Member{
		{UserName: "alice@example.com"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"id-first"}, ids)
}

func TestTeamIDCacheResolvesOnce(t *testing.T) {
	whoamiCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/aaa/identity/v1/whoami", r.URL.Path)
		whoamiCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"teamId":42,"teamName":"test"}`)
	}))
	t.Cleanup(server.Close)

	clientSet := openapicxsdk.NewClientSet(openapicxsdk.NewConfigBuilder().
		WithURL(server.URL).
		WithAPIKey("test").
		Build())
	cache := &teamIDCache{}

	id, err := cache.get(context.Background(), clientSet.Identity())
	require.NoError(t, err)
	require.Equal(t, int64(42), id)

	id, err = cache.get(context.Background(), clientSet.Identity())
	require.NoError(t, err)
	require.Equal(t, int64(42), id)
	require.Equal(t, 1, whoamiCalls)
}

func TestMemberUserIDsSkipsWhoAmIWhenNoMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("WhoAmI must not run when the Group has no members")
	}))
	t.Cleanup(server.Close)

	clientSet := openapicxsdk.NewClientSet(openapicxsdk.NewConfigBuilder().
		WithURL(server.URL).
		WithAPIKey("test").
		Build())
	r := &GroupReconciler{
		IdentityClient: clientSet.Identity(),
		UsersClient:    clientSet.Users(),
	}

	ids, err := r.memberUserIDs(context.Background(), &coralogixv1alpha1.Group{})
	require.NoError(t, err)
	require.Nil(t, ids)
}

func usersClientForSearch(t *testing.T, handler http.HandlerFunc) *users.UsersManagementServiceAPIService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return openapicxsdk.NewClientSet(openapicxsdk.NewConfigBuilder().
		WithURL(server.URL).
		WithAPIKey("test").
		Build()).Users()
}

func userJSON(id, username string) map[string]string {
	return map[string]string{"userId": id, "username": username}
}

func writeSearchUsers(t *testing.T, w http.ResponseWriter, nextPageToken int64, found ...map[string]string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	payload := users.SearchUsersResponse{Users: make([]users.RbacV2User, 0, len(found))}
	for _, user := range found {
		u := users.RbacV2User{}
		u.SetUserId(user["userId"])
		u.SetUsername(user["username"])
		payload.Users = append(payload.Users, u)
	}
	if nextPageToken != 0 {
		payload.SetNextPageToken(nextPageToken)
	}
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cxsdk

import (
	"context"
	"encoding/json"
	"fmt"
)

// UsersClient is a client for the SCIM Users API
type UsersClient struct {
	client *RestClient
}

// SCIMUser represents a SCIM User
type SCIMUser struct {
	Schemas  []string        `json:"schemas"`
	ID       *string         `json:"id,omitempty"`
	UserName string          `json:"userName"`
	Active   bool            `json:"active"`
	Name     *SCIMUserName   `json:"name,omitempty"`
	Groups   []SCIMUserGroup `json:"groups,omitempty"`
	Emails   []SCIMUserEmail `json:"emails,omitempty"`
}

// SCIMUserName represents a SCIM User Name
type SCIMUserName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

// SCIMUserEmail represents a SCIM User Email
type SCIMUserEmail struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
	Type    string `json:"type"`
}

// SCIMUserGroup represents a SCIM User Group
type SCIMUserGroup struct {
	Value string `json:"value"`
}

// Create creates a new SCIM User
func (c UsersClient) Create(ctx context.Context, userReq *SCIMUser) (*SCIMUser, error) {
	body, err := json.Marshal(userReq)
	if err != nil {
		return nil, err
	}

	bodyResp, err := c.client.Post(ctx, "", "application/json", string(body))
	if err != nil {
		return nil, err
	}

	var UserResp SCIMUser
	err = json.Unmarshal([]byte(bodyResp), &UserResp)
	if err != nil {
		return nil, err
	}

	return &UserResp, nil
}

// Get retrieves a SCIM User by ID
func (c UsersClient) Get(ctx context.Context, userID string) (*SCIMUser, error) {
	bodyResp, err := c.client.Get(ctx, fmt.Sprintf("/%s", userID))
	if err != nil {
		return nil, err
	}

	var UserResp SCIMUser
	err = json.Unmarshal([]byte(bodyResp), &UserResp)
	if err != nil {
		return nil, err
	}

	return &UserResp, nil
}

// Update updates a SCIM User by ID
func (c UsersClient) Update(ctx context.Context, userID string, userReq *SCIMUser) (*SCIMUser, error) {
	body, err := json.Marshal(userReq)
	if err != nil {
		return nil, err
	}

	bodyResp, err := c.client.Put(ctx, fmt.Sprintf("/%s", userID), "application/json", string(body))
	if err != nil {
		return nil, err
	}

	var UserResp SCIMUser
	err = json.Unmarshal([]byte(bodyResp), &UserResp)
	if err != nil {
		return nil, err
	}

	return &UserResp, nil
}

// Delete deletes a SCIM User by ID
func (c UsersClient) Delete(ctx context.Context, userID string) error {
	_, err := c.client.Delete(ctx, fmt.Sprintf("/%s", userID))
	return err

}

// NewUsersClient creates a new UsersClient
func NewUsersClient(c *CallPropertiesCreator) *UsersClient {
	restEndpoint := CoralogixRestEndpointFromRegion(c.coraglogixRegion)
	targetURL := restEndpoint + "/scim/Users"
	client := NewRestClient(targetURL, c.teamsLevelAPIKey)
	return &UsersClient{client: client}
}

// BaseURL is a function to fetch the base URL.
func (c UsersClient) BaseURL() string {
	return c.client.url
}

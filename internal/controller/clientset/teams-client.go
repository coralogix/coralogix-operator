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

package clientset

import (
	"context"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

//go:generate mockgen -destination=../mock_clientset/mock_teams-client.go -package=mock_clientset -source=teams-client.go TeamsClientInterface
type TeamsClientInterface interface {
	Create(ctx context.Context, req *cxsdk.CreateTeamInOrgRequest) (*cxsdk.CreateTeamInOrgResponse, error)
	Get(ctx context.Context, req *cxsdk.GetTeamRequest) (*cxsdk.GetTeamResponse, error)
	Update(ctx context.Context, req *cxsdk.UpdateTeamRequest) (*cxsdk.UpdateTeamResponse, error)
	Delete(ctx context.Context, req *cxsdk.DeleteTeamRequest) (*cxsdk.DeleteTeamResponse, error)
}

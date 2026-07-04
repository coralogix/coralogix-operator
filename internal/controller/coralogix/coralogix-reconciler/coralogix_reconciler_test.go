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

package coralogixreconciler

import (
	"errors"
	"net/http"
	"testing"

	grpcsdk "github.com/coralogix/coralogix-management-sdk/go"
	oapisdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsRemoteNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "grpc not found",
			err:  grpcsdk.NewSdkAPIError(status.Error(codes.NotFound, "not found"), "", ""),
			want: true,
		},
		{
			name: "openapi http not found",
			err:  oapisdk.NewAPIError(&http.Response{StatusCode: http.StatusNotFound}, errors.New("not found")),
			want: true,
		},
		{
			name: "openapi forbidden",
			err:  oapisdk.NewAPIError(&http.Response{StatusCode: http.StatusForbidden}, errors.New("forbidden")),
			want: false,
		},
		{
			name: "unknown",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRemoteNotFound(tt.err); got != tt.want {
				t.Fatalf("isRemoteNotFound() = %t, want %t", got, tt.want)
			}
		})
	}
}

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
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

func TestValidateNoEmbeddedIDWithImportRejectsNonEmptyIDWithImportAnnotation(t *testing.T) {
	dashboard := &cxsdk.Dashboard{Id: wrapperspb.String("3d7f1c2a-9e4b-4a11-8f2d-1a2b3c4d5e6f")}

	err := validateNoEmbeddedIDWithImport("some-import-id", dashboard)

	require.ErrorContains(t, err, "app.coralogix.com/import-id")
}

func TestValidateNoEmbeddedIDWithImportAllowsMissingID(t *testing.T) {
	require.NoError(t, validateNoEmbeddedIDWithImport("some-import-id", &cxsdk.Dashboard{}))
	require.NoError(t, validateNoEmbeddedIDWithImport("some-import-id", &cxsdk.Dashboard{Id: wrapperspb.String("")}))
}

func TestValidateNoEmbeddedIDWithImportAllowsNonEmptyIDWithoutImportAnnotation(t *testing.T) {
	dashboard := &cxsdk.Dashboard{Id: wrapperspb.String("3d7f1c2a-9e4b-4a11-8f2d-1a2b3c4d5e6f")}

	require.NoError(t, validateNoEmbeddedIDWithImport("", dashboard))
}

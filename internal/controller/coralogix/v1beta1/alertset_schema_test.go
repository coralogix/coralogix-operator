// Copyright 2026 Coralogix Ltd.
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

package v1beta1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

func TestAlertSetUsesTheAlertSpecSchema(t *testing.T) {
	alertCRD := readCRD(t, "coralogix.com_alerts.yaml")
	alertSetCRD := readCRD(t, "coralogix.com_alertsets.yaml")

	alertSchema := alertCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	alertSetRootSchema := alertSetCRD.Spec.Versions[0].Schema.OpenAPIV3Schema
	require.Contains(t, alertSetRootSchema.Required, "spec")
	alertSetSpec := alertSetRootSchema.Properties["spec"]
	alertsSchema := alertSetSpec.Properties["alerts"]
	require.NotNil(t, alertsSchema.Items)
	require.NotNil(t, alertsSchema.Items.Schema)
	alertSetItemSchema := alertsSchema.Items.Schema.Properties["spec"]

	require.Equal(t, schemaWithoutDescriptions(t, alertSchema), schemaWithoutDescriptions(t, alertSetItemSchema))
}

func readCRD(t *testing.T, name string) apiextensionsv1.CustomResourceDefinition {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "config", "crd", "bases", name)
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	var crd apiextensionsv1.CustomResourceDefinition
	require.NoError(t, yaml.Unmarshal(contents, &crd))
	return crd
}

func schemaWithoutDescriptions(t *testing.T, schema apiextensionsv1.JSONSchemaProps) any {
	t.Helper()
	contents, err := json.Marshal(schema)
	require.NoError(t, err)
	var value any
	require.NoError(t, json.Unmarshal(contents, &value))
	removeSchemaDescriptions(value)
	return value
}

func removeSchemaDescriptions(value any) {
	switch typed := value.(type) {
	case map[string]any:
		delete(typed, "description")
		for _, child := range typed {
			removeSchemaDescriptions(child)
		}
	case []any:
		for _, child := range typed {
			removeSchemaDescriptions(child)
		}
	}
}

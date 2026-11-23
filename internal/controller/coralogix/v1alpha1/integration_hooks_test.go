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
	"sort"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	integrations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/integration_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func integrationWithSecret(t *testing.T) *coralogixv1alpha1.Integration {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-role", Namespace: "default"},
		Data:       map[string][]byte{"role-arn": []byte("arn:aws:iam::123456789012:role/example-role")},
	}
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build())
	return &coralogixv1alpha1.Integration{
		ObjectMeta: metav1.ObjectMeta{Name: "integration", Namespace: "default"},
		Spec: coralogixv1alpha1.IntegrationSpec{
			IntegrationKey: "aws-metrics-collector",
			Version:        "0.1.0",
			Parameters: coralogixv1alpha1.IntegrationParameters{RawExtension: runtime.RawExtension{
				Raw: []byte(`{"ApplicationName":"cxo","MetricNamespaces":["AWS/S3"],"WithAggregations":false}`),
			}},
			ParametersFromSecret: coralogixv1alpha1.IntegrationParametersFromSecret{
				"AwsRoleArn": {LocalObjectReference: corev1.LocalObjectReference{Name: "aws-role"}, Key: "role-arn"},
			},
		},
	}
}

// sortedParameters returns the parameter list as JSON, sorted by key: the hooks build it from a
// map, so its order changes from run to run.
func sortedParameters(t *testing.T, parameters []integrations.Parameter) string {
	t.Helper()
	sorted := append([]integrations.Parameter(nil), parameters...)
	sort.Slice(sorted, func(i, j int) bool { return *sorted[i].Key < *sorted[j].Key })
	out, err := json.Marshal(sorted)
	require.NoError(t, err)
	return string(out)
}

// The parameter hooks run on every Create attempt and every reconcile. Running one twice on one
// body leaves the same parameter list.
func TestIntegrationParameterHooksAreSafeToRepeat(t *testing.T) {
	cr := integrationWithSecret(t)
	r := &IntegrationReconciler{}
	want := `[{"key":"ApplicationName","stringValue":"cxo"},` +
		`{"key":"AwsRoleArn","stringValue":"arn:aws:iam::123456789012:role/example-role"},` +
		`{"key":"MetricNamespaces","stringList":{"values":["AWS/S3"]}},` +
		`{"booleanValue":false,"key":"WithAggregations"}]`

	create, err := cr.CreateRequest(context.Background())
	require.NoError(t, err)
	require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), cr, create))
	require.JSONEq(t, want, sortedParameters(t, create.Metadata.IntegrationParameters.Parameters))
	require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), cr, create))
	require.JSONEq(t, want, sortedParameters(t, create.Metadata.IntegrationParameters.Parameters), "a second run must not add to the body")

	update, err := cr.UpdateRequest(context.Background())
	require.NoError(t, err)
	require.NoError(t, r.beforeUpdate(context.Background(), logr.Discard(), cr, update))
	require.JSONEq(t, want, sortedParameters(t, update.Metadata.IntegrationParameters.Parameters))
	require.NoError(t, r.beforeUpdate(context.Background(), logr.Discard(), cr, update))
	require.JSONEq(t, want, sortedParameters(t, update.Metadata.IntegrationParameters.Parameters), "a second run must not add to the body")
}

// Without parameters, the hooks leave the list empty, and the request keeps integrationParameters
// as an empty object: the API rejects a request without it.
func TestIntegrationHooksWithoutParametersSendAnEmptyObject(t *testing.T) {
	cr := integrationWithSecret(t)
	cr.Spec.Parameters = coralogixv1alpha1.IntegrationParameters{}
	cr.Spec.ParametersFromSecret = nil
	body, err := cr.CreateRequest(context.Background())
	require.NoError(t, err)
	require.NoError(t, (&IntegrationReconciler{}).beforeCreate(context.Background(), logr.Discard(), cr, body))
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	require.JSONEq(t, `{"metadata":{"integrationKey":"aws-metrics-collector","version":"0.1.0","integrationParameters":{}}}`, string(raw))
}

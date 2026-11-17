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

package baseline

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// admissionResult is what the API server answered to the create of one admission case.
type admissionResult struct {
	Accepted bool `json:"accepted"`
	// Causes are the field errors of a rejection, sorted by field and message.
	Causes []admissionCause `json:"causes,omitempty"`
	// Message is the error text of a rejection that has no field errors.
	Message string `json:"message,omitempty"`
}

type admissionCause struct {
	Type    string `json:"type"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// TestAdmission creates each admission case of a kind in envtest and records whether the API
// server accepts it. It covers the CRD schema and its CEL rules with real values, which the CRD
// golden shows only as text. No reconciler runs.
func TestAdmission(t *testing.T) {
	for _, k := range sortedKinds() {
		if len(k.admission) == 0 {
			continue
		}
		t.Run(k.name, func(t *testing.T) {
			c := startEnv(t)
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName("admission-" + k.name)}}
			if err := c.Create(context.Background(), ns); err != nil {
				t.Fatalf("create namespace: %v", err)
			}
			names := make([]string, 0, len(k.admission))
			for name := range k.admission {
				names = append(names, name)
			}
			sort.Strings(names)

			results := map[string]admissionResult{}
			for i, name := range names {
				obj := &unstructured.Unstructured{}
				if err := yaml.Unmarshal([]byte(k.admission[name]), &obj.Object); err != nil {
					t.Fatalf("%s: parse manifest: %v", name, err)
				}
				obj.SetNamespace(ns.Name)
				// Each case gets its own object name, so an accepted case never blocks the next.
				obj.SetName(fmt.Sprintf("%s-%03d", k.name, i))
				err := c.Create(context.Background(), obj)
				results[name] = admissionAnswer(t, name, err)
				if err == nil {
					if err := c.Delete(context.Background(), obj); err != nil {
						t.Fatalf("%s: delete: %v", name, err)
					}
				}
			}
			checkGolden(t, k.goldenDir(), "admission.json", marshalGolden(t, results))
		})
	}
}

func admissionAnswer(t *testing.T, name string, err error) admissionResult {
	t.Helper()
	if err == nil {
		return admissionResult{Accepted: true}
	}
	var status apierrors.APIStatus
	if !errors.As(err, &status) || !apierrors.IsInvalid(err) {
		t.Fatalf("%s: the create failed, but not with a validation error: %v", name, err)
	}
	result := admissionResult{}
	if details := status.Status().Details; details != nil {
		for _, cause := range details.Causes {
			// The API server adds this cause when a schema error stops it before the CEL rules.
			// Whether it appears depends on where a rule sits, not on what is accepted.
			if strings.Contains(cause.Message, "some validation rules were not checked") {
				continue
			}
			result.Causes = append(result.Causes, admissionCause{
				Type: string(cause.Type), Field: cause.Field, Message: cause.Message,
			})
		}
	}
	sort.Slice(result.Causes, func(i, j int) bool {
		a, b := result.Causes[i], result.Causes[j]
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		return a.Message < b.Message
	})
	if len(result.Causes) == 0 {
		result.Message = status.Status().Message
	}
	return result
}

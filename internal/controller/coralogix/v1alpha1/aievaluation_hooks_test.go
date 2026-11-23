// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The released kind rejects a threshold outside 0..1 before any request. The hook keeps the bound
// inclusive, reads every Quantity form, and gives the same answer each time it runs.
func TestAIEvaluationValidateChecksTheThreshold(t *testing.T) {
	for _, tc := range []struct {
		threshold string
		ok        bool
	}{
		{"0", true},
		{"1", true},
		{"0.8", true},
		{"750m", true},
		{"0.333333333", true},
		{"1.5", false},
		{"2", false},
		{"1001m", false},
		{"-0.1", false},
		{"-1", false},
	} {
		t.Run(tc.threshold, func(t *testing.T) {
			r := &AIEvaluationReconciler{}
			cr := &coralogixv1alpha1.AIEvaluation{Spec: coralogixv1alpha1.AIEvaluationSpec{Threshold: resource.MustParse(tc.threshold)}}
			for range 2 {
				err := r.validate(context.Background(), logr.Discard(), cr)
				if tc.ok {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, "spec.threshold must be between 0 and 1 inclusive")
				}
			}
		})
	}
}

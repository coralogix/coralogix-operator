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
	"errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/resource"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

var maxAIEvaluationThreshold = resource.MustParse("1")

// validate keeps the released check of the threshold: a value outside 0..1 fails before Create or
// Update sends a request. The CRD holds the threshold as a Quantity, which cannot carry the bound
// on the oldest Kubernetes version in CI.
func (r *AIEvaluationReconciler) validate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.AIEvaluation) error {
	threshold := cr.Spec.Threshold
	if threshold.Sign() < 0 || threshold.Cmp(maxAIEvaluationThreshold) > 0 {
		return errors.New("spec.threshold must be between 0 and 1 inclusive")
	}
	return nil
}

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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	recordingrulesservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/recording_rules_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func TestRecordingRuleGroupSetBeforeCreateNamesTheSet(t *testing.T) {
	previous := config.GetConfig().RecordingRuleGroupSetSuffix
	t.Cleanup(func() { config.GetConfig().RecordingRuleGroupSetSuffix = previous })

	r := &RecordingRuleGroupSetReconciler{}
	cr := &coralogixv1alpha1.RecordingRuleGroupSet{ObjectMeta: metav1.ObjectMeta{Name: "rules"}}
	for suffix, want := range map[string]string{"": "rules", "-eu2": "rules-eu2"} {
		config.GetConfig().RecordingRuleGroupSetSuffix = suffix
		body := &recordingrulesservice.CreateRuleGroupSet{}
		// The hook is safe to repeat: the second run leaves the same name.
		for range 2 {
			require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), cr, body))
			require.NotNil(t, body.Name)
			require.Equal(t, want, *body.Name)
		}
	}
}

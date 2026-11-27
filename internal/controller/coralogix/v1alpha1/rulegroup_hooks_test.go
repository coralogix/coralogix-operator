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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

func ruleGroupForHooks() *coralogixv1alpha1.RuleGroup {
	return &coralogixv1alpha1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "rulegroup", Namespace: "default"},
		Spec: coralogixv1alpha1.RuleGroupSpec{
			Name:         "group",
			Applications: []string{"shop"},
			Subsystems:   []string{"payments"},
			Severities:   []coralogixv1alpha1.RuleSeverity{coralogixv1alpha1.RuleSeverityWarning},
			Subgroups: []coralogixv1alpha1.RuleSubGroup{{
				Rules: []coralogixv1alpha1.Rule{
					{Name: "extract", Extract: &coralogixv1alpha1.Extract{SourceField: "text", Regex: "a"}},
					{Name: "allow", Block: &coralogixv1alpha1.Block{SourceField: "text", Regex: "b"}},
				},
			}},
		},
	}
}

// The hooks run on every Create attempt and every reconcile. Running one twice on one body leaves
// the same matchers and subgroups, and the same body as one run on a new body: each run builds both
// lists whole.
func TestRuleGroupHooksAreSafeToRepeat(t *testing.T) {
	ctx := context.Background()
	cr := ruleGroupForHooks()
	r := &RuleGroupReconciler{}
	marshal := func(v any) string {
		out, err := json.Marshal(v)
		require.NoError(t, err)
		return string(out)
	}

	for name, run := range map[string]func() (string, string){
		"beforeCreate": func() (string, string) {
			body, err := cr.CreateRequest(ctx)
			require.NoError(t, err)
			require.NoError(t, r.beforeCreate(ctx, logr.Discard(), cr, body))
			once := marshal(body)
			require.NoError(t, r.beforeCreate(ctx, logr.Discard(), cr, body))
			return once, marshal(body)
		},
		"beforeUpdate": func() (string, string) {
			body, err := cr.UpdateRequest(ctx)
			require.NoError(t, err)
			require.NoError(t, r.beforeUpdate(ctx, logr.Discard(), cr, body))
			once := marshal(body)
			require.NoError(t, r.beforeUpdate(ctx, logr.Discard(), cr, body))
			return once, marshal(body)
		},
	} {
		t.Run(name, func(t *testing.T) {
			once, twice := run()
			require.Equal(t, once, twice)
			require.Contains(t, once, `"ruleMatchers":[{"applicationName":{"value":"shop"}},{"subsystemName":{"value":"payments"}},{"severity":{"value":"VALUE_WARNING"}}]`)
			require.Contains(t, once, `"order":1`)
			require.Contains(t, once, `"allowParameters"`, "a block rule with blockingAllMatchingBlocks unset is an allow rule, as in the released controller")
		})
	}
}

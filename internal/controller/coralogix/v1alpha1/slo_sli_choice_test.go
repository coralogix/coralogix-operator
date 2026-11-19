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
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// sliType and sliDefinition are two ways to state the SLI. The API uses sliDefinition and ignores
// sliType when a request has both, so the CRD accepts exactly one of them.
var _ = Describe("SLO sliType or sliDefinition", func() {
	const (
		legacy = `"sliType": {"requestBasedMetric": {
			"goodEvents": {"query": "sum(rate(good_total[1m]))"},
			"totalEvents": {"query": "sum(rate(all_total[1m]))"}}}`
		definition = `"sliDefinition": {"requestBased": {"custom": {
			"goodEvents": {"query": "sum(rate(good_total[1m]))"},
			"totalEvents": {"query": "sum(rate(all_total[1m]))"}}}}`
		rest = `"targetThresholdPercentage": 99, "window": {"timeFrame": "7d"}`
	)
	spec := func(parts ...string) string {
		return "{" + strings.Join(append(parts, rest), ",") + "}"
	}
	for i, tc := range []struct {
		name  string
		spec  string
		valid bool
		// rejected is the text of the rule that refuses the spec.
		rejected string
		// request is a key that the Create request must hold, and absent one that it must not hold.
		request, absent string
	}{
		{name: "sliType alone", spec: spec(legacy), valid: true, request: `"requestBasedMetricSli":`, absent: `"sliDefinition":`},
		{name: "sliDefinition alone", spec: spec(definition), valid: true, request: `"sliDefinition":`, absent: `"requestBasedMetricSli":`},
		{name: "both", spec: spec(legacy, definition), rejected: "Exactly one of sliType, sliDefinition must be set"},
		{name: "neither", spec: spec(), rejected: "Exactly one of sliType, sliDefinition must be set"},
		{name: "an empty sliType", spec: spec(`"sliType": {}`), rejected: "Exactly one of requestBasedMetric, windowBasedMetric, apmSli must be set"},
		{name: "an empty sliType beside sliDefinition", spec: spec(`"sliType": {}`, definition), rejected: "Exactly one of sliType, sliDefinition must be set"},
		// The rule of the released product type reads sliType, which a sliDefinition replaces.
		{name: "product type apm with a sliDefinition", spec: spec(definition, `"productType": "apm"`), valid: true},
		{name: "product type apm with a custom sliType", spec: spec(legacy, `"productType": "apm"`), rejected: "productType 'apm' requires sliType.apmSli"},
	} {
		It(tc.name, func(ctx context.Context) {
			cr := newSLOCoverageCR(fmt.Sprintf("choice-%d", i), tc.spec)
			err := k8sClient.Create(ctx, cr.DeepCopy(), client.DryRunAll)
			if !tc.valid {
				Expect(err).To(MatchError(ContainSubstring(tc.rejected)))
				return
			}
			Expect(err).NotTo(HaveOccurred())
			body, err := cr.CreateRequest(ctx)
			Expect(err).NotTo(HaveOccurred())
			sent, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())
			if tc.request != "" {
				Expect(string(sent)).To(ContainSubstring(tc.request))
			}
			if tc.absent != "" {
				Expect(string(sent)).NotTo(ContainSubstring(tc.absent))
			}
		})
	}
})

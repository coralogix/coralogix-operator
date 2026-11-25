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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// The generated rules of the secretRef and inArms keys, on the synthetic ArmThing: each arm takes
// its URL as a literal or from a Secret, exactly one of them, and a header has only one source.
var _ = Describe("Generated Secret reference validation", Ordered, func() {
	BeforeAll(func() {
		_, err := envtest.InstallCRDs(cfg, envtest.CRDInstallOptions{Paths: []string{opgenCRDs}, ErrorIfPathMissing: true})
		Expect(err).NotTo(HaveOccurred())
	})

	newArmThing := func(name string, spec map[string]any) *unstructured.Unstructured {
		u := &unstructured.Unstructured{Object: map[string]any{"spec": spec}}
		u.SetGroupVersionKind(schema.GroupVersionKind{Group: "coralogix.com", Version: "v1alpha1", Kind: "ArmThing"})
		u.SetName(name)
		u.SetNamespace("default")
		return u
	}
	ref := func(key string) map[string]any { return map[string]any{"name": "values", "key": key} }

	It("accepts each source of a value", func(ctx context.Context) {
		for i, spec := range []map[string]any{
			{"name": "a", "quiet": map[string]any{"url": "https://example.com"}},
			{"name": "a", "quiet": map[string]any{"urlSecretKeyRef": ref("url")}},
			{"name": "a", "hook": map[string]any{"url": "https://example.com", "token": "t"}},
			{"name": "a", "hook": map[string]any{
				"urlSecretKeyRef": ref("url"), "tokenSecretKeyRef": ref("token"),
				"headers": map[string]any{"X-Team": "core"}, "headersFromSecret": map[string]any{"Authorization": ref("auth")},
			}},
		} {
			thing := newArmThing(fmt.Sprintf("accepted-%d", i), spec)
			Expect(k8sClient.Create(ctx, thing)).To(Succeed())
			Expect(k8sClient.Delete(ctx, thing)).To(Succeed())
		}
	})

	It("rejects two sources or none", func(ctx context.Context) {
		for i, tc := range []struct {
			spec map[string]any
			want string
		}{
			{map[string]any{"name": "a", "quiet": map[string]any{"url": "https://example.com", "urlSecretKeyRef": ref("url")}}, "Exactly one of url or urlSecretKeyRef must be set"},
			{map[string]any{"name": "a", "quiet": map[string]any{}}, "Exactly one of url or urlSecretKeyRef must be set"},
			{map[string]any{"name": "a", "hook": map[string]any{"url": "https://example.com"}}, "Exactly one of token or tokenSecretKeyRef must be set"},
			{map[string]any{"name": "a", "hook": map[string]any{
				"url": "https://example.com", "token": "t",
				"headers": map[string]any{"Authorization": "x"}, "headersFromSecret": map[string]any{"Authorization": ref("auth")},
			}}, "A key of headers can have only one source"},
		} {
			err := k8sClient.Create(ctx, newArmThing(fmt.Sprintf("rejected-%d", i), tc.spec))
			Expect(err).To(HaveOccurred(), "case %d", i)
			Expect(err.Error()).To(ContainSubstring(tc.want), "case %d", i)
		}
	})
})

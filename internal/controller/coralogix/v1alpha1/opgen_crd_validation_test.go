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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// The CRDs of the synthetic kinds of the operator generator. The generator test writes them with
// controller-gen from the generated types.
var opgenCRDs = filepath.Join("..", "..", "..", "..", "tools", "opgen", "internal", "generator", "testdata", "crd")

// The generated validation of a new kind: kubectl accepts a minimal Thing, and rejects a change of
// an immutable field, two oneOf arms, a value outside an enum, and a map value outside its limits.
var _ = Describe("Generated CRD validation", Ordered, func() {
	BeforeAll(func() {
		_, err := envtest.InstallCRDs(cfg, envtest.CRDInstallOptions{Paths: []string{opgenCRDs}, ErrorIfPathMissing: true})
		Expect(err).NotTo(HaveOccurred())
	})

	newThing := func(name string, spec map[string]any) *unstructured.Unstructured {
		u := &unstructured.Unstructured{Object: map[string]any{"spec": spec}}
		u.SetGroupVersionKind(schema.GroupVersionKind{Group: "coralogix.com", Version: "v1alpha1", Kind: "Thing"})
		u.SetName(name)
		u.SetNamespace("default")
		return u
	}
	minimalSpec := func() map[string]any {
		return map[string]any{
			"name":   "minimal",
			"kind":   "standard",
			"config": map[string]any{"http": map[string]any{"endpoint": "https://example.com"}},
		}
	}

	It("should accept a minimal Thing", func(ctx context.Context) {
		thing := newThing("minimal", minimalSpec())
		Expect(k8sClient.Create(ctx, thing)).To(Succeed())
		Expect(k8sClient.Delete(ctx, thing)).To(Succeed())
	})

	It("should reject a value outside the enum", func(ctx context.Context) {
		spec := minimalSpec()
		spec["kind"] = "THING_KIND_STANDARD"
		err := k8sClient.Create(ctx, newThing("bad-enum", spec))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`supported values: "standard", "advanced"`))
	})

	It("should check the limits of a map value", func(ctx context.Context) {
		spec := minimalSpec()
		spec["labels"] = map[string]any{"team": ""}
		err := k8sClient.Create(ctx, newThing("empty-label", spec))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.labels.team"))
		Expect(err.Error()).To(ContainSubstring("should be at least 1 chars long"))

		spec["labels"] = map[string]any{"team": "core"}
		thing := newThing("label", spec)
		Expect(k8sClient.Create(ctx, thing)).To(Succeed())
		Expect(k8sClient.Delete(ctx, thing)).To(Succeed())
	})

	It("should reject two oneOf arms", func(ctx context.Context) {
		spec := minimalSpec()
		spec["config"] = map[string]any{
			"http":  map[string]any{"endpoint": "https://example.com"},
			"queue": map[string]any{"topic": "events"},
		}
		err := k8sClient.Create(ctx, newThing("two-arms", spec))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of http, queue must be set"))
	})

	It("should reject a change, an add, or a removal of an immutable field", func(ctx context.Context) {
		tests := []struct {
			name   string
			before func(spec map[string]any)
			after  func(spec map[string]any)
			want   string
		}{
			{
				name:  "change",
				after: func(spec map[string]any) { spec["kind"] = "advanced" },
				want:  "spec.kind is immutable",
			},
			{
				name:   "nested-change",
				before: func(spec map[string]any) { spec["settings"] = map[string]any{"mode": "fast", "region": "eu"} },
				after:  func(spec map[string]any) { spec["settings"].(map[string]any)["region"] = "us" },
				want:   "spec.settings.region is immutable",
			},
			{
				name:   "nested-add",
				before: func(spec map[string]any) { spec["settings"] = map[string]any{"mode": "fast"} },
				after:  func(spec map[string]any) { spec["settings"].(map[string]any)["region"] = "us" },
				want:   "spec.settings.region is immutable",
			},
			{
				name:   "nested-removal",
				before: func(spec map[string]any) { spec["settings"] = map[string]any{"mode": "fast", "region": "eu"} },
				after:  func(spec map[string]any) { delete(spec["settings"].(map[string]any), "region") },
				want:   "spec.settings.region is immutable",
			},
			{
				name:  "parent-add",
				after: func(spec map[string]any) { spec["settings"] = map[string]any{"mode": "fast", "region": "eu"} },
				want:  "spec.settings.region is immutable",
			},
			{
				name:   "parent-removal",
				before: func(spec map[string]any) { spec["settings"] = map[string]any{"mode": "fast", "region": "eu"} },
				after:  func(spec map[string]any) { delete(spec, "settings") },
				want:   "spec.settings.region is immutable",
			},
		}
		for _, tt := range tests {
			spec := minimalSpec()
			if tt.before != nil {
				tt.before(spec)
			}
			thing := newThing(fmt.Sprintf("immutable-%s", tt.name), spec)
			Expect(k8sClient.Create(ctx, thing)).To(Succeed())

			changed, _, err := unstructured.NestedMap(thing.Object, "spec")
			Expect(err).NotTo(HaveOccurred())
			tt.after(changed)
			Expect(unstructured.SetNestedMap(thing.Object, changed, "spec")).To(Succeed())
			err = k8sClient.Update(ctx, thing)
			Expect(err).To(HaveOccurred(), tt.name)
			Expect(err.Error()).To(ContainSubstring(tt.want), tt.name)

			Expect(k8sClient.Delete(ctx, thing)).To(Succeed())
		}
	})

	It("should reject a reference that sets both backendRef and resourceRef", func(ctx context.Context) {
		refThing := &unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{
			"name": "both",
			"thingRef": map[string]any{
				"backendRef":  map[string]any{"id": "thing-id"},
				"resourceRef": map[string]any{"name": "thing"},
			},
		}}}
		refThing.SetGroupVersionKind(schema.GroupVersionKind{Group: "coralogix.com", Version: "v1alpha1", Kind: "RefThing"})
		refThing.SetName("both-refs")
		refThing.SetNamespace("default")
		err := k8sClient.Create(ctx, refThing)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of backendRef or resourceRef must be set"))
	})

	It("should allow a mutable field to change", func(ctx context.Context) {
		thing := newThing("mutable", minimalSpec())
		Expect(k8sClient.Create(ctx, thing)).To(Succeed())
		Expect(unstructured.SetNestedField(thing.Object, "renamed", "spec", "name")).To(Succeed())
		Expect(k8sClient.Update(ctx, thing)).To(Succeed())
		Expect(k8sClient.Delete(ctx, thing)).To(Succeed())
	})

	It("should allow the other fields of an object with an immutable field to change", func(ctx context.Context) {
		spec := minimalSpec()
		spec["settings"] = map[string]any{"mode": "fast", "region": "eu"}
		thing := newThing("mutable-settings", spec)
		Expect(k8sClient.Create(ctx, thing)).To(Succeed())
		Expect(unstructured.SetNestedField(thing.Object, "slow", "spec", "settings", "mode")).To(Succeed())
		Expect(unstructured.SetNestedField(thing.Object, int64(3), "spec", "settings", "retries")).To(Succeed())
		Expect(k8sClient.Update(ctx, thing)).To(Succeed())
		Expect(k8sClient.Delete(ctx, thing)).To(Succeed())
	})

	It("should allow an object without an immutable field to be added and removed", func(ctx context.Context) {
		thing := newThing("settings-without-region", minimalSpec())
		Expect(k8sClient.Create(ctx, thing)).To(Succeed())
		Expect(unstructured.SetNestedMap(thing.Object, map[string]any{"mode": "fast"}, "spec", "settings")).To(Succeed())
		Expect(k8sClient.Update(ctx, thing)).To(Succeed())
		unstructured.RemoveNestedField(thing.Object, "spec", "settings")
		Expect(k8sClient.Update(ctx, thing)).To(Succeed())
		Expect(k8sClient.Delete(ctx, thing)).To(Succeed())
	})
})

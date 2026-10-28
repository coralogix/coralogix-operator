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
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func TestResolveResourceID(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	folder := func(name, namespace, id string, labels map[string]string) *ViewFolder {
		f := &ViewFolder{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels}}
		if id != "" {
			f.Status.ID = ptr.To(id)
		}
		return f
	}
	originalClient, originalSelector := config.GetClient(), config.GetConfig().Selector
	t.Cleanup(func() {
		config.InitClient(originalClient)
		config.GetConfig().Selector = originalSelector
	})
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		folder("same", "team", "same-id", map[string]string{"team": "a"}),
		folder("other", "shared", "other-id", map[string]string{"team": "a"}),
		folder("pending", "team", "", map[string]string{"team": "a"}),
		folder("foreign", "team", "foreign-id", map[string]string{"team": "b"}),
	).Build())
	config.GetConfig().Selector = config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "a"})}

	tests := []struct {
		name    string
		ref     ResourceIDRef
		want    string
		wantErr string
	}{
		{name: "backendRef", ref: ResourceIDRef{BackendRef: &NCBackendRef{ID: "backend-id"}}, want: "backend-id"},
		{name: "same namespace", ref: ResourceIDRef{ResourceRef: &ResourceRef{Name: "same"}}, want: "same-id"},
		{name: "other namespace", ref: ResourceIDRef{ResourceRef: &ResourceRef{Name: "other", Namespace: ptr.To("shared")}}, want: "other-id"},
		{name: "no id yet", ref: ResourceIDRef{ResourceRef: &ResourceRef{Name: "pending"}}, wantErr: "ID is not populated for ViewFolder pending"},
		{name: "selector mismatch", ref: ResourceIDRef{ResourceRef: &ResourceRef{Name: "foreign"}}, wantErr: "ViewFolder foreign does not match selector"},
		{name: "missing", ref: ResourceIDRef{ResourceRef: &ResourceRef{Name: "missing"}}, wantErr: "not found"},
		{name: "empty", ref: ResourceIDRef{}, wantErr: "sets neither backendRef nor resourceRef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveResourceID(context.Background(), "team", tt.ref, "ViewFolder")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want an error with %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

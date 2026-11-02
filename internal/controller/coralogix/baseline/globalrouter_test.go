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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const globalRouterPath = "/notifications/notification-center/v1/routers"

func init() {
	register(kind{
		name:     "globalrouter",
		crdFile:  "coralogix.com_globalrouters.yaml",
		resource: "globalrouters",
		newFake:  func() apiFake { return &globalRouterFake{routers: map[string]any{}} },
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.GlobalRouterReconciler{
				GlobalRoutersClient: cs.GlobalRouters(), Interval: reconcileInterval,
			}
		},
		scenarios: globalRouterScenarios,
	})
}

const globalRouterBackendManifest = `apiVersion: coralogix.com/v1alpha1
kind: GlobalRouter
metadata:
  name: router
spec:
  name: Baseline router
  description: Baseline description
  id: router_spec_id
  entityLabels:
    team: example
  routingLabels:
    environment: test
    service: api
    team: example
  disabled: true
  fallbackTargets:
    - entityType: cases
      target:
        connector:
          backendRef:
            id: connector_cases
  rules:
    - name: urgent
      entityType: alerts
      condition: 'alertDef.priority == "P1"'
      customDetails:
        source: baseline
      targets:
        - connector:
            backendRef:
              id: connector_alerts
          preset:
            backendRef:
              id: preset_alerts
          customDetails:
            channel: primary
`

const globalRouterFallbackManifest = `apiVersion: coralogix.com/v1alpha1
kind: GlobalRouter
metadata:
  name: router
spec:
  name: Deprecated fallback router
  description: Deprecated fallback baseline
  fallback:
    - connector:
        backendRef:
          id: connector_fallback
`

const globalRouterReferenceManifest = `apiVersion: coralogix.com/v1alpha1
kind: GlobalRouter
metadata:
  name: router
spec:
  name: Reference router
  description: Reference baseline
  rules:
    - name: with-preset
      condition: 'alertDef.priority == "P1"'
      targets:
        - connector:
            resourceRef:
              name: connector
          preset:
            resourceRef:
              name: preset
    - name: without-preset
      condition: 'alertDef.priority == "P2"'
      targets:
        - connector:
            resourceRef:
              name: connector
`

var globalRouterScenarios = map[string]func(h *harness){
	"backend-shapes": func(h *harness) {
		h.Create("create with backend references and released shapes", globalRouterBackendManifest)
		h.Reconcile("update with status id")
		h.Delete("delete with status id")
	},
	"deprecated-fallback": func(h *harness) {
		h.Create("create with deprecated fallback", globalRouterFallbackManifest)
	},
	"resource-references": func(h *harness) {
		createGlobalRouterReference(h, "config/samples/v1alpha1/connectors/slack.yaml", "connector", "connector_status_id")
		createGlobalRouterReference(h, "config/samples/v1alpha1/presets/slack.yaml", "preset", "preset_status_id")
		h.Create("create with connector and preset resource references", globalRouterReferenceManifest)
	},
	"missing-reference-id": func(h *harness) {
		createGlobalRouterReference(h, "config/samples/v1alpha1/connectors/slack.yaml", "connector", "")
		h.Create("do not call the API before the connector has a status id", globalRouterReferenceManifest)
	},
	"missing-preset-id": func(h *harness) {
		createGlobalRouterReference(h, "config/samples/v1alpha1/connectors/slack.yaml", "connector", "connector_status_id")
		createGlobalRouterReference(h, "config/samples/v1alpha1/presets/slack.yaml", "preset", "")
		h.Create("do not call the API before the preset has a status id", globalRouterReferenceManifest)
	},
}

func createGlobalRouterReference(h *harness, file, name, id string) {
	h.t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, file))
	if err != nil {
		h.t.Fatal(err)
	}
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(raw, &obj.Object); err != nil {
		h.t.Fatal(err)
	}
	obj.SetName(name)
	obj.SetNamespace(h.namespace)
	if err := h.client.Create(context.Background(), obj); err != nil {
		h.t.Fatalf("create %s: %v", name, err)
	}
	if id == "" {
		return
	}
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set %s status id: %v", name, err)
	}
}

// globalRouterFake only models the responses needed to record the handwritten requests.
type globalRouterFake struct {
	mu      sync.Mutex
	routers map[string]any
}

func (f *globalRouterFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, _ := body.(map[string]any)
	router, _ := in["router"].(map[string]any)
	switch {
	case method == http.MethodPost && path == globalRouterPath:
		id := newUUID()
		f.routers[id] = router
		return http.StatusOK, map[string]any{"router": map[string]any{"id": id, "name": router["name"]}}
	case method == http.MethodPut && path == globalRouterPath:
		id, _ := router["id"].(string)
		f.routers[id] = router
		return http.StatusOK, map[string]any{"router": router}
	case method == http.MethodDelete && strings.HasPrefix(path, globalRouterPath+"/"):
		id := strings.TrimPrefix(path, globalRouterPath+"/")
		delete(f.routers, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

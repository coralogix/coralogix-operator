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

// Package baseline records what a kind does today: the API requests, the CR status after each
// step, the CRD, the RBAC rules, and the finalizer. The goldens are recorded on the handwritten
// kind. A later change to the kind, such as a switch to generated code, shows each behavior
// change as a golden diff.
//
//	UPDATE_GOLDEN=1 go test ./internal/controller/coralogix/baseline
//
// The scenario tests need envtest. `make unit-tests` sets KUBEBUILDER_ASSETS.
package baseline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// repoRoot is the path from this package to the repository root.
const repoRoot = "../../../.."

// kind is one Coralogix kind under test. A new kind adds one file that calls register.
type kind struct {
	// name is the testdata directory and the subtest name.
	name string
	// crdFile is the CRD file name under config/crd/bases.
	crdFile string
	// resource is the plural resource name in the RBAC rules.
	resource string
	// newFake returns an empty fake of the API that the kind calls.
	newFake func() apiFake
	// newReconciler returns the kind's reconciler, with SDK clients built from the client set.
	newReconciler func(cs *openapicxsdk.ClientSet) reconcile.Reconciler
	scenarios     map[string]func(h *harness)
	fidelity      fidelityConfig
}

var kinds = map[string]kind{}

func register(k kind) {
	if _, ok := kinds[k.name]; ok {
		panic("kind registered twice: " + k.name)
	}
	kinds[k.name] = k
}

// sortedKinds returns the kinds in name order, so test output is stable.
func sortedKinds() []kind {
	out := make([]kind, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (k kind) goldenDir() string { return filepath.Join("testdata", k.name) }

var (
	envOnce   sync.Once
	envErr    error
	testEnv   *envtest.Environment
	k8sClient client.Client
)

func TestMain(m *testing.M) {
	logf.SetLogger(logr.Discard())
	code := m.Run()
	if testEnv != nil {
		if err := testEnv.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "stop envtest: %v\n", err)
		}
	}
	os.Exit(code)
}

// startEnv starts envtest once, with every CRD of the operator. The shared reconciler reads the
// Kubernetes client and scheme from the config package, so they are set here.
func startEnv(t *testing.T) client.Client {
	t.Helper()
	envOnce.Do(func() {
		scheme := runtime.NewScheme()
		for _, add := range []func(*runtime.Scheme) error{
			clientgoscheme.AddToScheme, coralogixv1alpha1.AddToScheme, coralogixv1beta1.AddToScheme,
		} {
			if envErr = add(scheme); envErr != nil {
				return
			}
		}
		testEnv = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join(repoRoot, "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
		}
		cfg, err := testEnv.Start()
		if err != nil {
			envErr = err
			return
		}
		// A direct client, with no cache, so each step reads what the previous step wrote.
		if k8sClient, envErr = client.New(cfg, client.Options{Scheme: scheme}); envErr != nil {
			return
		}
		config.InitScheme(scheme)
		config.InitClient(k8sClient)
	})
	if envErr != nil {
		t.Fatalf("start envtest: %v\nRun `make unit-tests`, or set KUBEBUILDER_ASSETS.", envErr)
	}
	return k8sClient
}

// harness runs one scenario: one CR, the real reconciler, and a fake Coralogix API.
type harness struct {
	t          *testing.T
	client     client.Client
	server     *fakeServer
	reconciler reconcile.Reconciler
	namespace  string
	key        types.NamespacedName
	gvk        schema.GroupVersionKind
	steps      []step
}

// step is one entry of a scenario golden.
type step struct {
	Step string `json:"step"`
	// Spec is the spec that the step applied, if any.
	Spec     any               `json:"spec,omitempty"`
	Requests []recordedRequest `json:"requests"`
	Result   reconcileResult   `json:"result"`
	// Object is the CR after the step. It is null when the CR no longer exists.
	Object *objectState `json:"object"`
}

type reconcileResult struct {
	Error        bool   `json:"error"`
	Requeue      bool   `json:"requeue,omitempty"`
	RequeueAfter string `json:"requeueAfter,omitempty"`
}

type objectState struct {
	Finalizers []string `json:"finalizers"`
	Status     any      `json:"status"`
}

func newHarness(t *testing.T, k kind, scenario string) *harness {
	t.Helper()
	c := startEnv(t)
	server := newFakeServer(t, k.newFake())
	cs := openapicxsdk.NewClientSet(openapicxsdk.NewConfigBuilder().WithURL(server.baseURL).Build())

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName(k.name + "-" + scenario)}}
	if err := c.Create(context.Background(), ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	return &harness{t: t, client: c, server: server, reconciler: k.newReconciler(cs), namespace: ns.Name}
}

var notDNSLabel = regexp.MustCompile(`[^a-z0-9-]+`)

func namespaceName(s string) string {
	s = strings.Trim(notDNSLabel.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(s) > 63 {
		s = strings.TrimRight(s[:63], "-")
	}
	return s
}

// Create creates the CR from a YAML manifest in the scenario namespace, then reconciles once.
func (h *harness) Create(name, manifest string) {
	h.t.Helper()
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(manifest), &obj.Object); err != nil {
		h.t.Fatalf("%s: parse manifest: %v", name, err)
	}
	obj.SetNamespace(h.namespace)
	h.key = client.ObjectKeyFromObject(obj)
	h.gvk = obj.GroupVersionKind()
	if err := h.client.Create(context.Background(), obj); err != nil {
		h.t.Fatalf("%s: create: %v", name, err)
	}
	h.reconcile(name, obj.Object["spec"])
}

// CreateFromFile creates the CR from a manifest file, given relative to the repository root.
func (h *harness) CreateFromFile(name, path string) {
	h.t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, path))
	if err != nil {
		h.t.Fatal(err)
	}
	h.Create(name, string(raw))
}

// SetSpec replaces the spec of the CR, then reconciles once.
func (h *harness) SetSpec(name string, spec map[string]any) {
	h.t.Helper()
	obj := h.get()
	obj.Object["spec"] = spec
	if err := h.client.Update(context.Background(), obj); err != nil {
		h.t.Fatalf("%s: update: %v", name, err)
	}
	h.reconcile(name, spec)
}

// Reconcile reconciles once, as a periodic resync or a retry does.
func (h *harness) Reconcile(name string) {
	h.t.Helper()
	h.reconcile(name, nil)
}

// Delete deletes the CR, then reconciles once.
func (h *harness) Delete(name string) {
	h.t.Helper()
	if err := h.client.Delete(context.Background(), h.get()); err != nil {
		h.t.Fatalf("%s: delete: %v", name, err)
	}
	h.reconcile(name, nil)
}

// StatusID returns status.id of the CR.
func (h *harness) StatusID() string {
	h.t.Helper()
	id, _, err := unstructured.NestedString(h.get().Object, "status", "id")
	if err != nil || id == "" {
		h.t.Fatalf("the CR has no status.id (err: %v)", err)
	}
	return id
}

// API returns the fake API, for a scenario that changes the remote state.
func (h *harness) API() apiFake { return h.server.api }

func (h *harness) get() *unstructured.Unstructured {
	h.t.Helper()
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(h.gvk)
	if err := h.client.Get(context.Background(), h.key, obj); err != nil {
		h.t.Fatalf("get the CR: %v", err)
	}
	return obj
}

func (h *harness) reconcile(name string, spec any) {
	h.t.Helper()
	res, err := h.reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: h.key})
	result := reconcileResult{
		Error: err != nil,
		// Requeue is deprecated, so it is read as "anything other than RequeueAfter".
		Requeue: res != reconcile.Result{RequeueAfter: res.RequeueAfter},
	}
	if res.RequeueAfter != 0 {
		result.RequeueAfter = res.RequeueAfter.String()
	}
	h.steps = append(h.steps, step{
		Step:     name,
		Spec:     spec,
		Requests: h.server.takeRequests(),
		Result:   result,
		Object:   h.observe(),
	})
}

func (h *harness) observe() *objectState {
	h.t.Helper()
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(h.gvk)
	if err := h.client.Get(context.Background(), h.key, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		h.t.Fatalf("get the CR: %v", err)
	}
	finalizers := obj.GetFinalizers()
	if finalizers == nil {
		finalizers = []string{}
	}
	return &objectState{Finalizers: finalizers, Status: normalizeStatus(obj.Object["status"])}
}

// normalizeStatus keeps type, status, reason, and observedGeneration of each condition. The time
// changes on each run, and the message holds error text that may change wording.
func normalizeStatus(status any) any {
	m, ok := status.(map[string]any)
	if !ok {
		return status
	}
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	if conditions, ok := m["conditions"].([]any); ok {
		kept := make([]any, 0, len(conditions))
		for _, c := range conditions {
			cm, _ := c.(map[string]any)
			kept = append(kept, map[string]any{
				"type":               cm["type"],
				"status":             cm["status"],
				"reason":             cm["reason"],
				"observedGeneration": cm["observedGeneration"],
			})
		}
		out["conditions"] = kept
	}
	return out
}

func TestScenarios(t *testing.T) {
	for _, k := range sortedKinds() {
		names := make([]string, 0, len(k.scenarios))
		for name := range k.scenarios {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			t.Run(k.name+"/"+name, func(t *testing.T) {
				h := newHarness(t, k, name)
				k.scenarios[name](h)
				checkGolden(t, k.goldenDir(), "scenario_"+name+".json", maskIDs(marshalGolden(t, h.steps)))
			})
		}
	}
}

var uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// reconcileInterval is the reconcile interval of every recorded reconciler. It is not zero, so
// the goldens show that a successful reconcile asks to run again after the interval.
const reconcileInterval = 10 * time.Minute

// fixedIDPrefix marks the ids that a scenario or case writes itself. They stay as they are.
const fixedIDPrefix = "00000000-0000-4000-8000-"

// maskIDs replaces each id that the API made with <id-N>, in order of first appearance.
func maskIDs(golden []byte) []byte {
	seen := map[string]string{}
	return uuidPattern.ReplaceAllFunc(golden, func(id []byte) []byte {
		if bytes.HasPrefix(id, []byte(fixedIDPrefix)) {
			return id
		}
		if _, ok := seen[string(id)]; !ok {
			seen[string(id)] = fmt.Sprintf("<id-%d>", len(seen)+1)
		}
		return []byte(seen[string(id)])
	})
}

func marshalGolden(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

// checkGolden compares got with the golden file. UPDATE_GOLDEN=1 rewrites the file.
func checkGolden(t *testing.T, dir, name string, got []byte) {
	t.Helper()
	file := filepath.Join(dir, name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%v\nRun with UPDATE_GOLDEN=1 to create it.", err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("%s differs from the golden file.\n%s\nRun with UPDATE_GOLDEN=1 to accept the change.", file, firstDifference(string(want), string(got)))
	}
}

func firstDifference(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return fmt.Sprintf("line %d\n  golden: %s\n  actual: %s", i+1, wl, gl)
		}
	}
	return ""
}

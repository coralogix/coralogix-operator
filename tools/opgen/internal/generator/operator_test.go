package generator

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/coralogix/coralogix-management-sdk/iac/source"
)

// The synthetic kinds, generated into a copy of the operator, compile with the real operator code
// and pass controller-gen (make generate manifests). The CRDs that controller-gen writes are the
// golden CRDs that the envtest test of the operator installs.
//
// The test needs controller-gen (make controller-gen, or CONTROLLER_GEN) and the modules of the
// operator in the module cache. It is skipped in short mode.
func TestGeneratedKindsBuildInTheOperator(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the operator")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	controllerGen := os.Getenv("CONTROLLER_GEN")
	if controllerGen == "" {
		controllerGen = filepath.Join(root, "bin", "controller-gen")
	}
	if _, err := os.Stat(controllerGen); err != nil {
		t.Skipf("controller-gen not found at %s; run make controller-gen in the operator root", controllerGen)
	}
	pinned, err := source.Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	sandbox, sdk := t.TempDir(), t.TempDir()
	for _, path := range []string{"go.mod", "go.sum", "PROJECT", "hack", "api", "internal", "cmd", "tests",
		"config/crd/kustomization.yaml", "charts/coralogix-operator/templates/cluster_role.yaml"} {
		if err := copyPath(filepath.Join(root, path), filepath.Join(sandbox, path)); err != nil {
			t.Fatal(err)
		}
	}
	// The run generates the real kinds of the operator next to the synthetic ones, as make codegen
	// does, so it keeps the generated files of the real kinds.
	for _, dir := range []string{filepath.Join(root, toolDir), filepath.Join("testdata", "root", toolDir)} {
		if err := copyTree(filepath.Join(dir, "kinds"), filepath.Join(sandbox, kindsDir)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
	}
	writeKinds(t, filepath.Join(sandbox, kindsFile), filepath.Join(root, kindsFile), filepath.Join("testdata", "root", kindsFile))
	overlaySDK(t, pinned.SDKDir, sdk)
	goMod := filepath.Join(sandbox, "go.mod")
	appendFile(t, goMod, fmt.Sprintf("\nreplace %s => %s\n", source.SDKModule, sdk))

	openAPI := mergeOpenAPI(t, pinned.OpenAPI, filepath.Join("testdata", "openapi.yaml"))
	input := source.Input{ModuleRoot: sandbox, Module: operatorModule, SDKDir: sdk, SDKModule: source.SDKModule, OpenAPI: openAPI}
	if err := generateFromInput(input, sandbox, sandbox); err != nil {
		t.Fatal(err)
	}
	run(t, sandbox, controllerGen, `object:headerFile=hack/boilerplate.go.txt`, `paths=./...`)
	run(t, sandbox, controllerGen, `rbac:roleName=manager-role`, `crd`, `paths=./...`, `output:crd:artifacts:config=config/crd/bases`)
	run(t, sandbox, "go", "build", "./...")
	run(t, sandbox, "go", "vet", "./api/...", "./internal/controller/...", "./internal/utils/...", "./cmd/...", "./tests/e2e/")

	golden := filepath.Join("testdata", "crd")
	for _, name := range []string{"coralogix.com_things.yaml", "coralogix.com_patchthings.yaml", "coralogix.com_refthings.yaml", "coralogix.com_bodyidthings.yaml"} {
		got, err := os.ReadFile(filepath.Join(sandbox, "config", "crd", "bases", name))
		if err != nil {
			t.Fatal(err)
		}
		if *updateGolden {
			if err := os.MkdirAll(golden, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(golden, name), got, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(filepath.Join(golden, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("golden CRD %s differs; run go test ./internal/generator -run TestGeneratedKindsBuildInTheOperator -update:\n%s", name, firstDifference(string(want), string(got)))
		}
	}
	role, err := os.ReadFile(filepath.Join(sandbox, "config", "rbac", "role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range []string{"things", "things/finalizers", "things/status", "patchthings"} {
		if !bytes.Contains(role, []byte("- "+resource+"\n")) {
			t.Errorf("the role that controller-gen writes has no %s", resource)
		}
	}
}

// writeKinds writes the kinds of every listed kinds file to path.
func writeKinds(t *testing.T, path string, files ...string) {
	t.Helper()
	var all kindList
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var list kindList
		if err := yaml.Unmarshal(data, &list); err != nil {
			t.Fatal(err)
		}
		all.Kinds = append(all.Kinds, list.Kinds...)
	}
	data, err := yaml.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// mergeOpenAPI adds the paths and the component schemas of the synthetic contract to the OpenAPI
// of the pinned SDK. It keeps the order of every key, as the order of oneOf arms is output.
func mergeOpenAPI(t *testing.T, pinned []byte, synthetic string) []byte {
	t.Helper()
	extra, err := os.ReadFile(synthetic)
	if err != nil {
		t.Fatal(err)
	}
	var base, add yaml.Node
	if err := yaml.Unmarshal(pinned, &base); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(extra, &add); err != nil {
		t.Fatal(err)
	}
	for _, path := range [][]string{{"paths"}, {"components", "schemas"}} {
		into, from := mappingAt(t, base.Content[0], path), mappingAt(t, add.Content[0], path)
		into.Content = append(into.Content, from.Content...)
	}
	data, err := yaml.Marshal(&base)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// mappingAt returns the mapping node at the key path.
func mappingAt(t *testing.T, n *yaml.Node, path []string) *yaml.Node {
	t.Helper()
	for _, key := range path {
		found := false
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				n, found = n.Content[i+1], true
				break
			}
		}
		if !found || n.Kind != yaml.MappingNode {
			t.Fatalf("the OpenAPI has no mapping %v", path)
		}
	}
	return n
}

// overlaySDK builds a copy of the pinned SDK at dir from symbolic links, with the stand-in packages of
// the synthetic contracts and their client set accessors added.
func overlaySDK(t *testing.T, pinned, dir string) {
	t.Helper()
	linkExcept(t, pinned, dir, "go")
	linkExcept(t, filepath.Join(pinned, "go"), filepath.Join(dir, "go"), "openapi")
	openapi := filepath.Join("go", "openapi")
	linkExcept(t, filepath.Join(pinned, openapi), filepath.Join(dir, openapi), "gen", "cxsdk")
	linkExcept(t, filepath.Join(pinned, openapi, "gen"), filepath.Join(dir, openapi, "gen"))
	linkExcept(t, filepath.Join(pinned, openapi, "cxsdk"), filepath.Join(dir, openapi, "cxsdk"))
	for _, pkg := range []string{"things_service", "patch_things_service", "ref_things_service", "body_id_things_service"} {
		if err := copyTree(filepath.Join("testdata", "sdk", openapi, "gen", pkg), filepath.Join(dir, openapi, "gen", pkg)); err != nil {
			t.Fatal(err)
		}
	}
	accessors := `package cxsdk

import (
	bodyidthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/body_id_things_service"
	patchthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/patch_things_service"
	refthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ref_things_service"
	things "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/things_service"
)

func (c *ClientSet) Things() *things.ThingsServiceAPIService { return things.NewThingsServiceAPIService() }

func (c *ClientSet) PatchThings() *patchthings.PatchThingsServiceAPIService {
	return patchthings.NewPatchThingsServiceAPIService()
}

func (c *ClientSet) RefThings() *refthings.RefThingsServiceAPIService {
	return refthings.NewRefThingsServiceAPIService()
}

func (c *ClientSet) BodyIDThings() *bodyidthings.BodyIDThingsServiceAPIService {
	return bodyidthings.NewBodyIDThingsServiceAPIService()
}
`
	if err := os.WriteFile(filepath.Join(dir, openapi, "cxsdk", "opgen_synthetic.go"), []byte(accessors), 0o644); err != nil {
		t.Fatal(err)
	}
}

// linkExcept creates dir with a symbolic link to each entry of pinned, except the named ones.
func linkExcept(t *testing.T, pinned, dir string, except ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(pinned)
	if err != nil {
		t.Fatal(err)
	}
	skip := map[string]bool{}
	for _, name := range except {
		skip[name] = true
	}
	for _, e := range entries {
		if skip[e.Name()] {
			continue
		}
		if err := os.Symlink(filepath.Join(pinned, e.Name()), filepath.Join(dir, e.Name())); err != nil {
			t.Fatal(err)
		}
	}
}

func copyPath(from, to string) error {
	info, err := os.Stat(from)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyTree(from, to)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}

func appendFile(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

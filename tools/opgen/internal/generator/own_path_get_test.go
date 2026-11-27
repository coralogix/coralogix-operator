package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// The synthetic OwnPathThing has its Get on /own-path-things/id/{thing_id}, while Update and Delete
// are on /own-path-things/{thing_id}.
const ownPathOverrides = `resource: OwnPathThing
mode: existing
validators:
  inferred: true
api:
  operations:
    create: OwnPathThingsService_CreateOwnPathThing
    get: OwnPathThingsService_GetOwnPathThing
    update: OwnPathThingsService_UpdateOwnPathThing
    delete: OwnPathThingsService_DeleteOwnPathThing
  idProperty: thingId
  serverSetID: true
`

func checkOwnPathThing(t *testing.T, file string) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "behavior-overrides.yaml")
	if err := os.WriteFile(path, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	return Check(CheckOptions{Resource: "OwnPathThing", OpenAPIPath: filepath.Join("testdata", "ownpathget.yaml"), OverridesPath: path})
}

// Without the key, the shared module wants Get on the path of Update. With it, opgen gives the
// module a contract where Get is on that path.
func TestGetAtOwnPathMovesGetToTheItemPath(t *testing.T) {
	err := checkOwnPathThing(t, ownPathOverrides)
	if err == nil || !strings.Contains(err.Error(), "RESOURCE_LIFECYCLE_INCOMPLETE") {
		t.Fatalf("without the key, want RESOURCE_LIFECYCLE_INCOMPLETE, got %v", err)
	}
	if err := checkOwnPathThing(t, ownPathOverrides+"  getAtOwnPath: true\n"); err != nil {
		t.Fatalf("with the key: %v", err)
	}

	file, err := overrides.Parse([]byte(ownPathOverrides + "  getAtOwnPath: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := os.ReadFile(filepath.Join("testdata", "ownpathget.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := reshape(contract, "OwnPathThing", file)
	if err != nil {
		t.Fatal(err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if mappingPath(&doc, "paths", "/own-path-things/id/{thing_id}") != nil {
		t.Error("the path of Get has no other operation, so it must be gone")
	}
	item := mappingPath(&doc, "paths", "/own-path-things/{thing_id}")
	for _, method := range []string{"get", "put", "delete"} {
		if mapValue(item, method) == nil {
			t.Errorf("the item path lacks %s:\n%s", method, out)
		}
	}
}

// Another operation on the path of Get keeps the path.
func TestGetAtOwnPathKeepsAPathThatHasAnotherOperation(t *testing.T) {
	file, err := overrides.Parse([]byte(ownPathOverrides + "  getAtOwnPath: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := os.ReadFile(filepath.Join("testdata", "ownpathget.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	extra := strings.Replace(string(contract), "  /own-path-things/{thing_id}:\n", "  /own-path-things/id/{thing_id}/touch:\n    post: {operationId: Touch, responses: {'200': {description: ok}}}\n  /own-path-things/{thing_id}:\n", 1)
	out, _, err := reshape([]byte(extra), "OwnPathThing", file)
	if err != nil {
		t.Fatal(err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if mappingPath(&doc, "paths", "/own-path-things/id/{thing_id}/touch") == nil {
		t.Error("an unrelated path must stay")
	}
}

func TestGetAtOwnPathIsRejectedWhereItDoesNotApply(t *testing.T) {
	for name, test := range map[string]struct{ file, want string }{
		"no operations":         {"resource: T\nmode: existing\nvalidators:\n  inferred: true\napi:\n  getAtOwnPath: true\n", "needs api.operations"},
		"id in the body":        {"resource: T\nmode: existing\nvalidators:\n  inferred: true\napi:\n  operations: {create: C, get: G, update: U, delete: D}\n  updateIDInBody: true\n  getAtOwnPath: true\n", "cannot go with"},
		"a new kind":            {"resource: T\nmode: new\napi:\n  getAtOwnPath: true\n", "mode new sets no api key"},
		"batch get":             {"resource: T\nmode: existing\nvalidators:\n  inferred: true\napi:\n  operations: {create: C, get: G, update: U, delete: D}\n  getByIDs: ids\n  getAtOwnPath: true\n", "cannot go with"},
		"an upsert":             {"resource: T\nmode: existing\nvalidators:\n  inferred: true\napi:\n  operations: {create: C, get: G, update: C, delete: none}\n  getAtOwnPath: true\n", "cannot go with"},
		"a key that is not set": {"resource: T\nmode: existing\nvalidators:\n  inferred: true\napi:\n  operations: {create: C, get: G, update: U, delete: D}\n  getAtOwnPath: false\n", ""},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := overrides.Parse([]byte(test.file))
			switch {
			case test.want == "" && err != nil:
				t.Fatalf("a written false must parse, got %v", err)
			case test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)):
				t.Fatalf("want an error with %q, got %v", test.want, err)
			}
		})
	}
}

// The golden OwnPathThing has its Get on /own-path-things/id/{thing_id}. The generated code calls
// each operation by its SDK name, with the id as an argument, so the path of Get never shows.
func TestGetAtOwnPathGeneratesCallsByName(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	controller := readGenerated(t, root, controllerDir+"/ownpaththing_controller.go")
	for _, want := range []string{
		"r.Client.OwnPathThingsServiceCreateOwnPathThing(ctx)",
		"r.Client.OwnPathThingsServiceUpdateOwnPathThing(ctx, id)",
		"r.Client.OwnPathThingsServiceDeleteOwnPathThing(ctx, id)",
	} {
		if !strings.Contains(controller, want) {
			t.Errorf("the controller lacks %q:\n%s", want, controller)
		}
	}
	if strings.Contains(controller, "GetOwnPathThing") {
		t.Error("the controller must not call Get: a resync is one Update")
	}
	e2e := readGenerated(t, root, e2eDir+"/ownpaththing_generated_test.go")
	if !strings.Contains(e2e, "apiClient.OwnPathThingsServiceGetOwnPathThing(ctx, ") {
		t.Errorf("the e2e test reads the resource with Get and the id:\n%s", e2e)
	}
	if strings.Contains(controller+e2e, "/id/") {
		t.Error("a generated file names the path of Get")
	}
}

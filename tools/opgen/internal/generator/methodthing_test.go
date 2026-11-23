package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/source"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

const methodThingOverrides = `resource: MethodThing
mode: existing
validators: {inferred: true}
api:
  operations:
    create: MethodThingsService_CreateMethodThing
    get: MethodThingsService_GetMethodThing
    update: MethodThingsService_UpdateMethodThing
    delete: MethodThingsService_DeleteMethodThing
  methods: {create: PUT, update: POST}
  idProperty: thingId
  createResponseOnlyIDProperty: id
updateMask: omit
unwrap: [MethodThingPermissions]
types:
  MethodThing:
    fields:
      parentId: {skip: true}
      parentName: {required: true}
      name: {updateAs: newName, required: true}
      description: {updateAs: newDescription, required: true}
      permissions: {updateAs: newPermissions, required: true}
`

func TestMethodAndUpdateNameGapsRender(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "methodthing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	file, err := overrides.Parse([]byte(methodThingOverrides))
	if err != nil {
		t.Fatal(err)
	}
	entry := kindEntry{Kind: "MethodThing", Resource: "MethodThing"}
	doc, err := (&documents{raw: contract}).forKind(entry, file)
	if err != nil {
		t.Fatal(err)
	}
	v, err := validate(doc, entry, source.SDKModule, operatorModule, &e2eFile{}, file)
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := renderOne(v, operatorModule)
	if err != nil {
		t.Fatal(err)
	}
	controller := string(files["internal/controller/coralogix/v1alpha1/methodthing_controller.go"])
	for _, want := range []string{"MethodThingsServiceCreateMethodThing(ctx)", "MethodThingsServiceUpdateMethodThing(ctx, backendID)", "id := strconv.FormatInt(int64(idValue), 10)"} {
		if !strings.Contains(controller, want) {
			t.Errorf("controller misses %q", want)
		}
	}
	if strings.Contains(controller, "UpdateMask(") {
		t.Error("POST Update must not send an update mask")
	}
	conversion := string(files["api/coralogix/v1alpha1/methodthing_convert.go"])
	for _, want := range []string{"out.NewName", "out.NewDescription", "out.NewPermissions"} {
		if !strings.Contains(conversion, want) {
			t.Errorf("conversion misses %q", want)
		}
	}
}

func TestMethodAndUpdateNameKeysAreChecked(t *testing.T) {
	for _, tc := range []struct{ name, override, code string }{
		{"method policy is required", strings.ReplaceAll(strings.Replace(methodThingOverrides, "  methods: {create: PUT, update: POST}\n", "", 1), "updateMask: omit\n", ""), "OPERATION_METHOD_INCOMPATIBLE"},
		{"an update name must exist", strings.Replace(methodThingOverrides, "updateAs: newName", "updateAs: missingName", 1), "REQUEST_NAME_NOT_FOUND"},
		{"POST Update needs an explicit omitted mask", strings.Replace(methodThingOverrides, "updateMask: omit\n", "", 1), codeOverrideInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "behavior-overrides.yaml")
			if err := os.WriteFile(path, []byte(tc.override), 0o644); err != nil {
				t.Fatal(err)
			}
			wantCode(t, Check(CheckOptions{Resource: "MethodThing", OpenAPIPath: filepath.Join("testdata", "methodthing.yaml"), OverridesPath: path}), tc.code)
		})
	}
}

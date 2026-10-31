package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/source"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

const wrappedRequestValues = "  requestValues:\n    create:\n      label: {source: spec.name}\n      requestId: {source: uuid, value: 'cx-operator-create-'}\n    update:\n      label: {source: spec.name}\n      requestId: {source: uuid, value: 'cx-operator-replace-'}\n"

func TestWrappedRequestsAndResponses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "behavior-overrides.yaml")
	file := "resource: WrappedThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  requestWrapper: item\n  serverSetID: true\n  idProperty: id\n  updateIDInBody: true\n" + wrappedRequestValues + "types:\n  WrappedThing:\n    fields:\n      id:\n        readOnly: true\n"
	if err := os.WriteFile(path, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Check(CheckOptions{Resource: "WrappedThing", OpenAPIPath: filepath.Join("testdata", "wrapped.yaml"), OverridesPath: path})
	if err != nil {
		t.Fatal(err)
	}
	contract, err := os.ReadFile(filepath.Join("testdata", "wrapped.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := model.Load(contract)
	if err != nil {
		t.Fatal(err)
	}
	override, err := overrides.Parse([]byte(file))
	if err != nil {
		t.Fatal(err)
	}
	v, err := validate(doc, kindEntry{Kind: "WrappedThing", Resource: "WrappedThing"}, source.SDKModule, operatorModule, &e2eFile{}, override)
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := renderOne(v, operatorModule)
	if err != nil {
		t.Fatal(err)
	}
	controller := string(files["internal/controller/coralogix/v1alpha1/wrappedthing_controller.go"])
	conversion := string(files["api/coralogix/v1alpha1/wrappedthing_convert.go"])
	for _, want := range []string{"resp.Item.Id", "if resp == nil", "if resp.Item == nil"} {
		if !strings.Contains(controller, want) {
			t.Errorf("controller misses %q", want)
		}
	}
	for _, want := range []string{"out.Item = resource", "out.Id = ptr.To(ptr.Deref(in.Status.ID, \"\"))", "out.Label = ptr.To(in.Spec.Name)", "gouuid.NewString()", "toWrappedThingCreate", "toWrappedThingUpdate"} {
		if !strings.Contains(conversion, want) {
			t.Errorf("conversion misses %q", want)
		}
	}
}

func TestSupportedAPIKeysParse(t *testing.T) {
	for _, key := range []string{
		"operations: {create: C, get: G, update: U, delete: D}",
		"customMethods: true", "serverSetID: true", "idProperty: uniqueIdentifier",
		"clientSetID: true", "delete: {operation: ArchiveThing}",
		"updateMaskInBody: true", "requestWrapper: thing",
	} {
		t.Run(strings.Split(key, ":")[0], func(t *testing.T) {
			_, err := overrides.Parse([]byte("resource: Thing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  " + key + "\n"))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequestValueSourcesAreChecked(t *testing.T) {
	base := "resource: Thing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  requestWrapper: thing\n"
	for _, test := range []struct {
		name, value string
		valid       bool
	}{
		{"spec field", "requestValues: {create: {label: {source: spec.name}}}", true},
		{"uuid", "requestValues: {create: {requestId: {source: uuid, value: prefix-}}}", true},
		{"literal", "requestValues: {create: {mode: {source: literal, value: active}}}", true},
		{"unknown source", "requestValues: {create: {label: {source: arbitrary}}}", false},
		{"unknown operation", "requestValues: {delete: {label: {source: spec.name}}}", false},
		{"empty literal", "requestValues: {create: {mode: {source: literal}}}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := overrides.Parse([]byte(base + "  " + test.value + "\n"))
			if (err == nil) != test.valid {
				t.Fatalf("err = %v, want valid %t", err, test.valid)
			}
		})
	}
}

func TestClientSetIDUsesSpecID(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "wrapped.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	file := "resource: WrappedThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  requestWrapper: item\n  clientSetID: true\n  idProperty: id\n  updateIDInBody: true\n" + wrappedRequestValues
	files := checkLifecycleFixture(t, "WrappedThing", string(contract), file)
	types := string(files["api/coralogix/v1alpha1/wrappedthing_types.go"])
	conversion := string(files["api/coralogix/v1alpha1/wrappedthing_convert.go"])
	if !strings.Contains(types, "Id *string") || !strings.Contains(conversion, "out.Id = in.Id") {
		t.Fatal("client-set id is missing from the spec or Create request")
	}
}

func TestUnmappedRequestSiblingIsRejected(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "wrapped.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "behavior-overrides.yaml")
	file := "resource: WrappedThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  requestWrapper: item\n  clientSetID: true\n  idProperty: id\n  updateIDInBody: true\n"
	if err := os.WriteFile(path, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	openAPIPath := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(openAPIPath, contract, 0o644); err != nil {
		t.Fatal(err)
	}
	wantCode(t, Check(CheckOptions{Resource: "WrappedThing", OpenAPIPath: openAPIPath, OverridesPath: path}), codeShapeUnsupported)
}

func TestRequestValueNeedsSiblingField(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "wrapped.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	file := "resource: WrappedThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  requestWrapper: item\n  clientSetID: true\n  idProperty: id\n  updateIDInBody: true\n" + wrappedRequestValues
	file = strings.Replace(file, "label: {source: spec.name}", "missing: {source: spec.name}", 1)
	path := filepath.Join(t.TempDir(), "behavior-overrides.yaml")
	if err := os.WriteFile(path, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	openAPIPath := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(openAPIPath, contract, 0o644); err != nil {
		t.Fatal(err)
	}
	wantCode(t, Check(CheckOptions{Resource: "WrappedThing", OpenAPIPath: openAPIPath, OverridesPath: path}), codeOverrideUnused)
}

func TestCustomMethodsUseDifferentGetAndDeletePath(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := replaceOnce(t, string(contract), "/body-id-things/{id}:", "/body-id-things/id/{id}:")
	file := "resource: BodyIDThing\nmode: existing\nvalidators:\n  inferred: false\napi:\n  updateIDInBody: true\n  customMethods: true\n"
	checkLifecycleFixture(t, "BodyIDThing", text, file)
}

func TestOperationsOverrideSelectsNonstandardMethods(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(contract)
	for old, replacement := range map[string]string{
		"ThingsService_CreateThing":  "ThingsService_AddThing",
		"ThingsService_GetThing":     "ThingsService_FindThing",
		"ThingsService_ReplaceThing": "ThingsService_PutThing",
		"ThingsService_DeleteThing":  "ThingsService_RemoveThing",
	} {
		text = replaceOnce(t, text, old, replacement)
	}
	file := "resource: Thing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  operations: {create: ThingsService_AddThing, get: ThingsService_FindThing, update: ThingsService_PutThing, delete: ThingsService_RemoveThing}\n"
	files := checkLifecycleFixture(t, "Thing", text, file)
	controller := string(files["internal/controller/coralogix/v1alpha1/thing_controller.go"])
	if !strings.Contains(controller, "ThingsServiceAddThing") || !strings.Contains(controller, "ThingsServiceRemoveThing") {
		t.Fatal("generated controller does not call the selected methods")
	}
}

func TestDeleteOperationSelectsPost(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(contract)
	old := "    delete:\n      tags: [Things Service]\n      operationId: ThingsService_DeleteThing\n      responses:"
	replacement := "  /things/{id}/archive:\n    parameters:\n      - {name: id, in: path, required: true, schema: {type: string}}\n    post:\n      tags: [Things Service]\n      operationId: ThingsService_ArchiveThing\n      responses:"
	text = replaceOnce(t, text, old, replacement)
	file := "resource: Thing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  delete: {operation: ThingsService_ArchiveThing}\n"
	files := checkLifecycleFixture(t, "Thing", text, file)
	controller := string(files["internal/controller/coralogix/v1alpha1/thing_controller.go"])
	if !strings.Contains(controller, "ThingsServiceArchiveThing") {
		t.Fatal("generated controller does not call the POST delete method")
	}
}

func TestUpdateMaskInBody(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(contract)
	old := "      parameters:\n        - name: updateMask\n          in: query\n          required: true\n          schema:\n            type: string\n            pattern: '^[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*(,[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*)*$'\n"
	text = replaceOnce(t, text, old, "")
	text = replaceOnce(t, text, "title: UpdatePatchThingRequest\n              type: object\n              properties:\n", "title: UpdatePatchThingRequest\n              type: object\n              properties:\n                updateMask: {type: string, pattern: '^[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*(,[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*)*$'}\n")
	file := "resource: PatchThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  updateMaskInBody: true\n"
	files := checkLifecycleFixture(t, "PatchThing", text, file)
	conversion := string(files["api/coralogix/v1alpha1/patchthing_convert.go"])
	if !strings.Contains(conversion, "out.UpdateMask = ptr.To(PatchThingUpdateMask)") {
		t.Fatal("generated Update request does not put the mask in its body")
	}
}

func checkLifecycleFixture(t *testing.T, resource, contract, override string) map[string][]byte {
	t.Helper()
	dir := t.TempDir()
	openAPIPath := filepath.Join(dir, "openapi.yaml")
	overridePath := filepath.Join(dir, "behavior-overrides.yaml")
	if err := os.WriteFile(openAPIPath, []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overridePath, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Check(CheckOptions{Resource: resource, OpenAPIPath: openAPIPath, OverridesPath: overridePath}); err != nil {
		t.Fatal(err)
	}
	doc, err := model.Load([]byte(contract))
	if err != nil {
		t.Fatal(err)
	}
	file, err := overrides.Parse([]byte(override))
	if err != nil {
		t.Fatal(err)
	}
	v, err := validate(doc, kindEntry{Kind: resource, Resource: resource}, source.SDKModule, operatorModule, &e2eFile{}, file)
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := renderOne(v, operatorModule)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

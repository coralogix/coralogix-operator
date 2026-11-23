package generator

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/coralogix/coralogix-management-sdk/iac/model"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// The golden SettingThing is a singleton that one POST sets (api.operations with delete: none).
// Create and Update call the POST, Create writes the fixed id, and Delete makes no call. Get nests
// isActive and level in spec, which the CR holds flat (inline on the resource), and the arm that no
// request sends is gone (skip).
func TestSingletonKindSetsOneResource(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	controller := readGenerated(t, root, controllerDir+"/settingthing_controller.go")
	if n := strings.Count(controller, "r.Client.SettingThingsServiceSetSettingThing(ctx).SetSettingThingRequest(*body).Execute()"); n != 2 {
		t.Errorf("Create and Update call the POST without an id; found %d calls:\n%s", n, controller)
	}
	for _, want := range []string{`id := "setting-thing"`, "The API keeps the remote SettingThing: it has no Delete"} {
		if !strings.Contains(controller, want) {
			t.Errorf("the controller lacks %q", want)
		}
	}
	for _, unwanted := range []string{"SettingThingsServiceGetSettingThing", "UpsertPut", "UpsertDelete"} {
		if strings.Contains(controller, unwanted) {
			t.Errorf("the controller must not call %s", unwanted)
		}
	}
	types := readGenerated(t, root, apiDir+"/settingthing_types.go")
	for _, want := range []string{"IsActive *bool", "Level *string", "Item SettingThingItem", "Bucket *string"} {
		if !strings.Contains(types, want) {
			t.Errorf("the spec lacks %q", want)
		}
	}
	for _, unwanted := range []string{"OtherItem", "SettingThingSpec SettingThingSpec", "Spec *"} {
		if strings.Contains(types, unwanted) {
			t.Errorf("the spec must not have %q", unwanted)
		}
	}
	// An optional line on a field that the SDK holds as a value in the requests and the response.
	convert := readGenerated(t, root, apiDir+"/settingthing_convert.go")
	if !strings.Contains(convert, "out.Bucket = (*in.Bucket)") {
		t.Errorf("the conversion sets the SDK value of an optional field:\n%s", convert)
	}
	e2e := readGenerated(t, root, e2eDir+"/settingthing_generated_test.go")
	for _, want := range []string{
		"Ordered, Serial",
		"apiClient.SettingThingsServiceGetSettingThing(ctx).Execute()",
		"waitForSettingThingGoneGenerated(ctx, crClient, fullName)",
		`Equal("setting-thing")`,
		// The update step reads an inlined field through its object.
		"got.Setting.Spec.",
	} {
		if !strings.Contains(e2e, want) {
			t.Errorf("the e2e test of a singleton lacks %q", want)
		}
	}
	if strings.Contains(e2e, "GetSettingThing(ctx, ") {
		t.Error("the Get of a singleton takes no id")
	}
}

func TestSingletonKeysAreChecked(t *testing.T) {
	const head = "resource: S\nmode: existing\nvalidators: {inferred: true}\n"
	const ops = "api:\n  operations: {create: S_Set, get: S_Get, update: S_Set, delete: none}\n"
	for name, tc := range map[string]string{
		"singletonID without delete none":    head + "singletonID: {fixed: s}\napi:\n  operations: {create: C, get: G, update: U, delete: D}\n",
		"delete none without singletonID":    head + ops,
		"delete none with an empty id":       head + "singletonID: {fixed: \"\"}\n" + ops,
		"create and update differ":           head + "singletonID: {fixed: s}\napi:\n  operations: {create: S_Set, get: S_Get, update: S_Put, delete: none}\n",
		"delete none with a request wrapper": head + "singletonID: {fixed: s}\napi:\n  requestWrapper: s\n  operations: {create: S_Set, get: S_Get, update: S_Set, delete: none}\n",
		"delete none in a new kind":          "resource: S\nmode: new\nsingletonID: {fixed: s}\n" + ops,
		"an operation with the added suffix": head + "singletonID: {fixed: s}\napi:\n  operations: {create: S_Set_UpsertPut, get: S_Get, update: S_Set_UpsertPut, delete: none}\n",
	} {
		if _, err := overrides.Parse([]byte(tc)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	file, err := overrides.Parse([]byte(head + "singletonID: {fixed: s}\n" + ops + "types:\n  S:\n    fields:\n      spec: {inline: true}\n"))
	if err != nil {
		t.Fatalf("the singleton keys and inline on the resource must be accepted: %v", err)
	}
	if !file.Upserts() || file.UpsertOperations().Update != "S_Set"+overrides.UpsertPutSuffix || file.UpsertOperations().Delete != "S_Get"+overrides.UpsertDeleteSuffix {
		t.Errorf("the upsert names the added PUT and DELETE: %+v", file.UpsertOperations())
	}
}

const settingOverrides = `resource: SettingThing
mode: existing
validators: {inferred: true}
singletonID: {fixed: s}
api:
  operations: {create: SettingThingsService_SetSettingThing, get: SettingThingsService_GetSettingThing, update: SettingThingsService_SetSettingThing, delete: none}
types:
  SettingThing:
    fields:
`

// reshapeSetting reshapes the synthetic contract, changed by edit, for SettingThing with the field
// lines.
func reshapeSetting(t *testing.T, edit func(string) string, lines string) (*overrides.File, []byte, error) {
	t.Helper()
	file, err := overrides.Parse([]byte(settingOverrides + lines))
	if err != nil {
		t.Fatal(err)
	}
	out, parents, err := reshape([]byte(edit(syntheticContract(t))), "SettingThing", file)
	file.Inlined = parents
	return file, out, err
}

func same(s string) string { return s }

func TestReshapeAddsTheUpsertAndInlines(t *testing.T) {
	file, out, err := reshapeSetting(t, same, "      spec: {inline: true}\n      otherItem: {skip: true}\n")
	if err != nil {
		t.Fatal(err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	item := mappingPath(&doc, "paths", "/setting-things")
	for method, id := range map[string]string{"put": "SettingThingsService_SetSettingThing_UpsertPut", "delete": "SettingThingsService_GetSettingThing_UpsertDelete"} {
		if got := mapValue(mapValue(item, method), "operationId"); got == nil || got.Value != id {
			t.Errorf("the path lacks the %s %s", method, id)
		}
	}
	setting := mappingPath(&doc, "components", "schemas", "SettingThing")
	props := mapValue(setting, "properties")
	for _, name := range []string{"isActive", "level", "item", "otherItem"} {
		if mapValue(props, name) == nil {
			t.Errorf("the resource lacks %s", name)
		}
	}
	if mapValue(props, "spec") != nil || mapValue(setting, "oneOf") != nil || mapValue(setting, "required") != nil {
		t.Errorf("spec, its required line, and the group of the skipped arm must be gone:\n%s", out)
	}
	if want := (overrides.InlineParent{Field: "spec", Schema: "SettingThingSpec"}); file.Inlined["isActive"] != want || file.Inlined["level"] != want {
		t.Errorf("parents = %v", file.Inlined)
	}
	if !slices.Equal(file.PrunedArms, []string{"otherItem"}) {
		t.Errorf("pruned arms = %v", file.PrunedArms)
	}
}

// A skipped arm that a request sends keeps its group: the shared module then decides.
func TestReshapeKeepsAnArmThatARequestSends(t *testing.T) {
	file, out, err := reshapeSetting(t, same, "      item: {skip: true}\n")
	if err != nil {
		t.Fatal(err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if mapValue(mappingPath(&doc, "components", "schemas", "SettingThing"), "oneOf") == nil || len(file.PrunedArms) != 0 {
		t.Errorf("the group must stay; pruned %v", file.PrunedArms)
	}
}

func TestReshapeRejects(t *testing.T) {
	const inline = "      spec: {inline: true}\n"
	for name, tc := range map[string]struct {
		edit  func(string) string
		lines string
		code  string
	}{
		"a DELETE on the path": {func(s string) string {
			return strings.Replace(s, "  /setting-things:\n", "  /setting-things:\n    delete:\n      operationId: SettingThingsService_DeleteSettingThing\n      responses: {'200': {description: Deleted}}\n", 1)
		}, inline, "UPSERT_SHAPE_UNSUPPORTED"},
		"a Set that is a PUT": {func(s string) string {
			return strings.Replace(s, "    post:\n      tags: [Setting Things Service]", "    put:\n      tags: [Setting Things Service]", 1)
		}, inline, "UPSERT_SHAPE_UNSUPPORTED"},
		"a Get that the contract lacks": {func(s string) string {
			return strings.Replace(s, "operationId: SettingThingsService_GetSettingThing", "operationId: SettingThingsService_ReadSettingThing", 1)
		}, inline, "UPSERT_SHAPE_UNSUPPORTED"},
		"an optional inlined object": {func(s string) string {
			return strings.Replace(s, "      required: [spec]\n      oneOf:", "      oneOf:", 1)
		}, inline, "INLINE_UNSUPPORTED"},
		"an inlined object that the request holds": {func(s string) string {
			return strings.Replace(s, "                item: {$ref: '#/components/schemas/SettingThingItem'}\n", "                item: {$ref: '#/components/schemas/SettingThingItem'}\n                spec: {$ref: '#/components/schemas/SettingThingSpec'}\n", 1)
		}, inline, "INLINE_UNSUPPORTED"},
		"an inlined arm":               {same, "      otherItem: {inline: true}\n", "INLINE_UNSUPPORTED"},
		"an inlined field that is not": {same, "      nothing: {inline: true}\n", "INLINE_UNSUPPORTED"},
		"a PUT on the path": {func(s string) string {
			return strings.Replace(s, "  /setting-things:\n", "  /setting-things:\n    put:\n      operationId: SettingThingsService_ReplaceSettingThing\n      responses: {'200': {description: Replaced}}\n", 1)
		}, inline, "UPSERT_SHAPE_UNSUPPORTED"},
	} {
		_, _, err := reshapeSetting(t, tc.edit, tc.lines)
		var shape *shapeError
		if !errors.As(err, &shape) || shape.code != tc.code {
			t.Errorf("%s: want %s, got %v", name, tc.code, err)
		}
	}
}

// A skipped arm needs the requests: without them it cannot be known to be response-only.
func TestPruneNeedsTheRequests(t *testing.T) {
	file, err := overrides.Parse([]byte("resource: SettingThing\nmode: existing\nvalidators: {inferred: true}\napi:\n  operations: {create: Missing_Create, get: SettingThingsService_GetSettingThing, update: Missing_Create, delete: D}\ntypes:\n  SettingThing:\n    fields:\n      otherItem: {skip: true}\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = reshape([]byte(syntheticContract(t)), "SettingThing", file)
	var shape *shapeError
	if !errors.As(err, &shape) || shape.code != "OPERATION_NOT_FOUND" {
		t.Errorf("want OPERATION_NOT_FOUND, got %v", err)
	}
}

// The content key cannot go with a reshape: it cuts the resource down to its id.
func TestContentCannotGoWithAReshape(t *testing.T) {
	file, err := overrides.Parse([]byte(settingOverrides + "      spec: {inline: true}\n"))
	if err != nil {
		t.Fatal(err)
	}
	file.Content = &overrides.Content{Decode: "example.com/x.Unmarshal"}
	_, err = (&documents{raw: []byte(syntheticContract(t))}).forKind(kindEntry{Kind: "SettingThing", Resource: "SettingThing"}, file)
	var shape *shapeError
	if !errors.As(err, &shape) || shape.code != "OPERATOR_SHAPE_UNSUPPORTED" {
		t.Errorf("want OPERATOR_SHAPE_UNSUPPORTED, got %v", err)
	}
}

// Only a singleton that one POST sets has a fixed id. Another singleton is not supported.
func TestSingletonWithoutUpsertIsRejected(t *testing.T) {
	err := checkLifecycle(&model.Resource{Singleton: true}, nil)
	if err == nil || !strings.Contains(err.Error(), "delete: none") {
		t.Errorf("want the singleton message, got %v", err)
	}
}

// The e2e test of a singleton reads the env values of e2e.yaml as the other e2e tests do: it
// requires each variable and expands @{env.NAME} in the spec. The offline coverage test does not.
func TestSingletonE2EReadsTheEnvValues(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	e2e := readGenerated(t, root, e2eDir+"/settingthing_generated_test.go")
	for _, want := range []string{`"os"`, "requireSettingThingEnv()", `spec = expandSettingThingEnv(spec)`, `[]string{"SETTING_BUCKET"}`} {
		if !strings.Contains(e2e, want) {
			t.Errorf("the e2e test lacks %q", want)
		}
	}
	coverage := readGenerated(t, root, controllerDir+"/settingthing_coverage_test.go")
	if strings.Contains(coverage, "SETTING_BUCKET") && strings.Contains(coverage, "os.Getenv") {
		t.Error("the offline coverage test must not read the environment")
	}
}

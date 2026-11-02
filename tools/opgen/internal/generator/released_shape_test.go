package generator

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

func checkG14(t *testing.T, resource, override string) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "behavior-overrides.yaml")
	if err := os.WriteFile(path, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	options := CheckOptions{Resource: resource, OpenAPIPath: filepath.Join("testdata", "openapi.yaml"), OverridesPath: path}
	if resource == "Thing" {
		options.E2EPath = filepath.Join("testdata", "root", "tools", "opgen", "kinds", "thing", "e2e.yaml")
	}
	return Check(options)
}

func TestReleasedShapeGroupsAndRequireOne(t *testing.T) {
	groups := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ngroups:\n  routing: {fields: [config]}\n"
	if err := checkG14(t, "Thing", groups); err != nil {
		t.Fatalf("full-replace group: %v", err)
	}
	nested := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  ThingConfig:\n    groups:\n      choice: {fields: [http, queue]}\n"
	if err := checkG14(t, "Thing", nested); err != nil {
		t.Fatalf("nested oneOf group: %v", err)
	}
	patch := strings.Replace(groups, "Thing", "PatchThing", 1)
	wantCode(t, checkG14(t, "PatchThing", patch), codeShapeUnsupported)
	requireOne := "resource: PatchThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  PatchThing:\n    fields:\n      target: {requireOne: true}\n"
	if err := checkG14(t, "PatchThing", requireOne); err != nil {
		t.Fatalf("optional oneOf becomes required: %v", err)
	}
	wantCode(t, checkG14(t, "PatchThing", strings.Replace(requireOne, "target:", "name:", 1)), codeOverrideInvalid)
	badUnwrap := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  Thing:\n    fields:\n      name: {unwrap: true}\n"
	wantCode(t, checkG14(t, "Thing", badUnwrap), "UNWRAP_FIELD_UNSUPPORTED")
}

func TestReleasedShapeGroupMovesTheCRFieldAndConversion(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, kindsDir, "thing", "behavior-overrides.yaml")
	override := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ngroups:\n  routing: {fields: [config]}\n"
	if err := os.WriteFile(path, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct{ file, part string }{
		{filepath.Join(root, apiDir, "thing_types.go"), "Routing ThingSpecRouting `json:\"routing\"`"},
		{filepath.Join(root, apiDir, "thing_convert.go"), "in.Routing.Config"},
	} {
		data, err := os.ReadFile(check.file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), check.part) {
			t.Errorf("%s omits %q", check.file, check.part)
		}
	}
}

func TestReleasedShapeGroupSuppliesE2EUpdate(t *testing.T) {
	for _, tt := range []struct {
		name, update, assignment, assertion string
	}{
		{name: "automatic", assignment: `updated.Spec.Routing.Name = fullName + "-updated"`, assertion: `g.Expect(got.Name).To(Equal(fullName + "-updated"))`},
		{name: "configured", update: "update:\n  field: routing.description\n  value: changed\n", assignment: `updated.Spec.Routing.Description = ptr.To("changed")`, assertion: `g.Expect(ptr.Deref(got.Description, "")).To(Equal("changed"))`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root, input := syntheticRoot(t)
			kindDir := filepath.Join(root, kindsDir, "thing")
			override := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ngroups:\n  routing: {fields: [name, description, enabled, quota]}\n"
			if err := os.WriteFile(filepath.Join(kindDir, "behavior-overrides.yaml"), []byte(override), 0o644); err != nil {
				t.Fatal(err)
			}
			if tt.update != "" {
				if err := os.WriteFile(filepath.Join(kindDir, "e2e.yaml"), []byte("values:\n  destinations: [https://example.com]\n"+tt.update), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := generateFromInput(input, root, root); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(root, "tests", "e2e", "thing_generated_test.go"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parser.ParseFile(token.NewFileSet(), "thing_generated_test.go", data, 0); err != nil {
				t.Fatalf("generated e2e test is invalid Go: %v", err)
			}
			for _, want := range []string{tt.assignment, tt.assertion, "cr.Spec.Routing.Name = name"} {
				if !strings.Contains(string(data), want) {
					t.Errorf("generated e2e test omits %q", want)
				}
			}
		})
	}
}

func TestReleasedShapeGroupRequireOneMovesTheCELRule(t *testing.T) {
	file := &overrides.File{Resource: "Thing", Groups: map[string]overrides.Group{"choice": {Fields: []string{"left", "right", "timeout"}, RequireOne: true}}}
	b := &builder{file: file, types: map[string]*crType{}, usedGroups: map[string]bool{}}
	owner := &crType{Name: "ThingSpec", Fields: []*crField{{API: "left", JSON: "left", GoName: "Left", Optional: true}, {API: "right", JSON: "right", GoName: "Right", Optional: true}, {API: "timeout", JSON: "timeout", GoName: "Timeout", Optional: true}}}
	remaining, err := b.groupFields("Thing", owner, []model.OneOfGroup{{Arms: []string{"left", "right"}, AllowNone: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 || len(owner.Fields) != 1 || !owner.Fields[0].group {
		t.Fatal("the oneOf rule or fields did not move into the group")
	}
	markers := strings.Join(owner.Fields[0].object.Markers, "\n")
	if !strings.Contains(markers, "== 1") || strings.Contains(markers, "timeout") || len(owner.Fields[0].object.Fields) != 3 {
		t.Fatalf("the group lost its shared field or exactly-one rule: %s", markers)
	}
}

func TestReleasedShapeRequestValueBy(t *testing.T) {
	cr := &crType{Name: "Choice", Fields: []*crField{{API: "left", GoName: "Left", Optional: true, typ: &model.Type{Kind: model.Object}}, {API: "right", GoName: "Right", Optional: true, typ: &model.Type{Kind: model.Object}}}, groups: []model.OneOfGroup{{Arms: []string{"left", "right"}, AllowNone: true}}}
	ix := &refIndex{fields: map[string]sdkcheck.Ref{"Choice.Type": {Name: "Type", Want: "*ChoiceType"}}, bySchema: map[string]sdkcheck.Ref{"ChoiceType": {Name: "ChoiceType"}}, consts: map[string]bool{"CHOICETYPE_LEFT": true, "CHOICETYPE_RIGHT": true}}
	b := &builder{file: &overrides.File{Resource: "Choice"}, ix: ix, alias: "sdk"}
	field := &model.Field{Name: "type", Type: &model.Type{Kind: model.Enum, Schema: "ChoiceType", EnumZero: "UNSPECIFIED", Values: []string{"LEFT", "RIGHT"}}, Attrs: model.Attrs{Required: true}}
	code, err := b.requestValueBy(cr, "Choice", field, map[string]string{"left": "LEFT", "right": "RIGHT"}, "spec")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"if in.Left != nil", "else if in.Right != nil", "out.Type =", "exactly one source arm is required"} {
		if !strings.Contains(code, part) {
			t.Errorf("conversion omits %q:\n%s", part, code)
		}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", "package generated\nfunc convert() {\n"+code+"}\n", 0); err != nil {
		t.Fatalf("requestValueBy writes invalid Go: %v\n%s", err, code)
	}
	if _, err := b.requestValueBy(cr, "Choice", field, map[string]string{"left": "MISSING", "right": "RIGHT"}, "spec"); err == nil {
		t.Fatal("an unknown API enum value passed")
	}
	if _, err := b.requestValueBy(cr, "Choice", field, map[string]string{"left": "UNSPECIFIED", "right": "RIGHT"}, "spec"); err == nil {
		t.Fatal("an enum zero value passed as a selected-arm value")
	} else if shape, ok := isShapeError(err); !ok || shape.code != codeOverrideInvalid {
		t.Fatalf("want an invalid override for the enum zero value, got %v", err)
	}
}

func TestReleasedShapeUnwrapRestoresTheRequestWrapper(t *testing.T) {
	value := &model.Type{Kind: model.String, Wrappers: []model.Wrapper{{Schema: "ValueWrapper", Field: "value"}}}
	ix := &refIndex{fields: map[string]sdkcheck.Ref{"ValueWrapper.Value": {Name: "Value", Want: "*string"}}, bySchema: map[string]sdkcheck.Ref{"ValueWrapper": {Name: "ValueWrapper"}}}
	b := &builder{ix: ix, alias: "sdk"}
	field := &crField{API: "query", GoName: "Query", typ: value}
	code, expr, _, err := b.value(field, value, "in.Query", "*ValueWrapper", "spec.query")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "w1 := sdk.ValueWrapper{}") || !strings.Contains(code, "v2 := in.Query\nw1.Value = &v2") || expr != "w1" {
		t.Fatalf("the API wrapper was not restored:\n%s\nexpr: %s", code, expr)
	}
}

func TestReleasedShapeUnwrapSyntheticKind(t *testing.T) {
	root, input := syntheticRoot(t)
	api := string(input.OpenAPI)
	api = replaceOnce(t, api, "description: {type: string, description: An optional description.}", "description: {$ref: '#/components/schemas/ThingDescription'}")
	if strings.Count(api, "description: {type: string, x-coralogix-presence: true}") < 2 {
		t.Fatal("the synthetic Create and Update descriptions changed")
	}
	api = strings.Replace(api, "description: {type: string, x-coralogix-presence: true}", "description: {$ref: '#/components/schemas/ThingDescription'}", 2)
	api = replaceOnce(t, api, "    ThingKind:\n", "    ThingDescription:\n      type: object\n      required: [value]\n      properties:\n        value: {type: string, x-coralogix-presence: true}\n    ThingKind:\n")
	input.OpenAPI = []byte(api)
	sdkDir := filepath.Join(t.TempDir(), "sdk")
	if err := copyTree(filepath.Join("testdata", "sdk"), sdkDir); err != nil {
		t.Fatal(err)
	}
	sdkFile := filepath.Join(sdkDir, "go", "openapi", "gen", "things_service", "things.go")
	sdk, err := os.ReadFile(sdkFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(sdk), "Description  *string") != 3 {
		t.Fatal("the synthetic SDK description fields changed")
	}
	changed := strings.ReplaceAll(string(sdk), "Description  *string", "Description  *ThingDescription")
	changed = replaceOnce(t, changed, "type ThingKind string", "type ThingDescription struct { Value string `json:\"value\"` }\n\ntype ThingKind string")
	if err := os.WriteFile(sdkFile, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	originalSDKDir := input.SDKDir
	input.SDKDir = sdkDir
	modPath := filepath.Join(root, "go.mod")
	mod, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modPath, []byte(strings.Replace(string(mod), originalSDKDir, sdkDir, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, kindsDir, "thing", "behavior-overrides.yaml")
	if err := os.WriteFile(path, []byte("resource: Thing\nmode: existing\nvalidators: {inferred: true}\nunwrap: [ThingDescription]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types, err := os.ReadFile(filepath.Join(root, apiDir, "thing_types.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "Description *string") || strings.Contains(string(types), "type ThingDescription struct") {
		t.Fatal("the CR still contains the API wrapper")
	}
	conversion, err := os.ReadFile(filepath.Join(root, apiDir, "thing_convert.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(conversion), ".ThingDescription{") {
		t.Fatal("Create or Update does not rebuild the wrapper")
	}
}

func TestReleasedShapeRequestValueBySyntheticKind(t *testing.T) {
	root, input := syntheticRoot(t)
	api := string(input.OpenAPI)
	api = replaceOnce(t, api, "    ThingConfig:\n      type: object\n      properties:\n", "    ThingConfig:\n      type: object\n      required: [type]\n      properties:\n        type: {$ref: '#/components/schemas/ThingConfigType'}\n")
	api = replaceOnce(t, api, "    HttpThingConfig:\n", "    ThingConfigType:\n      type: string\n      enum: [THING_CONFIG_TYPE_UNSPECIFIED, THING_CONFIG_TYPE_HTTP, THING_CONFIG_TYPE_QUEUE]\n    HttpThingConfig:\n")
	input.OpenAPI = []byte(api)
	sdkDir := filepath.Join(t.TempDir(), "sdk")
	if err := copyTree(filepath.Join("testdata", "sdk"), sdkDir); err != nil {
		t.Fatal(err)
	}
	sdkFile := filepath.Join(sdkDir, "go", "openapi", "gen", "things_service", "things.go")
	sdk, err := os.ReadFile(sdkFile)
	if err != nil {
		t.Fatal(err)
	}
	changed := replaceOnce(t, string(sdk), "type ThingConfig struct {\n", "type ThingConfigType string\nconst (\nTHINGCONFIGTYPE_THING_CONFIG_TYPE_UNSPECIFIED ThingConfigType = \"THING_CONFIG_TYPE_UNSPECIFIED\"\nTHINGCONFIGTYPE_THING_CONFIG_TYPE_HTTP ThingConfigType = \"THING_CONFIG_TYPE_HTTP\"\nTHINGCONFIGTYPE_THING_CONFIG_TYPE_QUEUE ThingConfigType = \"THING_CONFIG_TYPE_QUEUE\"\n)\n\ntype ThingConfig struct {\n Type ThingConfigType `json:\"type\"`\n")
	if err := os.WriteFile(sdkFile, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	originalSDKDir := input.SDKDir
	input.SDKDir = sdkDir
	modPath := filepath.Join(root, "go.mod")
	mod, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modPath, []byte(strings.Replace(string(mod), originalSDKDir, sdkDir, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, kindsDir, "thing", "behavior-overrides.yaml")
	override := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  ThingConfig:\n    fields:\n      type: {requestValueBy: {http: THING_CONFIG_TYPE_HTTP, queue: THING_CONFIG_TYPE_QUEUE}}\n"
	if err := os.WriteFile(path, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types, err := os.ReadFile(filepath.Join(root, apiDir, "thing_types.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(types), "Type ThingConfigType") {
		t.Fatal("the hidden discriminator is still in the CR")
	}
	conversion, err := os.ReadFile(filepath.Join(root, apiDir, "thing_convert.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(conversion), "if in.Http != nil") || !strings.Contains(string(conversion), "THINGCONFIGTYPE_THING_CONFIG_TYPE_HTTP") {
		t.Fatal("Create or Update does not derive the discriminator")
	}
	zeroOverride := strings.Replace(override, "http: THING_CONFIG_TYPE_HTTP", "http: THING_CONFIG_TYPE_UNSPECIFIED", 1)
	if err := os.WriteFile(path, []byte(zeroOverride), 0o644); err != nil {
		t.Fatal(err)
	}
	wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
}

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

func TestSplitReferenceKeepsTwoReleasedFields(t *testing.T) {
	file, err := overrides.Parse([]byte("resource: RefThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  RefThing:\n    fields:\n      thingId: {name: thing, ref: {kind: Thing, type: NCRef}}\n      parentId: {name: parentFolderId, ref: {kind: Thing, form: split, resourceField: parentFolderRef}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := model.Load([]byte(syntheticContract(t)))
	if err != nil {
		t.Fatal(err)
	}
	e2e := &e2eFile{Values: map[string]any{
		"thing":           map[string]any{"resourceRef": map[string]any{"name": "thing-sample"}},
		"parentFolderRef": map[string]any{"name": "thing-sample"},
	}}
	v, err := validate(contract, kindEntry{Kind: "RefThing", Resource: "RefThing"}, source.SDKModule, operatorModule, e2e, file)
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := renderOne(v, operatorModule)
	if err != nil {
		t.Fatal(err)
	}
	types := string(files["api/coralogix/v1alpha1/refthing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/refthing_convert.go"])
	if !strings.Contains(types, "ParentFolderId *string `json:\"parentFolderId,omitempty\"`") || !strings.Contains(types, "ParentFolderRef *ResourceRef `json:\"parentFolderRef,omitempty\"`") {
		t.Error("the split reference must keep both released CR fields")
	}
	if !strings.Contains(types, "!(has(self.parentFolderId) && has(self.parentFolderRef))") || !strings.Contains(convert, "if in.ParentFolderId != nil") || !strings.Contains(convert, "ResolveResourceID(ctx, namespace, ResourceIDRef{ResourceRef: in.ParentFolderRef}") {
		t.Error("the split reference must choose one source and resolve the CR when selected")
	}
}

func TestNumericAndVersionedReferenceConversion(t *testing.T) {
	b := &builder{}
	f := &crField{ref: "Alert", refIDType: "int32", refVersion: "v1beta1"}
	code, value := b.referenceValue(f, "in.Alert", "spec.alert")
	if !strings.Contains(code, "ResolveNumericResourceID(ctx, namespace") || !strings.Contains(code, "\"Alert\", \"v1beta1\", 32)") || !strings.Contains(value, "int32(") {
		t.Errorf("numeric cross-version reference conversion is incomplete: %s %s", code, value)
	}
	if referenceWant(f) != "int32" {
		t.Error("numeric list element must use the SDK integer type")
	}
}

func TestNamedReferencePassesSecondClientAndLookup(t *testing.T) {
	file, err := overrides.Parse([]byte("resource: RefThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  RefThing:\n    fields:\n      thingId: {name: thing, ref: {kind: Thing, type: NCRef}}\n      parentId: {ref: {kind: Thing, form: named, lookup: LookupThingName}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := model.Load([]byte(syntheticContract(t)))
	if err != nil {
		t.Fatal(err)
	}
	e2e := &e2eFile{Values: map[string]any{
		"thing":     map[string]any{"resourceRef": map[string]any{"name": "thing-sample"}},
		"parentRef": map[string]any{"backendRef": map[string]any{"name": "parent"}},
	}}
	v, err := validate(contract, kindEntry{Kind: "RefThing", Resource: "RefThing"}, source.SDKModule, operatorModule, e2e, file)
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := renderOne(v, operatorModule)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		"api/coralogix/v1alpha1/refthing_types.go":                      "ParentRef *NamedResourceIDRef",
		"api/coralogix/v1alpha1/refthing_convert.go":                    "lookup, LookupThingName",
		"internal/controller/coralogix/v1alpha1/refthing_controller.go": "cr.CreateRequest(ctx, r.ClientSet)",
	} {
		if !strings.Contains(string(files[path]), want) {
			t.Errorf("%s does not contain %q", path, want)
		}
	}
}

func TestNumericAndListReferenceTypes(t *testing.T) {
	file, err := overrides.Parse([]byte("resource: BodyIDThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  BodyIDThing:\n    fields:\n      roleId: {ref: {kind: CustomRole, backendIDType: uint32}}\n      alertIds: {ref: {kind: Alert, version: v1beta1}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{file: file, used: map[overrides.Line]bool{}}
	integer := &crField{API: "roleId", JSON: "roleId", Optional: true, typ: &model.Type{Kind: model.Integer, Format: "int64"}}
	got, err := b.refField(integer, "BodyIDThing", file.Ref("BodyIDThing", "roleId"), "spec.roleId")
	if err != nil || got.GoType != "*UInt32ResourceIDRef" || got.refIDType != "int64" {
		t.Fatalf("numeric reference: %v, %+v", err, got)
	}
	list := &crField{API: "alertIds", JSON: "alertIds", Optional: true, typ: &model.Type{Kind: model.List, Elem: &model.Type{Kind: model.String}}}
	got, err = b.refField(list, "BodyIDThing", file.Ref("BodyIDThing", "alertIds"), "spec.alertIds")
	if err != nil || got.GoType != "[]ResourceIDRef" || got.refVersion != "v1beta1" {
		t.Fatalf("versioned reference list: %v, %+v", err, got)
	}
}

func TestNamedNumericReferenceUsesNumericCRField(t *testing.T) {
	file, err := overrides.Parse([]byte("resource: BodyIDThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  BodyIDThing:\n    fields:\n      webhookId: {ref: {kind: OutboundWebhook, form: named, lookup: LookupWebhookName}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{file: file, used: map[overrides.Line]bool{}}
	f := &crField{API: "webhookId", JSON: "webhookId", Optional: true, typ: &model.Type{Kind: model.Integer, Format: "int64"}}
	got, err := b.refField(f, "BodyIDThing", file.Ref("BodyIDThing", "webhookId"), "spec.webhookId")
	if err != nil || got.GoType != "*NamedInt64ResourceIDRef" || !b.hasNameLookup {
		t.Fatalf("named numeric reference: %v, %+v", err, got)
	}
	code, _ := b.referenceValue(got, "(*in.WebhookRef)", "spec.webhookRef")
	if !strings.Contains(code, "ResolveNamedNumericResourceID(") || !strings.Contains(code, "LookupWebhookName, 64)") {
		t.Errorf("the numeric name must resolve through the second client: %s", code)
	}
}

func TestReleasedNamedBackendOnlyReferenceUsesItsGoType(t *testing.T) {
	file, err := overrides.Parse([]byte("resource: BodyIDThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  BodyIDThing:\n    fields:\n      retentionId: {ref: {kind: ArchiveRetention, form: namedBackendOnly, lookup: LookupArchiveRetentionName, type: ArchiveRetention, idField: Id}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{file: file, used: map[overrides.Line]bool{}}
	f := &crField{API: "retentionId", JSON: "retentionId", Optional: true, typ: &model.Type{Kind: model.String}}
	got, err := b.refField(f, "BodyIDThing", file.Ref("BodyIDThing", "retentionId"), "spec.retentionId")
	if err != nil || got.GoType != "*ArchiveRetention" {
		t.Fatalf("released named reference: %v, %+v", err, got)
	}
	code, _ := b.referenceValue(got, "(*in.RetentionRef)", "spec.retentionRef")
	if !strings.Contains(code, "NamedBackendRef{ID: (*in.RetentionRef).BackendRef.Id, Name: (*in.RetentionRef).BackendRef.Name}") {
		t.Errorf("released backend type was not converted: %s", code)
	}
}

func syntheticContract(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReleasedFieldPresenceAndName(t *testing.T) {
	file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\ntypes:\n  BodyIDThing:\n    fields:\n      name: {name: title}\n      limit: {required: true}\n"
	files := checkLifecycleFixture(t, "BodyIDThing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/bodyidthing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/bodyidthing_convert.go"])
	for _, want := range []string{"Title string `json:\"title\"`", "Limit int32 `json:\"limit\"`"} {
		if !strings.Contains(types, want) {
			t.Errorf("types lack %q", want)
		}
	}
	if strings.Count(convert, "in.Title") != 2 || strings.Count(convert, "out.Name =") != 2 || strings.Count(convert, "out.Limit =") != 2 {
		t.Error("both request conversions must use the API properties")
	}
	if !strings.Contains(string(files["config/samples/v1alpha1/generated/bodyidthing.yaml"]), "title:") {
		t.Error("sample must use the CR property")
	}
}

func TestNestedReleasedFieldPresenceAndName(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  ThingItem:\n    fields:\n      name: {name: label}\n      weight: {required: true}\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/thing_convert.go"])
	if !strings.Contains(types, "Label string `json:\"label\"`") || !strings.Contains(types, "Weight int32 `json:\"weight\"`") {
		t.Error("nested CR fields must keep their released names and presence")
	}
	if !strings.Contains(convert, "in.Label") || !strings.Contains(convert, "out.Name =") || !strings.Contains(convert, "out.Weight =") {
		t.Error("nested request conversions must use API properties")
	}
}

func TestReleasedRequiredScalarWithoutPresence(t *testing.T) {
	contract := syntheticContract(t)
	start := strings.Index(contract, "  /body-id-things:\n")
	if start < 0 {
		t.Fatal("Body ID Thing operations not found")
	}
	body := contract[start:]
	body = replaceOnce(t, body, "required: [name]\n              properties:", "required: [name, limit]\n              properties:")
	body = strings.ReplaceAll(body,
		"limit: {type: integer, format: int32, minimum: 1, maximum: 10, x-coralogix-presence: true}",
		"limit: {type: integer, format: int32, minimum: 0, maximum: 10}")
	contract = contract[:start] + body

	dir := t.TempDir()
	openAPI := filepath.Join(dir, "openapi.yaml")
	override := filepath.Join(dir, "behavior-overrides.yaml")
	if err := os.WriteFile(openAPI, []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\n"
	if err := os.WriteFile(override, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	options := CheckOptions{Resource: "BodyIDThing", OpenAPIPath: openAPI, OverridesPath: override}
	wantCode(t, Check(options), "REQUIRED_SCALAR_PRESENCE_UNKNOWN")

	file += "types:\n  BodyIDThing:\n    fields:\n      limit: {required: true}\n"
	if err := os.WriteFile(override, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Check(options); err != nil {
		t.Fatalf("released required field: %v", err)
	}
}

func TestReleasedFieldValidationOverride(t *testing.T) {
	file := "resource: BodyIDThing\nmode: existing\napi: {updateIDInBody: true}\nvalidators:\n  inferred: true\n  fields:\n    spec.name: {inferred: false}\n    spec.limit:\n      remove: ['+kubebuilder:validation:Maximum=10']\n      add: ['+kubebuilder:validation:Maximum=9']\n"
	files := checkLifecycleFixture(t, "BodyIDThing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/bodyidthing_types.go"])
	if strings.Contains(types, "+kubebuilder:validation:MaxLength=100") || strings.Contains(types, "+kubebuilder:validation:MinLength=1") {
		t.Error("the released name must keep its absent length limits")
	}
	if !strings.Contains(types, "+kubebuilder:validation:Minimum=1") || !strings.Contains(types, "+kubebuilder:validation:Maximum=9") || strings.Contains(types, "+kubebuilder:validation:Maximum=10") {
		t.Error("the limit override must keep the API minimum and replace only the maximum")
	}
}

func TestReleasedCELAndDefaultOverrides(t *testing.T) {
	base := checkLifecycleFixture(t, "PatchThing", syntheticContract(t), "resource: PatchThing\nmode: existing\nvalidators: {inferred: true}\n")
	var oldRule string
	for _, line := range strings.Split(string(base["api/coralogix/v1alpha1/patchthing_types.go"]), "\n") {
		if strings.Contains(line, "At most one of some, prefix") {
			oldRule = strings.TrimPrefix(line, "// ")
			break
		}
	}
	if oldRule == "" {
		t.Fatal("the API oneOf has no generated CEL rule")
	}
	newRule := celMarker("!(has(self.some) && has(self.prefix))", "Only one target can be set")
	file := "resource: PatchThing\nmode: existing\nvalidators:\n  inferred: true\n  fields:\n    spec.target#type:\n      remove: ['" + oldRule + "']\n      add: ['" + newRule + "']\n    spec.description:\n      add: ['+kubebuilder:default=`memo`']\n"
	files := checkLifecycleFixture(t, "PatchThing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/patchthing_types.go"])
	for _, marker := range []string{newRule, "+kubebuilder:default=`memo`"} {
		if !strings.Contains(types, marker) {
			t.Errorf("released marker %q is missing", marker)
		}
	}
	if strings.Contains(types, oldRule) {
		t.Error("the replaced API CEL rule is still present")
	}
}

func TestDateTimeUsesMetav1TimeAndSDKTime(t *testing.T) {
	typ := &model.Type{Kind: model.String, Format: "date-time"}
	goType, err := scalarType(typ, "spec.fromTime")
	if err != nil || goType != "metav1.Time" {
		t.Fatalf("date-time CR type: %q, %v", goType, err)
	}
	b := &builder{}
	statements, value, pointer, err := b.value(&crField{typ: typ}, typ, "in.FromTime", "time.Time", "spec.fromTime")
	if err != nil || statements != "" || value != "in.FromTime.Time" || pointer {
		t.Fatalf("date-time SDK conversion: %q, %q, %v, %v", statements, value, pointer, err)
	}
	if got := limitMarkers(typ); len(got) != 0 {
		t.Fatalf("metav1.Time cannot carry string validation markers: %v", got)
	}
}

func TestEmptyCollectionAndObjectContractDefaults(t *testing.T) {
	for _, tc := range []struct {
		kind  model.Kind
		value string
	}{
		{model.List, "[]"},
		{model.Map, "{}"},
	} {
		field := &crField{typ: &model.Type{Kind: tc.kind}}
		if err := (&builder{}).contractDefault(field, &model.Attrs{Default: &tc.value}, "spec.newField"); err != nil {
			t.Fatal(err)
		}
		want := tc.value
		if tc.kind == model.List {
			want = "{}"
		}
		if len(field.Markers) != 1 || field.Markers[0] != "+kubebuilder:default="+want {
			t.Fatalf("%s default markers: %v", tc.kind, field.Markers)
		}
	}
}

func TestReleasedEnumValues(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    zero: unspecified\n    values: [THING_PRIORITY_LOW]\n    rejected: [THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    names: {THING_PRIORITY_LOW: alerts}\n    aliases: {unspecified: alerts}\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/thing_convert.go"])
	if !strings.Contains(types, "+kubebuilder:validation:Enum=unspecified;alerts") {
		t.Error("CR enum marker has the wrong values")
	}
	if !strings.Contains(convert, `"unspecified": thingsservice.THINGPRIORITY_THING_PRIORITY_LOW`) ||
		!strings.Contains(convert, `"alerts":      thingsservice.THINGPRIORITY_THING_PRIORITY_LOW`) {
		t.Error("alias and accepted value must map to the checked SDK constant")
	}
}

func TestReleasedEnumCanRejectExplicitZero(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    zero: unspecified\n    rejectZero: true\n    values: [THING_PRIORITY_LOW]\n    rejected: [THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/thing_convert.go"])
	if !strings.Contains(types, "+kubebuilder:validation:Enum=low") || strings.Contains(types, "Enum=unspecified") {
		t.Error("CR enum must accept only the business value")
	}
	if strings.Contains(convert, `"unspecified":`) || !strings.Contains(convert, `"low":`) {
		t.Error("conversion must reject an explicit zero value")
	}
}

func TestReleasedRequiredAndNameLinesMustChangeOutput(t *testing.T) {
	for _, line := range []string{"name: {required: true}", "name: {name: name}", "limit: {optional: true}"} {
		t.Run(line, func(t *testing.T) {
			file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\ntypes:\n  BodyIDThing:\n    fields:\n      " + line + "\n"
			dir := t.TempDir()
			openAPI := filepath.Join(dir, "openapi.yaml")
			override := filepath.Join(dir, "behavior-overrides.yaml")
			if err := os.WriteFile(openAPI, []byte(syntheticContract(t)), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(override, []byte(file), 0o644); err != nil {
				t.Fatal(err)
			}
			wantCode(t, Check(CheckOptions{Resource: "BodyIDThing", OpenAPIPath: openAPI, OverridesPath: override}), codeOverrideUnused)
		})
	}
}

func TestReleasedOptionalField(t *testing.T) {
	file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\ntypes:\n  BodyIDThing:\n    fields:\n      name: {optional: true}\n"
	files := checkLifecycleFixture(t, "BodyIDThing", syntheticContract(t), file)
	if !strings.Contains(string(files["api/coralogix/v1alpha1/bodyidthing_types.go"]), "Name *string `json:\"name,omitempty\"`") {
		t.Error("released name must be optional in the CR")
	}
}

func TestSkipReleasedTopLevelField(t *testing.T) {
	file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\ntypes:\n  BodyIDThing:\n    fields:\n      limit: {skip: true}\n"
	files := checkLifecycleFixture(t, "BodyIDThing", syntheticContract(t), file)
	for path, absent := range map[string][]string{
		"api/coralogix/v1alpha1/bodyidthing_types.go":        {"Limit", "json:\"limit"},
		"api/coralogix/v1alpha1/bodyidthing_convert.go":      {"out.Limit", "in.Limit"},
		"config/samples/v1alpha1/generated/bodyidthing.yaml": {"limit:"},
	} {
		for _, text := range absent {
			if strings.Contains(string(files[path]), text) {
				t.Errorf("%s still contains skipped field %q", path, text)
			}
		}
	}
}

func TestSkipRejectsInvalidLines(t *testing.T) {
	for _, tc := range []struct {
		name, field, code string
	}{
		{"false has no effect", "limit: {skip: false}", codeOverrideInvalid},
		{"conflicts with name", "limit: {skip: true, name: quota}", codeOverrideInvalid},
		{"required request field", "name: {skip: true}", "SKIP_REQUIRED_FIELD"},
		{"unknown field", "missing: {skip: true}", codeOverrideUnused},
		{"already absent from requests", "limit: {skip: true}", codeOverrideUnused},
		{"nested field", "ThingItem: {fields: {name: {skip: true}}}", codeOverrideInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := "types:\n  BodyIDThing:\n    fields:\n      " + tc.field + "\n"
			if tc.name == "nested field" {
				fields = "types:\n  " + tc.field + "\n"
			}
			file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\n" + fields
			dir := t.TempDir()
			openAPI, override := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
			contract := syntheticContract(t)
			if tc.name == "already absent from requests" {
				property := "                limit: {type: integer, format: int32, minimum: 1, maximum: 10, x-coralogix-presence: true}\n"
				contract = strings.Replace(contract, property, "", 2)
			}
			if err := os.WriteFile(openAPI, []byte(contract), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(override, []byte(file), 0o644); err != nil {
				t.Fatal(err)
			}
			wantCode(t, Check(CheckOptions{Resource: "BodyIDThing", OpenAPIPath: openAPI, OverridesPath: override}), tc.code)
		})
	}
}

func TestReleasedEnumVerbatimTrimmedAndBusinessFirst(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    zero: unspecified\n    values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    trimPrefix: THING_PRIORITY_\n    verbatim: true\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	if !strings.Contains(types, "Enum=unspecified;LOW;PAGER_DUTY_INCIDENTS") {
		t.Error("verbatim and trimPrefix must set the CR enum values")
	}
	contract := strings.Replace(syntheticContract(t), "enum: [THING_PRIORITY_UNSPECIFIED, THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]", "enum: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]", 1)
	file = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    verbatim: true\n"
	files = checkLifecycleFixture(t, "Thing", contract, file)
	if !strings.Contains(string(files["api/coralogix/v1alpha1/thing_types.go"]), "Enum=THING_PRIORITY_LOW;THING_PRIORITY_PAGER_DUTY_INCIDENTS") {
		t.Error("a business first value must stay accepted")
	}
}

func TestReleasedEnumNameKeepsOtherDefaultNames(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    zero: unspecified\n    values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    names: {THING_PRIORITY_LOW: alerts}\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	if !strings.Contains(string(files["api/coralogix/v1alpha1/thing_types.go"]), "Enum=unspecified;alerts;pagerDutyIncidents") {
		t.Error("the name override must keep the other CR spelling")
	}
}

func TestCompatibilityLinesRejectInvalidValues(t *testing.T) {
	cases := []struct {
		resource, fields string
		code             string
	}{
		{"BodyIDThing", "types:\n  BodyIDThing:\n    fields:\n      name: {name: limit}\n", codeNameCollision},
		{"BodyIDThing", "types:\n  BodyIDThing:\n    fields:\n      name: {required: true, optional: true}\n", codeOverrideInvalid},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW], rejected: [NOT_IN_API]}\n", codeOverrideInvalid},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW], rejected: [THING_PRIORITY_UNSPECIFIED, THING_PRIORITY_PAGER_DUTY_INCIDENTS]}\n", codeOverrideInvalid},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW], names: {THING_PRIORITY_LOW: low}}\n", codeOverrideInvalid},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS], names: {THING_PRIORITY_LOW: same, THING_PRIORITY_PAGER_DUTY_INCIDENTS: same}}\n", codeNameCollision},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW], rejected: [THING_PRIORITY_PAGER_DUTY_INCIDENTS], aliases: {unspecified: missing}}\n", codeOverrideInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.resource+tc.fields, func(t *testing.T) {
			head := "resource: " + tc.resource + "\nmode: existing\nvalidators: {inferred: true}\n"
			if tc.resource == "BodyIDThing" {
				head = "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\n"
			}
			dir := t.TempDir()
			openAPI, override := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
			if err := os.WriteFile(openAPI, []byte(syntheticContract(t)), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(override, []byte(head+tc.fields), 0o644); err != nil {
				t.Fatal(err)
			}
			wantCode(t, Check(CheckOptions{Resource: tc.resource, OpenAPIPath: openAPI, OverridesPath: override}), tc.code)
		})
	}
}

func TestEnumLineWithoutOutputChangeIsUnused(t *testing.T) {
	contract := strings.Replace(syntheticContract(t), "enum: [THING_PRIORITY_UNSPECIFIED, THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]", "enum: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]", 1)
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority: {values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]}\n"
	dir := t.TempDir()
	openAPI, override := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
	if err := os.WriteFile(openAPI, []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	wantCode(t, Check(CheckOptions{Resource: "Thing", OpenAPIPath: openAPI, OverridesPath: override}), codeOverrideUnused)
}

func TestEnumMarkerDelimitersAreRejected(t *testing.T) {
	for name, line := range map[string]string{
		"zero":    "zero: 'unspec,ified'\n    values: [THING_PRIORITY_LOW]",
		"names":   "values: [THING_PRIORITY_LOW]\n    names: {THING_PRIORITY_LOW: 'foo,bar'}",
		"aliases": "values: [THING_PRIORITY_LOW]\n    aliases: {'foo,bar': low}",
	} {
		t.Run(name, func(t *testing.T) {
			data := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    " + line + "\n"
			if _, err := overrides.Parse([]byte(data)); err == nil || !strings.Contains(err.Error(), "enums.ThingPriority") {
				t.Fatalf("Parse() error = %v, want an enum path error", err)
			}
		})
	}
}

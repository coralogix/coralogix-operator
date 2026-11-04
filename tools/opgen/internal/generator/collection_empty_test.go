package generator

import (
	"slices"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

func testBuilder(field overrides.Field) *builder {
	return &builder{
		kind: "Thing",
		file: &overrides.File{Types: map[string]overrides.Type{
			"ThingConfig": {Fields: map[string]overrides.Field{"empty": field, "items": field}},
		}},
		used:  map[overrides.Line]bool{},
		types: map[string]*crType{},
		convs: map[string]string{},
		alias: "sdk",
		ix: &refIndex{fields: map[string]sdkcheck.Ref{
			"SDKConfig.Empty":    {Name: "Empty", Want: "map[string]interface{}"},
			"SDKConfig.Items":    {Name: "Items", Want: "[]string"},
			"SDKItem.EntityType": {Name: "EntityType", Want: "string"},
		}},
	}
}

func TestRequiredListCanOmitEmptyRequestValue(t *testing.T) {
	b := testBuilder(overrides.Field{OmitEmptyInRequest: true})
	list := &model.Type{Kind: model.List, Elem: &model.Type{Kind: model.String}}
	f, err := b.field("ThingConfig", "ThingConfig", "items", "", list, true, "spec.items", false)
	if err != nil {
		t.Fatal(err)
	}
	if f.Optional {
		t.Fatal("the field must stay required in the CRD")
	}
	code, err := b.assignFrom(f, list, "SDKConfig", "config.items", "in.Items")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "if len(in.Items) != 0 {") {
		t.Fatalf("the empty list must stay out of the request:\n%s", code)
	}
}

func TestOmitEmptyRequestNeedsRequiredCollection(t *testing.T) {
	for _, tt := range []struct {
		name     string
		typ      *model.Type
		required bool
	}{
		{name: "scalar", typ: &model.Type{Kind: model.String}, required: true},
		{name: "optional list", typ: &model.Type{Kind: model.List, Elem: &model.Type{Kind: model.String}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := testBuilder(overrides.Field{OmitEmptyInRequest: true})
			_, err := b.field("ThingConfig", "ThingConfig", "items", "", tt.typ, tt.required, "spec.items", false)
			if err == nil || !strings.Contains(err.Error(), "omitEmptyInRequest needs a required collection") {
				t.Fatalf("want an invalid override, got %v", err)
			}
		})
	}
}

func TestEmptyObjectArm(t *testing.T) {
	empty := &model.Type{Kind: model.Object, Schema: "EmptyConfig"}
	for _, tt := range []struct {
		name, shape string
		wantType    string
	}{
		{name: "struct", wantType: "*ThingEmptyConfig"},
		{name: "released map", shape: "map", wantType: "*map[string]string"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := testBuilder(overrides.Field{EmptyObject: tt.shape})
			f, err := b.field("ThingConfig", "ThingConfig", "empty", "", empty, false, "spec.config.empty", false)
			if err != nil {
				t.Fatal(err)
			}
			if f.GoType != tt.wantType {
				t.Fatalf("Go type = %s, want %s", f.GoType, tt.wantType)
			}
			if tt.shape == "map" && !slices.Contains(f.Markers, "+kubebuilder:validation:MaxProperties=0") {
				t.Fatal("the map does not reject nonempty data")
			}
			code, err := b.assignFrom(f, empty, "SDKConfig", "config.empty", "in.Empty")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(code, "if in.Empty != nil {") || !strings.Contains(code, "out.Empty = map[string]interface{}{}") {
				t.Fatalf("the request must omit an absent arm and send an empty map for a present arm:\n%s", code)
			}
		})
	}
}

func TestSetAsReleasedList(t *testing.T) {
	item := &model.Type{Kind: model.Object, Schema: "Item", Fields: []*model.Field{{
		Name: "entityType", Type: &model.Type{Kind: model.String}, Behavior: model.Normal,
		Create: &model.Attrs{Required: true},
	}}}
	for _, tt := range []struct {
		name, key string
		elem      *model.Type
		markers   []string
	}{
		{name: "scalar", elem: &model.Type{Kind: model.String}},
		{name: "object", elem: item},
		{name: "list-map", elem: item, key: "entityType", markers: []string{"+listType=map", "+listMapKey=entityType"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := testBuilder(overrides.Field{Collection: "list", ListMapKey: tt.key})
			f, err := b.field("ThingConfig", "ThingConfig", "items", "", &model.Type{Kind: model.Set, Elem: tt.elem}, false, "spec.items", false)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(f.GoType, "[]") || slices.Contains(f.Markers, "+listType=set") {
				t.Fatalf("the released list gained set behavior: %s, %v", f.GoType, f.Markers)
			}
			for _, marker := range tt.markers {
				if !slices.Contains(f.Markers, marker) {
					t.Fatalf("missing %s in %v", marker, f.Markers)
				}
			}
			want := "[]string"
			if tt.elem.Kind == model.Object {
				want = "[]SDKItem"
			}
			stmts, expr, _, err := b.value(f, f.typ, "in.Items", want, "config.items")
			if err != nil {
				t.Fatal(err)
			}
			if tt.elem.Kind == model.String && expr != "in.Items" {
				t.Fatalf("the scalar list changed its values: %s", expr)
			}
			if tt.elem.Kind == model.Object && (!strings.Contains(stmts, "range in.Items") || !strings.Contains(stmts, "append(")) {
				t.Fatalf("object items must keep their input order:\n%s", stmts)
			}
		})
	}
}

func TestCollectionOverrideNeedsASetAndAValidKey(t *testing.T) {
	for _, tt := range []struct {
		name     string
		field    overrides.Field
		typ      *model.Type
		keyIsRef bool
	}{
		{name: "list is not a set", field: overrides.Field{Collection: "list"}, typ: &model.Type{Kind: model.List, Elem: &model.Type{Kind: model.String}}},
		{name: "scalar has no key", field: overrides.Field{Collection: "list", ListMapKey: "entityType"}, typ: &model.Type{Kind: model.Set, Elem: &model.Type{Kind: model.String}}},
		{name: "reference is not a string key", field: overrides.Field{Collection: "list", ListMapKey: "entityType"}, typ: &model.Type{Kind: model.Set, Elem: &model.Type{Kind: model.Object, Schema: "Item", Fields: []*model.Field{{
			Name: "entityType", Type: &model.Type{Kind: model.String}, Behavior: model.Normal, Create: &model.Attrs{Required: true},
		}}}}, keyIsRef: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := testBuilder(tt.field)
			if tt.keyIsRef {
				b.file.Types["Item"] = overrides.Type{Fields: map[string]overrides.Field{
					"entityType": {Name: "entityType", Ref: &overrides.Ref{Kind: "Thing"}},
				}}
			}
			_, err := b.field("ThingConfig", "ThingConfig", "items", "", tt.typ, false, "spec.items", false)
			if err == nil || !strings.Contains(err.Error(), "behavior-overrides.yaml") || !strings.Contains(err.Error(), "collection") && !strings.Contains(err.Error(), "listMapKey") {
				t.Fatalf("want an invalid override, got %v", err)
			}
		})
	}
}

func TestCollectionOverrideParsing(t *testing.T) {
	for _, tt := range []struct {
		name, mode, field string
		valid             bool
	}{
		{name: "plain list", mode: "existing", field: "collection: list", valid: true},
		{name: "omit empty request", mode: "existing", field: "omitEmptyInRequest: true", valid: true},
		{name: "omit empty new kind", mode: "new", field: "omitEmptyInRequest: true"},
		{name: "list-map", mode: "existing", field: "collection: list\n        listMapKey: entityType", valid: true},
		{name: "unknown collection", mode: "existing", field: "collection: bag"},
		{name: "key without list", mode: "existing", field: "listMapKey: entityType"},
		{name: "new kind", mode: "new", field: "collection: list"},
		{name: "unknown key", mode: "existing", field: "collection: list\n        listMapKeys: entityType"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := "resource: Thing\nmode: " + tt.mode + "\nvalidators:\n  inferred: true\ntypes:\n  ThingConfig:\n    fields:\n      items:\n        " + tt.field + "\n"
			_, err := overrides.Parse([]byte(data))
			if (err == nil) != tt.valid {
				t.Fatalf("valid = %t, want %t: %v", err == nil, tt.valid, err)
			}
		})
	}
}

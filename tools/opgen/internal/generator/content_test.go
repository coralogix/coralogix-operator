package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

const contentFileHead = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n"

func TestContentSpecAndAdoptKeysParse(t *testing.T) {
	for _, tt := range []struct {
		name, file, wantErr string
	}{
		{name: "content", file: contentFileHead + "content: {decode: example.com/sdk/go/openapi/thingjson.Unmarshal}\n"},
		{name: "content in mode new", file: "resource: Thing\nmode: new\ncontent: {decode: example.com/sdk/thingjson.Unmarshal}\n", wantErr: "a new kind follows the contract"},
		{name: "content without a package", file: contentFileHead + "content: {decode: Unmarshal}\n", wantErr: "is not <import path>.<Func>"},
		{name: "content with an unexported function", file: contentFileHead + "content: {decode: example.com/sdk/thingjson.unmarshal}\n", wantErr: "is not <import path>.<Func>"},
		{name: "content with an unknown key", file: contentFileHead + "content: {decode: example.com/sdk/thingjson.Unmarshal, sources: [json]}\n", wantErr: "sources"},
		{name: "spec with a hook", file: contentFileHead + "hooks: [beforeCreate]\nspec: {folder: {type: ThingFolder}}\n"},
		{name: "spec without a hook", file: contentFileHead + "spec: {folder: {type: ThingFolder}}\n", wantErr: "needs a hook"},
		{name: "spec with an unexported type", file: contentFileHead + "hooks: [beforeCreate]\nspec: {folder: {type: thingFolder}}\n", wantErr: "not an exported Go type"},
		{name: "spec with a bad name", file: contentFileHead + "hooks: [beforeCreate]\nspec: {bad-name: {type: ThingFolder}}\n", wantErr: "not a spec property name"},
		{name: "spec that names a content field", file: contentFileHead + "hooks: [beforeCreate]\ncontent: {decode: example.com/sdk/thingjson.Unmarshal}\nspec: {json: {type: ThingJSON}}\n", wantErr: "the content key has this field"},
		{name: "spec in mode new", file: "resource: Thing\nmode: new\nspec: {folder: {type: ThingFolder}}\n", wantErr: "a new kind follows the contract"},
		{name: "adopt", file: contentFileHead + "hooks: [adopt]\n"},
		{name: "adopt with afterCreate", file: contentFileHead + "hooks: [adopt, afterCreate, afterUpdate]\n", wantErr: "declare one"},
		{name: "field and omit sources", file: contentFileHead + "api:\n  requestWrapper: item\n  requestValues:\n    create: {policy: {source: field}, locked: {source: omit}}\n"},
		{name: "field source with a value", file: contentFileHead + "api:\n  requestWrapper: item\n  requestValues:\n    create: {policy: {source: field, value: x}}\n", wantErr: "field cannot set value"},
		{name: "omit source with a value", file: contentFileHead + "api:\n  requestWrapper: item\n  requestValues:\n    create: {locked: {source: omit, value: x}}\n", wantErr: "omit cannot set value"},
		{name: "createResponseOnlyIDProperty", file: contentFileHead + "api: {createResponseOnlyIDProperty: thingId}\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := overrides.Parse([]byte(tt.file))
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatal(err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("want an error with %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// editContentThing changes the behavior-overrides file of the synthetic ContentThing.
func editContentThing(t *testing.T, root string, edit func(string) string) {
	t.Helper()
	path := filepath.Join(root, kindsDir, "contentthing", overrides.FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(edit(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The CR of a content kind holds the resource as JSON text. A new field of the resource is in
// the SDK type, so the decoded content carries it: the generated files do not change.
func TestContentKindTakesANewResourceFieldWithoutChange(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	before := readTree(t, root)
	input.OpenAPI = []byte(replaceOnce(t, string(input.OpenAPI),
		"        mode: {type: string, enum: [MODE_A, MODE_B]}\n",
		"        mode: {type: string, enum: [MODE_A, MODE_B]}\n        theme: {type: string, description: A field that the API adds later.}\n"))
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	if diff := compareTrees(before, readTree(t, root)); diff != "" {
		t.Fatalf("a new field of the content changed the generated files:\n%s", diff)
	}
}

// A new field beside the request wrapper is a CR field or is left out: the run stops until a
// requestValues line says which.
func TestContentKindStopsOnANewRequestField(t *testing.T) {
	root, input := syntheticRoot(t)
	input.OpenAPI = []byte(strings.Replace(string(input.OpenAPI),
		"                locked: {type: boolean, x-coralogix-presence: true}\n",
		"                locked: {type: boolean, x-coralogix-presence: true}\n                pinned: {type: boolean, x-coralogix-presence: true}\n", 2))
	wantCode(t, generateFromInput(input, root, root), "RENDERER_SHAPE_UNSUPPORTED")
}

func TestContentKindChecksTheRequestValueLines(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(string) string
		code string
	}{
		{
			name: "a field source that the request requires",
			edit: func(s string) string {
				return strings.Replace(s, "policy: {source: field}", "requestId: {source: field}", -1)
			},
			code: codeOverrideInvalid,
		},
		{
			name: "a field source in create only",
			edit: func(s string) string {
				return strings.Replace(s, "      policy: {source: field}\n      locked: {source: omit}\ntypes:", "      locked: {source: omit}\ntypes:", 1)
			},
			code: "RENDERER_SHAPE_UNSUPPORTED",
		},
		{
			name: "a modelled writable field",
			edit: func(s string) string {
				return strings.Replace(s, "      id: {readOnly: true}\n", "      id: {readOnly: true}\n      name: {name: title}\n", 1)
			},
			code: "RENDERER_SHAPE_UNSUPPORTED",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root, input := syntheticRoot(t)
			editContentThing(t, root, tt.edit)
			err := generateFromInput(input, root, root)
			if err == nil {
				t.Fatal("want an error")
			}
			if tt.code == "RENDERER_SHAPE_UNSUPPORTED" {
				// Either the renderer or the request-value check reports it.
				if !strings.Contains(err.Error(), "RENDERER_SHAPE_UNSUPPORTED") && !strings.Contains(err.Error(), codeOverrideInvalid) {
					t.Fatalf("want a renderer or override error, got %v", err)
				}
				return
			}
			wantCode(t, err, tt.code)
		})
	}
}

// Create keeps a keepOnCleanup status field when it sets status.id, and the content kind reads
// its id from the Create response property.
func TestContentKindCreateReadsTheIDAndKeepsImported(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	controller := readKindFile(t, root, "contentthing", "controller")
	for _, want := range []string{
		`id := ptr.Deref(resp.ThingId, "")`,
		`cr.Status = coralogixv1alpha1.ContentThingStatus{ID: ptr.To(id), Imported: cr.Status.Imported}`,
		`adoptedID, err := r.adopt(ctx, log, cr, body)`,
		`// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch`,
	} {
		if !strings.Contains(controller, want) {
			t.Errorf("the controller has no %s", want)
		}
	}
	convert := readKindFile(t, root, "contentthing", "convert")
	for _, want := range []string{"out.Policy = in.Spec.Policy", "contentdecode.Unmarshal(text, out)"} {
		if !strings.Contains(convert, want) {
			t.Errorf("the conversion has no %s", want)
		}
	}
	if strings.Contains(convert, "Locked") {
		t.Error("the conversion sends the omitted field locked")
	}
}

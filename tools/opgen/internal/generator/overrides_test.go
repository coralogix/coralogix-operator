package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkRefThing runs opgen check on RefThing with the overrides file that edit makes from the
// file of the seed root.
func checkRefThing(t *testing.T, edit func(string) string) error {
	t.Helper()
	dir := filepath.Join("testdata", "root", "tools", "opgen", "kinds", "refthing")
	data, err := os.ReadFile(filepath.Join(dir, "behavior-overrides.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "behavior-overrides.yaml")
	if err := os.WriteFile(path, []byte(edit(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
	return Check(CheckOptions{Resource: "RefThing", OpenAPIPath: filepath.Join("testdata", "openapi.yaml"),
		E2EPath: filepath.Join(dir, "e2e.yaml"), OverridesPath: path})
}

func TestOverridesFile(t *testing.T) {
	if err := checkRefThing(t, func(s string) string { return s }); err != nil {
		t.Fatalf("the seed file: %v", err)
	}
	// A new kind can use ref too. It keeps the contract checks of a new resource.
	if err := checkRefThing(t, func(s string) string {
		return replaceOnce(t, replaceOnce(t, s, "validators:\n  inferred: true\n", ""), "mode: existing", "mode: new")
	}); err != nil {
		t.Fatalf("mode new: %v", err)
	}
	// A new kind keeps the contract limits by default. inferred: false drops them, for an API whose
	// server does not apply them, and inferred: true states the default. A section without the key
	// says nothing.
	for validators, wantErr := range map[string]bool{
		"validators:\n  inferred: false\n": false,
		"validators:\n  inferred: true\n":  false,
		"validators: null\n":               true,
		"validators: {}\n":                 true,
	} {
		err := checkRefThing(t, func(s string) string {
			return replaceOnce(t, replaceOnce(t, s, "validators:\n  inferred: true\n", validators), "mode: existing", "mode: new")
		})
		if wantErr {
			wantCode(t, err, codeOverrideInvalid)
		} else if err != nil {
			t.Errorf("mode new with %q: %v", validators, err)
		}
	}
	tests := []struct {
		name string
		edit func(t *testing.T, s string) string
		code string
	}{
		{
			name: "an unknown key",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "ref: {kind: Thing}\n      parentId:", "ref: {kind: Thing}\n        name: thing\n      parentId:")
			},
			code: codeOverrideInvalid,
		},
		{
			name: "a line for a field that the contract does not have",
			edit: func(t *testing.T, s string) string { return replaceOnce(t, s, "parentId:", "ownerId:") },
			code: codeOverrideUnused,
		},
		{
			name: "a line for a component that the resource does not use",
			edit: func(t *testing.T, s string) string {
				return s + "  ThingItem:\n    fields:\n      name:\n        ref: {kind: Thing}\n"
			},
			code: codeOverrideUnused,
		},
		{
			name: "a file of another resource",
			edit: func(t *testing.T, s string) string { return replaceOnce(t, s, "resource: RefThing", "resource: Thing") },
			code: "OVERRIDE_RESOURCE_MISMATCH",
		},
		{
			name: "a wrong mode",
			edit: func(t *testing.T, s string) string { return replaceOnce(t, s, "mode: existing", "mode: released") },
			code: codeOverrideInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantCode(t, checkRefThing(t, func(s string) string { return tt.edit(t, s) }), tt.code)
		})
	}
}

// A ref line on a field that is not a string id is an error.
func TestRefNeedsAStringID(t *testing.T) {
	dir := t.TempDir()
	overridesPath := filepath.Join(dir, "behavior-overrides.yaml")
	file := "resource: Thing\nmode: new\ntypes:\n  Thing:\n    fields:\n      enabled:\n        ref: {kind: Thing}\n"
	if err := os.WriteFile(overridesPath, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Check(CheckOptions{Resource: "Thing", OpenAPIPath: filepath.Join("testdata", "openapi.yaml"), OverridesPath: overridesPath,
		E2EPath: filepath.Join("testdata", "root", "tools", "opgen", "kinds", "thing", "e2e.yaml")})
	wantCode(t, err, codeOverrideInvalid)
}

// A ref line names a kind of the operator.
func TestRefKindMustExist(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, kindsDir, "refthing", "behavior-overrides.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(replaceOnce(t, string(data), "ref: {kind: Thing}\n      parentId", "ref: {kind: Thingy}\n      parentId")), 0o644); err != nil {
		t.Fatal(err)
	}
	wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
}

// checkBodyIDThing runs opgen check on BodyIDThing with the overrides file that edit makes from the
// file of the seed root.
func checkBodyIDThing(t *testing.T, edit func(string) string) error {
	t.Helper()
	dir := filepath.Join("testdata", "root", "tools", "opgen", "kinds", "bodyidthing")
	data, err := os.ReadFile(filepath.Join(dir, "behavior-overrides.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "behavior-overrides.yaml")
	if err := os.WriteFile(path, []byte(edit(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
	return Check(CheckOptions{Resource: "BodyIDThing", OpenAPIPath: filepath.Join("testdata", "openapi.yaml"), OverridesPath: path})
}

// The validators and api keys of a kind that users already have.
func TestValidatorsAndAPIKeys(t *testing.T) {
	if err := checkBodyIDThing(t, func(s string) string { return s }); err != nil {
		t.Fatalf("the seed file: %v", err)
	}
	tests := []struct {
		name string
		edit func(t *testing.T, s string) string
		code string
	}{
		{
			name: "without api.updateIDInBody the Replace has no id in its path",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "api:\n  updateIDInBody: true   # Replace sends the id in the body\n", "")
			},
			code: "",
		},
		{
			name: "an api key that opgen does not support yet",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "  updateIDInBody: true", "  updateIDInBody: true\n  clientSetID: true")
			},
			code: codeOverrideInvalid,
		},
		{
			name: "an api key that opgen does not support yet, set to false",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "  updateIDInBody: true", "  updateIDInBody: true\n  clientSetID: false")
			},
			code: codeOverrideInvalid,
		},
		{
			name: "api.updateIDInBody set to false",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "  updateIDInBody: true", "  updateIDInBody: false")
			},
			code: codeOverrideInvalid,
		},
		{
			name: "an empty api section",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "api:\n  updateIDInBody: true   # Replace sends the id in the body\n", "api: {}\n")
			},
			code: codeOverrideInvalid,
		},
		{
			name: "mode existing without validators.inferred",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "validators:\n  inferred: false        # the released kind has no length or bound limits\n", "")
			},
			code: codeOverrideInvalid,
		},
		{
			name: "mode new with an api key",
			edit: func(t *testing.T, s string) string {
				s = replaceOnce(t, s, "validators:\n  inferred: false        # the released kind has no length or bound limits\n", "")
				return replaceOnce(t, s, "mode: existing", "mode: new")
			},
			code: codeOverrideInvalid,
		},
		{
			name: "mode new with an api key set to false",
			edit: func(t *testing.T, s string) string {
				s = replaceOnce(t, s, "validators:\n  inferred: false        # the released kind has no length or bound limits\n", "")
				s = replaceOnce(t, s, "  updateIDInBody: true", "  updateIDInBody: false")
				return replaceOnce(t, s, "mode: existing", "mode: new")
			},
			code: codeOverrideInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkBodyIDThing(t, func(s string) string { return tt.edit(t, s) })
			if tt.code == "" {
				if err == nil {
					t.Fatal("want an error")
				}
				return
			}
			wantCode(t, err, tt.code)
		})
	}
}

// validators.inferred: false on a contract without limits changes nothing, so the line is unused.
func TestInferredFalseWithoutLimitsIsUnused(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// An optional name without limits: a required string always has minLength 1.
	contract := string(data)
	for old, replacement := range map[string]string{
		"title: CreateBodyIDThingRequest\n              type: object\n              required: [name]": "title: CreateBodyIDThingRequest\n              type: object",
		"title: BodyIDThing\n              type: object\n              required: [id, name]":          "title: BodyIDThing\n              type: object\n              required: [id]",
		"BodyIDThing:\n      type: object\n      required: [id, name]":                                "BodyIDThing:\n      type: object\n      required: [id]",
		"minLength: 1, maxLength: 100, ":                                                              "",
		"minimum: 1, maximum: 10, ":                                                                   "",
	} {
		if !strings.Contains(contract, old) {
			t.Fatalf("the contract has no %q", old)
		}
		contract = strings.ReplaceAll(contract, old, replacement)
	}
	dir := t.TempDir()
	openAPI := filepath.Join(dir, "openapi.yaml")
	if err := os.WriteFile(openAPI, []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	overridesPath := filepath.Join("testdata", "root", "tools", "opgen", "kinds", "bodyidthing", "behavior-overrides.yaml")
	wantCode(t, Check(CheckOptions{Resource: "BodyIDThing", OpenAPIPath: openAPI, OverridesPath: overridesPath}), codeOverrideUnused)
}

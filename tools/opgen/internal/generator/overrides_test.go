package generator

import (
	"os"
	"path/filepath"
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
	if err := checkRefThing(t, func(s string) string { return replaceOnce(t, s, "mode: existing", "mode: new") }); err != nil {
		t.Fatalf("mode new: %v", err)
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

package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/source"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// topLevelMaskContract is the synthetic contract with the PatchThing mask in the Update body, and
// a mask pattern that rejects dotted paths: Update cannot name an arm of the target oneOf.
func topLevelMaskContract(t *testing.T) string {
	t.Helper()
	text := syntheticContract(t)
	old := "      parameters:\n        - name: updateMask\n          in: query\n          required: true\n          schema:\n            type: string\n            pattern: '^[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*(,[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*)*$'\n"
	text = replaceOnce(t, text, old, "")
	return replaceOnce(t, text, "title: UpdatePatchThingRequest\n              type: object\n              properties:\n",
		"title: UpdatePatchThingRequest\n              type: object\n              properties:\n                updateMask: {type: string, pattern: '^[a-zA-Z_][a-zA-Z0-9_]*(,[a-zA-Z_][a-zA-Z0-9_]*)*$'}\n")
}

// renderKindFixture checks one kind as generate does, through the per-kind contract, and returns
// its files.
func renderKindFixture(t *testing.T, contract, override string) (map[string][]byte, error) {
	t.Helper()
	dir := t.TempDir()
	openAPIPath, overridePath := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
	if err := os.WriteFile(openAPIPath, []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overridePath, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	const resource = "PatchThing"
	if err := Check(CheckOptions{Resource: resource, OpenAPIPath: openAPIPath, OverridesPath: overridePath}); err != nil {
		return nil, err
	}
	file, err := overrides.Parse([]byte(override))
	if err != nil {
		t.Fatal(err)
	}
	entry := kindEntry{Kind: resource, Resource: resource}
	doc, err := (&documents{raw: []byte(contract)}).forKind(entry, file)
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
	return files, nil
}

const patchThingExisting = "resource: PatchThing\nmode: existing\nvalidators: {inferred: true}\napi:\n  updateMaskInBody: true\n"

// A mask of top-level paths cannot select an arm of a nested oneOf. With updateMask: omit, Update
// sends no mask and the server replaces the whole target, so the kind generates.
func TestUpdateMaskOmitSendsNoMask(t *testing.T) {
	contract := topLevelMaskContract(t)
	_, err := renderKindFixture(t, contract, patchThingExisting)
	wantCode(t, err, "UPDATE_MASK_NESTED_ONEOF_UNSUPPORTED")

	files, err := renderKindFixture(t, contract, patchThingExisting+"updateMask: omit\n")
	if err != nil {
		t.Fatal(err)
	}
	convert := string(files["api/coralogix/v1alpha1/patchthing_convert.go"])
	if strings.Contains(convert, "UpdateMask") {
		t.Fatalf("Update still sends a mask:\n%s", convert)
	}
	if !strings.Contains(convert, "out.Target") {
		t.Fatal("Update no longer sends the target")
	}
	controller := string(files["internal/controller/coralogix/v1alpha1/patchthing_controller.go"])
	if strings.Contains(controller, "UpdateMask") {
		t.Fatal("the controller still sets a mask")
	}
}

// The key needs a mask in the body, and an existing kind.
func TestUpdateMaskOmitNeedsABodyMaskOfAnExistingKind(t *testing.T) {
	for name, file := range map[string]string{
		"a query mask":        "resource: PatchThing\nmode: existing\nvalidators: {inferred: true}\nupdateMask: omit\n",
		"a new kind":          "resource: PatchThing\nmode: new\nupdateMask: omit\n",
		"another value":       patchThingExisting + "updateMask: all\n",
		"with api.updateMask": "resource: PatchThing\nmode: existing\nvalidators: {inferred: true}\napi:\n  updateMaskInBody: true\n  updateMask: {nested: [target]}\nupdateMask: omit\n",
	} {
		if _, err := overrides.Parse([]byte(file)); err == nil {
			t.Errorf("%s: the file parses", name)
		}
	}
}

// An unsupported arm leaves the CR, the requests, and the oneOf rule. The other arm stays.
func TestUnsupportedArmLeavesTheCR(t *testing.T) {
	file := patchThingExisting + "updateMask: omit\ntypes:\n  PatchThingTarget:\n    unsupportedArms:\n      - {arm: prefix, reason: another kind manages prefixes}\n"
	files, err := renderKindFixture(t, topLevelMaskContract(t), file)
	if err != nil {
		t.Fatal(err)
	}
	types := string(files["api/coralogix/v1alpha1/patchthing_types.go"])
	if strings.Contains(types, "Prefix") {
		t.Fatalf("the unsupported arm is still in the CR:\n%s", types)
	}
	if !strings.Contains(types, "Some ") {
		t.Fatal("the other arm is gone")
	}
	if strings.Contains(string(files["api/coralogix/v1alpha1/patchthing_convert.go"]), "Prefix") {
		t.Fatal("a request still sends the unsupported arm")
	}
	if strings.Contains(string(files["config/samples/v1alpha1/generated/patchthing.yaml"]), "prefix") {
		t.Fatal("the sample still sets the unsupported arm")
	}
}

// Each arm must be a oneOf arm of its component, with a reason, and no other line may name it.
func TestUnsupportedArmRejectsWrongLines(t *testing.T) {
	contract := topLevelMaskContract(t)
	base := patchThingExisting + "updateMask: omit\ntypes:\n"
	for name, tc := range map[string]struct{ types, code string }{
		"a missing property":  {"  PatchThingTarget:\n    unsupportedArms:\n      - {arm: suffix, reason: r}\n", codeOverrideUnused},
		"a missing component": {"  PatchThingNope:\n    unsupportedArms:\n      - {arm: some, reason: r}\n", codeOverrideUnused},
		"a plain field":       {"  PatchThing:\n    unsupportedArms:\n      - {arm: description, reason: r}\n", codeOverrideInvalid},
		"every arm":           {"  PatchThingTarget:\n    unsupportedArms:\n      - {arm: some, reason: r}\n      - {arm: prefix, reason: r}\n", codeOverrideInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := renderKindFixture(t, contract, base+tc.types)
			wantCode(t, err, tc.code)
		})
	}
	for name, types := range map[string]string{
		"no reason":    "  PatchThingTarget:\n    unsupportedArms:\n      - {arm: prefix}\n",
		"listed twice": "  PatchThingTarget:\n    unsupportedArms:\n      - {arm: prefix, reason: r}\n      - {arm: prefix, reason: r}\n",
		"a field line": "  PatchThingTarget:\n    unsupportedArms:\n      - {arm: prefix, reason: r}\n    fields:\n      prefix: {name: start}\n",
	} {
		if _, err := overrides.Parse([]byte(base + types)); err == nil {
			t.Errorf("%s: the file parses", name)
		}
	}
	if _, err := overrides.Parse([]byte("resource: PatchThing\nmode: new\ntypes:\n  PatchThingTarget:\n    unsupportedArms:\n      - {arm: prefix, reason: r}\n")); err == nil {
		t.Error("a new kind can list an unsupported arm")
	}
}

// An immutable line makes a field that Update accepts one that only Create sends: the CRD rejects
// a change of it, and Update leaves it out, so a PATCH keeps the stored value.
func TestImmutableFieldLeavesUpdate(t *testing.T) {
	contract := topLevelMaskContract(t)
	file := patchThingExisting + "updateMask: omit\ntypes:\n  PatchThing:\n    fields:\n      name: {immutable: true}\n"
	files, err := renderKindFixture(t, contract, file)
	if err != nil {
		t.Fatal(err)
	}
	types := string(files["api/coralogix/v1alpha1/patchthing_types.go"])
	if !strings.Contains(types, `rule="self == oldSelf",message="spec.name is immutable"`) {
		t.Fatalf("the CRD does not keep the field:\n%s", types)
	}
	convert := string(files["api/coralogix/v1alpha1/patchthing_convert.go"])
	if strings.Count(convert, "out.Name =") != 1 {
		t.Fatalf("want the name in Create only:\n%s", convert)
	}
	// A field that Update does not have is immutable already.
	_, err = renderKindFixture(t, contract, patchThingExisting+"updateMask: omit\ntypes:\n  PatchThing:\n    fields:\n      kind: {immutable: true}\n")
	wantCode(t, err, codeOverrideUnused)
	// A full replace would clear the field.
	err = checkImmutableOnReplace(t)
	wantCode(t, err, codeOverrideInvalid)
	// A nested field and a new kind follow the contract.
	for name, types := range map[string]string{
		"a nested field": "types:\n  PatchThingPrefix:\n    fields:\n      value: {immutable: true}\n",
		"with skip":      "types:\n  PatchThing:\n    fields:\n      name: {immutable: true, skip: true}\n",
	} {
		if _, err := overrides.Parse([]byte(patchThingExisting + types)); err == nil {
			t.Errorf("%s: the file parses", name)
		}
	}
	if _, err := overrides.Parse([]byte("resource: PatchThing\nmode: new\ntypes:\n  PatchThing:\n    fields:\n      name: {immutable: true}\n")); err == nil {
		t.Error("a new kind can set immutable")
	}
}

// checkImmutableOnReplace checks an immutable line on BodyIDThing, whose Update is a full replace.
func checkImmutableOnReplace(t *testing.T) error {
	t.Helper()
	dir := t.TempDir()
	openAPIPath, overridePath := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
	if err := os.WriteFile(openAPIPath, []byte(syntheticContract(t)), 0o644); err != nil {
		t.Fatal(err)
	}
	file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\ntypes:\n  BodyIDThing:\n    fields:\n      name: {immutable: true}\n"
	if err := os.WriteFile(overridePath, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	return Check(CheckOptions{Resource: "BodyIDThing", OpenAPIPath: openAPIPath, OverridesPath: overridePath})
}

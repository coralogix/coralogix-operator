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

// The synthetic Thing is a full replace (PUT). The helpers below give its Create and Replace bodies
// the shapes that the request-shape keys describe.
//
//	Get:     note, alpha, beta
//	Create:  memo, bundle {alpha, beta}, teamId
//	Replace: memo, bundle {alpha, beta}, teamId

const (
	quotaLine = "                quota: {type: string, format: int64, x-coralogix-presence: true}\n"
	getQuota  = "        quota: {type: string, format: int64, description: \"The quota, a 64-bit number.\"}\n"
)

// requestShape edits the contract of Thing.
type requestShape struct {
	note         bool // Get has note. Create and Replace have memo.
	bundle       bool // Get has alpha and beta. Create and Replace have bundle {alpha, beta}.
	bundleOnly   string
	teamID       string // "", "optional", or "required"
	memoRequired bool   // Create and Replace require memo
}

func (c requestShape) contract(t *testing.T) func(string) string {
	return func(s string) string {
		if strings.Count(s, quotaLine) != 2 {
			t.Fatal("the synthetic Create and Replace bodies changed")
		}
		var create, replace, get, components string
		if c.note {
			create += "                memo: {type: string, x-coralogix-presence: true}\n"
			replace += "                memo: {type: string, x-coralogix-presence: true}\n"
			get += "        note: {type: string, description: A note.}\n"
		}
		if c.bundle {
			bundle := "                bundle: {$ref: '#/components/schemas/ThingBundle'}\n"
			if c.bundleOnly != "replace" {
				create += bundle
			}
			if c.bundleOnly != "create" {
				replace += bundle
			}
			get += "        alpha: {type: string, description: The first part.}\n        beta: {type: array, items: {type: string}, description: The second part.}\n"
			components = "    ThingBundle:\n      type: object\n      properties:\n        alpha: {type: string, x-coralogix-presence: true}\n        beta: {type: array, items: {type: string}}\n"
		}
		if c.teamID != "" {
			create += "                teamId: {type: integer, format: int32, x-coralogix-presence: true}\n"
			replace += "                teamId: {type: integer, format: int32, x-coralogix-presence: true}\n"
		}
		i := strings.Index(s, quotaLine) + len(quotaLine)
		s = s[:i] + create + s[i:]
		i = strings.Index(s[i:], quotaLine) + i + len(quotaLine)
		s = s[:i] + replace + s[i:]
		s = replaceOnce(t, s, getQuota, getQuota+get)
		if c.memoRequired {
			s = replaceOnce(t, s, "required: [name, kind, config]", "required: [name, kind, config, memo]")
			s = replaceOnce(t, s, "required: [name, config]", "required: [name, config, memo]")
		}
		if c.teamID == "required" {
			s = replaceOnce(t, s, "required: [name, kind, config]", "required: [name, kind, config, teamId]")
		}
		return replaceOnce(t, s, "    ThingKind:\n", components+"    ThingKind:\n")
	}
}

func (c requestShape) sdk(t *testing.T) func(string) string {
	return func(s string) string {
		var create, replace, get, types string
		if c.note {
			create += "	Memo *string `json:\"memo,omitempty\"`\n"
			replace += "	Memo *string `json:\"memo,omitempty\"`\n"
			get += "	Note *string `json:\"note,omitempty\"`\n"
		}
		if c.bundle {
			if c.bundleOnly != "replace" {
				create += "	Bundle *ThingBundle `json:\"bundle,omitempty\"`\n"
			}
			if c.bundleOnly != "create" {
				replace += "	Bundle *ThingBundle `json:\"bundle,omitempty\"`\n"
			}
			get += "	Alpha *string `json:\"alpha,omitempty\"`\n	Beta []string `json:\"beta,omitempty\"`\n"
			types = "type ThingBundle struct {\n	Alpha *string `json:\"alpha,omitempty\"`\n	Beta []string `json:\"beta,omitempty\"`\n}\n\n"
		}
		if c.teamID != "" {
			create += "	TeamId *int32 `json:\"teamId,omitempty\"`\n"
			replace += "	TeamId *int32 `json:\"teamId,omitempty\"`\n"
		}
		s = replaceOnce(t, s, "type CreateThingRequest struct {\n", "type CreateThingRequest struct {\n"+create)
		s = replaceOnce(t, s, "type ReplaceThingRequest struct {\n", "type ReplaceThingRequest struct {\n"+replace)
		s = replaceOnce(t, s, "type Thing struct {\n", "type Thing struct {\n"+get)
		return replaceOnce(t, s, "type ThingKind string", types+"type ThingKind string")
	}
}

func (c requestShape) generate(t *testing.T, types string) (root string, err error) {
	t.Helper()
	root, input := skipRoot(t, c.contract(t), c.sdk(t))
	if types != "" {
		types = "types:\n  Thing:\n    fields:\n" + types
	}
	writeThingOverrides(t, root, types)
	return root, generateFromInput(input, root, root)
}

// functionBody returns the text from the first from to the next function.
func functionBody(t *testing.T, s, from string) string {
	t.Helper()
	i := strings.Index(s, from)
	if i < 0 {
		t.Fatalf("no %q in\n%s", from, s)
	}
	s = s[i:]
	if j := strings.Index(s[len(from):], "\nfunc "); j >= 0 {
		return s[:len(from)+j]
	}
	return s
}

// A top-level field that Create and Update have, and Get does not, has no CR field when a skip
// line names it. The requests never send it.
func TestSkipRemovesARequestOnlyField(t *testing.T) {
	c := requestShape{teamID: "optional"}
	root, err := c.generate(t, "      teamId: {skip: true}\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{apiDir + "/thing_types.go", apiDir + "/thing_convert.go", "config/samples/v1alpha1/generated/thing.yaml"} {
		if strings.Contains(strings.ToLower(readGenerated(t, root, path)), "teamid") {
			t.Errorf("%s has the skipped request-only field", path)
		}
	}
}

// A request that requires the field stays an error: the CR cannot send a value that it does not have.
func TestSkipRejectsARequiredRequestOnlyField(t *testing.T) {
	c := requestShape{teamID: "required"}
	_, err := c.generate(t, "      teamId: {skip: true}\n")
	if err == nil || !strings.Contains(err.Error(), "SKIP_REQUIRED_FIELD") {
		t.Fatalf("want SKIP_REQUIRED_FIELD, got %v", err)
	}
}

// Without the line, the field has no place in the CR, and the run stops.
func TestRequestOnlyFieldNeedsASkipLine(t *testing.T) {
	c := requestShape{teamID: "optional"}
	_, err := c.generate(t, "")
	if err == nil {
		t.Fatal("a request-only field without a skip line must stop the run")
	}
}

// Only skip can name a field that the resource component lacks. Another key on it stays an error,
// and a field that no body has stays an error.
func TestSkipLineOfARequestOnlyFieldIsChecked(t *testing.T) {
	c := requestShape{teamID: "optional"}
	for name, tc := range map[string]string{
		"another key":         "      teamId: {readOnly: true}\n",
		"a field no body has": "      missing: {skip: true}\n",
	} {
		_, err := c.generate(t, tc)
		if err == nil || !strings.Contains(err.Error(), codeOverrideUnused) {
			t.Errorf("%s: want %s, got %v", name, codeOverrideUnused, err)
		}
	}
}

// The CR field keeps the name of the Get response. The requests use the other name.
func TestRequestAsRenamesARequestField(t *testing.T) {
	c := requestShape{note: true}
	root, err := c.generate(t, "      note: {requestAs: memo}\n")
	if err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	if !strings.Contains(types, "Note *string") || strings.Contains(types, "Memo") {
		t.Fatalf("the CR must have note and no memo:\n%s", types)
	}
	convert := readGenerated(t, root, apiDir+"/thing_convert.go")
	create := functionBody(t, convert, "func (in *ThingSpec) toCreateThingRequest")
	update := functionBody(t, convert, "func (in *ThingSpec) toReplaceThingRequest")
	for name, body := range map[string]string{"Create": create, "Update": update} {
		if !strings.Contains(body, "out.Memo = in.Note") || strings.Contains(body, "out.Note") {
			t.Errorf("%s must set memo from note:\n%s", name, body)
		}
	}
}

// A PATCH sends an update mask. The mask path of a renamed field is not defined, so the line stops the run.
func TestRequestAsIsRejectedOnAPatchResource(t *testing.T) {
	edit := func(s string) string {
		s = replaceOnce(t, s, "                description: {type: string, x-coralogix-presence: true}\n                enabled: {type: boolean, default: true, x-coralogix-presence: true}\n                kind: {$ref: '#/components/schemas/PatchThingKind'}\n",
			"                summary: {type: string, x-coralogix-presence: true}\n                enabled: {type: boolean, default: true, x-coralogix-presence: true}\n                kind: {$ref: '#/components/schemas/PatchThingKind'}\n")
		return replaceOnce(t, s, "                description: {type: string, x-coralogix-presence: true}\n                enabled: {type: boolean, default: true, x-coralogix-presence: true}\n                target: {$ref: '#/components/schemas/PatchThingTarget'}\n      responses:\n        '200':\n          description: Updated",
			"                summary: {type: string, x-coralogix-presence: true}\n                enabled: {type: boolean, default: true, x-coralogix-presence: true}\n                target: {$ref: '#/components/schemas/PatchThingTarget'}\n      responses:\n        '200':\n          description: Updated")
	}
	editSDK := func(s string) string {
		s = strings.Replace(s, "type CreatePatchThingRequest struct {\n", "type CreatePatchThingRequest struct {\n	Summary *string `json:\"summary,omitempty\"`\n", 1)
		return strings.Replace(s, "type UpdatePatchThingRequest struct {\n", "type UpdatePatchThingRequest struct {\n	Summary *string `json:\"summary,omitempty\"`\n", 1)
	}
	root, input := skipRoot(t, edit, editSDK)
	dir := filepath.Join(root, kindsDir, "patchthing")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := "resource: PatchThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  PatchThing:\n    fields:\n      description: {requestAs: summary}\n"
	if err := os.WriteFile(filepath.Join(dir, "behavior-overrides.yaml"), []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	err := generateFromInput(input, root, root)
	if err == nil || !strings.Contains(err.Error(), "requestAs needs a full-replace Update") {
		t.Fatalf("want the PATCH error, got %v", err)
	}
}

// wrappedShape renders WrappedThing with a request wrapper that holds a request component of its
// own, as an API that takes its metadata in a wrapper does:
//
//	Get item:     id, name, title, note, tag
//	Create item:  id, name, heading, extra {note, tag}
//	Update item:  the same, or without extra when updateHasExtra is false
//
// It returns the generated conversions, or the error of the run.
func wrappedShape(t *testing.T, updateHasExtra, requireNote bool, lines string) (string, error) {
	t.Helper()
	contract, err := os.ReadFile(filepath.Join("testdata", "wrapped.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	spec := string(contract)
	spec = replaceOnce(t, spec, "                item: {$ref: '#/components/schemas/WrappedThing'}\n                label: {type: string}\n                requestId: {type: string}\n      responses:\n        '200': {description: Created",
		"                item: {$ref: '#/components/schemas/WrappedThingCreate'}\n                label: {type: string}\n                requestId: {type: string}\n      responses:\n        '200': {description: Created")
	spec = replaceOnce(t, spec, "                item: {$ref: '#/components/schemas/WrappedThing'}\n                id: {type: string}",
		"                item: {$ref: '#/components/schemas/WrappedThingUpdate'}\n                id: {type: string}")
	extra := "    WrappedExtra:\n      type: object\n      properties:\n"
	if requireNote {
		extra = "    WrappedExtra:\n      type: object\n      required: [note]\n      properties:\n"
	}
	update := "        extra: {$ref: '#/components/schemas/WrappedExtra'}\n"
	if !updateHasExtra {
		update = ""
	}
	required := "[name]"
	if requireNote {
		required = "[name, extra]"
	}
	spec = replaceOnce(t, spec, "    WrappedThing:\n", extra+"        note: {type: string, x-coralogix-presence: true}\n        tag: {type: array, items: {type: string}}\n"+
		"    WrappedThingCreate:\n      type: object\n      required: "+required+"\n      properties:\n        id: {type: string}\n        name: {type: string, minLength: 1, x-coralogix-presence: true}\n        heading: {type: string, x-coralogix-presence: true}\n        extra: {$ref: '#/components/schemas/WrappedExtra'}\n"+
		"    WrappedThingUpdate:\n      type: object\n      required: "+required+"\n      properties:\n        id: {type: string}\n        name: {type: string, minLength: 1, x-coralogix-presence: true}\n        heading: {type: string, x-coralogix-presence: true}\n"+update+
		"    WrappedThing:\n")
	spec = strings.TrimRight(spec, "\n") + "\n        title: {type: string, x-coralogix-presence: true}\n        note: {type: string, x-coralogix-presence: true}\n        tag: {type: array, items: {type: string}}\n"
	file := "resource: WrappedThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  requestWrapper: item\n  serverSetID: true\n  idProperty: id\n  updateIDInBody: true\n" + wrappedRequestValues + "types:\n  WrappedThing:\n    fields:\n      id:\n        readOnly: true\n" + lines
	doc, err := model.Load([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	override, err := overrides.Parse([]byte(file))
	if err != nil {
		t.Fatal(err)
	}
	v, err := validate(doc, kindEntry{Kind: "WrappedThing", Resource: "WrappedThing"}, source.SDKModule, operatorModule, &e2eFile{}, override)
	if err != nil {
		return "", err
	}
	files, _, err := renderOne(v, operatorModule)
	if err != nil {
		return "", err
	}
	return string(files["api/coralogix/v1alpha1/wrappedthing_convert.go"]), nil
}

const wrappedRenames = "      title: {requestAs: heading}\n      note: {requestAs: extra.note}\n      tag: {requestAs: extra.tag}\n"

// The path is read inside the request wrapper, and the wrapper can hold a request component that
// is not the resource. A path of one segment sets a field of that component.
func TestRequestAsBehindARequestWrapper(t *testing.T) {
	conversion, err := wrappedShape(t, true, false, wrappedRenames)
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range []string{"toWrappedThingCreateCreate", "toWrappedThingUpdateUpdate"} {
		body := functionBody(t, conversion, ") "+fn+"(")
		if !strings.Contains(body, "out.Heading = in.Title") {
			t.Errorf("%s must set heading from title:\n%s", fn, body)
		}
	}
}

// A path with a dot builds the outer object. Two fields under the same object share it: the
// second field must not replace the object of the first.
func TestRequestAsWithDotsBuildsOneSharedObject(t *testing.T) {
	conversion, err := wrappedShape(t, true, false, wrappedRenames)
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range []string{"toWrappedThingCreateCreate", "toWrappedThingUpdateUpdate"} {
		body := functionBody(t, conversion, ") "+fn+"(")
		if strings.Count(body, "WrappedExtra{}") != 1 || strings.Count(body, "out.Extra = ") != 1 {
			t.Errorf("%s must build one extra object:\n%s", fn, body)
		}
		for _, want := range []string{".Note = in.Note", ".Tag = ", "if in.Note != nil || len(in.Tag) != 0 {"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s misses %q:\n%s", fn, want, body)
			}
		}
	}
	if strings.Contains(conversion, "in.Extra") {
		t.Error("the CR must not have the request object")
	}
}

// A path that only one request has sends the field only in that request.
func TestRequestAsPathInOneRequestOnly(t *testing.T) {
	conversion, err := wrappedShape(t, false, false, wrappedRenames)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(functionBody(t, conversion, ") toWrappedThingCreateCreate("), "out.Extra = ") {
		t.Error("Create must send the object")
	}
	update := functionBody(t, conversion, ") toWrappedThingUpdateUpdate(")
	if strings.Contains(update, "Extra") || strings.Contains(update, "in.Note") {
		t.Errorf("Update must not send the fields:\n%s", update)
	}
}

// The model reads a path with a dot in the request wrapper. Without a wrapper it records no object
// of the path, and the SDK check stops the run.
func TestRequestAsWithDotsNeedsARequestWrapper(t *testing.T) {
	c := requestShape{bundle: true}
	_, err := c.generate(t, "      alpha: {requestAs: bundle.alpha}\n      beta: {requestAs: bundle.beta}\n")
	if err == nil || !strings.Contains(err.Error(), "goes through 0 objects") {
		t.Fatalf("want the iac error, got %v", err)
	}
}

// An iac code reaches the user when the path is not in a request.
func TestRequestAsRejectsAPathThatNoRequestHas(t *testing.T) {
	c := requestShape{note: true}
	_, err := c.generate(t, "      note: {requestAs: missing}\n")
	if err == nil || !strings.Contains(err.Error(), "REQUEST_NAME_NOT_FOUND") {
		t.Fatalf("want REQUEST_NAME_NOT_FOUND, got %v", err)
	}
	_, err = c.generate(t, "      note: {requestAs: memo.deeper}\n")
	if err == nil || !strings.Contains(err.Error(), "REQUEST_NAME_NOT_OBJECT") {
		t.Fatalf("want REQUEST_NAME_NOT_OBJECT, got %v", err)
	}
}

// A line that names the field itself changes nothing, so the run stops.
func TestRequestAsOfTheSameNameIsUnused(t *testing.T) {
	c := requestShape{note: true}
	_, err := c.generate(t, "      description: {requestAs: description}\n")
	if err == nil || !strings.Contains(err.Error(), codeOverrideUnused) {
		t.Fatalf("want %s, got %v", codeOverrideUnused, err)
	}
}

func TestRequestAsLineIsChecked(t *testing.T) {
	const head = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n"
	for name, tc := range map[string]string{
		"a new kind":           "resource: Thing\nmode: new\ntypes:\n  Thing:\n    fields:\n      note: {requestAs: memo}\n",
		"a nested path":        head + "types:\n  ThingItem:\n    fields:\n      name: {requestAs: a.memo}\n",
		"an empty segment":     head + "types:\n  Thing:\n    fields:\n      note: {requestAs: a..b}\n",
		"a trailing dot":       head + "types:\n  Thing:\n    fields:\n      note: {requestAs: a.}\n",
		"skip on the line":     head + "types:\n  Thing:\n    fields:\n      note: {requestAs: memo, skip: true}\n",
		"readOnly on the line": head + "types:\n  Thing:\n    fields:\n      note: {requestAs: memo, readOnly: true}\n",
		"createAs on the line": head + "types:\n  Thing:\n    fields:\n      note: {requestAs: memo, createAs: {x: y}}\n",
	} {
		if _, err := overrides.Parse([]byte(tc)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := overrides.Parse([]byte(head + "types:\n  Thing:\n    fields:\n      note: {requestAs: bundle.alpha}\n")); err != nil {
		t.Errorf("a path with dots must be accepted: %v", err)
	}
	if _, err := overrides.Parse([]byte(head + "types:\n  ThingItem:\n    fields:\n      name: {requestAs: memo}\n")); err != nil {
		t.Errorf("a name for a field of a nested component must be accepted: %v", err)
	}
}

// A released optional field can map to a required request property. The check "does a request
// require this field" follows the request path, not the name in Get.
func TestOptionalOverrideFollowsARequestAsRename(t *testing.T) {
	c := requestShape{note: true, memoRequired: true}
	root, err := c.generate(t, "      note: {requestAs: memo, optional: true}\n")
	if err != nil {
		t.Fatalf("a renamed field that a request requires can be optional in the CR: %v", err)
	}
	if !strings.Contains(readGenerated(t, root, apiDir+"/thing_types.go"), "Note *string") {
		t.Error("the CR field must be optional")
	}
	// The line stays unused when the request does not require the renamed field.
	c = requestShape{note: true}
	_, err = c.generate(t, "      note: {requestAs: memo, optional: true}\n")
	if err == nil || !strings.Contains(err.Error(), codeOverrideUnused) {
		t.Fatalf("want %s when memo is optional, got %v", codeOverrideUnused, err)
	}
}

// The same for a path with dots: the field is required when every segment of the path is required.
func TestOptionalOverrideFollowsADottedRequestAs(t *testing.T) {
	lines := "      title: {requestAs: heading}\n      note: {requestAs: extra.note, optional: true}\n      tag: {requestAs: extra.tag}\n"
	if _, err := wrappedShape(t, true, true, lines); err != nil {
		t.Fatalf("note is required in extra, and extra is required: %v", err)
	}
	_, err := wrappedShape(t, true, false, lines)
	if err == nil || !strings.Contains(err.Error(), codeOverrideUnused) {
		t.Fatalf("want %s when note is optional, got %v", codeOverrideUnused, err)
	}
}

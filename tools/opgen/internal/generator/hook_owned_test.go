package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// A field that a hook sets is not rendered, so the shared checks of its contract limits do not
// apply to it. A required string that the API accepts empty is no rule of the generated CR.
func TestRequiredEmptyStringBelowAHookFieldIsNotChecked(t *testing.T) {
	spec := wrappedContract(t, true, true)
	_, err := renderWrapped(t, spec, setByHookFile(wrappedRenames))
	wantCode(t, err, "REQUIRED_VALUE_CAN_BE_EMPTY")
	if _, err := renderWrapped(t, spec, setByHookFile(
		"      title: {requestAs: heading}\n      note: {requestAs: extra.note, setByHook: true}\n      tag: {requestAs: extra.tag}\n")); err != nil {
		t.Fatalf("a string below the hook field is not checked: %v", err)
	}
}

// wrappedModeContract holds an enum whose business values do not use the prefix of its zero value,
// as the note of the resource.
func wrappedModeContract(t *testing.T) string {
	t.Helper()
	spec := strings.ReplaceAll(wrappedContract(t, true, false), "note: {type: string, x-coralogix-presence: true}", "note: {$ref: '#/components/schemas/WrappedMode'}")
	return spec + "    WrappedMode:\n      type: string\n      enum: [WRAPPED_MODE_UNSPECIFIED, FAST, SLOW]\n"
}

// An enum below a field that a hook sets is read with any value prefix: no enums line can name the
// spellings of values that the generator never renders, and the line would change no generated code.
func TestEnumBelowAHookFieldNeedsNoEnumsLine(t *testing.T) {
	spec := wrappedModeContract(t)
	_, err := renderWrapped(t, spec, setByHookFile(wrappedRenames))
	wantCode(t, err, "ENUM_ZERO_INVALID")
	if _, err := renderWrapped(t, spec, setByHookFile(
		"      title: {requestAs: heading}\n      note: {requestAs: extra.note, setByHook: true}\n      tag: {requestAs: extra.tag}\n")); err != nil {
		t.Fatalf("an enum below the hook field has any prefix: %v", err)
	}
}

// A hook field exempts only the enum components below it. An enum component of another field keeps
// the strict prefix rule.
func TestEnumOfAnotherFieldKeepsThePrefixRule(t *testing.T) {
	spec := strings.ReplaceAll(wrappedModeContract(t), "title: {type: string, x-coralogix-presence: true}", "title: {$ref: '#/components/schemas/WrappedOther'}")
	spec = strings.ReplaceAll(spec, "heading: {type: string, x-coralogix-presence: true}", "heading: {$ref: '#/components/schemas/WrappedOther'}")
	spec += "    WrappedOther:\n      type: string\n      enum: [WRAPPED_OTHER_UNSPECIFIED, ONE, TWO]\n"
	_, err := renderWrapped(t, spec, setByHookFile(
		"      title: {requestAs: heading}\n      note: {requestAs: extra.note, setByHook: true}\n      tag: {requestAs: extra.tag}\n"))
	wantCode(t, err, "ENUM_ZERO_INVALID")
}

// sendNullAsEmpty on a top-level field states what the server stores for an omitted value, so the
// contract need not declare a default for a field that Get requires and Create leaves optional.
func TestSendNullAsEmptyStatesTheOmittedValue(t *testing.T) {
	spec := replaceOnce(t, wrappedIDBeside(t), "    WrappedThing:\n      type: object\n      required: [name]\n", "    WrappedThing:\n      type: object\n      required: [name, title]\n")
	rest := "      note: {requestAs: extra.note}\n      tag: {requestAs: extra.tag}\n"
	_, err := renderWrapped(t, spec, wrappedFile("      title: {requestAs: heading}\n"+rest))
	wantCode(t, err, "FIELD_SERVER_DEFAULT_UNDECLARED")
	files, err := renderWrapped(t, spec, wrappedFile("      title: {requestAs: heading, sendNullAsEmpty: true}\n"+rest))
	if err != nil {
		t.Fatal(err)
	}
	create := functionBody(t, string(files["api/coralogix/v1alpha1/wrappedthing_convert.go"]), ") toWrappedThingCreateCreate(")
	if !strings.Contains(create, `""`) {
		t.Errorf("an omitted title must be sent as empty:\n%s", create)
	}
}

// A spec field of a list type keeps a released field that is a plain slice. The list is the value
// itself, with omitempty.
func TestSpecFieldOfAListType(t *testing.T) {
	file := func(fieldType string, value bool) string {
		v := ""
		if value {
			v = ", value: true"
		}
		return wrappedFile(wrappedRenames) + "spec:\n  names: {type: '" + fieldType + "'" + v + "}\nhooks: [beforeCreate, beforeUpdate]\n"
	}
	files, err := renderWrapped(t, wrappedIDBeside(t), file("[]string", true))
	if err != nil {
		t.Fatal(err)
	}
	if types := string(files["api/coralogix/v1alpha1/wrappedthing_types.go"]); !strings.Contains(types, "Names []string `json:\"names,omitempty\"`") {
		t.Errorf("the spec must hold the list itself:\n%s", types)
	}
	if _, err := renderWrapped(t, wrappedIDBeside(t), file("[]WrappedNoteSource", true)); err != nil {
		t.Fatalf("a list of a handwritten type: %v", err)
	}
	for name, text := range map[string]string{
		"a list without value":    file("[]string", false),
		"a list of a number":      file("[]int", true),
		"a list of a lower name":  file("[]note", true),
		"a map":                   file("map[string]string", true),
		"a pointer to a list":     file("*[]string", true),
		"a list of a list":        file("[][]string", true),
		"a bare string":           file("string", true),
		"a list of a qualified":   file("[]pkg.Name", true),
		"an unexported list type": file("[]noteSource", true),
	} {
		if _, err := overrides.Parse([]byte(text)); err == nil {
			t.Errorf("%s: the file parses", name)
		}
	}
}

// A spec field can state its description, which is its doc in the type and so its description in
// the CRD. Without it the field has none.
func TestSpecFieldDescription(t *testing.T) {
	file := wrappedFile(wrappedRenames) + "spec:\n  names: {type: '[]string', value: true, description: The names that the hook reads.}\n  more: {type: '[]string', value: true}\nhooks: [beforeCreate, beforeUpdate]\n"
	files, err := renderWrapped(t, wrappedIDBeside(t), file)
	if err != nil {
		t.Fatal(err)
	}
	types := string(files["api/coralogix/v1alpha1/wrappedthing_types.go"])
	if !strings.Contains(types, "\t// The names that the hook reads.\n\t// +optional\n\tNames []string") {
		t.Errorf("the field must have its description:\n%s", types)
	}
	if !strings.Contains(types, "\t// +optional\n\tMore []string") {
		t.Errorf("a field without a description has only its markers:\n%s", types)
	}
}

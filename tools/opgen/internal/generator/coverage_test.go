package generator

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"go.yaml.in/yaml/v4"
)

func enumField(api, schema string, values ...string) *crField {
	m := &enumMap{schema: schema}
	for _, v := range values {
		m.Values = append(m.Values, enumValue{API: strings.ToUpper(v), Name: v})
	}
	return &crField{API: api, JSON: api, GoName: goName(api), Optional: true, enum: m,
		typ: &model.Type{Kind: model.Enum, Schema: schema, EnumZero: "ZERO"}}
}

func enumListField(api, schema string, values ...string) *crField {
	f := enumField(api, schema, values...)
	f.typ = &model.Type{Kind: model.List, Elem: f.typ}
	return f
}

func objectField(api string, kind model.Kind, t *crType) *crField {
	return &crField{API: api, JSON: api, GoName: goName(api), Optional: true, object: t, typ: &model.Type{Kind: kind}}
}

func listOf(f *crField) *crField {
	f.typ = &model.Type{Kind: model.List, Elem: f.typ}
	return f
}

// coverageSpec is a spec with an enum, a list of enums, a oneOf whose arms hold enums, and a list of
// objects that hold a oneOf.
func coverageSpec() *kindData {
	logs := &crType{schema: "LogsQuery", Fields: []*crField{enumField("severity", "Severity", "d", "e")}}
	spans := &crType{schema: "SpansQuery", Fields: []*crField{enumField("agg", "Agg", "p", "q", "r")}}
	query := &crType{schema: "Query", Fields: []*crField{objectField("logs", model.Object, logs), objectField("spans", model.Object, spans)}}
	query.groups = []model.OneOfGroup{{Arms: []string{"logs", "spans"}}}
	armA := &crType{schema: "RuleA", Fields: []*crField{enumField("kind", "KindA", "k1", "k2")}}
	armB := &crType{schema: "RuleB", Fields: []*crField{enumField("level", "Level", "l1")}}
	rule := &crType{schema: "Rule", Fields: []*crField{objectField("a", model.Object, armA), objectField("b", model.Object, armB)}}
	rule.groups = []model.OneOfGroup{{Arms: []string{"a", "b"}}}
	spec := &crType{Name: "ThingSpec", Fields: []*crField{
		enumField("mode", "Mode", "a", "b", "c"),
		enumListField("tags", "Tag", "x", "y", "z"),
		objectField("query", model.OneOf, query),
		listOf(objectField("rules", model.Object, rule)),
	}}
	return &kindData{Kind: "Thing", Spec: spec}
}

func specsOf(t *testing.T, d *kindData, file *e2eFile) ([][]*omap, *sampleValues, error) {
	t.Helper()
	v := &sampleValues{file: file, used: map[string]bool{}}
	_, _, steps, err := v.specs(d)
	return steps, v, err
}

func stepsJSON(t *testing.T, steps [][]*omap) string {
	t.Helper()
	data, err := json.Marshal(steps)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func wantShapeCode(t *testing.T, err error, code string) {
	t.Helper()
	shapeOf(t, err, code)
}

func shapeOf(t *testing.T, err error, code string) *shapeError {
	t.Helper()
	var shape *shapeError
	if !errors.As(err, &shape) || shape.code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
	return shape
}

func TestSweepReachesEveryValueAndArm(t *testing.T) {
	steps, v, err := specsOf(t, coverageSpec(), &e2eFile{})
	if err != nil {
		t.Fatal(err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("the sweep left %v", missing)
	}
	all := stepsJSON(t, steps)
	// An enum list gets every open value in one list.
	if !strings.Contains(all, `"tags":["y","z"]`) && !strings.Contains(all, `"tags":["x","y","z"]`) {
		t.Errorf("enum list items must go into one list: %s", all)
	}
	// A list of objects takes items while each item adds coverage.
	if !strings.Contains(all, `"rules":[{"b"`) && !strings.Contains(all, `"rules":[{"a"`) {
		t.Errorf("rules missing: %s", all)
	}
	for _, want := range []string{`"spans"`, `"logs"`, `"agg":"r"`, `"level":"l1"`} {
		if !strings.Contains(all, want) && want != `"logs"` {
			t.Errorf("no step sends %s: %s", want, all)
		}
	}
}

func TestSweepArmsDoNotStarve(t *testing.T) {
	// Step 0 sets the first arm (logs). The arm with more open values must not wait behind it.
	steps, v, err := specsOf(t, coverageSpec(), &e2eFile{})
	if err != nil {
		t.Fatal(err)
	}
	if !v.cov.covered[coverageKey{"arm", "Query", "spans"}] || !v.cov.covered[coverageKey{"arm", "Query", "logs"}] {
		t.Fatal("an arm was not covered")
	}
	if len(steps) > 3 {
		t.Errorf("the sweep needs %d steps for 3 enum values under one arm: %s", len(steps), stepsJSON(t, steps))
	}
}

func TestSweepUsesCopiesForARootOneOf(t *testing.T) {
	d := coverageSpec()
	armA := &crType{schema: "RootA", Fields: []*crField{enumField("kind", "RootKindA", "k1")}}
	armB := &crType{schema: "RootB", Fields: []*crField{enumField("kind", "RootKindB", "k2")}}
	d.Spec = &crType{Name: "ThingSpec", Fields: []*crField{objectField("a", model.Object, armA), objectField("b", model.Object, armB)}}
	d.Spec.groups = []model.OneOfGroup{{Arms: []string{"a", "b"}}}
	steps, _, err := specsOf(t, d, &e2eFile{})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 || len(steps[0]) != 1 {
		t.Fatalf("step 0 sets arm a; one copy with arm b covers the rest: %s", stepsJSON(t, steps))
	}
	if !strings.Contains(stepsJSON(t, steps), `"b"`) {
		t.Errorf("the copy must set arm b: %s", stepsJSON(t, steps))
	}
}

func TestFixedValueCoversOnlyWhatItSends(t *testing.T) {
	file := &e2eFile{Values: map[string]any{"query": map[string]any{"spans": map[string]any{"agg": "p"}}, "mode": "a"}}
	_, _, err := specsOf(t, coverageSpec(), file)
	shape := shapeOf(t, err, codeE2ECoverageMissing)
	for _, want := range []string{
		"the oneOf arm logs of query (component Query)",
		"the value Q of query.spans.agg (component Agg)",
		"the value R of query.spans.agg (component Agg)",
		"the value B of mode (component Mode)",
		"the value C of mode (component Mode)",
	} {
		if !strings.Contains(shape.msg, want) {
			t.Errorf("the guard must name %q:\n%s", want, shape.msg)
		}
	}
	for _, covered := range []string{"the oneOf arm spans of query", "the value P of query.spans.agg", "the value A of mode"} {
		if strings.Contains(shape.msg, covered) {
			t.Errorf("the fixed value sends %q: the guard must not name it:\n%s", covered, shape.msg)
		}
	}
}

func TestGuardCountsEnumListItems(t *testing.T) {
	file := &e2eFile{Values: map[string]any{"tags": []any{"x"}}}
	_, _, err := specsOf(t, coverageSpec(), file)
	shape := shapeOf(t, err, codeE2ECoverageMissing)
	if !strings.Contains(shape.msg, "the value Y of tags (component Tag)") || strings.Contains(shape.msg, "the value X of tags") {
		t.Errorf("the guard must name the open list items only:\n%s", shape.msg)
	}
}

func TestSkipsRemoveValuesAndArmsFromTheGuardAndTheSweep(t *testing.T) {
	file := &e2eFile{
		SkipValues:  map[string]e2eSkipValues{"Mode": {Values: []string{"B"}, Reason: "the API rejects it"}},
		SkipChoices: map[string]e2eSkipChoices{"Query": {Choices: []string{"spans"}, Reason: "the team has no span data"}},
	}
	steps, v, err := specsOf(t, coverageSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	all := stepsJSON(t, steps)
	if strings.Contains(all, `"spans"`) || strings.Contains(all, `"mode":"b"`) {
		t.Errorf("a skipped value or arm must not be sent: %s", all)
	}
	if v.cov.covered[coverageKey{"enum", "Mode", "B"}] {
		t.Error("a skip must not count as covered")
	}
	if !strings.Contains(all, `"mode":"c"`) {
		t.Errorf("the other values must still be sent: %s", all)
	}
}

func TestSkipThatNamesNothingIsUnused(t *testing.T) {
	for name, file := range map[string]*e2eFile{
		"enum component": {SkipValues: map[string]e2eSkipValues{"Nope": {Values: []string{"A"}, Reason: "r"}}},
		"enum value":     {SkipValues: map[string]e2eSkipValues{"Mode": {Values: []string{"NOPE"}, Reason: "r"}}},
		"zero value":     {SkipValues: map[string]e2eSkipValues{"Mode": {Values: []string{"ZERO"}, Reason: "r"}}},
		"group":          {SkipChoices: map[string]e2eSkipChoices{"Nope": {Choices: []string{"a"}, Reason: "r"}}},
		"arm":            {SkipChoices: map[string]e2eSkipChoices{"Query": {Choices: []string{"nope"}, Reason: "r"}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := specsOf(t, coverageSpec(), file)
			wantShapeCode(t, err, codeE2ESkipUnused)
		})
	}
}

func TestSkipNeedsAReasonAndAValue(t *testing.T) {
	for name, text := range map[string]string{
		"no reason":  "skipValues:\n  Mode: {values: [A]}\n",
		"no values":  "skipValues:\n  Mode: {reason: r}\n",
		"no choices": "skipChoices:\n  Query: {reason: r}\n",
		"no reason2": "skipChoices:\n  Query: {choices: [spans]}\n",
		"blank":      "skipValues:\n  Mode: {values: [A], reason: ' '}\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), e2eFileName)
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := readE2E(path)
			wantShapeCode(t, err, codeE2EInvalid)
		})
	}
	var file e2eFile
	if err := yaml.Unmarshal([]byte("skipValues:\n  Mode: {values: [A], reason: ok}\n"), &file); err != nil || file.SkipValues["Mode"].Reason != "ok" {
		t.Fatalf("the skip keys must decode: %v", err)
	}
}

func TestOmittedFieldIsNotAnExemption(t *testing.T) {
	file := &e2eFile{Omit: []string{"mode"}}
	_, _, err := specsOf(t, coverageSpec(), file)
	shape := shapeOf(t, err, codeE2ECoverageMissing)
	if !strings.Contains(shape.msg, "the value A of mode (component Mode)") {
		t.Errorf("an omitted field must not hide its values:\n%s", shape.msg)
	}
}

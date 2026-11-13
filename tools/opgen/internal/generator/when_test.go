package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
)

func stringField(api string) *crField {
	return &crField{API: api, JSON: api, GoName: goName(api), Optional: true, typ: &model.Type{Kind: model.String}}
}

// whenSpec has a connector enum with a parent id that each value needs, a oneOf with a source field
// that each arm needs, and a list of aggregations whose meta oneOf depends on the type.
func whenSpec() *kindData {
	samples := &crType{schema: "Samples", Fields: []*crField{enumField("sampleType", "SampleType", "s1", "s2")}}
	histogram := &crType{schema: "Histogram", Fields: []*crField{stringField("buckets")}}
	meta := &crType{schema: "Meta", Fields: []*crField{objectField("samples", model.Object, samples), objectField("histogram", model.Object, histogram)}}
	meta.groups = []model.OneOfGroup{{Arms: []string{"samples", "histogram"}}}
	aggregation := &crType{schema: "Aggregation", Fields: []*crField{enumField("type", "AggType", "min", "max", "hist"), objectField("meta", model.OneOf, meta)}}
	logs := &crType{schema: "LogsQuery", Fields: []*crField{stringField("lucene")}}
	spans := &crType{schema: "SpansQuery", Fields: []*crField{stringField("filter")}}
	query := &crType{schema: "Query", Fields: []*crField{objectField("logs", model.Object, logs), objectField("spans", model.Object, spans)}}
	query.groups = []model.OneOfGroup{{Arms: []string{"logs", "spans"}}}
	spec := &crType{Name: "ThingSpec", Fields: []*crField{
		enumField("kind", "Conn", "slack", "http", "mail"),
		stringField("parent"),
		stringField("source"),
		objectField("query", model.OneOf, query),
		listOf(objectField("aggs", model.Object, aggregation)),
	}}
	return &kindData{Kind: "Thing", Spec: spec}
}

func kindEntries() []e2eWhen {
	return []e2eWhen{
		{Match: map[string]any{"kind": "slack"}, Set: map[string]any{"parent": "p-slack"}},
		{Match: map[string]any{"kind": "http"}, Set: map[string]any{"parent": "p-http"}},
		{Match: map[string]any{"kind": "mail"}, Set: map[string]any{"parent": "p-mail"}},
	}
}

func queryEntries() []e2eWhen {
	return []e2eWhen{
		{Match: map[string]any{"query": "logs"}, Set: map[string]any{"source": "s-logs"}},
		{Match: map[string]any{"query": "spans"}, Set: map[string]any{"source": "s-spans"}},
	}
}

func aggEntries() []e2eWhen {
	return []e2eWhen{
		{Match: map[string]any{"aggs.type": "min"}, Set: map[string]any{"aggs.meta": map[string]any{"samples": map[string]any{"sampleType": "s1"}}}},
		{Match: map[string]any{"aggs.type": "max"}, Set: map[string]any{"aggs.meta": map[string]any{"samples": map[string]any{"sampleType": "s2"}}}},
		{Match: map[string]any{"aggs.type": "hist"}, Set: map[string]any{"aggs.meta": map[string]any{"histogram": map[string]any{"buckets": "1"}}}},
	}
}

// specsAsJSON returns the full spec first, then the copies of each step.
func copiesOf(t *testing.T, steps [][]sweepCopy) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, step := range steps {
		for _, spec := range step {
			data, err := json.Marshal(spec.spec)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatal(err)
			}
			out = append(out, m)
		}
	}
	return out
}

func TestWhenEntriesSetTheirValuesInTheCopyOfTheirMatch(t *testing.T) {
	file := &e2eFile{When: kindEntries()}
	file.Values = map[string]any{"kind": "slack", "parent": "p-slack"} // the full spec: a consistent base
	steps, _, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, copy := range copiesOf(t, steps) {
		kind, parent := copy["kind"], copy["parent"]
		if parent != "p-"+kind.(string) {
			t.Errorf("kind %v needs parent p-%v, got %v", kind, kind, parent)
		}
		seen[kind.(string)] = true
	}
	if !seen["http"] || !seen["mail"] {
		t.Errorf("each value needs a copy with its own entry: %v", seen)
	}
}

func TestWhenEntriesThatSetOnePathLandInDifferentCopies(t *testing.T) {
	file := &e2eFile{When: kindEntries(), Values: map[string]any{"kind": "slack", "parent": "p-slack"}}
	steps, _, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(copiesOf(t, steps)); n < 2 {
		t.Fatalf("http and mail set one path to two values and need two copies, got %d", n)
	}
}

func TestWhenMatchesAnArmAndSetsItsData(t *testing.T) {
	file := &e2eFile{When: queryEntries(), Values: map[string]any{"query": map[string]any{"logs": map[string]any{"lucene": "a"}}, "source": "s-logs"}}
	steps, _, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, copy := range copiesOf(t, steps) {
		query := copy["query"].(map[string]any)
		if _, ok := query["spans"]; ok {
			found = true
			if copy["source"] != "s-spans" {
				t.Errorf("the spans arm needs source s-spans: %v", copy)
			}
		}
	}
	if !found {
		t.Error("no copy sends the spans arm")
	}
}

func TestWhenAppliesPerListItem(t *testing.T) {
	file := &e2eFile{When: aggEntries()}
	steps, v, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("left %v", missing)
	}
	want := map[string]string{"min": "s1", "max": "s2", "hist": ""}
	seen := map[string]bool{}
	for _, copy := range copiesOf(t, steps) {
		items, _ := copy["aggs"].([]any)
		for _, raw := range items {
			item := raw.(map[string]any)
			kind := item["type"].(string)
			meta := item["meta"].(map[string]any)
			if kind == "hist" {
				if _, ok := meta["histogram"]; !ok {
					t.Errorf("hist needs the histogram arm: %v", item)
				}
			} else if meta["samples"].(map[string]any)["sampleType"] != want[kind] {
				t.Errorf("%s needs sampleType %s: %v", kind, want[kind], item)
			}
			seen[kind] = true
		}
	}
	if len(seen) != 3 {
		t.Errorf("each type needs an item: %v", seen)
	}
}

func TestWhenListInMatchCoversEachValue(t *testing.T) {
	entries := []e2eWhen{
		{Match: map[string]any{"aggs.type": []any{"min", "max"}}, Set: map[string]any{"aggs.meta": map[string]any{"samples": map[string]any{"sampleType": "s1"}}}},
		{Match: map[string]any{"aggs.type": "hist"}, Set: map[string]any{"aggs.meta": map[string]any{"histogram": map[string]any{"buckets": "1"}}}},
	}
	_, v, err := specsOf(t, whenSpec(), &e2eFile{When: entries})
	if err != nil {
		t.Fatal(err)
	}
	for _, api := range []string{"MIN", "MAX", "HIST"} {
		if !v.cov.covered[coverageKey{"enum", "AggType", api}] {
			t.Errorf("%s was not sent", api)
		}
	}
}

func TestWhenCoupledPathNeedsAnEntryForEveryValue(t *testing.T) {
	entries := kindEntries()[:1]
	_, _, err := specsOf(t, whenSpec(), &e2eFile{When: entries})
	shape := shapeOf(t, err, codeE2EWhenMissing)
	for _, want := range []string{"the value HTTP of kind", "the value MAIL of kind"} {
		if !strings.Contains(shape.msg, want) {
			t.Errorf("the guard must name %q:\n%s", want, shape.msg)
		}
	}
	if strings.Contains(shape.msg, "the value SLACK") {
		t.Errorf("slack has an entry:\n%s", shape.msg)
	}
	// A skip covers a value that has no entry.
	file := &e2eFile{When: entries, SkipValues: map[string]e2eSkipValues{"Conn": {Values: []string{"HTTP", "MAIL"}, Reason: "no parent"}},
		Values: map[string]any{"kind": "slack", "parent": "p-slack"}}
	if _, _, err := specsOf(t, whenSpec(), file); err != nil {
		t.Fatalf("a skip must cover the values without an entry: %v", err)
	}
}

func TestWhenCoupledArmNeedsAnEntry(t *testing.T) {
	_, _, err := specsOf(t, whenSpec(), &e2eFile{When: queryEntries()[:1]})
	shape := shapeOf(t, err, codeE2EWhenMissing)
	if !strings.Contains(shape.msg, "the oneOf arm spans of query") {
		t.Errorf("the guard must name the arm:\n%s", shape.msg)
	}
}

func TestWhenValueWithoutAnEntryIsNotSweptAtACoupledPath(t *testing.T) {
	// The sweep must not pick an open value at a coupled path: it has no data for it.
	file := &e2eFile{When: kindEntries(), Values: map[string]any{"kind": "slack", "parent": "p-slack"}}
	steps, _, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range copiesOf(t, steps) {
		if copy["parent"] == nil {
			t.Errorf("a copy sends a kind with no parent: %v", copy)
		}
	}
}

func TestWhenEntriesWithAnInvalidShapeFail(t *testing.T) {
	cases := map[string]struct {
		entry e2eWhen
		code  string
	}{
		"empty match":       {e2eWhen{Set: map[string]any{"parent": "p"}}, codeE2EWhenInvalid},
		"unknown path":      {e2eWhen{Match: map[string]any{"nope": "slack"}}, codeE2EWhenUnused},
		"unknown value":     {e2eWhen{Match: map[string]any{"kind": "nope"}}, codeE2EWhenUnused},
		"unknown arm":       {e2eWhen{Match: map[string]any{"query": "nope"}}, codeE2EWhenUnused},
		"not an enum":       {e2eWhen{Match: map[string]any{"parent": "x"}}, codeE2EWhenInvalid},
		"match not string":  {e2eWhen{Match: map[string]any{"kind": 3}}, codeE2EWhenInvalid},
		"unknown set path":  {e2eWhen{Match: map[string]any{"kind": "slack"}, Set: map[string]any{"nope": "x"}}, codeE2EWhenUnused},
		"set repeats match": {e2eWhen{Match: map[string]any{"kind": "slack"}, Set: map[string]any{"kind": "http"}}, codeE2EWhenConflict},
		"set outside item":  {e2eWhen{Match: map[string]any{"aggs.type": "min"}, Set: map[string]any{"parent": "x"}}, codeE2EWhenInvalid},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := specsOf(t, whenSpec(), &e2eFile{When: []e2eWhen{tc.entry}})
			wantShapeCode(t, err, tc.code)
		})
	}
}

// Two entries with different matches can set one path to two values. They must not share a copy.
func TestWhenEntriesWithDifferentMatchesAndOneSetPathDoNotShareACopy(t *testing.T) {
	entries := []e2eWhen{
		{Match: map[string]any{"kind": "http"}, Set: map[string]any{"parent": "p-http"}},
		{Match: map[string]any{"query": "spans"}, Set: map[string]any{"parent": "p-spans"}},
		{Match: map[string]any{"kind": "slack"}, Set: map[string]any{"parent": "p-base"}},
		{Match: map[string]any{"query": "logs"}, Set: map[string]any{"parent": "p-base"}},
	}
	file := &e2eFile{When: entries, Values: map[string]any{"kind": "slack", "parent": "p-base"},
		SkipValues:  map[string]e2eSkipValues{"Conn": {Values: []string{"MAIL"}, Reason: "no parent"}},
		SkipChoices: map[string]e2eSkipChoices{}}
	steps, _, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range copiesOf(t, steps) {
		_, spans := copy["query"].(map[string]any)["spans"]
		if (copy["kind"] == "http" && copy["parent"] != "p-http") || (spans && copy["parent"] != "p-spans") {
			t.Errorf("a copy mixes the data of two entries: %v", copy)
		}
	}
}

func TestWhenDecodesFromE2EYAML(t *testing.T) {
	text := "when:\n  - match: {aggs.type: [min, max], kind: slack}\n    set:\n      parent: p\n      aggs.meta: {samples: {sampleType: s1}}\n"
	path := filepath.Join(t.TempDir(), e2eFileName)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := readE2E(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.When) != 1 || file.When[0].Set["parent"] != "p" {
		t.Fatalf("when did not decode: %+v", file.When)
	}
	if types, ok := file.When[0].Match["aggs.type"].([]any); !ok || len(types) != 2 {
		t.Errorf("a match list must decode as a list: %+v", file.When[0].Match)
	}
}

// A fixed list in values: is the base of the full spec. The items that a copy plans for the same
// list win over it, so a when entry in a list works with a fixed base.
func TestPlannedItemsWinOverAFixedList(t *testing.T) {
	fixed := []any{map[string]any{"type": "min", "meta": map[string]any{"samples": map[string]any{"sampleType": "s1"}}}}
	file := &e2eFile{When: aggEntries(), Values: map[string]any{"aggs": fixed}}
	steps, v, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("left %v", missing)
	}
	planned := false
	for _, copy := range copiesOf(t, steps) {
		if items, _ := copy["aggs"].([]any); len(items) == 2 {
			planned = true // the fixed base already covers min: one item each for max and hist
		}
	}
	if !planned {
		t.Error("no copy has the two planned items: the fixed list hid them")
	}
}

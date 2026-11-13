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

// A list has a contract limit. A copy must not hold more planned items than the limit allows. The
// entries that do not fit wait for another copy: none is dropped.
func TestWhenPlannedItemsRespectTheListLimit(t *testing.T) {
	d := whenSpec()
	one := int64(1)
	for _, f := range d.Spec.Fields {
		if f.JSON == "aggs" {
			f.typ.MaxItems = &one
		}
	}
	steps, v, err := specsOf(t, d, &e2eFile{When: aggEntries()})
	if err != nil {
		t.Fatal(err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("an entry that does not fit must wait, not vanish: %v", missing)
	}
	copies := 0
	for _, copy := range copiesOf(t, steps) {
		if items, _ := copy["aggs"].([]any); len(items) > 1 {
			t.Errorf("a copy holds %d items, the contract allows 1: %v", len(items), copy)
		}
		copies++
	}
	if copies < 2 {
		t.Errorf("the entries for max and hist (the full spec covers min) need their own copies, got %d", copies)
	}
}

// A when entry that selects an arm must stay in play while values below the arm are open: at a
// coupled path the sweep does not pick the arm by itself.
func TestWhenArmIsRevisitedWhileItsValuesAreOpen(t *testing.T) {
	entries := []e2eWhen{
		{Match: map[string]any{"query": "logs"}},
		{Match: map[string]any{"query": "spans"}},
	}
	steps, v, err := specsOf(t, coverageSpec(), &e2eFile{When: entries})
	if err != nil {
		t.Fatalf("every value below a coupled arm must be reached: %v", err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("left %v", missing)
	}
	all := stepsJSON(t, steps)
	for _, want := range []string{`"agg":"q"`, `"agg":"r"`} {
		if !strings.Contains(all, want) {
			t.Errorf("no copy sends %s: %s", want, all)
		}
	}
}

// One entry can list several arms. The sweep must come back to an arm of the list while values
// below it are open, not fall back to the first arm.
func TestWhenListOfArmsIsWorkedOffUntilEveryValueIsSent(t *testing.T) {
	entries := []e2eWhen{{Match: map[string]any{"query": []any{"logs", "spans"}}}}
	steps, v, err := specsOf(t, coverageSpec(), &e2eFile{When: entries})
	if err != nil {
		t.Fatalf("every value below every listed arm must be reached: %v", err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("left %v", missing)
	}
	all := stepsJSON(t, steps)
	for _, want := range []string{`"agg":"q"`, `"agg":"r"`, `"severity":"e"`} {
		if !strings.Contains(all, want) {
			t.Errorf("no copy sends %s: %s", want, all)
		}
	}
}

// An entry that sets a whole object and an entry that matches an arm at the same path must not
// share a copy: the literal would win, and the data of the arm entry would apply to the wrong arm.
func TestWhenLiteralObjectAndForcedArmDoNotShareACopy(t *testing.T) {
	entries := append(queryEntries(),
		e2eWhen{Match: map[string]any{"kind": "http"}, Set: map[string]any{"query": map[string]any{"logs": map[string]any{"lucene": "http-only"}}}},
		e2eWhen{Match: map[string]any{"kind": "mail"}},
	)
	// The full spec uses mail, so the http entry is open and the spans entry is open too.
	file := &e2eFile{When: entries, Values: map[string]any{"kind": "mail"},
		SkipValues: map[string]e2eSkipValues{"Conn": {Values: []string{"SLACK"}, Reason: "no data"}}}
	steps, _, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range copiesOf(t, steps) {
		_, spans := copy["query"].(map[string]any)["spans"]
		if !spans && copy["source"] == "s-spans" {
			t.Errorf("the data of the spans entry is applied to a copy without the spans arm: %v", copy)
		}
	}
}

func TestOverlaysConflictOnASharedPathOrAParentPath(t *testing.T) {
	mk := func(values map[string]any, arms map[string]string) *overlay {
		o := newOverlay()
		for k, v := range values {
			o.values[k] = v
		}
		for k, a := range arms {
			o.arms[k] = a
		}
		return o
	}
	for name, tc := range map[string]struct {
		a, b *overlay
		want bool
	}{
		"same value":                  {mk(map[string]any{"p": "x"}, nil), mk(map[string]any{"p": "x"}, nil), false},
		"other value":                 {mk(map[string]any{"p": "x"}, nil), mk(map[string]any{"p": "y"}, nil), true},
		"same arm":                    {mk(nil, map[string]string{"q": "a"}), mk(nil, map[string]string{"q": "a"}), false},
		"other arm":                   {mk(nil, map[string]string{"q": "a"}), mk(nil, map[string]string{"q": "b"}), true},
		"value and arm, one path":     {mk(map[string]any{"q": map[string]any{}}, nil), mk(nil, map[string]string{"q": "a"}), true},
		"path and a child":            {mk(map[string]any{"q": map[string]any{}}, nil), mk(map[string]any{"q.a.x": 1}, nil), true},
		"different paths":             {mk(map[string]any{"p": 1}, nil), mk(map[string]any{"q": 1}, nil), false},
		"a name that starts the same": {mk(map[string]any{"q": 1}, nil), mk(map[string]any{"query": 1}, nil), false},
	} {
		if got := tc.a.conflicts(tc.b); got != tc.want {
			t.Errorf("%s: conflicts = %v, want %v", name, got, tc.want)
		}
	}
}

// An entry that matches a root field and a list-item field is global. The overlay of a list item
// that sets the same item field differently would drop its match but keep its data: they must not
// share a copy.
func TestWhenGlobalEntryAndListItemEntriesDoNotShareACopyWhenTheyDisagree(t *testing.T) {
	entries := []e2eWhen{
		{Match: map[string]any{"kind": "http", "aggs.type": "min"}, Set: map[string]any{"parent": "p-http"}},
		{Match: map[string]any{"kind": "mail"}},
		aggEntries()[1], // max
		aggEntries()[2], // hist
	}
	file := &e2eFile{When: entries, Values: map[string]any{"kind": "mail"},
		SkipValues: map[string]e2eSkipValues{"Conn": {Values: []string{"SLACK"}, Reason: "no data"}}}
	steps, _, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range copiesOf(t, steps) {
		if copy["parent"] != "p-http" {
			continue
		}
		found := false
		items, _ := copy["aggs"].([]any)
		for _, raw := range items {
			if raw.(map[string]any)["type"] == "min" {
				found = true
			}
		}
		if copy["kind"] != "http" || !found {
			t.Errorf("a copy with the data of the http-and-min entry must match it: %v", copy)
		}
	}
}

// A fixed object in values: is the base. A when entry that sets a field inside it must reach the
// copy: the value of the entry replaces the value of the base.
func TestWhenSetsAFieldInsideAFixedObject(t *testing.T) {
	entries := []e2eWhen{
		{Match: map[string]any{"kind": "http"}, Set: map[string]any{"query.logs.lucene": "http-only"}},
		{Match: map[string]any{"kind": "mail"}},
	}
	file := &e2eFile{When: entries,
		Values:      map[string]any{"kind": "mail", "query": map[string]any{"logs": map[string]any{"lucene": "base"}}},
		SkipValues:  map[string]e2eSkipValues{"Conn": {Values: []string{"SLACK"}, Reason: "no data"}},
		SkipChoices: map[string]e2eSkipChoices{"Query": {Choices: []string{"spans"}, Reason: "the fixed query is logs"}}}
	steps, _, err := specsOf(t, whenSpec(), file)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, copy := range copiesOf(t, steps) {
		if copy["kind"] != "http" {
			continue
		}
		found = true
		logs, _ := copy["query"].(map[string]any)["logs"].(map[string]any)
		if logs["lucene"] != "http-only" {
			t.Errorf("the entry sets lucene inside the fixed query, got %v", copy["query"])
		}
	}
	if !found {
		t.Fatal("no copy sends http")
	}
	// The base value in the file is not changed by a copy.
	if file.Values["query"].(map[string]any)["logs"].(map[string]any)["lucene"] != "base" {
		t.Error("a copy must not change the fixed value of the file")
	}
}

// An entry that matches an arm and entries for the values below that arm belong in one copy: the
// arm selects the object, and the value fills it. They are no conflict.
func TestWhenArmAndEntriesForItsValuesCoexist(t *testing.T) {
	entries := append(queryEntries()[:0:0],
		e2eWhen{Match: map[string]any{"query": "logs"}},
		e2eWhen{Match: map[string]any{"query": "spans"}},
		e2eWhen{Match: map[string]any{"query.spans.agg": "p"}},
		e2eWhen{Match: map[string]any{"query.spans.agg": "q"}},
		e2eWhen{Match: map[string]any{"query.spans.agg": "r"}},
	)
	steps, v, err := specsOf(t, coverageSpec(), &e2eFile{When: entries})
	if err != nil {
		t.Fatalf("the arm and the values below it must be reachable: %v", err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("left %v", missing)
	}
	all := stepsJSON(t, steps)
	for _, want := range []string{`"agg":"q"`, `"agg":"r"`} {
		if !strings.Contains(all, want) {
			t.Errorf("no copy sends %s: %s", want, all)
		}
	}
}

func TestOverlaysOfAnArmAndOfAnotherArmsFieldConflict(t *testing.T) {
	arm := newOverlay()
	arm.arms["query"], arm.armNames["query"] = "spansQuery", "spans"
	same := newOverlay()
	same.values["query.spans.agg"] = "q"
	other := newOverlay()
	other.values["query.logs.severity"] = "e"
	if arm.conflicts(same) || same.conflicts(arm) {
		t.Error("a field below the selected arm belongs with it")
	}
	if !arm.conflicts(other) || !other.conflicts(arm) {
		t.Error("a field below another arm has no place next to the selected arm")
	}
}

// The contract can ask for more items than the selected entries give. The copy still needs them.
func TestWhenPlannedItemsRespectTheListMinimum(t *testing.T) {
	d := whenSpec()
	two := int64(2)
	for _, f := range d.Spec.Fields {
		if f.JSON == "aggs" {
			f.typ.MinItems = &two
		}
	}
	steps, _, err := specsOf(t, d, &e2eFile{When: aggEntries()})
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range copiesOf(t, steps) {
		if items, _ := copy["aggs"].([]any); len(items) < 2 {
			t.Errorf("a copy holds %d items, the contract needs 2: %v", len(items), copy)
		}
	}
}

// A coupled field below a skipped arm needs no entry for each value: nothing can send them.
func TestWhenCoupledValuesBelowASkippedArmNeedNoEntries(t *testing.T) {
	entries := []e2eWhen{
		{Match: map[string]any{"query": "logs"}},
		{Match: map[string]any{"query.spans.agg": "p"}},
	}
	file := &e2eFile{When: entries, SkipChoices: map[string]e2eSkipChoices{"Query": {Choices: []string{"spans"}, Reason: "no span data"}}}
	if _, _, err := specsOf(t, coverageSpec(), file); err != nil {
		t.Fatalf("the values below a skipped arm need no entry or skip: %v", err)
	}
}

func pairSpec() *kindData {
	item := &crType{schema: "Pair", Fields: []*crField{enumField("kind", "PairKind", "a", "b"), stringField("data")}}
	return &kindData{Kind: "Thing", Spec: &crType{Name: "ThingSpec", Fields: []*crField{listOf(objectField("items", model.Object, item))}}}
}

func pairEntries() []e2eWhen {
	return []e2eWhen{
		{Match: map[string]any{"items.kind": "a"}, Set: map[string]any{"items.data": "data-a"}},
		{Match: map[string]any{"items.kind": "b"}, Set: map[string]any{"items.data": "data-b"}},
	}
}

// Padding a list to its minimum must keep the coupling: a padded item repeats a finished item
// and does not pick a coupled value without its data.
func TestWhenPaddedItemsKeepTheirCoupling(t *testing.T) {
	d := pairSpec()
	two := int64(2)
	d.Spec.Fields[0].typ.MinItems = &two
	base := []any{map[string]any{"kind": "a", "data": "data-a"}, map[string]any{"kind": "a", "data": "data-a"}}
	steps, _, err := specsOf(t, d, &e2eFile{When: pairEntries(), Values: map[string]any{"items": base}})
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range copiesOf(t, steps) {
		items, _ := copy["items"].([]any)
		if len(items) < 2 {
			t.Errorf("a copy holds %d items, the contract needs 2: %v", len(items), copy)
		}
		for _, raw := range items {
			item := raw.(map[string]any)
			if item["data"] != "data-"+item["kind"].(string) {
				t.Errorf("an item breaks the coupling of its kind: %v", item)
			}
		}
	}
}

// A list of enums below its minimum takes values that it does not hold yet: a set has no duplicates.
func TestPaddedEnumListHasNoDuplicates(t *testing.T) {
	d := coverageSpec()
	three := int64(3)
	for _, f := range d.Spec.Fields {
		if f.JSON == "tags" {
			f.typ.MinItems = &three
		}
	}
	steps, v, err := specsOf(t, d, &e2eFile{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stepsJSON(t, steps), `"tags":["`) {
		t.Fatal("no copy sends tags")
	}
	full, _, _, err := (&sampleValues{file: &e2eFile{}, used: map[string]bool{}}).specs(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range append([]*omap{full}, func() []*omap {
		var all []*omap
		for _, step := range steps {
			for _, c := range step {
				all = append(all, c.spec)
			}
		}
		return all
	}()...) {
		tags, _ := spec.get("tags").([]any)
		seen := map[any]bool{}
		for _, tag := range tags {
			if seen[tag] {
				t.Errorf("duplicate tag %v in %v", tag, tags)
			}
			seen[tag] = true
		}
		if len(tags) < 3 {
			t.Errorf("tags needs 3 items: %v", tags)
		}
	}
	_ = v
}

// A map can need more than one entry.
func TestMapIsPaddedToItsMinimumEntries(t *testing.T) {
	two := int64(2)
	labels := &crField{API: "labels", JSON: "labels", GoName: "Labels", Optional: true,
		typ: &model.Type{Kind: model.Map, Elem: &model.Type{Kind: model.String}, MinItems: &two}}
	d := &kindData{Kind: "Thing", Spec: &crType{Name: "ThingSpec", Fields: []*crField{labels}}}
	v := &sampleValues{file: &e2eFile{}, used: map[string]bool{}}
	full, _, _, err := v.specs(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := full.json(); !strings.Contains(got, `"key"`) || !strings.Contains(got, `"key2"`) {
		t.Errorf("the map needs two entries: %s", got)
	}
}

// A fixed object above a list hides the items that entries plan below it: the object must come
// from the type tree, or the data of the entries is lost.
func TestFixedAncestorDoesNotHidePlannedItemsBelowIt(t *testing.T) {
	d := whenSpec()
	var aggs *crField
	kept := d.Spec.Fields[:0:0]
	for _, f := range d.Spec.Fields {
		if f.JSON == "aggs" {
			aggs = f
			continue
		}
		kept = append(kept, f)
	}
	wrapper := &crType{schema: "Wrapper", Fields: []*crField{aggs}}
	d.Spec.Fields = append(kept, objectField("wrapper", model.Object, wrapper))
	entries := []e2eWhen{
		{Match: map[string]any{"wrapper.aggs.type": "min"}, Set: map[string]any{"wrapper.aggs.meta": map[string]any{"samples": map[string]any{"sampleType": "s1"}}}},
		{Match: map[string]any{"wrapper.aggs.type": "max"}, Set: map[string]any{"wrapper.aggs.meta": map[string]any{"samples": map[string]any{"sampleType": "s2"}}}},
		{Match: map[string]any{"wrapper.aggs.type": "hist"}, Set: map[string]any{"wrapper.aggs.meta": map[string]any{"histogram": map[string]any{"buckets": "1"}}}},
		{Match: map[string]any{"kind": "slack"}}, {Match: map[string]any{"kind": "http"}}, {Match: map[string]any{"kind": "mail"}},
	}
	fixed := map[string]any{"aggs": []any{map[string]any{"type": "min", "meta": map[string]any{"samples": map[string]any{"sampleType": "s1"}}}}}
	steps, v, err := specsOf(t, d, &e2eFile{When: entries, Values: map[string]any{"wrapper": fixed}})
	if err != nil {
		t.Fatalf("the entries below a fixed object must still apply: %v", err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("left %v", missing)
	}
	if !strings.Contains(stepsJSON(t, steps), `"histogram"`) {
		t.Error("the histogram item of the entry never reached a copy")
	}
}

// An item of an outer list holds an inner list. The overlay of an inner item wins over the overlay
// of the outer item at a shared path: they must not share a copy when they disagree.
func TestNestedListItemEntriesThatDisagreeDoNotShareACopy(t *testing.T) {
	inner := &crType{schema: "Inner", Fields: []*crField{enumField("t", "InnerType", "t1", "t2")}}
	outer := &crType{schema: "Outer", Fields: []*crField{enumField("k", "OuterKind", "k1", "k2"), listOf(objectField("inner", model.Object, inner))}}
	d := &kindData{Kind: "Thing", Spec: &crType{Name: "ThingSpec", Fields: []*crField{listOf(objectField("outer", model.Object, outer))}}}
	entries := []e2eWhen{
		{Match: map[string]any{"outer.k": "k1"}, Set: map[string]any{"outer.inner.t": "t1"}},
		{Match: map[string]any{"outer.k": "k2"}},
		{Match: map[string]any{"outer.inner.t": "t1"}},
		{Match: map[string]any{"outer.inner.t": "t2"}},
	}
	// The full spec uses k2, so the entry for k1 and the entry for t2 are both open.
	steps, _, err := specsOf(t, d, &e2eFile{When: entries, Values: map[string]any{"outer.k": "k2"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range copiesOf(t, steps) {
		for _, raw := range copy["outer"].([]any) {
			o := raw.(map[string]any)
			if o["k"] != "k1" {
				continue
			}
			for _, in := range o["inner"].([]any) {
				if in.(map[string]any)["t"] != "t1" {
					t.Errorf("an outer k1 item holds an inner item that its entry did not allow: %v", o)
				}
			}
		}
	}
}

// A fixed object keeps the parts that entries do not touch. Only the planned list, or the forced
// arm, is rebuilt: a sibling that names a real server-side object must stay what the file says.
func TestFixedObjectKeepsItsSiblingsWhenAListBelowItIsPlanned(t *testing.T) {
	d := pairSpec()
	items := d.Spec.Fields[0]
	wrapper := &crType{schema: "Wrapper", Fields: []*crField{stringField("resourceId"), items}}
	d.Spec.Fields = []*crField{objectField("wrapper", model.Object, wrapper)}
	entries := []e2eWhen{
		{Match: map[string]any{"wrapper.items.kind": "a"}, Set: map[string]any{"wrapper.items.data": "data-a"}},
		{Match: map[string]any{"wrapper.items.kind": "b"}, Set: map[string]any{"wrapper.items.data": "data-b"}},
	}
	fixed := map[string]any{"resourceId": "real-id", "items": []any{map[string]any{"kind": "a", "data": "data-a"}}}
	steps, _, err := specsOf(t, d, &e2eFile{When: entries, Values: map[string]any{"wrapper": fixed}})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, copy := range copiesOf(t, steps) {
		n++
		if got := copy["wrapper"].(map[string]any)["resourceId"]; got != "real-id" {
			t.Errorf("the fixed sibling was rebuilt: %v", copy)
		}
	}
	if n == 0 {
		t.Fatal("arm b needs a copy")
	}
}

func TestFixedObjectKeepsItsSiblingsWhenAnArmIsForced(t *testing.T) {
	conf := &crType{schema: "Conf", Fields: []*crField{stringField("id"),
		objectField("x", model.Object, &crType{schema: "X", Fields: []*crField{stringField("note")}}),
		objectField("y", model.Object, &crType{schema: "Y", Fields: []*crField{stringField("note")}})}}
	conf.groups = []model.OneOfGroup{{Arms: []string{"x", "y"}}}
	d := &kindData{Kind: "Thing", Spec: &crType{Name: "ThingSpec", Fields: []*crField{objectField("conf", model.OneOf, conf)}}}
	entries := []e2eWhen{{Match: map[string]any{"conf": "x"}}, {Match: map[string]any{"conf": "y"}}}
	fixed := map[string]any{"id": "real-id", "x": map[string]any{"note": "base"}}
	steps, _, err := specsOf(t, d, &e2eFile{When: entries, Values: map[string]any{"conf": fixed}})
	if err != nil {
		t.Fatal(err)
	}
	sawY := false
	for _, copy := range copiesOf(t, steps) {
		c := copy["conf"].(map[string]any)
		if _, y := c["y"]; y {
			sawY = true
		}
		if c["id"] != "real-id" {
			t.Errorf("the fixed sibling of a forced arm was rebuilt: %v", copy)
		}
	}
	if !sawY {
		t.Fatal("arm y needs a copy")
	}
}

func listMapSpec(key string) *kindData {
	item := &crType{schema: "Rule", Fields: []*crField{stringField("id"), enumField("mode", "RuleMode", "a", "b", "c")}}
	list := listOf(objectField("rules", model.Object, item))
	list.Markers = []string{"+listType=map", "+listMapKey=" + key}
	return &kindData{Kind: "Thing", Spec: &crType{Name: "ThingSpec", Fields: []*crField{list}}}
}

// The items of a list-map need distinct keys. The extra items that the sweep adds, and the items
// that entries plan, must not repeat a key: Kubernetes rejects the list.
func TestListMapItemsGetDistinctKeys(t *testing.T) {
	steps, _, err := specsOf(t, listMapSpec("id"), &e2eFile{})
	if err != nil {
		t.Fatal(err)
	}
	multi := false
	for _, copy := range copiesOf(t, steps) {
		seen := map[any]bool{}
		items := copy["rules"].([]any)
		multi = multi || len(items) > 1
		for _, raw := range items {
			id := raw.(map[string]any)["id"]
			if seen[id] {
				t.Errorf("the list-map has the key %v twice: %v", id, items)
			}
			seen[id] = true
		}
	}
	if !multi {
		t.Error("the sweep should put several modes into one list")
	}
}

func TestListMapItemsOfWhenEntriesGetDistinctKeys(t *testing.T) {
	entries := []e2eWhen{
		{Match: map[string]any{"rules.mode": "a"}, Set: map[string]any{"rules.id": "same"}},
		{Match: map[string]any{"rules.mode": "b"}, Set: map[string]any{"rules.id": "same"}},
		{Match: map[string]any{"rules.mode": "c"}, Set: map[string]any{"rules.id": "same"}},
	}
	steps, v, err := specsOf(t, listMapSpec("id"), &e2eFile{When: entries})
	if err != nil {
		t.Fatal(err)
	}
	if missing := v.cov.missing(); len(missing) != 0 {
		t.Fatalf("left %v", missing)
	}
	for _, copy := range copiesOf(t, steps) {
		seen := map[any]bool{}
		for _, raw := range copy["rules"].([]any) {
			id := raw.(map[string]any)["id"]
			if seen[id] {
				t.Errorf("the planned items repeat the key %v: %v", id, copy["rules"])
			}
			seen[id] = true
		}
	}
}

// A key that cannot be changed, an enum, drops the item that repeats it, and what the item would
// have recorded. The list stays valid.
func TestListMapEnumKeyDropsARepeatedItem(t *testing.T) {
	steps, _, err := specsOf(t, listMapSpec("mode"), &e2eFile{})
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range copiesOf(t, steps) {
		seen := map[any]bool{}
		for _, raw := range copy["rules"].([]any) {
			mode := raw.(map[string]any)["mode"]
			if seen[mode] {
				t.Errorf("the list-map has the mode %v twice: %v", mode, copy["rules"])
			}
			seen[mode] = true
		}
	}
}

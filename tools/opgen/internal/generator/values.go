package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"io"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"go.yaml.in/yaml/v4"
)

// e2eFileName is the optional file of a kind with the values that the contract cannot state.
const e2eFileName = "e2e.yaml"

// Issue codes of the values of the sample and the e2e test.
const (
	codeE2EInvalid  = "E2E_INVALID"
	codeE2ERequired = "E2E_VALUE_REQUIRED"
	codeE2EUnused   = "E2E_VALUE_UNUSED"
)

// e2eFile is the content of e2e.yaml. It is handwritten and optional.
type e2eFile struct {
	// Serial keeps a kind whose resources affect the whole account in one Ginkgo process.
	Serial bool `yaml:"serial"`
	// Fixtures are resources that the generated test creates before this kind.
	Fixtures []e2eFixture `yaml:"fixtures"`
	// Minimal adds fields that the API needs although the CRD marks them optional.
	Minimal map[string]any `yaml:"minimal"`
	// Substitutions replace sample values with run-specific values in generated tests.
	Substitutions map[string]string `yaml:"substitutions"`
	// Upgrade freezes the spec accepted by a released chart.
	Upgrade *e2eUpgrade `yaml:"upgrade"`
	// Omit lists optional top-level fields that cannot appear with other full-spec fields.
	Omit []string `yaml:"omit"`
	// Values are spec values by field path, for example config.http.endpoint. A value replaces the
	// whole value that the generator makes up for that field. The path of a field inside a list item
	// or a map value has no index: the value is used for every item.
	Values map[string]any `yaml:"values"`
	// Update is the change of the update step.
	Update *e2eUpdate `yaml:"update"`
	// When gives the extra values that an enum value or a oneOf arm needs.
	When []e2eWhen `yaml:"when"`
	// SkipValues lists accepted enum values that no test sends, by enum component, with the reason.
	SkipValues map[string]e2eSkipValues `yaml:"skipValues"`
	// SkipChoices lists oneOf arms that no test sends, by component, with the reason.
	SkipChoices map[string]e2eSkipChoices `yaml:"skipChoices"`
}

// e2eSkipValues skips API values of one enum component. A skipped value is still in the CRD: a skip
// means "not tested", and rejected means "not accepted".
type e2eSkipValues struct {
	Values []string `yaml:"values"`
	Reason string   `yaml:"reason"`
}

// e2eSkipChoices skips API properties that are arms of one oneOf component.
type e2eSkipChoices struct {
	Choices []string `yaml:"choices"`
	Reason  string   `yaml:"reason"`
}

type e2eFixture struct {
	Kind        string         `yaml:"kind"`
	Suffix      string         `yaml:"suffix"`
	StatusField string         `yaml:"statusField"`
	Spec        map[string]any `yaml:"spec"`
}

type e2eUpgrade struct {
	From  string         `yaml:"from"`
	Spec  map[string]any `yaml:"spec"`
	Field string         `yaml:"field"`
	Value string         `yaml:"value"`
}

// e2eUpdate changes one scalar field that Update sends. A field in a root group uses a dotted path.
type e2eUpdate struct {
	Field string `yaml:"field"`
	Value any    `yaml:"value"`
}

func readE2E(path string) (*e2eFile, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &e2eFile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var f e2eFile
	if err := dec.Decode(&f); err != nil && !errors.Is(err, io.EOF) {
		return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName, msg: err.Error()}
	}
	if f.Upgrade != nil && (f.Upgrade.From == "" || f.Upgrade.Field == "" || f.Upgrade.Value == "" || len(f.Upgrade.Spec) == 0) {
		return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":upgrade", msg: "from, spec, field, and value are required"}
	}
	for _, component := range sortedKeys(f.SkipValues) {
		if skip := f.SkipValues[component]; len(skip.Values) == 0 || strings.TrimSpace(skip.Reason) == "" {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":skipValues." + component, msg: "values and reason are required"}
		}
	}
	for _, component := range sortedKeys(f.SkipChoices) {
		if skip := f.SkipChoices[component]; len(skip.Choices) == 0 || strings.TrimSpace(skip.Reason) == "" {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":skipChoices." + component, msg: "choices and reason are required"}
		}
	}
	return &f, nil
}

// e2eData is what the e2e test template needs.
type e2eData struct {
	Omit                  []string
	Serial                bool
	Fixtures              []fixtureData
	Substitutions         []substitutionData
	Upgrade               *upgradeData
	FullJSON, MinimalJSON string // Go string literals of the spec as JSON
	NameField             string // the Go path of a string name, "" when there is none
	NamePtr               bool
	Update                *updateData
	// FullKeys are the enum values and the oneOf arms that the full spec sends.
	FullKeys []coverageKey
	// CoverageSpecs are the specs that the offline coverage test checks: the full spec, the minimal
	// spec, and every copy of every step.
	CoverageSpecs []coverageSpecData
	// StepJSON are the specs of each coverage step as Go string literals, for the live coverage suite.
	StepJSON [][]string
	// Steps are the coverage steps after the full spec: the copies of the spec that each step creates.
	Steps [][]sweepCopy
}

// coverageSpecData is one spec of the offline coverage test. Expect holds Go string literals.
type coverageSpecData struct {
	Name, JSON string
	Expect     []string
}

type fixtureData struct {
	Kind, Suffix, StatusField, SpecJSON string
}

type substitutionData struct {
	From, To string
}

type upgradeData struct {
	From, SpecJSON, Field, Value string
}

// updateData is the update step as Go code: the field that it changes, the value that it sets,
// the value that the API then returns, and the expression that reads it from the response "got".
type updateData struct {
	Field, Set, Want, Got string
	// NotNil lists values that the check reads through: the nested object of a createAs field.
	NotNil []string
}

type updateCandidate struct {
	field            *crField
	jsonPath, goPath string
	value            any
	inOneOf          bool
}

// sampleValues makes up the full and the minimal spec of the sample and the e2e test.
type sampleValues struct {
	file *e2eFile
	used map[string]bool
	// cov holds the skips and the enum values and oneOf arms that the full spec sends.
	cov *coverage
	// record is true while the full spec is made: it records what it sends in cov.
	record bool
	// virtual holds the parts of a fixed object that stay fixed when the object is rebuilt from the type tree.
	virtual map[string]any
	// fullKeys are the enum values and oneOf arms of the full spec.
	fullKeys []coverageKey
	// sweeping picks open arms and values instead of the first ones.
	sweeping bool
	// layers hold the values and arms that when entries force in the current copy and list item.
	// The last layer wins. items are the overlays of the list items that a copy plans, by list path.
	layers []*overlay
	items  map[string][]*overlay
}

// buildValues makes up the full and the minimal spec from the contract and e2e.yaml.
func buildValues(d *kindData, ix *refIndex, file *e2eFile) (full, minimal *omap, e2e *e2eData, err error) {
	v := &sampleValues{file: file, used: map[string]bool{}}
	for _, name := range file.Omit {
		found := false
		for _, field := range d.Spec.Fields {
			if field.JSON == name && field.Optional {
				found = true
				break
			}
		}
		if !found {
			return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":omit", msg: "omit must name an optional top-level spec field"}
		}
	}
	var steps [][]sweepCopy
	if full, minimal, steps, err = v.specs(d); err != nil {
		return nil, nil, nil, err
	}
	for _, name := range sortedKeys(file.Minimal) {
		value := file.Minimal[name]
		known := false
		for _, field := range d.Spec.Fields {
			if field.JSON == name && field.Optional {
				known = true
			}
		}
		if !known {
			return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":minimal." + name, msg: "not an optional spec field"}
		}
		minimal.set(name, value)
	}
	for _, path := range sortedKeys(file.Values) {
		if !v.used[path] {
			return nil, nil, nil, &shapeError{code: codeE2EUnused, path: e2eFileName + ":values." + path, msg: "the spec has no such field"}
		}
	}
	e2e = &e2eData{FullJSON: goString(full.json()), MinimalJSON: goString(minimal.json()), Serial: file.Serial, Omit: file.Omit, Steps: steps, FullKeys: v.fullKeys}
	e2e.CoverageSpecs = v.coverageSpecs(full, minimal, steps)
	for _, step := range steps {
		var specs []string
		for _, c := range step {
			specs = append(specs, goString(c.spec.json()))
		}
		e2e.StepJSON = append(e2e.StepJSON, specs)
	}
	for _, fixture := range file.Fixtures {
		if !token.IsIdentifier(fixture.Kind) || fixture.Suffix == "" || !token.IsIdentifier(fixture.StatusField) || len(fixture.Spec) == 0 {
			return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":fixtures", msg: "kind, suffix, statusField, and spec are required"}
		}
		encoded, err := json.Marshal(fixture.Spec)
		if err != nil {
			return nil, nil, nil, err
		}
		e2e.Fixtures = append(e2e.Fixtures, fixtureData{fixture.Kind, fixture.Suffix, fixture.StatusField, goString(string(encoded))})
	}
	for _, from := range sortedKeys(file.Substitutions) {
		e2e.Substitutions = append(e2e.Substitutions, substitutionData{from, file.Substitutions[from]})
	}
	if file.Upgrade != nil {
		for name := range file.Upgrade.Spec {
			known := false
			for _, field := range d.Spec.Fields {
				if field.JSON == name {
					known = true
				}
			}
			if !known {
				return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":upgrade.spec." + name, msg: "not a spec field"}
			}
		}
		encoded, err := json.MarshalIndent(file.Upgrade.Spec, "", "  ")
		if err != nil {
			return nil, nil, nil, err
		}
		e2e.Upgrade = &upgradeData{file.Upgrade.From, string(encoded), file.Upgrade.Field, file.Upgrade.Value}
	}
	candidates := updateCandidates(d.Spec, full)
	for _, candidate := range candidates {
		if candidate.field.API == "name" && candidate.field.typ.Kind == model.String {
			e2e.NameField, e2e.NamePtr = candidate.goPath, candidate.field.Optional
			break
		}
	}
	if e2e.Update, err = updateStep(d, ix, candidates, file, e2e.NameField); err != nil {
		return nil, nil, nil, err
	}
	return full, minimal, e2e, nil
}

// specs makes up the full spec, the minimal spec, and the coverage steps. It fails when an accepted
// enum value or oneOf arm is neither sent nor skipped.
func (v *sampleValues) specs(d *kindData) (full, minimal *omap, steps [][]sweepCopy, err error) {
	if v.cov, err = newCoverage(d, v.file); err != nil {
		return nil, nil, nil, err
	}
	v.record = true
	v.cov.startSpec()
	if full, err = v.object(d.Spec, "", true); err != nil {
		return nil, nil, nil, err
	}
	v.fullKeys = v.cov.finishSpec()
	v.record = false
	if minimal, err = v.object(d.Spec, "", false); err != nil {
		return nil, nil, nil, err
	}
	v.record, v.sweeping = true, true
	if steps, err = v.cov.sweep(v, d); err != nil {
		return nil, nil, nil, err
	}
	v.record, v.sweeping = false, false
	if err := v.cov.guard(); err != nil {
		return nil, nil, nil, err
	}
	return full, minimal, steps, nil
}

// coverageSpecs lists the specs of the offline coverage test, with what each one must send.
func (v *sampleValues) coverageSpecs(full, minimal *omap, steps [][]sweepCopy) []coverageSpecData {
	specs := []coverageSpecData{
		{Name: "full spec", JSON: goString(full.json()), Expect: v.cov.expectations(v.fullKeys)},
		{Name: "minimal spec", JSON: goString(minimal.json())},
	}
	for i, step := range steps {
		for j, c := range step {
			specs = append(specs, coverageSpecData{
				Name: fmt.Sprintf("step %d copy %d", i+1, j+1), JSON: goString(c.spec.json()), Expect: v.cov.expectations(c.keys),
			})
		}
	}
	return specs
}

// object makes up the value of a struct: every field when full is true, only the required fields
// otherwise. Of each oneOf group, it sets the first arm, when the group needs one or full is true.
func (v *sampleValues) object(t *crType, prefix string, full bool) (*omap, error) {
	skip := map[string]bool{}
	selected := map[string]bool{}
	for _, g := range t.groups {
		chosen := v.arm(t, g, full, strings.TrimSuffix(prefix, "."))
		for _, arm := range g.Arms {
			skip[arm] = arm != chosen
			selected[arm] = arm == chosen
		}
	}
	out := &omap{}
	for _, f := range t.Fields {
		if f.synthetic {
			if _, ok := v.file.Values[prefix+f.JSON]; !ok {
				continue
			}
		}
		if f.refForm == "split" {
			if _, ok := v.file.Values[prefix+f.refSibling]; ok {
				continue
			}
		}
		_, inGroup := skip[f.API]
		if skip[f.API] || (full && prefix == "" && slices.Contains(v.file.Omit, f.JSON)) || (!full && f.Optional && !inGroup) {
			continue
		}
		value, err := v.field(f, prefix+f.JSON, full)
		if err != nil {
			return nil, err
		}
		out.set(f.JSON, value)
		if v.record && selected[f.API] {
			// The arm counts only when its field is in the spec: omit and the full-versus-minimal rule can drop it.
			v.cov.mark(coverageKey{"arm", v.cov.owner[t], f.API})
		}
	}
	return out, nil
}

func (v *sampleValues) field(f *crField, path string, full bool) (any, error) {
	if value, ok := v.layerValue(path); ok {
		if v.record {
			v.cov.literal(f, value)
		}
		return value, nil
	}
	_, armForced := v.layerArm(path)
	planned := len(v.items[path]) != 0
	if value, ok := v.fixed(path); ok {
		v.used[path] = true
		if !armForced && !planned && !v.armForcedBelow(path) {
			value = v.withDescendantValues(path, value)
			if v.record {
				v.cov.literal(f, value)
			}
			return value, nil
		}
		// A forced arm or planned items win over the fixed value. The rest of the fixed object stays:
		// its parts become fixed values of their own, and only the forced subtree is rebuilt.
		v.keepFixedParts(path, value)
	}
	if f.ref != "" {
		return nil, &shapeError{code: codeE2ERequired, path: e2eFileName + ":values." + path,
			msg: fmt.Sprintf("a reference to a %s needs a value: a backendRef id or a resourceRef that the test can use", f.ref)}
	}
	switch f.typ.Kind {
	case model.List, model.Set:
		return v.list(f, path, full)
	case model.Map:
		item, err := v.item(f, f.typ.Elem, path, full)
		if err != nil {
			return nil, err
		}
		out := (&omap{}).set("key", item)
		// A map can need more entries than one. Repeat the finished value under other keys.
		for n := 2; f.typ.MinItems != nil && int64(len(out.keys)) < *f.typ.MinItems; n++ {
			out.set(fmt.Sprintf("key%d", n), cloneSpecValue(item))
		}
		return out, nil
	case model.String, model.Bool, model.Number, model.Integer, model.Enum, model.Object, model.OneOf:
		// One value, below.
	}
	return v.item(f, f.typ, path, full)
}

func (v *sampleValues) item(f *crField, t *model.Type, path string, full bool) (any, error) {
	switch t.Kind {
	case model.Object, model.OneOf:
		return v.object(f.object, path+".", full)
	case model.Enum:
		return v.enumName(f, path), nil
	case model.Bool:
		return true, nil
	case model.Integer:
		return integerValue(t), nil
	case model.Number:
		return numberValue(t), nil
	case model.String, model.List, model.Set, model.Map:
		// A string, below. The renderer rejects a collection here.
	}
	return stringValue(f.API, t, path)
}

// arm returns the API property of the arm that a value sets, "" for none. The first arm that is not
// skipped is the default. A sweep takes the arm that adds the most coverage.
func (v *sampleValues) arm(t *crType, g model.OneOfGroup, full bool, path string) string {
	if !full && g.AllowNone {
		return ""
	}
	chosen := ""
	if forced, ok := v.layerArm(path); ok && slices.Contains(g.Arms, forced) {
		chosen = forced
	} else if v.sweeping && v.record && !v.cov.coupled[path] {
		chosen = v.cov.pickArm(t, g)
	}
	for _, arm := range g.Arms {
		if chosen == "" && !v.cov.skipped[coverageKey{"arm", v.cov.owner[t], arm}] {
			chosen = arm
		}
	}
	if chosen == "" {
		if g.AllowNone {
			return "" // every arm is skipped and the group may be empty: send none
		}
		chosen = g.Arms[0] // every arm is skipped, and the group needs one
	}
	return chosen
}

// enumName returns the CR spelling of the value that a value sets. The first value that is not
// skipped is the default. A sweep takes the first value that no test sends yet.
func (v *sampleValues) enumName(f *crField, path string) string {
	return v.enumNames(f, path, 1)[0]
}

// enumNames returns up to limit CR spellings. A sweep returns the open values first, and the
// default does not look at them.
func (v *sampleValues) enumNames(f *crField, path string, limit int) []string {
	var open, usable []enumValue
	for _, value := range f.enum.Values {
		key := coverageKey{"enum", f.enum.schema, value.sent()}
		if v.cov.skipped[key] {
			continue
		}
		usable = append(usable, value)
		if v.sweeping && v.record && !v.cov.coupled[path] && v.cov.open(key) && value.API != enumZero(f) {
			open = append(open, value)
		}
	}
	chosen := open
	if len(chosen) == 0 {
		chosen = usable
		if len(chosen) > 1 {
			chosen = chosen[:1]
		}
	}
	if len(chosen) == 0 {
		chosen = f.enum.Values[:1] // every value is skipped: the field still needs one
	}
	if len(chosen) > limit {
		chosen = chosen[:limit]
	}
	names := make([]string, len(chosen))
	for i, value := range chosen {
		names[i] = value.Name
		if v.record {
			v.cov.mark(coverageKey{"enum", f.enum.schema, value.sent()})
		}
	}
	return names
}

// list makes up the items of a list. The default is one item. A sweep sends every open enum value
// of a list of enums, and adds items of objects while each item adds coverage.
func (v *sampleValues) list(f *crField, path string, full bool) (any, error) {
	value, err := v.listItems(f, path, full)
	if err != nil {
		return nil, err
	}
	items, ok := value.([]any)
	if !ok || f.typ.MinItems == nil {
		return value, nil
	}
	// The contract may need more items than the entries or the sweep gave. Pad with items that keep
	// what the others hold: a coupled value must keep its data.
	for int64(len(items)) < *f.typ.MinItems {
		item, err := v.padItem(f, path, full, items)
		if err != nil {
			return nil, err
		}
		v.uniqueKey(f, items, item) // a repeated item gets a key of its own, when it can
		items = append(items, item)
	}
	return items, nil
}

// padItem makes one more item for a list that is below its contract minimum. A list of objects
// repeats a finished item, because a new one could pick a coupled value without its data. A list
// of enums takes a value that the list does not hold yet, and a list of strings or integers a
// distinct value, so a set does not get a duplicate.
func (v *sampleValues) padItem(f *crField, path string, full bool, items []any) (any, error) {
	elem := f.typ.Elem
	switch {
	case elem.Kind == model.Enum && f.enum != nil:
		used := map[string]bool{}
		for _, item := range items {
			if name, ok := item.(string); ok {
				used[name] = true
			}
		}
		for _, value := range f.enum.Values {
			if !used[value.Name] && value.API != enumZero(f) && !v.cov.skipped[coverageKey{"enum", f.enum.schema, value.API}] {
				if v.record {
					v.cov.mark(coverageKey{"enum", f.enum.schema, value.sent()})
				}
				return value.Name, nil
			}
		}
	case (elem.Kind == model.Object || elem.Kind == model.OneOf) && f.typ.Kind == model.List && len(items) > 0:
		return cloneSpecValue(items[len(items)-1]), nil
	case elem.Kind == model.String && elem.Pattern == "" && elem.Format != "date-time" && len(items) > 0:
		if s, ok := items[0].(string); ok {
			next := fmt.Sprintf("%s-%d", s, len(items)+1)
			if elem.MaxLength == nil || int64(len(next)) <= *elem.MaxLength {
				return next, nil
			}
		}
	case elem.Kind == model.Integer && len(items) > 0:
		if n, ok := items[0].(int64); ok {
			next := n + int64(len(items))
			if elem.Maximum == nil || float64(next) <= *elem.Maximum {
				return next, nil
			}
		}
	}
	return v.item(f, elem, path, full)
}

// cloneSpecValue copies a made-up value, so a repeated item can be changed without changing the first.
func cloneSpecValue(value any) any {
	switch x := value.(type) {
	case *omap:
		out := &omap{}
		for _, key := range x.keys {
			out.set(key, cloneSpecValue(x.values[key]))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for key, child := range x {
			out[key] = cloneSpecValue(child)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, child := range x {
			out[i] = cloneSpecValue(child)
		}
		return out
	}
	return value
}

func (v *sampleValues) listItems(f *crField, path string, full bool) (any, error) {
	if !v.sweeping || !v.record {
		item, err := v.item(f, f.typ.Elem, path, full)
		return []any{item}, err
	}
	if f.typ.Elem.Kind == model.Enum && f.enum != nil {
		limit := maxItemsPerList
		if f.typ.MaxItems != nil && int(*f.typ.MaxItems) < limit {
			limit = int(*f.typ.MaxItems)
		}
		items := []any{}
		for _, name := range v.enumNames(f, path, max(limit, 1)) {
			items = append(items, name)
		}
		return items, nil
	}
	if planned := v.items[path]; len(planned) != 0 {
		return v.plannedItems(f, path, full, planned)
	}
	first, err := v.item(f, f.typ.Elem, path, full)
	if err != nil {
		return nil, err
	}
	items := []any{first}
	for len(items) < maxItemsPerList && v.cov.extra < maxExtraItemsPerStep {
		if f.typ.MaxItems != nil && int64(len(items)) >= *f.typ.MaxItems {
			break
		}
		before := len(v.cov.covered)
		trial := v.cov.snapshot()
		item, err := v.item(f, f.typ.Elem, path, full)
		if err != nil {
			return nil, err
		}
		if len(v.cov.covered) == before || !v.uniqueKey(f, items, item) {
			v.cov.restore(trial) // the item adds nothing, or has no free key: drop it, and what it would have recorded
			break
		}
		items = append(items, item)
		v.cov.extra++
	}
	return items, nil
}

// plannedItems makes one item for each when entry that the copy plans in this list.
func (v *sampleValues) plannedItems(f *crField, path string, full bool, planned []*overlay) (any, error) {
	items := []any{}
	for _, layer := range planned {
		v.layers = append(v.layers, layer)
		trial := v.cov.snapshot()
		item, err := v.item(f, f.typ.Elem, path, full)
		v.layers = v.layers[:len(v.layers)-1]
		if err != nil {
			return nil, err
		}
		if !v.uniqueKey(f, items, item) {
			v.cov.restore(trial) // no free key: the entry waits, and records nothing
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// fixed returns the fixed value of a path: a value of e2e.yaml, or a part of a fixed object that is being rebuilt.
func (v *sampleValues) fixed(path string) (any, bool) {
	if value, ok := v.file.Values[path]; ok {
		return value, true
	}
	value, ok := v.virtual[path]
	return value, ok
}

// keepFixedParts makes each part of a fixed object a fixed value of its own path.
func (v *sampleValues) keepFixedParts(path string, value any) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	if v.virtual == nil {
		v.virtual = map[string]any{}
	}
	for key, part := range object {
		v.virtual[path+"."+key] = part
	}
}

// armForcedBelow reports whether the layers force an arm, or plan list items, at a path below a
// fixed object. The fixed value cannot hold them, so the object is made from the type tree.
func (v *sampleValues) armForcedBelow(path string) bool {
	for _, layer := range v.layers {
		for p := range layer.arms {
			if strings.HasPrefix(p, path+".") {
				return true
			}
		}
	}
	for p, planned := range v.items {
		if len(planned) != 0 && strings.HasPrefix(p, path+".") {
			return true
		}
	}
	return false
}

// withDescendantValues merges the values that the layers force below a fixed object into a copy of
// the object, so a when entry that sets a field inside it is not lost. A path through a list
// applies to every item, as a path with no index does elsewhere.
func (v *sampleValues) withDescendantValues(path string, fixed any) any {
	merged := fixed
	copied := false
	for _, layer := range v.layers {
		for _, p := range sortedKeys(layer.values) {
			if !strings.HasPrefix(p, path+".") {
				continue
			}
			if !copied {
				merged, copied = deepCopyLiteral(fixed), true
			}
			merged = setLiteralPath(merged, strings.Split(strings.TrimPrefix(p, path+"."), "."), layer.values[p])
		}
	}
	return merged
}

func deepCopyLiteral(value any) any {
	switch x := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, child := range x {
			out[k] = deepCopyLiteral(child)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, child := range x {
			out[i] = deepCopyLiteral(child)
		}
		return out
	}
	return value
}

// setLiteralPath sets a value at a path inside a literal that is already a copy.
func setLiteralPath(node any, segments []string, value any) any {
	switch x := node.(type) {
	case map[string]any:
		if len(segments) == 1 {
			x[segments[0]] = value
			return x
		}
		x[segments[0]] = setLiteralPath(x[segments[0]], segments[1:], value)
		return x
	case []any:
		for i := range x {
			x[i] = setLiteralPath(x[i], segments, value)
		}
		return x
	case nil:
		if len(segments) == 1 {
			return map[string]any{segments[0]: value}
		}
		return map[string]any{segments[0]: setLiteralPath(nil, segments[1:], value)}
	}
	return node
}

// layerValue returns the value that the layers force at a path.
func (v *sampleValues) layerValue(path string) (any, bool) {
	for i := len(v.layers) - 1; i >= 0; i-- {
		if value, ok := v.layers[i].values[path]; ok {
			return value, true
		}
	}
	return nil, false
}

// layerArm returns the arm that the layers force at the field that holds a oneOf.
func (v *sampleValues) layerArm(path string) (string, bool) {
	for i := len(v.layers) - 1; i >= 0; i-- {
		if arm, ok := v.layers[i].arms[path]; ok {
			return arm, true
		}
	}
	return "", false
}

func integerValue(t *model.Type) int64 {
	n := int64(1)
	if t.Minimum != nil {
		n = int64(*t.Minimum)
	}
	if t.Maximum != nil && float64(n) > *t.Maximum {
		n = int64(*t.Maximum)
	}
	return n
}

// numberValue makes up a number inside the contract bounds. The CR holds it as a quantity, which
// is int-or-string in the CRD: a whole number can be a YAML number, and a fraction must be a
// decimal string, or admission rejects it.
func numberValue(t *model.Type) any {
	n := 1.0
	if t.Minimum != nil && n < *t.Minimum {
		n = *t.Minimum
	}
	if t.Maximum != nil && n > *t.Maximum {
		n = *t.Maximum
	}
	if n == math.Trunc(n) && math.Abs(n) < 1e15 {
		return int64(n)
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}

// stringValue makes up a string that meets the length limits and the pattern of the contract.
func stringValue(name string, t *model.Type, path string) (string, error) {
	if t.Format == "date-time" {
		return "2024-01-01T00:00:00Z", nil
	}
	s := "sample-" + kebab(goName(name))
	if t.MaxLength != nil && int64(len(s)) > *t.MaxLength {
		s = s[:*t.MaxLength]
	}
	if t.MinLength != nil && int64(len(s)) < *t.MinLength {
		s += strings.Repeat("x", int(*t.MinLength)-len(s))
	}
	if t.Pattern != "" {
		re, err := regexp.Compile(t.Pattern)
		if err != nil || !re.MatchString(s) {
			return "", &shapeError{code: codeE2ERequired, path: e2eFileName + ":values." + path,
				msg: fmt.Sprintf("the made-up value %q does not match the pattern %s; set the value in %s", s, t.Pattern, e2eFileName)}
		}
	}
	return s, nil
}

// updateCandidates lists top-level fields first, then fields in root groups. A group keeps its
// API fields at the root, even though the CR and its sample put them in an object.
func updateCandidates(spec *crType, full *omap) []updateCandidate {
	var candidates []updateCandidate
	appendFields := func(owner *crType, jsonPrefix, goPrefix string, value any) {
		inOneOf := map[string]bool{}
		for _, group := range owner.groups {
			for _, arm := range group.Arms {
				inOneOf[arm] = true
			}
		}
		for _, field := range owner.Fields {
			if field.group {
				continue
			}
			candidates = append(candidates, updateCandidate{
				field: field, jsonPath: jsonPrefix + field.JSON, goPath: goPrefix + field.GoName,
				value: objectValue(value, field.JSON), inOneOf: inOneOf[field.API],
			})
		}
	}
	appendFields(spec, "", "", full)
	for _, group := range spec.Fields {
		if group.group {
			appendFields(group.object, group.JSON+".", group.GoName+".", full.get(group.JSON))
		}
	}
	return candidates
}

func objectValue(object any, field string) any {
	switch value := object.(type) {
	case *omap:
		return value.get(field)
	case map[string]any:
		return value[field]
	}
	return nil
}

// updateStep picks the change of the update step: the update line of e2e.yaml, or the first
// mutable scalar outside a oneOf. A top-level field takes precedence over a grouped field.
func updateStep(d *kindData, ix *refIndex, candidates []updateCandidate, file *e2eFile, nameField string) (*updateData, error) {
	resource, err := ix.typeRef("fields")
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		f := candidate.field
		if file.Update != nil && file.Update.Field != candidate.jsonPath {
			continue
		}
		if f.immutable || candidate.inOneOf || f.enum != nil || f.ref != "" || f.typ.Pattern != "" || f.typ.WireString ||
			!slices.Contains([]model.Kind{model.String, model.Bool, model.Integer}, f.typ.Kind) {
			if file.Update != nil {
				return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.field",
					msg: fmt.Sprintf("%s is not a mutable string, bool, or integer outside a oneOf", candidate.jsonPath)}
			}
			continue
		}
		value, err := updateValue(f, candidate.value, file, nameField == candidate.goPath)
		if err != nil {
			return nil, err
		}
		base := "got."
		if d.GetResponseField != "" {
			base += d.GetResponseField + "."
		}
		// The CR holds a createAs field flat. The response nests it in an object.
		var notNil []string
		owner, name := resource.Name, f.API
		if f.fold != nil {
			object, err := ix.field(resource.Name, f.fold.Object)
			if err != nil {
				return nil, err
			}
			base += object.Name + "."
			if strings.HasPrefix(object.Want, "*") {
				notNil = append(notNil, strings.TrimSuffix(base, "."))
			}
			owner, name = strings.TrimPrefix(object.Want, "*"), f.fold.Property
		}
		got, err := ix.field(owner, name)
		if err != nil {
			return nil, err
		}
		read := base + got.Name
		if strings.HasPrefix(got.Want, "*") {
			read = fmt.Sprintf("ptr.Deref(%s, %s)", read, zeroLiteral(f.typ))
		}
		set := value
		if f.Optional {
			set = "ptr.To(" + value + ")"
		}
		return &updateData{Field: candidate.goPath, Set: set, Want: value, Got: read, NotNil: notNil}, nil
	}
	if file.Update != nil {
		return nil, &shapeError{code: codeE2EUnused, path: e2eFileName + ":update.field", msg: "the spec has no such field"}
	}
	return nil, &shapeError{code: codeE2ERequired, path: e2eFileName + ":update",
		msg: "the spec has no mutable string, bool, or integer outside a oneOf that the update step can change; set update in " + e2eFileName}
}

// updateValue is the new value of the update step as Go code.
func updateValue(f *crField, current any, file *e2eFile, nameField bool) (string, error) {
	goType := strings.TrimPrefix(f.GoType, "*")
	if file.Update != nil {
		switch value := file.Update.Value.(type) {
		case string:
			return strconv.Quote(value), nil
		case bool:
			return strconv.FormatBool(value), nil
		case int:
			return fmt.Sprintf("%s(%d)", goType, value), nil
		}
		return "", &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.value", msg: "the value must be a string, a bool, or an integer"}
	}
	if nameField {
		return "fullName + \"-updated\"", nil
	}
	switch value := current.(type) {
	case bool:
		return strconv.FormatBool(!value), nil
	case int64:
		next := value + 1
		if f.typ.Maximum != nil && float64(next) > *f.typ.Maximum {
			next = value - 1
		}
		return fmt.Sprintf("%s(%d)", goType, next), nil
	case string:
		next := value + "-updated"
		if f.typ.MaxLength != nil && int64(len(next)) > *f.typ.MaxLength {
			next = strings.Repeat("u", len(value))
		}
		return strconv.Quote(next), nil
	}
	return "", fmt.Errorf("spec.%s: no value to update", f.API)
}

func zeroLiteral(t *model.Type) string {
	switch t.Kind {
	case model.Bool:
		return "false"
	case model.Integer:
		return "0"
	case model.String, model.Number, model.Enum, model.Object, model.OneOf, model.List, model.Set, model.Map:
		// The zero value of a string, below.
	}
	return `""`
}

// goString is s as a Go string literal: a raw string when it can be one.
func goString(s string) string {
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

// omap is a JSON or YAML object that keeps the order of its keys.
type omap struct {
	keys   []string
	values map[string]any
}

func (m *omap) set(key string, value any) *omap {
	if m.values == nil {
		m.values = map[string]any{}
	}
	if _, ok := m.values[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
	return m
}

func (m *omap) get(key string) any { return m.values[key] }

// MarshalJSON writes the keys in order.
func (m *omap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(m.values[k])
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(value)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func (m *omap) json() string {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err) // the values are strings, numbers, bools, lists, and objects
	}
	return string(data)
}

// yamlNode is the value as a YAML node, with the keys in order.
func yamlNode(value any) (*yaml.Node, error) {
	switch v := value.(type) {
	case *omap:
		n := &yaml.Node{Kind: yaml.MappingNode}
		for _, k := range v.keys {
			child, err := yamlNode(v.values[k])
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: k}, child)
		}
		return n, nil
	case []any:
		n := &yaml.Node{Kind: yaml.SequenceNode}
		for _, item := range v {
			child, err := yamlNode(item)
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, child)
		}
		return n, nil
	}
	n := &yaml.Node{}
	if err := n.Encode(value); err != nil {
		return nil, err
	}
	return n, nil
}

// listMapKeys returns the item fields that identify the items of a Kubernetes list-map.
func listMapKeys(f *crField) []string {
	var keys []string
	for _, marker := range f.Markers {
		if key, ok := strings.CutPrefix(marker, "+listMapKey="); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

// uniqueKey makes the key of a new item differ from the keys of the items before it. A string key
// gets a suffix. It reports false when the key cannot be made unique: an enum key that repeats, or
// a string key with a pattern or a length limit that the suffix would break.
func (v *sampleValues) uniqueKey(f *crField, items []any, item any) bool {
	keys := listMapKeys(f)
	if len(keys) == 0 {
		return true
	}
	get := func(x any, key string) any {
		switch o := x.(type) {
		case *omap:
			return o.get(key)
		case map[string]any:
			return o[key]
		}
		return nil
	}
	set := func(x any, key string, value any) {
		switch o := x.(type) {
		case *omap:
			o.set(key, value)
		case map[string]any:
			o[key] = value
		}
	}
	tuple := func(x any) string {
		var b strings.Builder
		for _, key := range keys {
			fmt.Fprintf(&b, "%v\x00", get(x, key))
		}
		return b.String()
	}
	taken := map[string]bool{}
	for _, other := range items {
		taken[tuple(other)] = true
	}
	if !taken[tuple(item)] {
		return true
	}
	for _, key := range keys {
		original, ok := get(item, key).(string)
		if !ok || f.object == nil {
			continue
		}
		i := slices.IndexFunc(f.object.Fields, func(c *crField) bool { return c.JSON == key })
		if i < 0 || f.object.Fields[i].typ == nil || f.object.Fields[i].typ.Kind != model.String || f.object.Fields[i].typ.Pattern != "" {
			continue
		}
		limit := f.object.Fields[i].typ.MaxLength
		for n := len(items) + 1; n <= len(items)+25; n++ {
			candidate := fmt.Sprintf("%s-%d", original, n)
			if limit != nil && int64(len(candidate)) > *limit {
				break
			}
			set(item, key, candidate)
			if !taken[tuple(item)] {
				return true
			}
		}
		set(item, key, original)
	}
	return false
}

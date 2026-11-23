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
	// Substitutions replace sample values with run-specific values in generated tests. A value
	// can name @{run}, the name of the CR, and @{uuid}, a new UUID for each CR, for an id that the
	// API keeps unique across the account.
	Substitutions map[string]string `yaml:"substitutions"`
	// Upgrade freezes the spec accepted by a released chart.
	Upgrade *e2eUpgrade `yaml:"upgrade"`
	// Omit lists optional fields that cannot appear with other full-spec fields. A field of a list
	// item takes the path without an index, for example policies.targets.
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
	// Env lists environment variables that the live tests read. A substitution, a fixture spec, or
	// the upgrade spec names one as @{env.NAME}, for an account value that the public repository
	// cannot hold, such as a resource that only the test account has. A value cannot: it reaches
	// the sample.
	Env []string `yaml:"env"`
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

// apiVersionPattern matches a Kubernetes API version, such as v1alpha1 or v1beta1.
var apiVersionPattern = regexp.MustCompile(`^v[0-9]+(?:(?:alpha|beta)[0-9]+)?$`)

type e2eFixture struct {
	Kind string `yaml:"kind"`
	// Version is the API version of the kind, such as v1beta1 for Alert. The default is v1alpha1.
	Version     string         `yaml:"version"`
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
	// Check is the property of the resource that the update step compares, for a kind with the
	// content key: the step sets json to Value and reads Check from the JSON of the Get response.
	Check string `yaml:"check"`
	// Skip leaves out the update step, with the reason: the API accepts no other value of any
	// field that the step can change, such as a key whose versions need different parameters.
	Skip string `yaml:"skip"`
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
	seen := map[string]bool{}
	for _, name := range f.Env {
		if !envName.MatchString(name) || seen[name] {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":env", msg: fmt.Sprintf("%q is not an upper-case variable name, or it is listed twice", name)}
		}
		seen[name] = true
	}
	return &f, nil
}

// uuidToken in a substitution is a new UUID for each CR that a generated test builds.
const uuidToken = "@{uuid}"

var (
	envName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	envRef  = regexp.MustCompile(`@\{env\.([^}]*)\}`)
)

// envIssues checks that each @{env.NAME} of the e2e texts names a variable of env, and that each
// variable of env is used.
func envIssues(env []string, texts ...string) error {
	used := map[string]bool{}
	for _, text := range texts {
		for _, m := range envRef.FindAllStringSubmatch(text, -1) {
			if !slices.Contains(env, m[1]) {
				return &shapeError{code: codeE2EInvalid, path: e2eFileName + ":env", msg: fmt.Sprintf("@{env.%s} names a variable that env does not list", m[1])}
			}
			used[m[1]] = true
		}
	}
	for _, name := range env {
		if !used[name] {
			return &shapeError{code: codeE2EUnused, path: e2eFileName + ":env." + name, msg: "no value names @{env." + name + "}"}
		}
	}
	return nil
}

// e2eData is what the e2e test template needs.
type e2eData struct {
	Omit     []string
	Serial   bool
	Fixtures []fixtureData
	// FixtureVersions are the API versions of the fixtures other than v1alpha1, for the imports.
	FixtureVersions       []string
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
	// Overwrite is the e2e test of an overwrite resource, nil for another resource.
	Overwrite *overwriteData
	// Env are the environment variables that the live tests read.
	Env []string
	// UUID is true when a substitution names @{uuid}: the tests import the uuid package.
	UUID bool
}

// overwriteData is the e2e test of an overwrite resource. The spec holds one list; the test gives its
// item a unique name, finds the item by that name in the list that Get returns, and changes one field.
type overwriteData struct {
	List, ListGet   string // the Go field of the list in the spec and in the Get response
	Name, NameGet   string // the Go field of the name of an item in the spec and in the Get response
	NamePtr         bool   // the spec name is a pointer
	NameGetPtr      bool   // the Get name is a pointer
	Field, FieldGet string // the Go field that the update step changes, in the spec and in the Get response
	Set, Want, Got  string
}

// coverageSpecData is one spec of the offline coverage test. Expect holds Go string literals.
type coverageSpecData struct {
	Name, JSON string
	Expect     []string
}

type fixtureData struct {
	Kind, Suffix, StatusField, SpecJSON string
	// Alias is the import alias of the API package of the kind, such as coralogixv1beta1.
	Alias string
}

type substitutionData struct {
	From, To string
	// UUID is true when To names @{uuid}.
	UUID bool
}

type upgradeData struct {
	From, SpecJSON, Field, Value string
	// Nested is true for a dotted field: the patch sets the path below a spec field.
	Nested bool
}

// updateData is the update step as Go code: the field that it changes, the value that it sets,
// the value that the API then returns, and the expression that reads it from the response "got".
type updateData struct {
	Field, Set, Want, Got string
	// ListGet and Match select the updated item when the mutable field is inside a list.
	ListGet, Match string
	// Check is the resource property that the step compares for a kind with the content key.
	// Want is then the new JSON text.
	Check string
	// NotNil lists values that the check reads through: the nested object of a createAs field.
	NotNil []string
	// Skip is the reason of an update step that e2e.yaml leaves out.
	Skip string
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
		if !optionalPath(d.Spec, strings.Split(name, ".")) {
			return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":omit", msg: "omit must name an optional spec field; a field of a list item takes the path without an index"}
		}
	}
	var steps [][]sweepCopy
	if full, minimal, steps, err = v.specs(d); err != nil {
		return nil, nil, nil, err
	}
	for _, name := range sortedKeys(file.Minimal) {
		value := file.Minimal[name]
		known, inGroup := false, false
		for _, field := range d.Spec.Fields {
			// The list of an overwrite resource is required, and its items need the fields that the
			// API needs.
			if field.JSON == name && (field.Optional || d.Overwrite) {
				known = true
			}
			// A required group holds optional fields. The value names them.
			if field.JSON == name && field.group && optionalGroupFields(field, value) {
				known, inGroup = true, true
			}
		}
		if !known {
			return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":minimal." + name, msg: "not an optional spec field, or a group object of its optional fields"}
		}
		if group, ok := minimal.get(name).(*omap); ok && inGroup {
			// The minimal group keeps its required fields. The value adds optional ones.
			fields := value.(map[string]any)
			for _, key := range sortedKeys(fields) {
				group.set(key, fields[key])
			}
			continue
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
		version := fixture.Version
		if version == "" {
			version = "v1alpha1"
		}
		if !apiVersionPattern.MatchString(version) {
			return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":fixtures", msg: fmt.Sprintf("version %q is not a Kubernetes API version", fixture.Version)}
		}
		if version != "v1alpha1" && !slices.Contains(e2e.FixtureVersions, version) {
			e2e.FixtureVersions = append(e2e.FixtureVersions, version)
		}
		e2e.Fixtures = append(e2e.Fixtures, fixtureData{fixture.Kind, fixture.Suffix, fixture.StatusField, goString(string(encoded)), "coralogix" + version})
	}
	for _, from := range sortedKeys(file.Substitutions) {
		to := file.Substitutions[from]
		hasUUID := strings.Contains(to, uuidToken)
		e2e.Substitutions = append(e2e.Substitutions, substitutionData{from, to, hasUUID})
		e2e.UUID = e2e.UUID || hasUUID
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
		nested := !d.Overwrite && strings.Contains(file.Upgrade.Field, ".")
		if nested {
			top, _, _ := strings.Cut(file.Upgrade.Field, ".")
			if _, ok := file.Upgrade.Spec[top]; !ok {
				return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":upgrade.field", msg: "a dotted field must start with a field of upgrade.spec"}
			}
		}
		e2e.Upgrade = &upgradeData{From: file.Upgrade.From, SpecJSON: string(encoded), Field: file.Upgrade.Field, Value: file.Upgrade.Value, Nested: nested}
	}
	// The values also make the sample, which must not name a variable.
	if envRef.MatchString(e2e.FullJSON) || envRef.MatchString(e2e.MinimalJSON) {
		return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":values", msg: "a value reaches the sample; name the variable in a substitution of the value"}
	}
	// A new UUID is made for each CR when a substitution is applied. The sample, the fixtures, and
	// the upgrade check apply no substitution.
	uuidTexts := []string{e2e.FullJSON, e2e.MinimalJSON}
	for _, fixture := range e2e.Fixtures {
		uuidTexts = append(uuidTexts, fixture.SpecJSON)
	}
	if e2e.Upgrade != nil {
		uuidTexts = append(uuidTexts, e2e.Upgrade.SpecJSON, e2e.Upgrade.Value)
	}
	for _, text := range uuidTexts {
		if strings.Contains(text, uuidToken) {
			return nil, nil, nil, &shapeError{code: codeE2EInvalid, path: e2eFileName, msg: uuidToken + " is valid only in a substitution; keep a fixed value in values and map it to " + uuidToken + " in substitutions"}
		}
	}
	var texts []string
	for _, fixture := range e2e.Fixtures {
		texts = append(texts, fixture.SpecJSON)
	}
	for _, sub := range e2e.Substitutions {
		texts = append(texts, sub.To)
	}
	if e2e.Upgrade != nil {
		texts = append(texts, e2e.Upgrade.SpecJSON)
	}
	if err := envIssues(file.Env, texts...); err != nil {
		return nil, nil, nil, err
	}
	e2e.Env = file.Env
	if d.Overwrite {
		if e2e.Overwrite, err = overwriteStep(d, ix, file, full); err != nil {
			return nil, nil, nil, err
		}
		e2e.Serial = true // every write replaces the list of the whole account
		return full, minimal, e2e, nil
	}
	if d.Singleton {
		e2e.Serial = true // every CR writes the one resource of the team
	}
	candidates := updateCandidates(d.Spec, full)
	for _, candidate := range candidates {
		if candidate.field.API == "name" && candidate.field.typ.Kind == model.String {
			e2e.NameField, e2e.NamePtr = candidate.goPath, candidate.field.Optional
			break
		}
	}
	if e2e.Update, err = updateStep(d, ix, candidates, file, e2e.NameField, full); err != nil {
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

// optionalPath reports whether path names an optional field of t: a top-level field, or a field
// inside an object or the items of a list, named without an index.
func optionalPath(t *crType, path []string) bool {
	for _, f := range t.Fields {
		if f.JSON != path[0] {
			continue
		}
		if len(path) == 1 {
			return f.Optional
		}
		return f.object != nil && optionalPath(f.object, path[1:])
	}
	return false
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
	dateTimes := 0
	patterned := map[string]int{}
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
		if skip[f.API] || (full && slices.Contains(v.file.Omit, prefix+f.JSON)) || (!full && f.Optional && !inGroup && !f.secretExactlyOne) {
			continue
		}
		value, err := v.field(f, prefix+f.JSON, full)
		if err != nil {
			return nil, err
		}
		if selected[f.API] && !full {
			if value, err = v.nonEmptyArm(f, prefix+f.JSON, value); err != nil {
				return nil, err
			}
		}
		if text, ok := value.(string); ok && f.typ != nil && f.typ.Pattern != "" && !v.hasValue(prefix+f.JSON) {
			// Two fields of one object with the same made-up text, such as the two ends of a range
			// whose pattern the plain sample does not match, take other characters of the pattern.
			if n := patterned[text]; n > 0 {
				if other, ok := patternString(f.typ.Pattern, minLength(f.typ), maxLength(f.typ), n); ok {
					value, text = other, other
				}
			}
			patterned[text]++
		}
		if text, ok := value.(string); ok && text == sampleDateTime && f.typ != nil && f.typ.Format == "date-time" {
			// Fields of one object that make a range, such as from and to, must not be equal: the
			// API rejects a range that is empty or reversed. Each date-time field is one hour later.
			value = sampleDateTimeAt(dateTimes)
			dateTimes++
		}
		out.set(f.JSON, value)
		if v.record && selected[f.API] {
			// The arm counts only when its field is in the spec: omit and the full-versus-minimal rule can drop it.
			v.cov.mark(coverageKey{"arm", v.cov.owner[t], f.API})
		}
	}
	return out, nil
}

// nonEmptyArm gives a selected oneOf arm that came out as an empty object its first optional
// field that is not a collection. Kubernetes before 1.28 does not see an empty object as present in a CEL rule such
// as has(self.action), so an arm of {} fails admission on the oldest cluster of the live tests.
// An arm with no field stays empty: the contract states it that way.
func (v *sampleValues) nonEmptyArm(f *crField, path string, value any) (any, error) {
	arm, ok := value.(*omap)
	if !ok || len(arm.keys) != 0 || f.object == nil {
		return value, nil
	}
	for _, child := range f.object.Fields {
		if child.synthetic || child.group || !child.Optional || child.ref != "" || child.joinedBy != "" {
			continue
		}
		if strings.Contains(strings.Join(child.Doc, " "), "Requires ") {
			continue // the contract says that another field must come with it, as in "Requires event"
		}
		switch child.typ.Kind {
		case model.String, model.Bool, model.Integer, model.Number, model.Enum, model.Object, model.OneOf:
		case model.List, model.Set, model.Map:
			continue
		}
		item, err := v.field(child, path+"."+child.JSON, false)
		if err != nil {
			return nil, err
		}
		arm.set(child.JSON, item)
		break
	}
	return arm, nil
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
	if f.joinedBy != "" {
		item, err := stringValue(f.API, f.typ, path)
		return []any{item}, err
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
		if f.intNames != nil {
			return f.intNames.Values[0].Name, nil
		}
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
		// The default is the first business value. The zero value is the API default, and an API can
		// reject it when it is sent, so it is the default only when the field has no other value.
		chosen = usable
		if i := slices.IndexFunc(usable, func(value enumValue) bool { return value.API != enumZero(f) }); i > 0 {
			chosen = usable[i:]
		}
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
		v.uniqueKey(f, path, items, item, true) // a repeated item gets a key of its own, when it can
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
	case elem.Kind == model.Integer && f.intNames != nil:
		used := map[string]bool{}
		for _, item := range items {
			if name, ok := item.(string); ok {
				used[name] = true
			}
		}
		for _, value := range f.intNames.Values {
			if !used[value.Name] {
				return value.Name, nil
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
		if len(v.cov.covered) == before || !v.uniqueKey(f, path, items, item, false) {
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
		fits := err == nil && v.uniqueKey(f, path, items, item, false) // the layer still holds the keys that the entry sets
		v.layers = v.layers[:len(v.layers)-1]
		if err != nil {
			return nil, err
		}
		if !fits {
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

// sampleDateTime is the made-up value of a date-time field.
const sampleDateTime = "2024-01-01T00:00:00Z"

// sampleDateTimeAt is the made-up date-time that is n hours after sampleDateTime.
func sampleDateTimeAt(n int) string {
	return fmt.Sprintf("2024-01-01T%02d:00:00Z", n%24)
}

// stringValue makes up a string that meets the length limits and the pattern of the contract.
func stringValue(name string, t *model.Type, path string) (string, error) {
	if t.Format == "date-time" {
		return sampleDateTime, nil
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
			// The plain sample does not match: take a string that the pattern builds.
			if built, ok := patternString(t.Pattern, minLength(t), maxLength(t), 0); ok {
				return built, nil
			}
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
func updateStep(d *kindData, ix *refIndex, candidates []updateCandidate, file *e2eFile, nameField string, full *omap) (*updateData, error) {
	if file.Update != nil && file.Update.Skip != "" {
		if file.Update.Field != "" || file.Update.Value != nil || file.Update.Check != "" {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.skip", msg: "a skipped update step sets no field, value, or check"}
		}
		if strings.TrimSpace(file.Update.Skip) == "" {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.skip", msg: "the reason is blank"}
		}
		return &updateData{Skip: file.Update.Skip}, nil
	}
	if d.Content != nil {
		return contentUpdateStep(file)
	}
	if file.Update != nil && file.Update.Check != "" {
		return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.check", msg: "only a kind with the content key uses check"}
	}
	resource, err := ix.typeRef("fields")
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		f := candidate.field
		if file.Update != nil && file.Update.Field != candidate.jsonPath {
			continue
		}
		// A synthetic field, such as an ignored field or the CR arm of a split reference, has no API
		// property for the step to read back, and no contract type.
		if f.synthetic || f.immutable || candidate.inOneOf || f.enum != nil || f.intNames != nil || f.ref != "" || f.typ.Pattern != "" || f.typ.WireString ||
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
		base := d.GetItem() + "."
		// The CR holds a createAs field flat. The response nests it in an object.
		var notNil []string
		owner, name := resource.Name, f.API
		if parent, ok := d.inlined[f.API]; ok {
			// An inline line on the resource: the response nests the field in an object.
			object, err := ix.field(resource.Name, parent.Field)
			if err != nil {
				return nil, err
			}
			base += object.Name + "."
			if strings.HasPrefix(object.Want, "*") {
				notNil = append(notNil, strings.TrimSuffix(base, "."))
			}
			owner = strings.TrimPrefix(object.Want, "*")
		}
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
		if step, err := listUpdateStep(d, ix, file, resource.Name, full); step != nil || err != nil {
			return step, err
		}
		return nil, &shapeError{code: codeE2EUnused, path: e2eFileName + ":update.field", msg: "the spec has no such field"}
	}
	return nil, &shapeError{code: codeE2ERequired, path: e2eFileName + ":update",
		msg: "the spec has no mutable string, bool, or integer outside a oneOf that the update step can change; set update in " + e2eFileName}
}

// listUpdateStep changes the first item of a list-map. It reads the item back by its key because
// the API may return list items in another order.
func listUpdateStep(d *kindData, ix *refIndex, file *e2eFile, resourceName string, full *omap) (*updateData, error) {
	listName, fieldName, ok := strings.Cut(file.Update.Field, ".")
	if !ok {
		return nil, nil
	}
	for _, list := range d.Spec.Fields {
		if list.JSON != listName || list.object == nil || !strings.HasPrefix(list.GoType, "[]") {
			continue
		}
		keys := listMapKeys(list)
		if len(keys) != 1 {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.field", msg: "a list item update needs one listMapKey"}
		}
		var target, key *crField
		for _, f := range list.object.Fields {
			if f.JSON == fieldName {
				target = f
			}
			if f.JSON == keys[0] {
				key = f
			}
		}
		if target == nil || key == nil {
			return nil, nil
		}
		if target == key {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.field", msg: "the update cannot change a listMapKey"}
		}
		if target.synthetic || target.immutable || target.enum != nil || target.ref != "" || target.typ.Pattern != "" || target.typ.WireString ||
			!slices.Contains([]model.Kind{model.String, model.Bool, model.Integer}, target.typ.Kind) {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.field", msg: "the list item field is not a mutable string, bool, or integer"}
		}
		values, ok := full.get(list.JSON).([]any)
		if !ok || len(values) == 0 {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.field", msg: "the full spec needs one list item"}
		}
		item := values[0]
		if _, ok := objectValue(item, key.JSON).(string); !ok {
			return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.field", msg: "the list-map key needs a string value"}
		}
		value, err := updateValue(target, objectValue(item, target.JSON), file, false)
		if err != nil {
			return nil, err
		}
		listRef, err := ix.field(resourceName, list.API)
		if err != nil {
			return nil, err
		}
		itemType := strings.TrimPrefix(strings.TrimPrefix(listRef.Want, "[]"), "*")
		keyRef, err := ix.field(itemType, key.API)
		if err != nil {
			return nil, err
		}
		fieldRef, err := ix.field(itemType, target.API)
		if err != nil {
			return nil, err
		}
		match := "item." + keyRef.Name
		if strings.HasPrefix(keyRef.Want, "*") {
			match = "ptr.Deref(" + match + ", \"\")"
		}
		got := "item." + fieldRef.Name
		if strings.HasPrefix(fieldRef.Want, "*") {
			got = fmt.Sprintf("ptr.Deref(%s, %s)", got, zeroLiteral(target.typ))
		}
		set := value
		if target.Optional {
			set = "ptr.To(" + value + ")"
		}
		read := "got."
		if d.GetResponseField != "" {
			read += d.GetResponseField + "."
		}
		read += listRef.Name
		return &updateData{Field: list.GoName + "[0]." + target.GoName, Set: set, Want: value, Got: got,
			ListGet: read, Match: match + " == updated.Spec." + list.GoName + "[0]." + key.GoName}, nil
	}
	return nil, nil
}

// overwriteStep builds the e2e test of an overwrite resource: the list field, the name of an item,
// and the update step, the update line of e2e.yaml or the first mutable string, bool, or integer of
// an item other than its name. A path of e2e.yaml names a field of an item without the list.
func overwriteStep(d *kindData, ix *refIndex, file *e2eFile, full *omap) (*overwriteData, error) {
	var list *crField
	for _, f := range d.Spec.Fields {
		if f.object != nil && strings.HasPrefix(f.GoType, "[]") {
			if list != nil {
				return nil, unsupported("spec", "the e2e test of an overwrite resource needs one list of objects")
			}
			list = f
		}
	}
	if list == nil {
		return nil, unsupported("spec", "the e2e test of an overwrite resource needs a list of objects")
	}
	resource, err := ix.typeRef("fields")
	if err != nil {
		return nil, err
	}
	listGet, err := ix.field(resource.Name, list.API)
	if err != nil {
		return nil, err
	}
	itemGet := strings.TrimPrefix(strings.TrimPrefix(listGet.Want, "[]"), "*")
	d.ItemGetType = itemGet
	out := &overwriteData{List: list.GoName, ListGet: listGet.Name}
	var item any
	if values, ok := objectValue(full, list.JSON).([]any); ok && len(values) != 0 {
		item = values[0]
	}
	for _, f := range list.object.Fields {
		if f.API != "name" || f.typ.Kind != model.String || f.joinedBy != "" {
			continue
		}
		got, err := ix.field(itemGet, f.API)
		if err != nil {
			return nil, err
		}
		out.Name, out.NamePtr, out.NameGet, out.NameGetPtr = f.GoName, f.Optional, got.Name, strings.HasPrefix(got.Want, "*")
	}
	if out.Name == "" {
		return nil, unsupported(list.JSON, "the e2e test of an overwrite resource needs a string name in each item")
	}
	for _, f := range list.object.Fields {
		if file.Update != nil && file.Update.Field != f.JSON {
			continue
		}
		if f.GoName == out.Name || f.immutable || f.enum != nil || f.intNames != nil || f.ref != "" || f.joinedBy != "" || f.inlineParent != "" || f.typ.Pattern != "" ||
			!slices.Contains([]model.Kind{model.String, model.Bool, model.Integer}, f.typ.Kind) {
			if file.Update != nil {
				return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.field",
					msg: fmt.Sprintf("%s is not a mutable string, bool, or integer of an item", f.JSON)}
			}
			continue
		}
		value, err := updateValue(f, objectValue(item, f.JSON), file, false)
		if err != nil {
			return nil, err
		}
		got, err := ix.field(itemGet, f.API)
		if err != nil {
			return nil, err
		}
		read := "item." + got.Name
		if strings.HasPrefix(got.Want, "*") {
			read = fmt.Sprintf("ptr.Deref(%s, %s)", read, zeroLiteral(f.typ))
		}
		set := value
		if f.Optional {
			set = "ptr.To(" + value + ")"
		}
		out.Field, out.FieldGet, out.Set, out.Want, out.Got = f.GoName, got.Name, set, value, read
		return out, nil
	}
	if file.Update != nil {
		return nil, &shapeError{code: codeE2EUnused, path: e2eFileName + ":update.field", msg: "an item has no such field"}
	}
	return nil, &shapeError{code: codeE2ERequired, path: e2eFileName + ":update",
		msg: "an item has no mutable string, bool, or integer that the update step can change; set update in " + e2eFileName}
}

// contentUpdateStep sets json to new content, and compares one property of the resource.
func contentUpdateStep(file *e2eFile) (*updateData, error) {
	if file.Update == nil || file.Update.Field != "json" || file.Update.Check == "" {
		return nil, &shapeError{code: codeE2ERequired, path: e2eFileName + ":update",
			msg: "a kind with the content key needs update: {field: json, value: <JSON text>, check: <resource property>}"}
	}
	text, ok := file.Update.Value.(string)
	var parsed map[string]any
	if !ok || json.Unmarshal([]byte(text), &parsed) != nil {
		return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.value", msg: "the value must be a JSON object as text"}
	}
	if _, ok := parsed[file.Update.Check]; !ok {
		return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.check", msg: "the value has no such property"}
	}
	value := strconv.Quote(text)
	return &updateData{Field: "Json", Set: "ptr.To(" + value + ")", Want: value, Check: file.Update.Check}, nil
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

// explicitKey reports whether a value of e2e.yaml sets the key of the items at a path: a fixed
// value, or a value of a when entry. Such a key is sent as it is. Any other key is made up.
func (v *sampleValues) explicitKey(path, key string) bool {
	if _, ok := v.layerValue(path + "." + key); ok {
		return true
	}
	_, ok := v.fixed(path + "." + key)
	return ok
}

// uniqueKey makes the key of a new item differ from the keys of the items before it. A made-up
// string key gets a suffix. A key that e2e.yaml sets is never changed, so the item reports false
// and waits for another copy. A padded item (pad) is a copy that the contract minimum needs: its
// key is always made up. It also reports false when the key cannot be made unique: an enum key
// that repeats, or a string key with a pattern or a length limit that the suffix would break.
func (v *sampleValues) uniqueKey(f *crField, path string, items []any, item any, pad bool) bool {
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
		if !ok || f.object == nil || (!pad && v.explicitKey(path, key)) {
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

// optionalGroupFields reports whether value is an object of optional fields of the group.
func optionalGroupFields(group *crField, value any) bool {
	fields, ok := value.(map[string]any)
	if !ok || len(fields) == 0 || group.object == nil {
		return false
	}
	for name := range fields {
		found := false
		for _, f := range group.object.Fields {
			if f.JSON == name && f.Optional {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// minLength is the lower length limit of a string type, 0 for none.
func minLength(t *model.Type) int {
	if t.MinLength == nil {
		return 0
	}
	return int(*t.MinLength)
}

// maxLength is the upper length limit of a string type, -1 for none.
func maxLength(t *model.Type) int64 {
	if t.MaxLength == nil {
		return -1
	}
	return *t.MaxLength
}

// hasValue reports whether e2e.yaml or a sweep layer sets the value at a path.
func (v *sampleValues) hasValue(path string) bool {
	if _, ok := v.layerValue(path); ok {
		return true
	}
	_, ok := v.fixed(path)
	return ok
}

package generator

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
)

// Issue codes of the coverage of the e2e tests.
const (
	codeE2ECoverageMissing = "E2E_COVERAGE_MISSING"
	codeE2ESkipUnused      = "E2E_SKIP_UNUSED"
)

const (
	// A step makes copies of the spec until a copy adds no new value or arm.
	maxCopiesPerStep = 10
	// A list takes items while each item adds coverage.
	maxItemsPerList = 12
	// A step takes at most this many extra list items over all its lists.
	maxExtraItemsPerStep = 20
	// The sweep stops with an error after this many steps.
	maxSweepSteps = 100
	// An uncovered oneOf arm always wins over an arm with only uncovered values below it.
	uncoveredArmWeight = 1000
)

// coverageKey names one thing that a test must send: an accepted enum value, or a oneOf arm. The
// component is the contract component that holds it, so a component at two CR paths counts once.
type coverageKey struct {
	kind      string // "enum" or "arm"
	component string
	name      string // the API value of an enum, or the API property of an arm
}

// coverage is the set of enum values and oneOf arms of the spec tree, and the part that tests send.
// It reads the same crType tree that renders the types and the CRD.
type coverage struct {
	universe    []coverageKey
	label       map[coverageKey]string
	covered     map[coverageKey]bool
	skipped     map[coverageKey]bool
	owner       map[*crType]string // the contract component of each struct
	whens       []*whenEntry
	coupled     map[string]bool // the CR paths that a when entry matches
	coupledKeys map[coverageKey]bool
	// live holds the keys that a test can reach: the ones that are not only below a skipped arm.
	live map[coverageKey]bool
	// sent collects the keys that the spec in the making sends. It is nil when no spec is collected.
	sent map[coverageKey]bool
	// extra counts the list items that the current step added beyond the first.
	extra int
}

func newCoverage(d *kindData, file *e2eFile) (*coverage, error) {
	c := &coverage{coupled: map[string]bool{}, coupledKeys: map[coverageKey]bool{}, label: map[coverageKey]string{}, covered: map[coverageKey]bool{}, skipped: map[coverageKey]bool{}, owner: map[*crType]string{}}
	c.walk(d.Spec, d.Kind, "", map[*crType]bool{})
	if err := c.readSkips(file); err != nil {
		return nil, err
	}
	c.live = map[coverageKey]bool{}
	c.markLive(d.Spec, map[*crType]bool{})
	if err := c.parseWhen(d.Spec, file.When); err != nil {
		return nil, err
	}
	c.markCoupledKeys(d.Spec)
	if err := c.checkWhenCoverage(); err != nil {
		return nil, err
	}
	return c, nil
}

// markLive records the keys that the sweep can reach. It does not enter a skipped arm, so a skip of
// an arm needs no second skip for the values below it.
func (c *coverage) markLive(t *crType, seen map[*crType]bool) {
	if seen[t] {
		return
	}
	seen[t] = true
	component := c.owner[t]
	cut := map[string]bool{}
	for _, g := range t.groups {
		for _, arm := range g.Arms {
			c.live[coverageKey{"arm", component, arm}] = true
			cut[arm] = c.skipped[coverageKey{"arm", component, arm}]
		}
	}
	for _, f := range t.Fields {
		if f.synthetic || cut[f.API] {
			continue
		}
		if f.enum != nil {
			for _, v := range f.enum.Values {
				c.live[coverageKey{"enum", f.enum.schema, v.API}] = true
			}
		}
		if f.object != nil {
			c.markLive(f.object, seen)
		}
	}
}

// walk lists the enum values and the oneOf arms below a struct. A struct that two CR paths share is visited once.
func (c *coverage) walk(t *crType, parentComponent, path string, seen map[*crType]bool) {
	if seen[t] {
		return
	}
	seen[t] = true
	component := t.schema
	if component == "" {
		component = parentComponent // the spec, or the struct of a released group
	}
	c.owner[t] = component
	for _, g := range t.groups {
		for _, arm := range g.Arms {
			key := coverageKey{"arm", component, arm}
			c.add(key, fmt.Sprintf("the oneOf arm %s of %s (component %s)", arm, pathLabel(path), component))
		}
	}
	for _, f := range t.Fields {
		if f.synthetic {
			continue
		}
		child := path + f.JSON
		if f.enum != nil {
			for _, v := range f.enum.Values {
				if v.API == enumZero(f) {
					continue
				}
				c.add(coverageKey{"enum", f.enum.schema, v.API}, fmt.Sprintf("the value %s of %s (component %s)", v.API, child, f.enum.schema))
			}
		}
		if f.object != nil {
			c.walk(f.object, component, child+".", seen)
		}
	}
}

func (c *coverage) add(key coverageKey, label string) {
	if _, ok := c.label[key]; ok {
		return
	}
	c.universe = append(c.universe, key)
	c.label[key] = label
}

func enumZero(f *crField) string {
	if f.typ == nil {
		return ""
	}
	if f.typ.Kind == model.Enum {
		return f.typ.EnumZero
	}
	if f.typ.Elem != nil {
		return f.typ.Elem.EnumZero
	}
	return ""
}

func armJSON(t *crType, api string) string {
	for _, f := range t.Fields {
		if f.API == api {
			return f.JSON
		}
	}
	return api
}

func pathLabel(path string) string {
	if path == "" {
		return "the spec"
	}
	return strings.TrimSuffix(path, ".")
}

// readSkips marks the skipValues and skipChoices of e2e.yaml. A skip names a thing that the tree has.
func (c *coverage) readSkips(file *e2eFile) error {
	for _, component := range sortedKeys(file.SkipValues) {
		for _, value := range file.SkipValues[component].Values {
			key := coverageKey{"enum", component, value}
			if _, ok := c.label[key]; !ok {
				return &shapeError{code: codeE2ESkipUnused, path: e2eFileName + ":skipValues." + component, msg: fmt.Sprintf("the spec has no enum %s with the accepted value %s", component, value)}
			}
			c.skipped[key] = true
		}
	}
	for _, component := range sortedKeys(file.SkipChoices) {
		for _, arm := range file.SkipChoices[component].Choices {
			key := coverageKey{"arm", component, arm}
			if _, ok := c.label[key]; !ok {
				return &shapeError{code: codeE2ESkipUnused, path: e2eFileName + ":skipChoices." + component, msg: fmt.Sprintf("the spec has no oneOf in %s with the arm %s", component, arm)}
			}
			c.skipped[key] = true
		}
	}
	return nil
}

// mark records that a test sends the key.
func (c *coverage) mark(key coverageKey) {
	c.covered[key] = true
	if c.sent != nil {
		c.sent[key] = true
	}
}

// startSpec begins to collect the keys that one spec sends.
func (c *coverage) startSpec() { c.sent = map[coverageKey]bool{} }

// finishSpec returns the keys that the spec sent, in contract order.
func (c *coverage) finishSpec() []coverageKey {
	var keys []coverageKey
	for _, key := range c.universe {
		if c.sent[key] {
			keys = append(keys, key)
		}
	}
	c.sent = nil
	return keys
}

// literal records what a fixed value of e2e.yaml really sends. A fixed value never exempts the
// other values below it: the guard still names them.
func (c *coverage) literal(f *crField, value any) {
	for _, key := range c.literalKeys(f, value) {
		c.mark(key)
	}
}

// literalKeys lists the enum values and the oneOf arms that a literal value sends.
func (c *coverage) literalKeys(f *crField, value any) []coverageKey {
	var keys []coverageKey
	if f.enum != nil {
		for _, name := range literalStrings(value) {
			for _, v := range f.enum.Values {
				if v.Name == name {
					keys = append(keys, coverageKey{"enum", f.enum.schema, v.API})
				}
			}
		}
		return keys
	}
	if f.object == nil {
		return nil
	}
	switch x := value.(type) {
	case []any:
		for _, item := range x {
			keys = append(keys, c.literalObjectKeys(f.object, item)...)
		}
	case map[string]any:
		if f.typ != nil && f.typ.Kind == model.Map {
			for _, item := range x {
				keys = append(keys, c.literalObjectKeys(f.object, item)...)
			}
			return keys
		}
		keys = append(keys, c.literalObjectKeys(f.object, x)...)
	}
	return keys
}

func (c *coverage) literalObjectKeys(t *crType, value any) []coverageKey {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	var keys []coverageKey
	component := c.owner[t]
	for _, g := range t.groups {
		for _, arm := range g.Arms {
			if _, set := object[armJSON(t, arm)]; set {
				keys = append(keys, coverageKey{"arm", component, arm})
			}
		}
	}
	for _, f := range t.Fields {
		if child, ok := object[f.JSON]; ok {
			keys = append(keys, c.literalKeys(f, child)...)
		}
	}
	return keys
}

func literalStrings(value any) []string {
	switch x := value.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// open reports whether a test can still send the key: it is not covered and not skipped.
func (c *coverage) open(key coverageKey) bool {
	return !c.covered[key] && !c.skipped[key]
}

// openBelow counts the open keys of a field: its enum values and everything below it.
func (c *coverage) openBelow(f *crField, seen map[*crType]bool) int {
	n := 0
	if f.enum != nil {
		for _, v := range f.enum.Values {
			if v.API != enumZero(f) && c.open(coverageKey{"enum", f.enum.schema, v.API}) {
				n++
			}
		}
	}
	if f.object != nil {
		n += c.openInStruct(f.object, seen)
	}
	return n
}

func (c *coverage) openInStruct(t *crType, seen map[*crType]bool) int {
	if seen[t] {
		return 0
	}
	seen[t] = true
	n := 0
	for _, g := range t.groups {
		for _, arm := range g.Arms {
			if c.open(coverageKey{"arm", c.owner[t], arm}) {
				n++
			}
		}
	}
	for _, f := range t.Fields {
		if !f.synthetic {
			n += c.openBelow(f, seen)
		}
	}
	return n
}

// pickArm returns the API property of the arm that a copy sets. An open arm always wins. Then the
// arm with the most open values below it wins, so no branch starves. A tie takes the first arm.
func (c *coverage) pickArm(t *crType, g model.OneOfGroup) string {
	best, bestScore := "", -1
	for _, arm := range g.Arms {
		key := coverageKey{"arm", c.owner[t], arm}
		if c.skipped[key] {
			continue
		}
		score := 0
		if !c.covered[key] {
			score = uncoveredArmWeight
		}
		if i := slices.IndexFunc(t.Fields, func(f *crField) bool { return f.API == arm }); i >= 0 {
			score += c.openBelow(t.Fields[i], map[*crType]bool{})
		}
		if score > bestScore {
			best, bestScore = arm, score
		}
	}
	return best
}

// missing lists what no test sends, in contract order.
func (c *coverage) missing() []string {
	var out []string
	for _, key := range c.universe {
		if c.live[key] && !c.covered[key] && !c.skipped[key] {
			out = append(out, c.label[key])
		}
	}
	return out
}

// guard fails when an accepted value or arm is neither sent nor skipped.
func (c *coverage) guard() error {
	missing := c.missing()
	if len(missing) == 0 {
		return nil
	}
	return &shapeError{code: codeE2ECoverageMissing, path: e2eFileName,
		msg: fmt.Sprintf("no test sends %d accepted values or arms.\nFor each, add a when entry, or add it to skipValues or skipChoices with a reason:\n  %s", len(missing), strings.Join(missing, "\n  "))}
}

// sweepCopy is one spec of a step, with the enum values and the oneOf arms that it sends.
type sweepCopy struct {
	spec *omap
	keys []coverageKey
}

// sweep builds the extra steps. Step 0, the full spec, has already been recorded. Each step makes
// copies of the spec until a copy adds nothing. The sweep stops at the first step that adds nothing.
func (c *coverage) sweep(v *sampleValues, d *kindData) ([][]sweepCopy, error) {
	var steps [][]sweepCopy
	for len(c.missing()) != 0 {
		if len(steps) == maxSweepSteps {
			return nil, &shapeError{code: codeE2ECoverageMissing, path: e2eFileName, msg: fmt.Sprintf("the sweep did not finish in %d steps", maxSweepSteps)}
		}
		c.extra = 0
		var step []sweepCopy
		for len(step) < maxCopiesPerStep {
			before := len(c.covered)
			global, items := c.planCopy()
			v.layers, v.items = []*overlay{global}, items
			c.startSpec()
			spec, err := v.object(d.Spec, "", true)
			keys := c.finishSpec()
			v.layers, v.items = nil, nil
			if err != nil {
				return nil, err
			}
			if len(c.covered) == before {
				break
			}
			step = append(step, sweepCopy{spec: spec, keys: keys})
		}
		if len(step) == 0 {
			break // the rest cannot be sent: the guard names it
		}
		steps = append(steps, step)
	}
	return steps, nil
}

// expectations returns the Go string literals that a Create request must hold for the keys: the
// API value of an enum as a JSON string, and the API property of an arm as a JSON key.
func (c *coverage) expectations(keys []coverageKey) []string {
	var out []string
	for _, key := range keys {
		if key.kind == "arm" {
			out = append(out, strconv.Quote(`"`+key.name+`":`))
		} else {
			out = append(out, strconv.Quote(`"`+key.name+`"`))
		}
	}
	return out
}

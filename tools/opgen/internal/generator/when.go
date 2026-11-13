package generator

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
)

// Issue codes of the when key of e2e.yaml.
const (
	codeE2EWhenMissing  = "E2E_WHEN_MISSING"
	codeE2EWhenUnused   = "E2E_WHEN_UNUSED"
	codeE2EWhenConflict = "E2E_WHEN_CONFLICT"
	codeE2EWhenInvalid  = "E2E_WHEN_INVALID"
)

// e2eWhen gives the extra values that an enum value or a oneOf arm needs, such as the parent preset
// of a connector type. Paths are CR paths without an index, and values are CR spellings. A match
// value is a string, or a list of strings of which any one matches. A match on a oneOf names the
// field that holds it and the arm.
type e2eWhen struct {
	Match map[string]any `yaml:"match"`
	Set   map[string]any `yaml:"set"`
}

// whenEntry is a parsed when entry.
type whenEntry struct {
	index  int
	anchor string // the list that holds the matched fields, "" for the whole CR
	// anchorMax is the contract limit of the items of the anchor list, 0 for none.
	anchorMax int
	choices   []whenChoice
	set       map[string]any
	fields    map[string]*crField // the field of each set path
}

type whenChoice struct {
	path    string
	arm     bool
	options []whenOption
}

type whenOption struct {
	name string // the CR spelling of an enum value, or the CR name of an arm
	arm  string // the API property of an arm
	key  coverageKey
	// field is the field of an arm. The values below it stay open after the arm is covered.
	field *crField
}

// overlay holds the values and the arms that one copy, or one list item, forces.
type overlay struct {
	values map[string]any
	arms   map[string]string
}

func newOverlay() *overlay { return &overlay{values: map[string]any{}, arms: map[string]string{}} }

// conflicts reports whether the other overlay forces something that this one forces differently.
// The same path with the same value or the same arm is no conflict. A value and an arm at one
// path conflict, and so do a path and a path below it: a literal object at a path replaces what
// an entry sets or matches inside it.
func (o *overlay) conflicts(other *overlay) bool {
	type forced struct {
		path  string
		value any
		arm   string
		isArm bool
	}
	list := func(x *overlay) []forced {
		var out []forced
		for path, value := range x.values {
			out = append(out, forced{path: path, value: value})
		}
		for path, arm := range x.arms {
			out = append(out, forced{path: path, arm: arm, isArm: true})
		}
		return out
	}
	for _, a := range list(o) {
		for _, b := range list(other) {
			switch {
			case a.path == b.path:
				if a.isArm != b.isArm || (a.isArm && a.arm != b.arm) || (!a.isArm && !reflect.DeepEqual(a.value, b.value)) {
					return true
				}
			case strings.HasPrefix(a.path, b.path+".") || strings.HasPrefix(b.path, a.path+"."):
				return true
			}
		}
	}
	return false
}

func (o *overlay) merge(other *overlay) {
	for path, value := range other.values {
		o.values[path] = value
	}
	for path, arm := range other.arms {
		o.arms[path] = arm
	}
}

// resolvePath finds the field at a CR path and the lists above it.
func resolvePath(root *crType, path string) (field *crField, lists []string, ok bool) {
	t := root
	segments := strings.Split(path, ".")
	for i, segment := range segments {
		if t == nil {
			return nil, nil, false
		}
		j := slices.IndexFunc(t.Fields, func(f *crField) bool { return f.JSON == segment })
		if j < 0 {
			return nil, nil, false
		}
		field = t.Fields[j]
		if i < len(segments)-1 {
			if field.typ != nil && (field.typ.Kind == model.List || field.typ.Kind == model.Set) {
				lists = append(lists, strings.Join(segments[:i+1], "."))
			}
			t = field.object
		}
	}
	return field, lists, field != nil
}

// parseWhen reads the when entries against the type tree. A path must name a field, and a match value an accepted value or arm.
func (c *coverage) parseWhen(root *crType, entries []e2eWhen) error {
	for i, entry := range entries {
		where := fmt.Sprintf("%s:when[%d]", e2eFileName, i)
		if len(entry.Match) == 0 {
			return &shapeError{code: codeE2EWhenInvalid, path: where, msg: "match needs at least one path"}
		}
		parsed := &whenEntry{index: i, set: entry.Set, fields: map[string]*crField{}}
		var lists [][]string
		var paths []string
		for _, path := range sortedKeys(entry.Match) {
			field, above, ok := resolvePath(root, path)
			if !ok {
				return &shapeError{code: codeE2EWhenUnused, path: where + ".match." + path, msg: "the spec has no such field"}
			}
			choice, err := c.parseChoice(field, path, entry.Match[path], where+".match."+path)
			if err != nil {
				return err
			}
			parsed.choices = append(parsed.choices, choice)
			lists = append(lists, above)
			paths = append(paths, path)
		}
		parsed.anchor = commonList(lists[0], paths)
		if parsed.anchor != "" {
			if list, _, ok := resolvePath(root, parsed.anchor); ok && list.typ != nil && list.typ.MaxItems != nil {
				parsed.anchorMax = int(*list.typ.MaxItems)
			}
		}
		for _, path := range sortedKeys(entry.Set) {
			field, _, ok := resolvePath(root, path)
			if !ok {
				return &shapeError{code: codeE2EWhenUnused, path: where + ".set." + path, msg: "the spec has no such field"}
			}
			if slices.Contains(paths, path) {
				return &shapeError{code: codeE2EWhenConflict, path: where + ".set." + path, msg: "set repeats a path of match with another value"}
			}
			if parsed.anchor != "" && !strings.HasPrefix(path, parsed.anchor+".") {
				return &shapeError{code: codeE2EWhenInvalid, path: where + ".set." + path,
					msg: fmt.Sprintf("the match is inside the items of %s: set only paths inside the same item", parsed.anchor)}
			}
			parsed.fields[path] = field
		}
		c.whens = append(c.whens, parsed)
		for _, choice := range parsed.choices {
			c.coupled[choice.path] = true
		}
	}
	return nil
}

// commonList returns the deepest list of the first path that holds every match path, "" for none.
func commonList(lists, paths []string) string {
	best := ""
	for _, list := range lists {
		if len(list) > len(best) && !slices.ContainsFunc(paths, func(p string) bool { return !strings.HasPrefix(p, list+".") }) {
			best = list
		}
	}
	return best
}

func (c *coverage) parseChoice(f *crField, path string, raw any, where string) (whenChoice, error) {
	var names []string
	switch x := raw.(type) {
	case string:
		names = []string{x}
	case []any:
		for _, item := range x {
			s, ok := item.(string)
			if !ok {
				return whenChoice{}, &shapeError{code: codeE2EWhenInvalid, path: where, msg: "a match value is a string or a list of strings"}
			}
			names = append(names, s)
		}
	}
	if len(names) == 0 {
		return whenChoice{}, &shapeError{code: codeE2EWhenInvalid, path: where, msg: "a match value is a string or a list of strings"}
	}
	choice := whenChoice{path: path}
	switch {
	case f.enum != nil && f.typ != nil && f.typ.Kind == model.Enum:
		for _, name := range names {
			i := slices.IndexFunc(f.enum.Values, func(v enumValue) bool { return v.Name == name && v.API != enumZero(f) })
			if i < 0 {
				return whenChoice{}, &shapeError{code: codeE2EWhenUnused, path: where, msg: fmt.Sprintf("%q is not an accepted CR value of %s", name, f.enum.schema)}
			}
			choice.options = append(choice.options, whenOption{name: name, key: coverageKey{"enum", f.enum.schema, f.enum.Values[i].sent()}})
		}
	case f.object != nil && len(f.object.groups) != 0:
		choice.arm = true
		for _, name := range names {
			i := slices.IndexFunc(f.object.Fields, func(a *crField) bool {
				return a.JSON == name && slices.ContainsFunc(f.object.groups, func(g model.OneOfGroup) bool { return slices.Contains(g.Arms, a.API) })
			})
			if i < 0 {
				return whenChoice{}, &shapeError{code: codeE2EWhenUnused, path: where, msg: fmt.Sprintf("%q is not an arm of the oneOf at %s", name, path)}
			}
			api := f.object.Fields[i].API
			choice.options = append(choice.options, whenOption{name: name, arm: api, key: coverageKey{"arm", c.owner[f.object], api}, field: f.object.Fields[i]})
		}
	default:
		return whenChoice{}, &shapeError{code: codeE2EWhenInvalid, path: where, msg: "a match path names an enum field or the field that holds a oneOf"}
	}
	return choice, nil
}

// checkWhenCoverage fails when a coupled path has an accepted value or arm that no entry names and no skip covers.
func (c *coverage) checkWhenCoverage() error {
	named := map[coverageKey]bool{}
	for _, entry := range c.whens {
		for _, choice := range entry.choices {
			for _, option := range choice.options {
				named[option.key] = true
			}
		}
	}
	var missing []string
	for _, key := range c.universe {
		if c.coupledKeys[key] && !named[key] && !c.skipped[key] {
			missing = append(missing, c.label[key])
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return &shapeError{code: codeE2EWhenMissing, path: e2eFileName,
		msg: fmt.Sprintf("a when entry matches the field of %d accepted values or arms below, so each one needs its own when entry or a skip:\n  %s", len(missing), strings.Join(missing, "\n  "))}
}

// markCoupledKeys lists the keys of the fields that a when entry matches, whatever entry names them.
func (c *coverage) markCoupledKeys(root *crType) {
	for path := range c.coupled {
		field, _, ok := resolvePath(root, path)
		if !ok {
			continue
		}
		if field.enum != nil {
			for _, v := range field.enum.Values {
				if v.API != enumZero(field) {
					c.coupledKeys[coverageKey{"enum", field.enum.schema, v.API}] = true
				}
			}
			continue
		}
		for _, g := range field.object.groups {
			for _, arm := range g.Arms {
				c.coupledKeys[coverageKey{"arm", c.owner[field.object], arm}] = true
			}
		}
	}
}

// choose picks the option of each match path that an entry uses in a copy, and the keys that the copy then sends.
// The first open option wins. An entry with no usable option gets no overlay.
func (c *coverage) choose(entry *whenEntry, planned map[coverageKey]bool) (assign *overlay, open int, ok bool) {
	assign = newOverlay()
	keys := map[coverageKey]bool{}
	below := 0
	for _, choice := range entry.choices {
		// An open option wins. Among the rest, the arm with the most open values below it wins, so a
		// list of options is worked off until every value below every arm is sent. A tie takes the first.
		var picked *whenOption
		best := -1
		for i := range choice.options {
			option := &choice.options[i]
			if c.skipped[option.key] {
				continue
			}
			score := 0
			if c.open(option.key) && !planned[option.key] {
				score = uncoveredArmWeight
			}
			if option.field != nil {
				score += c.openBelow(option.field, map[*crType]bool{})
			}
			if score > best {
				picked, best = option, score
			}
		}
		if picked == nil {
			return nil, 0, false
		}
		if choice.arm {
			assign.arms[choice.path] = picked.arm
			// Automatic arm choice is off at a coupled path, so the entry is the only way to reach the
			// values below its arm. Keep it in play while any of them is open.
			below += c.openBelow(picked.field, map[*crType]bool{})
		} else {
			assign.values[choice.path] = picked.name
		}
		keys[picked.key] = true
	}
	for path, value := range entry.set {
		assign.values[path] = value
		for _, key := range c.literalKeys(entry.fields[path], value) {
			keys[key] = true
		}
	}
	for key := range keys {
		if c.open(key) && !planned[key] {
			open++
		}
	}
	return assign, open + below, true
}

// planCopy picks the entries that one copy applies. It takes the entry with the most open keys
// first, and skips an entry that forces another value than one already chosen: that entry waits
// for a later copy. An entry in a list gets its own item.
func (c *coverage) planCopy() (global *overlay, items map[string][]*overlay) {
	global, items = newOverlay(), map[string][]*overlay{}
	planned := map[coverageKey]bool{}
	used := map[int]bool{}
	for {
		var best *whenEntry
		var bestAssign *overlay
		bestOpen := 0
		for _, entry := range c.whens {
			if used[entry.index] {
				continue
			}
			assign, open, ok := c.choose(entry, planned)
			if !ok || open == 0 || open <= bestOpen {
				continue
			}
			if entry.anchor == "" && (global.conflicts(assign) || conflictsWithAnyItem(items, assign)) {
				continue
			}
			if entry.anchor != "" && global.conflicts(assign) {
				continue // an item overlay that overrides a global entry would drop its match but keep its data
			}
			if entry.anchor != "" && len(items[entry.anchor]) >= entry.itemLimit() {
				continue // the list is full: the entry waits for another copy
			}
			best, bestAssign, bestOpen = entry, assign, open
		}
		if best == nil {
			return global, items
		}
		used[best.index] = true
		for _, choice := range best.choices {
			for _, option := range choice.options {
				if (choice.arm && bestAssign.arms[choice.path] == option.arm) || (!choice.arm && bestAssign.values[choice.path] == option.name) {
					planned[option.key] = true
				}
			}
		}
		for path, value := range best.set {
			for _, key := range c.literalKeys(best.fields[path], value) {
				planned[key] = true
			}
		}
		if best.anchor == "" {
			global.merge(bestAssign)
		} else {
			items[best.anchor] = append(items[best.anchor], bestAssign)
		}
	}
}

// itemLimit is the number of items that one copy may put into the anchor list: the cap of the
// sweep, or the contract limit when it is lower.
func (e *whenEntry) itemLimit() int {
	if e.anchorMax > 0 && e.anchorMax < maxItemsPerList {
		return e.anchorMax
	}
	return maxItemsPerList
}

// conflictsWithAnyItem reports whether an overlay conflicts with the overlay of a planned list item.
func conflictsWithAnyItem(items map[string][]*overlay, assign *overlay) bool {
	for _, list := range items {
		for _, item := range list {
			if item.conflicts(assign) {
				return true
			}
		}
	}
	return false
}

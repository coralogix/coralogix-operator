package generator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
)

// treeBuilder makes random type trees. Every name is unique, so a recorded key can be found in a spec.
type treeBuilder struct {
	r    *rand.Rand
	n    int
	arms [][2]string // component and arm of every oneOf arm, to pick skips
	maps []listMap   // every list-map, to set explicit keys
}

// listMap is a list-map of a random tree: the field, its key, and the enum field that tells the items apart.
type listMap struct {
	field *crField
	key   string
	mode  *crField
}

// pathOf returns the CR path of a field below a type.
func pathOf(t *crType, prefix string, target *crField) (string, bool) {
	for _, f := range t.Fields {
		if f == target {
			return prefix + f.JSON, true
		}
		if f.object != nil {
			if path, ok := pathOf(f.object, prefix+f.JSON+".", target); ok {
				return path, true
			}
		}
	}
	return "", false
}

func (b *treeBuilder) name(prefix string) string {
	b.n++
	return fmt.Sprintf("%s%d", prefix, b.n)
}

// limits gives a list a minimum and a maximum. A list of enums has no more distinct items than values.
func (b *treeBuilder) limits(f *crField, most int) {
	upper := int64(1 + b.r.Intn(most))
	lower := int64(b.r.Intn(int(upper) + 1))
	f.typ.MaxItems = &upper
	if lower > 0 {
		f.typ.MinItems = &lower
	}
}

func (b *treeBuilder) enum(api string) *crField {
	var values []string
	for i := 0; i < 2+b.r.Intn(3); i++ {
		values = append(values, strings.ToLower(b.name("v")))
	}
	return enumField(api, b.name("Enum"), values...)
}

func (b *treeBuilder) object(depth int) *crType {
	t := &crType{schema: b.name("Obj")}
	for i := 0; i < 1+b.r.Intn(3); i++ {
		t.Fields = append(t.Fields, b.field(depth))
	}
	return t
}

func (b *treeBuilder) oneOf(depth int) *crType {
	t := &crType{schema: b.name("One")}
	var arms []string
	for i := 0; i < 2+b.r.Intn(2); i++ {
		api := b.name("arm")
		arms = append(arms, api)
		t.Fields = append(t.Fields, objectField(api, model.Object, b.object(depth+1)))
		b.arms = append(b.arms, [2]string{t.schema, api})
	}
	t.groups = []model.OneOfGroup{{Arms: arms, AllowNone: b.r.Intn(3) == 0}}
	return t
}

func (b *treeBuilder) field(depth int) *crField {
	api := b.name("f")
	kind := b.r.Intn(7)
	if depth >= 3 && kind >= 2 && kind <= 4 {
		kind = 0
	}
	switch kind {
	case 0:
		return b.enum(api)
	case 1:
		f := enumField(api, b.name("Enum"), strings.ToLower(b.name("v")), strings.ToLower(b.name("v")), strings.ToLower(b.name("v")))
		f.typ = &model.Type{Kind: model.List, Elem: f.typ}
		b.limits(f, 3)
		return f
	case 2:
		return objectField(api, model.Object, b.object(depth+1))
	case 3:
		return objectField(api, model.OneOf, b.oneOf(depth+1))
	case 4:
		f := listOf(objectField(api, model.Object, b.object(depth+1)))
		b.limits(f, 4)
		if b.r.Intn(2) == 0 {
			// A list-map: the items are identified by a string key.
			key := b.name("key")
			mode := b.enum(b.name("mode"))
			f.object.Fields = append([]*crField{stringField(key), mode}, f.object.Fields...)
			f.Markers = []string{"+listType=map", "+listMapKey=" + key}
			b.maps = append(b.maps, listMap{field: f, key: key, mode: mode})
		}
		return f
	case 5:
		f := &crField{API: api, JSON: api, GoName: goName(api), Optional: true, typ: &model.Type{Kind: model.Map, Elem: &model.Type{Kind: model.String}}}
		if b.r.Intn(2) == 0 {
			two := int64(2)
			f.typ.MinItems = &two
		}
		return f
	}
	return stringField(api)
}

// checkValue walks a spec value along its type and reports every breach of a contract limit: the
// size of a list or a map, and the arms of a oneOf group.
func checkValue(t *crType, value any, path string, report func(string)) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	for _, g := range t.groups {
		n := 0
		for _, arm := range g.Arms {
			if _, set := object[armJSON(t, arm)]; set {
				n++
			}
		}
		if n > 1 || (n == 0 && !g.AllowNone) {
			report(fmt.Sprintf("%s: %d arms of a group that allows %s", path, n, map[bool]string{true: "at most one", false: "exactly one"}[g.AllowNone]))
		}
	}
	for _, f := range t.Fields {
		child, ok := object[f.JSON]
		if !ok {
			continue
		}
		switch f.typ.Kind {
		case model.List, model.Set:
			items, _ := child.([]any)
			if f.typ.MinItems != nil && int64(len(items)) < *f.typ.MinItems {
				report(fmt.Sprintf("%s.%s: %d items, minimum %d", path, f.JSON, len(items), *f.typ.MinItems))
			}
			if f.typ.MaxItems != nil && int64(len(items)) > *f.typ.MaxItems {
				report(fmt.Sprintf("%s.%s: %d items, maximum %d", path, f.JSON, len(items), *f.typ.MaxItems))
			}
			if f.enum != nil {
				seen := map[any]bool{}
				for _, item := range items {
					if seen[item] {
						report(fmt.Sprintf("%s.%s: duplicate enum value %v", path, f.JSON, item))
					}
					seen[item] = true
				}
			}
			if keys := listMapKeys(f); len(keys) != 0 {
				seen := map[string]bool{}
				for _, item := range items {
					object, _ := item.(map[string]any)
					tuple := ""
					for _, key := range keys {
						tuple += fmt.Sprint(object[key]) + "\x00"
					}
					if seen[tuple] {
						report(fmt.Sprintf("%s.%s: the list-map has the key %q twice", path, f.JSON, tuple))
					}
					seen[tuple] = true
				}
			}
			if f.object != nil {
				for _, item := range items {
					checkValue(f.object, item, path+"."+f.JSON, report)
				}
			}
		case model.Map:
			entries, _ := child.(map[string]any)
			if f.typ.MinItems != nil && int64(len(entries)) < *f.typ.MinItems {
				report(fmt.Sprintf("%s.%s: %d entries, minimum %d", path, f.JSON, len(entries), *f.typ.MinItems))
			}
		case model.String, model.Bool, model.Number, model.Integer, model.Enum, model.Object, model.OneOf:
			if f.object != nil {
				checkValue(f.object, child, path+"."+f.JSON, report)
			}
		}
	}
}

// For any tree, every spec that the sweep makes meets the limits of the contract, and every key
// that a spec records is in the spec. These are the properties that the offline and live tests rely on.
func TestSweepSpecsHoldTheInvariantsOnRandomTrees(t *testing.T) {
	for seed := int64(1); seed <= 400; seed++ {
		b := &treeBuilder{r: rand.New(rand.NewSource(seed))}
		spec := &crType{Name: "ThingSpec"}
		for i := 0; i < 1+b.r.Intn(4); i++ {
			spec.Fields = append(spec.Fields, b.field(0))
		}
		d := &kindData{Kind: "Thing", Spec: spec}
		file := &e2eFile{}
		if len(b.arms) > 0 && b.r.Intn(3) == 0 {
			// Skip one arm of a group that has another arm to send.
			pick := b.arms[b.r.Intn(len(b.arms))]
			file.SkipChoices = map[string]e2eSkipChoices{pick[0]: {Choices: []string{pick[1]}, Reason: "random skip"}}
		}
		explicit := b.explicitKeys(spec, file)
		v := &sampleValues{file: file, used: map[string]bool{}}
		full, minimal, steps, err := v.specs(d)
		if err != nil {
			t.Errorf("seed %d: %v", seed, err)
			continue
		}
		var all strings.Builder
		report := func(msg string) { t.Errorf("seed %d: %s", seed, msg) }
		specs := []*omap{full, minimal}
		keysOf := [][]coverageKey{v.fullKeys, nil}
		for _, step := range steps {
			for _, c := range step {
				specs = append(specs, c.spec)
				keysOf = append(keysOf, c.keys)
			}
		}
		for i, s := range specs {
			var decoded map[string]any
			data, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			checkValue(spec, decoded, "spec", report)
			all.Write(data)
			for _, key := range keysOf[i] {
				token := `"` + key.name + `"`
				if key.kind == "enum" {
					token = `"` + strings.ToLower(key.name) + `"`
				}
				if !strings.Contains(string(data), token) {
					report(fmt.Sprintf("spec %d records %v as sent, but %s is not in it: %s", i, key, token, data))
				}
			}
		}
		for _, key := range explicit {
			if renamed := strings.TrimSuffix(key, `"`) + `-`; strings.Contains(all.String(), renamed) {
				t.Errorf("seed %d: a spec changes the explicit key %s", seed, key)
			}
		}
	}
}

// explicitKeys gives some list-maps keys that e2e.yaml sets: a fixed key, or a when entry for each
// mode, all with the same key or each with a key of its own. It returns the JSON of each key
// that a spec must send as it is.
func (b *treeBuilder) explicitKeys(spec *crType, file *e2eFile) []string {
	var sent []string
	for _, m := range b.maps {
		path, ok := pathOf(spec, "", m.field)
		// A list that needs more items than the entries give is padded with copies. A copy gets a
		// key of its own, so only lists without a larger minimum can tell a renamed key from a copy.
		if !ok || b.r.Intn(2) == 0 || (m.field.typ.MinItems != nil && *m.field.typ.MinItems > 1) {
			continue
		}
		if b.r.Intn(3) == 0 {
			value := b.name("fixed-key")
			if file.Values == nil {
				file.Values = map[string]any{}
			}
			file.Values[path+"."+m.key] = value
			sent = append(sent, fmt.Sprintf("%q:%q", m.key, value))
			continue
		}
		shared := b.name("shared-key")
		for _, value := range m.mode.enum.Values {
			key := shared
			if b.r.Intn(2) == 0 {
				key = b.name("own-key")
			}
			file.When = append(file.When, e2eWhen{
				Match: map[string]any{path + "." + m.mode.JSON: value.Name},
				Set:   map[string]any{path + "." + m.key: key},
			})
			sent = append(sent, fmt.Sprintf("%q:%q", m.key, key))
		}
	}
	return sent
}

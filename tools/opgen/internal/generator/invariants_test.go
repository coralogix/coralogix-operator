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
			f.object.Fields = append([]*crField{stringField(key)}, f.object.Fields...)
			f.Markers = []string{"+listType=map", "+listMapKey=" + key}
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
		v := &sampleValues{file: file, used: map[string]bool{}}
		full, minimal, steps, err := v.specs(d)
		if err != nil {
			t.Errorf("seed %d: %v", seed, err)
			continue
		}
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
	}
}

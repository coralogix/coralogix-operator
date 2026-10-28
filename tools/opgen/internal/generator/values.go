package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// Values are spec values by field path, for example config.http.endpoint. A value replaces the
	// whole value that the generator makes up for that field. The path of a field inside a list item
	// or a map value has no index: the value is used for every item.
	Values map[string]any `yaml:"values"`
	// Update is the change of the update step.
	Update *e2eUpdate `yaml:"update"`
}

// e2eUpdate changes one top-level field that Update sends.
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
	return &f, nil
}

// e2eData is what the e2e test template needs.
type e2eData struct {
	FullJSON, MinimalJSON string // Go string literals of the spec as JSON
	NameField             string // the Go field of a top-level string name, "" when there is none
	NamePtr               bool
	Update                *updateData
}

// updateData is the update step as Go code: the field that it changes, the value that it sets,
// the value that the API then returns, and the expression that reads it from the response "got".
type updateData struct {
	Field, Set, Want, Got string
}

// sampleValues makes up the full and the minimal spec of the sample and the e2e test.
type sampleValues struct {
	file *e2eFile
	used map[string]bool
}

// buildValues makes up the full and the minimal spec from the contract and e2e.yaml.
func buildValues(d *kindData, ix *refIndex, file *e2eFile) (full, minimal *omap, e2e *e2eData, err error) {
	v := &sampleValues{file: file, used: map[string]bool{}}
	if full, err = v.object(d.Spec, "", true); err != nil {
		return nil, nil, nil, err
	}
	if minimal, err = v.object(d.Spec, "", false); err != nil {
		return nil, nil, nil, err
	}
	for _, path := range sortedKeys(file.Values) {
		if !v.used[path] {
			return nil, nil, nil, &shapeError{code: codeE2EUnused, path: e2eFileName + ":values." + path, msg: "the spec has no such field"}
		}
	}
	e2e = &e2eData{FullJSON: goString(full.json()), MinimalJSON: goString(minimal.json())}
	for _, f := range d.Spec.Fields {
		if f.API == "name" && f.typ.Kind == model.String {
			e2e.NameField, e2e.NamePtr = f.GoName, f.Optional
		}
	}
	if e2e.Update, err = updateStep(d, ix, full, file, e2e.NameField); err != nil {
		return nil, nil, nil, err
	}
	return full, minimal, e2e, nil
}

// object makes up the value of a struct: every field when full is true, only the required fields
// otherwise. Of each oneOf group, it sets the first arm, when the group needs one or full is true.
func (v *sampleValues) object(t *crType, prefix string, full bool) (*omap, error) {
	skip := map[string]bool{}
	for _, g := range t.groups {
		for i, arm := range g.Arms {
			skip[arm] = i > 0 || (!full && g.AllowNone)
		}
	}
	out := &omap{}
	for _, f := range t.Fields {
		_, inGroup := skip[f.API]
		if skip[f.API] || (!full && f.Optional && !inGroup) {
			continue
		}
		value, err := v.field(f, prefix+f.API, full)
		if err != nil {
			return nil, err
		}
		out.set(f.API, value)
	}
	return out, nil
}

func (v *sampleValues) field(f *crField, path string, full bool) (any, error) {
	if value, ok := v.file.Values[path]; ok {
		v.used[path] = true
		return value, nil
	}
	switch f.typ.Kind {
	case model.List, model.Set:
		item, err := v.item(f, f.typ.Elem, path, full)
		return []any{item}, err
	case model.Map:
		item, err := v.item(f, f.typ.Elem, path, full)
		return (&omap{}).set("key", item), err
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
		return f.enum.Values[0].Name, nil
	case model.Bool:
		return true, nil
	case model.Integer:
		return integerValue(t), nil
	case model.String, model.Number, model.List, model.Set, model.Map:
		// A string, below. The renderer rejects a number and a collection here.
	}
	return stringValue(f.API, t, path)
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

// stringValue makes up a string that meets the length limits and the pattern of the contract.
func stringValue(name string, t *model.Type, path string) (string, error) {
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

// updateStep picks the change of the update step: the update line of e2e.yaml, or the first
// mutable top-level string, bool, or integer that is not in a oneOf group.
func updateStep(d *kindData, ix *refIndex, full *omap, file *e2eFile, nameField string) (*updateData, error) {
	resource, err := ix.typeRef("fields")
	if err != nil {
		return nil, err
	}
	inGroup := map[string]bool{}
	for _, g := range d.Spec.groups {
		for _, arm := range g.Arms {
			inGroup[arm] = true
		}
	}
	for _, f := range d.Spec.Fields {
		if file.Update != nil && file.Update.Field != f.API {
			continue
		}
		if f.immutable || inGroup[f.API] || f.enum != nil || f.typ.Pattern != "" || f.typ.WireString ||
			!slices.Contains([]model.Kind{model.String, model.Bool, model.Integer}, f.typ.Kind) {
			if file.Update != nil {
				return nil, &shapeError{code: codeE2EInvalid, path: e2eFileName + ":update.field",
					msg: fmt.Sprintf("%s is not a mutable top-level string, bool, or integer outside a oneOf", f.API)}
			}
			continue
		}
		value, err := updateValue(f, full, file, nameField)
		if err != nil {
			return nil, err
		}
		got, err := ix.field(resource.Name, f.API)
		if err != nil {
			return nil, err
		}
		read := "got." + got.Name
		if strings.HasPrefix(got.Want, "*") {
			read = fmt.Sprintf("ptr.Deref(%s, %s)", read, zeroLiteral(f.typ))
		}
		set := value
		if f.Optional {
			set = "ptr.To(" + value + ")"
		}
		return &updateData{Field: f.GoName, Set: set, Want: value, Got: read}, nil
	}
	if file.Update != nil {
		return nil, &shapeError{code: codeE2EUnused, path: e2eFileName + ":update.field", msg: "the spec has no such field"}
	}
	return nil, &shapeError{code: codeE2ERequired, path: e2eFileName + ":update",
		msg: "the spec has no mutable top-level string, bool, or integer that the update step can change; set update in " + e2eFileName}
}

// updateValue is the new value of the update step as Go code.
func updateValue(f *crField, full *omap, file *e2eFile, nameField string) (string, error) {
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
	if f.GoName == nameField {
		return "fullName + \"-updated\"", nil
	}
	switch value := full.get(f.API).(type) {
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

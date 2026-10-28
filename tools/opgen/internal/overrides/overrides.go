// Package overrides reads behavior-overrides.yaml of a kind. The file states where a kind differs
// from what the API contract alone gives: a field with a line keeps its released behavior, and a
// field without a line follows the contract.
//
// The reader is strict. An unknown key is an error. A line that matches nothing is an error too
// (OVERRIDE_UNUSED), so a stale line cannot stay.
package overrides

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"io"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	iacoverrides "github.com/coralogix/coralogix-management-sdk/iac/overrides"
)

// FileName is the name of the file in the directory of the kind.
const FileName = "behavior-overrides.yaml"

const (
	// ModeExisting is a kind that users already have. The contract checks of a released resource
	// apply.
	ModeExisting = "existing"
	// ModeNew is a new kind that uses an operator key. The contract checks of a new resource apply.
	ModeNew = "new"
)

// File is the content of behavior-overrides.yaml.
type File struct {
	// Resource is the OpenAPI component of the resource. It must match the resource of the kind.
	Resource string `yaml:"resource"`
	// Mode is "existing" or "new".
	Mode string `yaml:"mode"`
	// Types has one entry per OpenAPI component.
	Types map[string]Type `yaml:"types"`
}

// Type is the lines of one component.
type Type struct {
	Fields map[string]Field `yaml:"fields"`
}

// Field is one field line.
type Field struct {
	// Ref makes a string id of another resource a reference (way 2): the custom resource sets
	// backendRef.id, or resourceRef, the custom resource that manages the other resource.
	Ref *Ref `yaml:"ref"`
}

// Ref is the ref key of a field line.
type Ref struct {
	// Kind is the kind of the custom resource that resourceRef names. Its status id is sent.
	Kind string `yaml:"kind"`
}

// Line is one field line: the component and the field that it names.
type Line struct {
	Component, Field string
}

func (l Line) String() string { return "types." + l.Component + ".fields." + l.Field }

// Parse reads the file. An unknown key or a wrong value is an error.
func Parse(data []byte) (*File, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var f File
	if err := dec.Decode(&f); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s is empty", FileName)
		}
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	if err := f.check(); err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	return &f, nil
}

func (f *File) check() error {
	if f.Resource == "" {
		return errors.New("resource is required")
	}
	if f.Mode != ModeExisting && f.Mode != ModeNew {
		return fmt.Errorf("mode is %q, want %q or %q", f.Mode, ModeExisting, ModeNew)
	}
	for _, line := range f.Lines() {
		field := f.Types[line.Component].Fields[line.Field]
		if field.Ref == nil {
			return fmt.Errorf("%s: the line sets no key", line)
		}
		if !token.IsIdentifier(field.Ref.Kind) || !token.IsExported(field.Ref.Kind) {
			return fmt.Errorf("%s.ref.kind: %q is not a kind", line, field.Ref.Kind)
		}
	}
	return nil
}

// Policy returns the contract rules of the kind: the rules of a released resource in mode
// existing, and the rules of a new resource in mode new. No key of this file is an API key yet.
func (f *File) Policy() model.Policy {
	if f == nil || f.Mode == ModeNew {
		return model.Policy{}
	}
	return iacoverrides.Exceptions{}.Policy()
}

// Lines returns every field line in a stable order.
func (f *File) Lines() []Line {
	if f == nil {
		return nil
	}
	var out []Line
	for component, t := range f.Types {
		for field := range t.Fields {
			out = append(out, Line{Component: component, Field: field})
		}
	}
	slices.SortFunc(out, func(a, b Line) int {
		return strings.Compare(a.Component+"."+a.Field, b.Component+"."+b.Field)
	})
	return out
}

// Ref returns the ref key of component.field, or nil.
func (f *File) Ref(component, field string) *Ref {
	if f == nil {
		return nil
	}
	return f.Types[component].Fields[field].Ref
}

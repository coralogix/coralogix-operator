package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// exclusiveBoundsExact keeps the values that a schema accepts and gives them an inclusive form.
func TestExclusiveBoundsExact(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		want   map[string]any // the keys of the schema after the rewrite
	}{
		{
			name:   "number with a step: the least multiple above the bound",
			schema: `{type: number, multipleOf: 0.001, exclusiveMinimum: 0}`,
			want:   map[string]any{"type": "number", "multipleOf": 0.001, "minimum": 0.001},
		},
		{
			name:   "number with a step: the greatest multiple below the bound",
			schema: `{type: number, multipleOf: 0.5, exclusiveMaximum: 3}`,
			want:   map[string]any{"type": "number", "multipleOf": 0.5, "maximum": 2.5},
		},
		{
			name:   "bound that is not a multiple of the step",
			schema: `{type: number, multipleOf: 0.5, exclusiveMinimum: 0.7, exclusiveMaximum: 2.2}`,
			want:   map[string]any{"type": "number", "multipleOf": 0.5, "minimum": 1, "maximum": 2},
		},
		{
			name:   "integer",
			schema: `{type: integer, format: int32, exclusiveMinimum: 0, exclusiveMaximum: 11}`,
			want:   map[string]any{"type": "integer", "format": "int32", "minimum": 1, "maximum": 10},
		},
		{
			name:   "integer with a bound that is not whole",
			schema: `{type: integer, exclusiveMinimum: 0.5, exclusiveMaximum: 9.5}`,
			want:   map[string]any{"type": "integer", "minimum": 1, "maximum": 9},
		},
		{
			name:   "integer with a fractional step accepts every integer",
			schema: `{type: integer, multipleOf: 0.5, exclusiveMinimum: 0}`,
			want:   map[string]any{"type": "integer", "multipleOf": 0.5, "minimum": 1},
		},
		{
			name:   "integer with a fractional step moves by the integers that are multiples of it",
			schema: `{type: integer, multipleOf: 1.5, exclusiveMinimum: 0}`,
			want:   map[string]any{"type": "integer", "multipleOf": 1.5, "minimum": 3},
		},
		{
			name:   "integer with a whole step",
			schema: `{type: integer, multipleOf: 2, exclusiveMaximum: 10}`,
			want:   map[string]any{"type": "integer", "multipleOf": 2, "maximum": 8},
		},
		{
			name:   "a step finer than float64 at the bound has no safe form",
			schema: `{type: number, multipleOf: 1e-20, exclusiveMinimum: 1}`,
			want:   map[string]any{"type": "number", "multipleOf": 1e-20, "exclusiveMinimum": 1},
		},
		{
			name:   "a step finer than float32 at the bound has no safe form for a float",
			schema: `{type: number, format: float, multipleOf: 1e-8, exclusiveMinimum: 1}`,
			want:   map[string]any{"type": "number", "format": "float", "multipleOf": 1e-8, "exclusiveMinimum": 1},
		},
		{
			name:   "a float32 step that float32 holds is rewritten",
			schema: `{type: number, format: float, multipleOf: 0.5, exclusiveMinimum: 1}`,
			want:   map[string]any{"type": "number", "format": "float", "multipleOf": 0.5, "minimum": 1.5},
		},
		{
			name:   "a bound outside int32 has no safe form",
			schema: `{type: integer, format: int32, exclusiveMinimum: 2147483647}`,
			want:   map[string]any{"type": "integer", "format": "int32", "exclusiveMinimum": 2147483647},
		},
		{
			name:   "a bound that int32 holds is rewritten",
			schema: `{type: integer, format: int32, exclusiveMaximum: 2147483647}`,
			want:   map[string]any{"type": "integer", "format": "int32", "maximum": 2147483646},
		},
		{
			name:   "an int64 bound that float64 cannot hold has no safe form",
			schema: `{type: integer, format: int64, exclusiveMaximum: 9007199254740994}`,
			want:   map[string]any{"type": "integer", "format": "int64", "exclusiveMaximum": 9007199254740994},
		},
		{
			name:   "an int64 bound that float64 holds is rewritten",
			schema: `{type: integer, format: int64, exclusiveMaximum: 9007199254740993}`,
			want:   map[string]any{"type": "integer", "format": "int64", "maximum": 9007199254740992},
		},
		{
			name:   "a step finer than float64 at an edge that float64 cannot hold exactly has no safe form",
			schema: `{type: number, format: double, multipleOf: 1e-18, exclusiveMinimum: 0.1}`,
			want:   map[string]any{"type": "number", "format": "double", "multipleOf": 1e-18, "exclusiveMinimum": 0.1},
		},
		{
			name:   "a step finer than float32 at an edge that float32 cannot hold exactly has no safe form",
			schema: `{type: number, format: float, multipleOf: 1e-9, exclusiveMaximum: 0.1}`,
			want:   map[string]any{"type": "number", "format": "float", "multipleOf": 1e-9, "exclusiveMaximum": 0.1},
		},
		{
			name:   "an edge that float64 cannot hold exactly is rewritten when the step is coarse enough",
			schema: `{type: number, format: double, multipleOf: 0.001, exclusiveMinimum: 0.1}`,
			want:   map[string]any{"type": "number", "format": "double", "multipleOf": 0.001, "minimum": 0.101},
		},
		{
			name:   "OpenAPI 3.0 form",
			schema: `{type: integer, minimum: 5, exclusiveMinimum: true, maximum: 8, exclusiveMaximum: true}`,
			want:   map[string]any{"type": "integer", "minimum": 6, "maximum": 7},
		},
		{
			name:   "OpenAPI 3.0 form that is false changes nothing",
			schema: `{type: integer, minimum: 5, exclusiveMinimum: false}`,
			want:   map[string]any{"type": "integer", "minimum": 5},
		},
		{
			name:   "the tighter of an inclusive and an exclusive bound stays",
			schema: `{type: integer, minimum: 3, exclusiveMinimum: 0, maximum: 4, exclusiveMaximum: 100}`,
			want:   map[string]any{"type": "integer", "minimum": 3, "maximum": 4},
		},
		{
			name:   "a number with no step has no exact form",
			schema: `{type: number, exclusiveMinimum: 0}`,
			want:   map[string]any{"type": "number", "exclusiveMinimum": 0},
		},
		{
			name:   "a string is not a number",
			schema: `{type: string, exclusiveMinimum: 0}`,
			want:   map[string]any{"type": "string", "exclusiveMinimum": 0},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := "components:\n  schemas:\n    Thing: " + test.schema + "\n"
			out, err := exclusiveBoundsExact([]byte(spec))
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Components struct {
					Schemas map[string]map[string]any
				}
			}
			if err := yaml.Unmarshal(out, &got); err != nil {
				t.Fatal(err)
			}
			if schema := got.Components.Schemas["Thing"]; !sameNumbers(schema, test.want) {
				t.Errorf("schema = %v, want %v", schema, test.want)
			}
		})
	}
}

func sameNumbers(got, want map[string]any) bool {
	if len(got) != len(want) {
		return false
	}
	for key, w := range want {
		g, ok := got[key]
		if !ok {
			return false
		}
		if gn, gok := toFloat(g); gok {
			if wn, wok := toFloat(w); wok && gn == wn {
				continue
			}
			return false
		}
		if g != w {
			return false
		}
	}
	return true
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// A contract with no exclusive bound is returned byte for byte, and a property that is named
// exclusiveMinimum is not a bound.
func TestExclusiveBoundsExactLeavesOtherContractsAlone(t *testing.T) {
	plain := []byte("components:\n  schemas:\n    Thing: {type: integer, minimum: 1}\n")
	if out, err := exclusiveBoundsExact(plain); err != nil || !bytes.Equal(out, plain) {
		t.Errorf("a contract with no exclusive bound changed: %q, %v", out, err)
	}
	named := []byte("components:\n  schemas:\n    Thing:\n      type: object\n      properties:\n        exclusiveMinimum: {type: integer}\n")
	if out, err := exclusiveBoundsExact(named); err != nil || !bytes.Equal(out, named) {
		t.Errorf("a property named exclusiveMinimum changed the contract: %q, %v", out, err)
	}
}

// The shared module refuses an exclusive bound. With a step, the generator gives the contract the
// inclusive form that has the same values, so the kind passes the check. Without a step it stays
// refused, with the code of the shared module.
func TestExclusiveBoundsAreRewrittenBeforeTheCheck(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	const field = "ratio: {type: number, format: float, "
	if !bytes.Contains(data, []byte(field)) {
		t.Fatal("the test contract has no ratio field to change")
	}
	check := func(extra string) error {
		contract := filepath.Join(t.TempDir(), "openapi.yaml")
		if err := os.WriteFile(contract, bytes.Replace(data, []byte(field), []byte(field+extra), -1), 0o644); err != nil {
			t.Fatal(err)
		}
		return Check(CheckOptions{Resource: "Thing", OpenAPIPath: contract, E2EPath: filepath.Join("testdata", "root", "tools", "opgen", "kinds", "thing", "e2e.yaml")})
	}
	if err := check("multipleOf: 0.5, exclusiveMinimum: 0, "); err != nil {
		t.Fatalf("an exclusive bound with a step must pass: %v", err)
	}
	if err := check("exclusiveMinimum: 0, "); err == nil || !strings.Contains(err.Error(), "NUMERIC_EXCLUSIVE_BOUND_UNSUPPORTED") {
		t.Fatalf("an exclusive bound with no step must stay refused with NUMERIC_EXCLUSIVE_BOUND_UNSUPPORTED, got %v", err)
	}
}

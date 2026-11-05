package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
)

func TestNumberIsAQuantity(t *testing.T) {
	for _, format := range []string{"float", "double"} {
		goType, err := scalarType(&model.Type{Kind: model.Number, Format: format}, "spec.ratio")
		if err != nil || goType != "resource.Quantity" {
			t.Fatalf("number %s: %q, %v", format, goType, err)
		}
	}
	// The shared module rejects a number without a format first. The renderer checks it too.
	if _, err := scalarType(&model.Type{Kind: model.Number}, "spec.ratio"); err == nil {
		t.Fatal("a number without a format must be rejected")
	}
}

// A Quantity is int-or-string in the CRD. Minimum and Maximum do not apply to that schema.
func TestNumberHasNoBoundMarkers(t *testing.T) {
	low, high := 0.0, 1.0
	typ := &model.Type{Kind: model.Number, Format: "double", Minimum: &low, Maximum: &high}
	if got := limitMarkers(typ); len(got) != 0 {
		t.Fatalf("a Quantity cannot carry numeric bounds: %v", got)
	}
}

func TestNumberConvertsWithAsApproximateFloat64(t *testing.T) {
	typ := &model.Type{Kind: model.Number, Format: "float"}
	b := &builder{}
	for _, tc := range []struct{ want, src, expr string }{
		{"float32", "in.Ratio", "float32(in.Ratio.AsApproximateFloat64())"},
		{"float64", "in.Ratio", "in.Ratio.AsApproximateFloat64()"},
		{"float32", "(*in.Ratio)", "float32((*in.Ratio).AsApproximateFloat64())"},
	} {
		statements, expr, pointer, err := b.value(&crField{typ: typ}, typ, tc.src, tc.want, "spec.ratio")
		if err != nil || statements != "" || expr != tc.expr || pointer {
			t.Errorf("%s from %s: %q, %q, %v, %v", tc.want, tc.src, statements, expr, pointer, err)
		}
	}
	if _, _, _, err := b.value(&crField{typ: typ}, typ, "in.Ratio", "int32", "spec.ratio"); err == nil {
		t.Fatal("an SDK integer field must be rejected")
	}
}

// Kubernetes rejects a number literal as the default of an int-or-string field. The default is a
// string in plain decimal form.
func TestNumberDefaultIsAString(t *testing.T) {
	for value, want := range map[string]string{
		"0.5":   "`0.5`",
		"1e-07": "`0.0000001`",
		"3":     "`3`",
		"-2.50": "`-2.5`",
	} {
		field := &crField{typ: &model.Type{Kind: model.Number, Format: "float"}}
		b := &builder{}
		attrs := &model.Attrs{Default: &value}
		if err := b.contractDefault(field, attrs, "spec.ratio"); err != nil {
			t.Fatalf("default %s: %v", value, err)
		}
		if got := strings.Join(field.Markers, " "); got != "+kubebuilder:default="+want {
			t.Errorf("default %s: got %q", value, got)
		}
	}
	for _, bad := range []string{"NaN", "Inf", "abc"} {
		field := &crField{typ: &model.Type{Kind: model.Number, Format: "float"}}
		if err := (&builder{}).contractDefault(field, &model.Attrs{Default: &bad}, "spec.ratio"); err == nil {
			t.Errorf("default %q must be rejected", bad)
		}
	}
}

// The CRD schema of a Quantity is int-or-string. A fraction as a YAML number fails admission, so
// the sample holds a whole number as a number and a fraction as a decimal string.
func TestNumberSampleValueIsAdmissible(t *testing.T) {
	bound := func(v float64) *float64 { return &v }
	for name, tc := range map[string]struct {
		min, max *float64
		want     any
	}{
		"no bounds":          {nil, nil, int64(1)},
		"whole bound":        {bound(5), bound(10), int64(5)},
		"maximum below one":  {nil, bound(0.5), "0.5"},
		"fractional minimum": {bound(2.5), nil, "2.5"},
		"tiny maximum":       {nil, bound(0.0000001), "0.0000001"},
	} {
		got := numberValue(&model.Type{Kind: model.Number, Format: "float", Minimum: tc.min, Maximum: tc.max})
		if got != tc.want {
			t.Errorf("%s: got %#v, want %#v", name, got, tc.want)
		}
	}
}

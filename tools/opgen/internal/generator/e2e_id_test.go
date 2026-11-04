package generator

import (
	"strings"
	"testing"
)

// The status id is a string. A numeric API id must reach the SDK Get call as its own Go type.
func TestE2EParsesNumericStatusID(t *testing.T) {
	for _, tc := range []struct {
		bits   int
		goType string
		want   []string
	}{
		{0, "", []string{"fullID      string", ") string {", "return id"}},
		{32, "int32", []string{"fullID      int32", ") int32 {", "strconv.ParseInt(id, 10, 32)", "return int32(parsedID)"}},
		{64, "int64", []string{"fullID      int64", ") int64 {", "strconv.ParseInt(id, 10, 64)", "return int64(parsedID)"}},
	} {
		d := &kindData{
			Kind: "Thing", Lower: "thing", GetMethod: "ThingsServiceGetThing",
			SDKAlias: "thingsservice", SDKPath: "example.com/sdk/things", CxsdkPath: "example.com/sdk/cxsdk",
			Module: "example.com/operator", Client: "ThingsServiceAPIService", Accessor: "Things",
			IDBits: tc.bits, IDGoType: tc.goType,
			E2E: &e2eData{FullJSON: "`{}`", MinimalJSON: "`{}`"},
		}
		src, err := execute("e2e_test.go.tmpl", d)
		if err != nil {
			t.Fatalf("%d-bit id: %v", tc.bits, err)
		}
		for _, want := range tc.want {
			if !strings.Contains(string(src), want) {
				t.Errorf("%d-bit id: the e2e test has no %q", tc.bits, want)
			}
		}
		if tc.bits == 0 && strings.Contains(string(src), `"strconv"`) {
			t.Error("a string id needs no strconv import")
		}
	}
}

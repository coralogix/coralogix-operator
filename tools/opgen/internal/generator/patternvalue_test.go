package generator

import (
	"regexp"
	"testing"
)

func TestPatternString(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		minLen  int
		maxLen  int64
	}{
		{`^[1-9][0-9]{0,11}s$`, 0, -1},
		{`^-?[0-9]+$`, 0, -1},
		{`^[a-z]+$`, 5, 10},
		{`^(foo|bar)-[0-9]{2,4}$`, 0, -1},
		{`^[^/]+$`, 3, -1},
		{`^[A-Za-z_][A-Za-z0-9_]*(,[A-Za-z_][A-Za-z0-9_]*)*$`, 0, -1},
	} {
		got, ok := patternString(tc.pattern, tc.minLen, tc.maxLen)
		if !ok {
			t.Errorf("%s: no string", tc.pattern)
			continue
		}
		if !regexp.MustCompile(tc.pattern).MatchString(got) || len(got) < tc.minLen || (tc.maxLen >= 0 && int64(len(got)) > tc.maxLen) {
			t.Errorf("%s: %q does not fit", tc.pattern, got)
		}
	}
	if _, ok := patternString(`^(?=a)b$`, 0, -1); ok {
		t.Error("a pattern that the walk cannot build must give no string")
	}
}

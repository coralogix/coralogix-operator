package generator

import (
	"strings"
	"unicode"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
)

// goName is the exported Go name of an API name: "foo_bar" and "fooBar" become FooBar.
func goName(name string) string { return model.GoName(name) }

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// words splits a Go name at each upper-case letter that starts a word. An acronym stays one word:
// "AIEvaluation" is "AI" and "Evaluation", "ViewFolder" is "View" and "Folder".
func words(name string) []string {
	runes := []rune(name)
	var out []string
	start := 0
	for i := 1; i < len(runes); i++ {
		prev, cur := runes[i-1], runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		lowerToUpper := unicode.IsUpper(cur) && (unicode.IsLower(prev) || unicode.IsDigit(prev))
		acronymEnd := unicode.IsUpper(cur) && unicode.IsUpper(prev) && unicode.IsLower(next)
		if lowerToUpper || acronymEnd {
			out = append(out, string(runes[start:i]))
			start = i
		}
	}
	return append(out, string(runes[start:]))
}

// kebab is the kebab-case name of a kind: ViewFolder is view-folder.
func kebab(kind string) string {
	parts := words(kind)
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, "-")
}

// plural is the resource name of a kind in the API server: the lower-case kind in plural.
// The generated types state it with a resource marker, so the CRD, RBAC, and kustomization use
// the same name.
func plural(kind string) string {
	s := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "z"),
		strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"):
		return s + "es"
	case strings.HasSuffix(s, "y") && len(s) > 1 && !strings.ContainsRune("aeiou", rune(s[len(s)-2])):
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

// enumName is the spelling of an API enum value in a custom resource: the value after the prefix
// of the zero value, in lower camel case. ENTITY_TYPE_PAGER_DUTY_INCIDENTS is pagerDutyIncidents.
func enumName(value, zero string) string {
	if prefix, ok := strings.CutSuffix(zero, "UNSPECIFIED"); ok && prefix != "" {
		value = strings.TrimPrefix(value, prefix)
	}
	var b strings.Builder
	for i, part := range strings.Split(strings.ToLower(value), "_") {
		if part == "" {
			continue
		}
		if i == 0 || b.Len() == 0 {
			b.WriteString(part)
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]) + part[1:])
	}
	return b.String()
}

// enumConstName is the SDK constant of an enum value, as openapi-generator writes it: the
// upper-case type name, "_", and the value with "_" between a digit and the letter after it.
// sdkcheck lists the same name, so a difference fails the SDK check.
func enumConstName(typeName, value string) string {
	var b strings.Builder
	for i, r := range value {
		if i > 0 && unicode.IsLetter(r) && unicode.IsDigit(rune(value[i-1])) {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(typeName) + "_" + b.String()
}

// sdkAlias is the import name of an SDK package: things_service is thingsservice.
func sdkAlias(pkg string) string { return strings.ReplaceAll(pkg, "_", "") }

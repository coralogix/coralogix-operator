package generator

import (
	"slices"
	"testing"
)

func changedPaths(changes []schemaChange) []string {
	var paths []string
	for _, change := range changes {
		paths = append(paths, change.Path)
	}
	slices.Sort(paths)
	return paths
}

func enumSchema(values ...any) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind": map[string]any{"type": "string", "enum": values},
			"tags": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string", "enum": values},
			},
		},
	}
}

func TestReleasedSchemaAcceptsAnEnumSuperset(t *testing.T) {
	old := enumSchema("a", "b")
	for name, current := range map[string]map[string]any{
		"a value is added":      enumSchema("a", "b", "c"),
		"the values are moved":  enumSchema("b", "a"),
		"moved and added":       enumSchema("c", "b", "a"),
		"the same values again": enumSchema("a", "b"),
	} {
		var changes []schemaChange
		if err := compareSchema("Thing", "spec", old, current, &changes); err != nil {
			t.Fatal(err)
		}
		if len(changes) != 0 {
			t.Errorf("%s: an enum superset was reported: %v", name, changes)
		}
	}
}

func TestReleasedSchemaReportsEnumValuesThatAreRemovedOrTightened(t *testing.T) {
	old := enumSchema("a", "b")
	removed := enumSchema("a")
	var changes []schemaChange
	if err := compareSchema("Thing", "spec", old, removed, &changes); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(changedPaths(changes), []string{"spec.properties.kind", "spec.properties.tags[]"}) {
		t.Fatalf("a removed enum value was not reported on the field and on the list item: %v", changes)
	}

	replaced := enumSchema("a", "c")
	changes = nil
	if err := compareSchema("Thing", "spec", old, replaced, &changes); err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("a replaced enum value was not reported: %v", changes)
	}

	unbounded := map[string]any{"type": "object", "properties": map[string]any{"kind": map[string]any{"type": "string"}}}
	changes = nil
	if err := compareSchema("Thing", "spec", map[string]any{"type": "object", "properties": map[string]any{"kind": map[string]any{"type": "string"}}}, old, &changes); err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Path != "spec.properties.kind" {
		t.Fatalf("a new enum on a released field must be reported: %v", changes)
	}
	changes = nil
	if err := compareSchema("Thing", "spec", old, unbounded, &changes); err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("a dropped enum must be reported: %v", changes)
	}
}

func TestReleasedSchemaReportsOtherRuleChangesBesideAnEnumSuperset(t *testing.T) {
	old := enumSchema("a", "b")
	current := enumSchema("a", "b", "c")
	current["properties"].(map[string]any)["kind"].(map[string]any)["minLength"] = 1
	var changes []schemaChange
	if err := compareSchema("Thing", "spec", old, current, &changes); err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Path != "spec.properties.kind" {
		t.Fatalf("a changed rule beside an enum superset must be reported: %v", changes)
	}
}

func TestReleasedSchemaChecksOldFieldsAndAllowsNewOptionalFields(t *testing.T) {
	old := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "minLength": 1, "description": "Old text"},
		},
	}
	newField := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "minLength": 1, "description": "New text"},
			"note": map[string]any{"type": "string", "maxLength": 100},
		},
	}
	var changes []schemaChange
	compare := func() {
		t.Helper()
		if err := compareSchema("Thing", "spec", old, newField, &changes); err != nil {
			t.Fatal(err)
		}
	}
	compare()
	if len(changes) != 0 {
		t.Fatalf("new optional field changed released rules: %v", changes)
	}

	changedRule := newField["properties"].(map[string]any)
	changedRule["name"].(map[string]any)["minLength"] = 2
	compare()
	if len(changes) != 1 || changes[0].Path != "spec.properties.name" {
		t.Fatalf("old field rule was not reported: %v", changes)
	}

	changes = nil
	newField["required"] = []any{"note"}
	compare()
	if len(changes) != 2 || changes[0].Path != "spec" {
		t.Fatalf("a new required child must change the parent rule: %v", changes)
	}

	changes = nil
	delete(changedRule, "name")
	compare()
	if len(changes) != 2 || changes[1].After != "missing" {
		t.Fatalf("a removed old field was not reported: %v", changes)
	}
}

package generator

import "testing"

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

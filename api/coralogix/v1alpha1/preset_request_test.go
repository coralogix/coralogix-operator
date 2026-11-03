package v1alpha1

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// These request bodies were recorded from the handwritten Preset conversion.
func TestPresetRequestParity(t *testing.T) {
	tests := []struct {
		name   string
		spec   string
		create string
	}{
		{
			name:   "empty condition arm and attachment",
			spec:   `{"name":"test","description":"description","connectorType":"slack","entityType":"alerts","attachmentConfig":"ENABLED","configOverrides":[{"conditionType":{"matchEntityType":{}},"payloadType":"test-payload","messageConfig":{"fields":[{"fieldName":"title","template":"first"},{"fieldName":"body","template":"second"}]}}]}`,
			create: `{"preset":{"attachmentConfig":{"policy":"ENABLED"},"configOverrides":[{"conditionType":{"matchEntityType":{}},"messageConfig":{"fields":[{"fieldName":"title","template":"first"},{"fieldName":"body","template":"second"}]},"payloadType":"test-payload"}],"connectorType":"SLACK","description":"description","entityType":"ALERTS","name":"test"}}`,
		},
		{
			name:   "subtype condition arm without attachment",
			spec:   `{"name":"test","description":"description","connectorType":"slack","entityType":"alerts","configOverrides":[{"conditionType":{"matchEntityTypeAndSubType":{"entitySubType":"logsImmediateTriggered"}},"messageConfig":{"fields":[{"fieldName":"title","template":"first"}]}}]}`,
			create: `{"preset":{"configOverrides":[{"conditionType":{"matchEntityTypeAndSubType":{"entitySubType":"logsImmediateTriggered"}},"messageConfig":{"fields":[{"fieldName":"title","template":"first"}]}}],"connectorType":"SLACK","description":"description","entityType":"ALERTS","name":"test"}}`,
		},
		{
			name:   "selected empty arm and empty fields",
			spec:   `{"name":"test","description":"description","connectorType":"slack","entityType":"alerts","configOverrides":[{"conditionType":{"matchEntityType":{}},"messageConfig":{"fields":[]}}]}`,
			create: `{"preset":{"configOverrides":[{"conditionType":{"matchEntityType":{}},"messageConfig":{}}],"connectorType":"SLACK","description":"description","entityType":"ALERTS","name":"test"}}`,
		},
		{
			name:   "empty optional overrides",
			spec:   `{"name":"test","description":"description","connectorType":"slack","entityType":"alerts","configOverrides":[]}`,
			create: `{"preset":{"connectorType":"SLACK","description":"description","entityType":"ALERTS","name":"test"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var preset Preset
			if err := json.Unmarshal([]byte(tt.spec), &preset.Spec); err != nil {
				t.Fatal(err)
			}
			create, err := preset.CreateRequest(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			gotCreate, err := json.Marshal(create)
			if err != nil {
				t.Fatal(err)
			}
			if string(gotCreate) != tt.create {
				t.Errorf("create request = %s, want %s", gotCreate, tt.create)
			}

			id := "preset-id"
			preset.Status.ID = &id
			update, err := preset.UpdateRequest(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			gotUpdate, err := json.Marshal(update)
			if err != nil {
				t.Fatal(err)
			}
			wantUpdate := strings.Replace(tt.create, `"name":"test"`, `"id":"preset-id","name":"test"`, 1)
			if string(gotUpdate) != wantUpdate {
				t.Errorf("update request = %s, want %s", gotUpdate, wantUpdate)
			}
		})
	}
}

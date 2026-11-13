package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

const enumHead = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n"

// A contract value that the line does not name is accepted. A new API value reaches the CRD and
// the conversion of a released kind with no edit of behavior-overrides.yaml.
func TestEnumValueWithoutALineIsAccepted(t *testing.T) {
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), enumHead+"    zero: unspecified\n    names: {THING_PRIORITY_LOW: alerts}\n")
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/thing_convert.go"])
	if !strings.Contains(types, "Enum=unspecified;alerts;pagerDutyIncidents") {
		t.Errorf("the CR enum must hold every contract value:\n%s", types)
	}
	if !strings.Contains(convert, "THINGPRIORITY_THING_PRIORITY_PAGER_DUTY_INCIDENTS") {
		t.Error("the conversion must map the value that no line names")
	}
}

func TestEnumRejectedValueStaysOutOfTheCR(t *testing.T) {
	line := "    zero: unspecified\n    rejected: [{value: THING_PRIORITY_PAGER_DUTY_INCIDENTS, reason: the operator cannot send its config}]\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), enumHead+line)
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	if strings.Contains(types, "agerDuty") || !strings.Contains(types, "Enum=unspecified;low") {
		t.Errorf("a rejected value must not be in the CR enum:\n%s", types)
	}
}

func TestEnumOldValuesKeyFailsWithAMessage(t *testing.T) {
	_, err := overrides.Parse([]byte(enumHead + "    zero: unspecified\n    values: [THING_PRIORITY_LOW]\n"))
	if err == nil || !strings.Contains(err.Error(), "enums.ThingPriority.values: the key is removed") {
		t.Fatalf("Parse() error = %v, want the removed values key message", err)
	}
}

func TestEnumRejectedNeedsAMappingWithAReason(t *testing.T) {
	_, err := overrides.Parse([]byte(enumHead + "    zero: unspecified\n    rejected: [THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n"))
	if err == nil || !strings.Contains(err.Error(), "{value: THING_PRIORITY_PAGER_DUTY_INCIDENTS, reason:") {
		t.Fatalf("Parse() error = %v, want a message that shows the new form", err)
	}
	_, err = overrides.Parse([]byte(enumHead + "    zero: unspecified\n    rejected: [{value: THING_PRIORITY_PAGER_DUTY_INCIDENTS}]\n"))
	if err == nil || !strings.Contains(err.Error(), "needs a reason") {
		t.Fatalf("Parse() error = %v, want a missing reason message", err)
	}
}

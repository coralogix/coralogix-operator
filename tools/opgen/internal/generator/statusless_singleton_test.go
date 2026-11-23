package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

func TestStatuslessSingletonNeedsAnExistingLifecycle(t *testing.T) {
	const head = "resource: RuleSet\nmode: existing\nvalidators: {inferred: true}\n"
	const operations = "api:\n  operations: {create: Rules_Create, get: Rules_Get, update: Rules_Replace, delete: Rules_Delete}\n"
	valid, err := overrides.Parse([]byte(head + "singletonID: {none: true}\n" + operations))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkLifecycle(&model.Resource{Singleton: true}, valid); err != nil {
		t.Fatalf("the released singleton lifecycle must be supported: %v", err)
	}
	for name, text := range map[string]string{
		"fixed id":         head + "singletonID: {fixed: rules}\n" + operations,
		"new kind":         strings.Replace(head, "existing", "new", 1) + "singletonID: {none: true}\n" + operations,
		"no delete":        head + "singletonID: {none: true}\n" + strings.Replace(operations, "Rules_Delete", "none", 1),
		"ambiguous choice": head + "singletonID: {none: true, fixed: rules}\n" + operations,
	} {
		if _, err := overrides.Parse([]byte(text)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if err := checkLifecycle(&model.Resource{Singleton: true}, nil); err == nil {
		t.Error("a singleton without an override must remain unsupported")
	}
}

func TestListItemUpdateFindsTheSameItemInTheAPI(t *testing.T) {
	key := &crField{API: "entityType", JSON: "entityType", GoName: "EntityType", GoType: "string", typ: &model.Type{Kind: model.String}}
	enabled := &crField{API: "enabled", JSON: "enabled", GoName: "Enabled", GoType: "bool", typ: &model.Type{Kind: model.Bool}}
	list := &crField{API: "rules", JSON: "rules", GoName: "Rules", GoType: "[]Rule", object: &crType{Fields: []*crField{key, enabled}}, Markers: []string{"+listType=map", "+listMapKey=entityType"}}
	d := &kindData{Spec: &crType{Fields: []*crField{list}}, GetResponseField: "RuleSet"}
	ix := &refIndex{fields: map[string]sdkcheck.Ref{
		"RuleSet.Rules":   {Name: "Rules", Want: "[]Rule"},
		"Rule.EntityType": {Name: "EntityType", Want: "string"},
		"Rule.Enabled":    {Name: "Enabled", Want: "bool"},
	}}
	full := new(omap).set("rules", []any{new(omap).set("entityType", "logs").set("enabled", true)})
	step, err := listUpdateStep(d, ix, &e2eFile{Update: &e2eUpdate{Field: "rules.enabled", Value: false}}, "RuleSet", full)
	if err != nil {
		t.Fatal(err)
	}
	if step.Field != "Rules[0].Enabled" || step.ListGet != "got.RuleSet.Rules" || step.Match != `item.EntityType == "logs"` || step.Got != "item.Enabled" || step.Want != "false" {
		t.Fatalf("the update must identify the item and read its changed field: %+v", step)
	}
	list.Markers = nil
	if _, err := listUpdateStep(d, ix, &e2eFile{Update: &e2eUpdate{Field: "rules.enabled", Value: false}}, "RuleSet", full); err == nil {
		t.Error("a list without an item key must be rejected")
	}
}

func TestStatuslessSingletonRendersReplaceFinalizerAndDelete(t *testing.T) {
	d := &kindData{Kind: "RuleSet", Plural: "rulesets", Finalizer: "ruleset.coralogix.com/finalizer",
		Module: "example.com/operator", SDKPath: "example.com/sdk/rules", SDKAlias: "rules", CxsdkPath: "example.com/sdk/cxsdk",
		Client: "RulesClient", UpdateMethod: "Replace", UpdateSetter: "Body", UpdateBody: "ReplaceRequest", DeleteMethod: "Delete",
		Singleton: true, StatuslessSingleton: true, Spec: &crType{Name: "RuleSetSpec"}}
	controller, err := execute("controller.go.tmpl", d)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"return r.HandleUpdate(ctx, log, obj)", "r.Client.Replace(ctx).Body(*body).Execute()", "coralogixreconciler.AddFinalizer(ctx, log, cr, r)", "r.Client.Delete(ctx).Execute()"} {
		if !strings.Contains(string(controller), want) {
			t.Errorf("the controller lacks %q", want)
		}
	}
	if strings.Contains(string(controller), "cr.Status.ID") {
		t.Error("the controller must not read a status id")
	}
	types, err := execute("types.go.tmpl", d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(types), "ID *string") || !strings.Contains(string(types), "HasIDInStatus() bool") {
		t.Error("the status must have no id and must use the Update path")
	}
}

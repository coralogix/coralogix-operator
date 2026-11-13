package generator

import (
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// hookData is what the templates need to call the hooks of a kind. A hook is an unexported method
// of the reconciler that a handwritten file of the controller package defines.
type hookData struct {
	Validate     bool
	BeforeCreate bool
	Adopt        bool
	BeforeUpdate bool
	AfterCreate  bool
	AfterUpdate  bool
	BeforeDelete bool
	Delete       bool
	// Any is true when the kind declares at least one hook.
	Any bool
	// CreateResponse and UpdateResponse are the Go types that Execute returns, written for the
	// controller package, such as *thingsservice.Thing or map[string]interface{}.
	CreateResponse string
	UpdateResponse string
}

// statusField is an extra field of the status of a kind. A hook sets it.
type statusField struct {
	JSON          string
	GoName        string
	GoType        string // a pointer type: *string, *int32, *int64, or *bool
	KeepOnCleanup bool
}

// hooks sets the data of the hooks and of the extra status fields of the kind.
func (b *builder) hooks(d *kindData) error {
	file := b.file
	d.Hooks = hookData{
		Validate:     file.HasHook(overrides.HookValidate),
		BeforeCreate: file.HasHook(overrides.HookBeforeCreate),
		Adopt:        file.HasHook(overrides.HookAdopt),
		BeforeUpdate: file.HasHook(overrides.HookBeforeUpdate),
		AfterCreate:  file.HasHook(overrides.HookAfterCreate),
		AfterUpdate:  file.HasHook(overrides.HookAfterUpdate),
		BeforeDelete: file.HasHook(overrides.HookBeforeDelete),
		Delete:       file.HasHook(overrides.HookDelete),
	}
	if file != nil {
		d.Hooks.Any = len(file.Hooks) != 0
		for _, name := range sortedKeys(file.Status) {
			field := file.Status[name]
			d.Status = append(d.Status, statusField{JSON: name, GoName: goName(name), GoType: "*" + field.Type, KeepOnCleanup: field.KeepOnCleanup})
		}
	}
	if !d.Hooks.Any {
		return nil
	}
	created, err := b.ix.typeRef("create.response")
	if err != nil {
		return err
	}
	d.Hooks.CreateResponse = "*" + qualify(created.Name, b.alias)
	// An update with no response fields returns a plain map.
	d.Hooks.UpdateResponse = "map[string]interface{}"
	if updated, err := b.ix.typeRef("update.response"); err == nil {
		d.Hooks.UpdateResponse = "*" + qualify(updated.Name, b.alias)
	}
	return nil
}

// KeptStatusFields are the extra status fields with keepOnCleanup. Create keeps them when it sets
// status.id: they outlive the remote resource.
func (d *kindData) KeptStatusFields() []statusField {
	var out []statusField
	for _, f := range d.Status {
		if f.KeepOnCleanup {
			out = append(out, f)
		}
	}
	return out
}

// RemoteStatusFields are the extra status fields that a selector mismatch removes.
func (d *kindData) RemoteStatusFields() []string {
	var out []string
	for _, f := range d.Status {
		if !f.KeepOnCleanup {
			out = append(out, f.JSON)
		}
	}
	return out
}

// HasClientSet is true when the reconciler needs the SDK client set: a name lookup of a reference,
// or a hook that calls another API.
func (d *kindData) HasClientSet() bool { return d.HasNameLookup || d.Hooks.Any }

// CreateReturn is what a Create return statement starts with, before the error. With an afterCreate
// hook, Create also returns the follow-up step, and an error returns no step.
func (d *kindData) CreateReturn() string {
	if d.Hooks.AfterCreate {
		return "nil, "
	}
	return ""
}

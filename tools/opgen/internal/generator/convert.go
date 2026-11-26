package generator

import (
	"fmt"
	"go/token"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// operations reads the SDK names of the calls.
func (b *builder) operations(r *model.Resource, d *kindData) error {
	client, err := b.ix.typeRef("resource")
	if err != nil {
		return err
	}
	d.Client = client.Name
	accessor, err := b.ix.method("clientset", "ClientSet")
	if err != nil {
		return err
	}
	d.Accessor = accessor.Name
	for _, op := range []struct {
		path   string
		method *string
		setter *string
		body   *string
	}{
		{"create", &d.CreateMethod, &d.CreateSetter, &d.CreateBody},
		{"get", &d.GetMethod, nil, nil},
		{"update", &d.UpdateMethod, &d.UpdateSetter, &d.UpdateBody},
		{"delete", &d.DeleteMethod, nil, nil},
	} {
		if op.path == "delete" && r.AlwaysExists {
			continue // the API cannot remove the resource, and Delete makes no call
		}
		m, err := b.ix.method(op.path, client.Name)
		if err != nil {
			return err
		}
		*op.method = m.Name
		if op.setter == nil {
			continue
		}
		body, err := b.ix.typeRef(op.path + ".body")
		if err != nil {
			return unsupported(op.path, "the request has no body")
		}
		builderType, err := b.ix.typeRef(op.path)
		if err != nil {
			return err
		}
		setter, err := b.ix.method(op.path+".body", builderType.Name)
		if err != nil {
			return err
		}
		*op.body, *op.setter = body.Name, setter.Name
	}
	resource, err := b.ix.typeRef("fields")
	if err != nil {
		return err
	}
	if r.Overwrite {
		return b.overwriteOperations(r, d)
	}
	if r.Singleton {
		if err := b.singletonOperations(r, d); err != nil {
			return err
		}
		if r.AlwaysExists {
			// Its Update is a PATCH, and the API requires the update mask.
			return b.updateMask(r, d)
		}
		return nil
	}
	id, err := b.ix.field(resource.Name, r.ID())
	if err != nil {
		return err
	}
	d.IDField, d.IDValue = id.Name, !strings.HasPrefix(id.Want, "*")
	if r.IDType.Kind == model.Integer {
		d.IDGoType = r.IDType.Format
		d.IDZeroValid = r.IDType.ZeroValid
		if d.IDGoType == "int32" {
			d.IDBits = 32
		} else {
			d.IDBits = 64
		}
		if strings.TrimPrefix(id.Want, "*") != d.IDGoType {
			return unsupported("resource.id", "the SDK id type %s does not match the API %s", id.Want, d.IDGoType)
		}
	} else if strings.TrimPrefix(id.Want, "*") != "string" {
		return unsupported("resource.id", "the SDK id type %s does not match the API string", id.Want)
	}
	if r.Create.Response.Field != "" {
		response, err := b.ix.typeRef("create.response")
		if err != nil {
			return err
		}
		field, err := b.ix.field(response.Name, r.Create.Response.Field)
		if err != nil {
			return err
		}
		d.CreateResponseField, d.CreateResponseValue = field.Name, field.Want == resource.Name
	}
	if r.Create.Response.ID != "" {
		response, err := b.ix.typeRef("create.response")
		if err != nil {
			return err
		}
		field, err := b.ix.field(response.Name, r.Create.Response.ID)
		if err != nil {
			return err
		}
		want := "string"
		if d.IDBits != 0 {
			want = d.IDGoType
		}
		if strings.TrimPrefix(field.Want, "*") != want {
			return unsupported("create.response", "the SDK id type %s of the Create response does not match the API %s", field.Want, want)
		}
		d.CreateResponseIDField, d.CreateResponseIDValue = field.Name, field.Want == want
	}
	if r.Get.Response.Field != "" {
		response, err := b.ix.typeRef("get.response")
		if err != nil {
			return err
		}
		field, err := b.ix.field(response.Name, r.Get.Response.Field)
		if err != nil {
			return err
		}
		d.GetResponseField, d.GetResponseValue = field.Name, field.Want == resource.Name
	}
	if r.Policy.GetByIDs != "" {
		builderType, err := b.ix.typeRef("get")
		if err != nil {
			return err
		}
		setter, err := b.ix.method("get.ids", builderType.Name)
		if err != nil {
			return err
		}
		if !r.Get.Response.Items || d.GetResponseField == "" {
			return unsupported("api.getByIDs", "Get does not return a list of the resource")
		}
		d.GetByIDs = setter.Name
	}
	if r.GetFromList {
		if err := b.getFromList(r, d); err != nil {
			return err
		}
	}
	if r.Policy.RequestWrapper != "" {
		for _, request := range []struct {
			body  string
			field *string
			value *bool
			inner *string
		}{{d.CreateBody, &d.CreateWrapperField, &d.CreateWrapperValue, &d.CreateResourceBody}, {d.UpdateBody, &d.UpdateWrapperField, &d.UpdateWrapperValue, &d.UpdateResourceBody}} {
			ref, err := b.ix.field(request.body, r.Policy.RequestWrapper)
			if err != nil {
				return err
			}
			// The wrapper holds the resource, or a request component of its own.
			*request.field, *request.value, *request.inner = ref.Name, !strings.HasPrefix(ref.Want, "*"), strings.TrimPrefix(ref.Want, "*")
		}
	}
	return b.updateMask(r, d)
}

// updateMask reads the SDK names of the update mask: its setter on the builder of Update, or its
// field in the body of Update. A full replace has none.
func (b *builder) updateMask(r *model.Resource, d *kindData) error {
	if !r.Replace && !r.MaskInBody && !b.file.OmitsUpdateMask() {
		builderType, err := b.ix.typeRef("update")
		if err != nil {
			return err
		}
		mask, err := b.ix.method("update.mask", builderType.Name)
		if err != nil {
			return err
		}
		d.MaskSetter = mask.Name
	}
	if r.MaskInBody && !b.file.OmitsUpdateMask() {
		field, err := b.ix.field(d.UpdateBody, r.UpdateMask)
		if err != nil {
			return err
		}
		d.MaskInBody, d.MaskBodyField, d.MaskBodyValue = true, field.Name, field.Want == "string"
	}
	return nil
}

// singletonOperations reads the SDK names of a singleton that one POST sets (api.operations with
// delete: none). The POST is Create and Update, and the status id is the fixed singletonID. The
// generated Delete makes no call, so DeleteMethod is unused.
func (b *builder) singletonOperations(r *model.Resource, d *kindData) error {
	d.Singleton = true
	d.AlwaysExists = r.AlwaysExists
	d.StatuslessSingleton = b.file.StatuslessSingleton() || r.AlwaysExists
	if !d.StatuslessSingleton {
		d.SingletonID = b.file.SingletonID.Fixed
	}
	if !d.StatuslessSingleton && (d.CreateMethod != d.UpdateMethod || d.CreateBody != d.UpdateBody || d.CreateSetter != d.UpdateSetter) {
		return unsupported("api.operations", "a singleton uses one POST for Create and Update")
	}
	for _, response := range []struct {
		path  string
		field string
		name  *string
		value *bool
	}{{"create.response", r.Create.Response.Field, &d.CreateResponseField, &d.CreateResponseValue}, {"get.response", r.Get.Response.Field, &d.GetResponseField, &d.GetResponseValue}} {
		if response.field == "" {
			continue
		}
		owner, err := b.ix.typeRef(response.path)
		if err != nil {
			return err
		}
		field, err := b.ix.field(owner.Name, response.field)
		if err != nil {
			return err
		}
		resource, err := b.ix.typeRef("fields")
		if err != nil {
			return err
		}
		*response.name, *response.value = field.Name, field.Want == resource.Name
	}
	if r.Policy.RequestWrapper != "" {
		for _, request := range []struct {
			body  string
			field *string
			value *bool
			inner *string
		}{{d.CreateBody, &d.CreateWrapperField, &d.CreateWrapperValue, &d.CreateResourceBody}, {d.UpdateBody, &d.UpdateWrapperField, &d.UpdateWrapperValue, &d.UpdateResourceBody}} {
			ref, err := b.ix.field(request.body, r.Policy.RequestWrapper)
			if err != nil {
				return err
			}
			*request.field, *request.value, *request.inner = ref.Name, !strings.HasPrefix(ref.Want, "*"), strings.TrimPrefix(ref.Want, "*")
		}
	}
	return nil
}

// overwriteOperations reads the SDK names of an overwrite resource (api.overwrite): one write is
// Create, Update, and Delete. Delete sends every list of the write empty. The resource has no id.
func (b *builder) overwriteOperations(r *model.Resource, d *kindData) error {
	d.Overwrite = true
	if d.CreateMethod != d.UpdateMethod || d.CreateMethod != d.DeleteMethod || d.CreateBody != d.UpdateBody {
		return unsupported("api.operations", "an overwrite resource uses one write for Create, Update, and Delete")
	}
	if r.Create.Response.Field != "" || r.Get.Response.Field != "" {
		return unsupported("resource", "an overwrite resource has no response wrapper")
	}
	for _, f := range r.Fields {
		if f.Create == nil {
			continue
		}
		ref, err := b.ix.field(d.CreateBody, f.Name)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(ref.Want, "[]") && !strings.HasPrefix(ref.Want, "map[") {
			return unsupported("resource."+f.Name, "the write of an overwrite resource sends a list or a map, got %s", ref.Want)
		}
		d.WriteFields = append(d.WriteFields, writeField{Field: ref.Name, Empty: qualify(ref.Want, b.alias) + "{}"})
	}
	if len(d.WriteFields) == 0 {
		return unsupported("resource", "the write of an overwrite resource sends no field")
	}
	return nil
}

// specConversions writes the conversions of the spec fields to the Create and Update resources.
func (b *builder) specConversions(d *kindData, create, update []*model.Field) error {
	createType, updateType := d.CreateBody, d.UpdateBody
	createSuffix, updateSuffix := "", ""
	if d.CreateResourceBody != "" {
		createType, updateType = d.CreateResourceBody, d.UpdateResourceBody
		createSuffix, updateSuffix = "Create", "Update"
	}
	var err error
	if d.CreateConv, err = b.conversionAs(d.Spec, createType, create, "spec", createSuffix); err != nil {
		return err
	}
	d.UpdateConv, err = b.conversionAs(d.Spec, updateType, update, "spec", updateSuffix)
	return err
}

// nestedMask returns the update mask paths of an object field that api.updateMask.nested names: one
// path for each field of the object that Update sends, and for each oneOf of the object, one
// conditional path per arm. A CR sets one arm, so the mask names only that arm: an API that rejects
// the bare object path also rejects a mask with an arm that the body does not have. The object must
// be required, and each oneOf must require one arm. Otherwise a CR that leaves the object or the
// arm out adds no path, and the API would keep the stored value instead of clearing it.
func (b *builder) nestedMask(d *kindData, f *model.ResourceField) ([]string, []maskArm, error) {
	location := "api.updateMask.nested." + f.Name
	if f.Type == nil || f.Type.Kind != model.Object {
		return nil, nil, unsupported(location, "the field must be an object with fields of its own; an object that is only a oneOf is not supported yet")
	}
	i := slices.IndexFunc(d.Spec.Fields, func(c *crField) bool { return c.API == f.Name })
	if i < 0 || d.Spec.Fields[i].object == nil {
		return nil, nil, unsupported(location, "the custom resource has no object field %s", f.Name)
	}
	owner := d.Spec.Fields[i]
	if owner.Optional {
		return nil, nil, unsupported(location, "the object must be required: a CR that leaves it out would add no path, and the API would keep the stored value")
	}
	arms := map[string]bool{}
	for _, g := range owner.object.groups {
		if g.AllowNone {
			return nil, nil, unsupported(location, "the oneOf of %s permits no arm: a CR with no arm would add no path, and the API would keep the stored arm; add requireOne", strings.Join(g.Arms, ", "))
		}
		for _, arm := range g.Arms {
			arms[arm] = true
		}
	}
	// An arm of a oneOf of the API that the CR does not show as a choice, such as a group with one
	// settable arm, would be a plain path, and the mask would name it also when the body lacks it.
	for _, g := range f.Type.Groups {
		for _, arm := range g.Arms {
			if !arms[arm] && slices.ContainsFunc(owner.object.Fields, func(c *crField) bool { return c.API == arm }) {
				return nil, nil, unsupported(location+"."+arm, "an arm of a oneOf that the custom resource does not show as a choice is not supported in a nested mask")
			}
		}
	}
	var paths []string
	var out []maskArm
	for _, child := range f.Type.Fields {
		if child.Update == nil {
			continue // Update does not send it
		}
		j := slices.IndexFunc(owner.object.Fields, func(c *crField) bool { return c.API == child.Name })
		if j < 0 && b.file != nil && len(b.file.Field(f.Type.Schema, child.Name).RequestValueBy) != 0 {
			// Update sends a value that the selected arm decides, so the mask would have to name it
			// too, with the arm that the body has.
			return nil, nil, unsupported(location+"."+child.Name, "a field with requestValueBy is not supported in a nested mask: Update sends it, and the mask would not name it")
		}
		if j < 0 && slices.ContainsFunc(owner.object.Fields, func(c *crField) bool { return c.inlineParent == child.Name }) {
			// Update rebuilds the inlined object from its fields beside the others, and sends it.
			return nil, nil, unsupported(location+"."+child.Name, "an inlined object is not supported in a nested mask: Update sends it, and the mask would not name it")
		}
		if j < 0 {
			continue // the CR does not have it, so Update does not send it
		}
		field := owner.object.Fields[j]
		if field.synthetic || field.group || field.fold != nil || field.inlineParent != "" {
			return nil, nil, unsupported(location+"."+child.Name, "a nested mask path of a field that the CR moves or renames in the request is not supported")
		}
		path := f.Name + "." + child.Name
		if arms[child.Name] {
			out = append(out, maskArm{Path: path, Check: setCondition(field, "in.Spec."+owner.GoName+"."+field.GoName)})
			continue
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 && len(out) == 0 {
		return nil, nil, unsupported(location, "Update sends no field of %s", f.Name)
	}
	return paths, out, nil
}

// requests builds the conversions of the spec to the Create and Update bodies, and the update mask.
func (b *builder) requests(r *model.Resource, d *kindData) error {
	var create, update []*model.Field
	var mask []string
	nested, used := map[string]bool{}, map[string]bool{}
	if b.file != nil && b.file.API.UpdateMask != nil {
		if r.Replace {
			return unsupported("api.updateMask", "a full-replace Update has no update mask")
		}
		for _, name := range b.file.API.UpdateMask.Nested {
			nested[name] = true
		}
	}
	bodyID := r.Policy.UpdateIDInBody
	for _, f := range r.Fields {
		var createPath, updatePath []string
		if f.RequestPath != "" {
			if !r.Replace {
				return unsupported("types."+r.Name+".fields."+f.Name+".requestAs", "requestAs needs a full-replace Update: the update mask path of a renamed field is not defined")
			}
			createPath = strings.Split(f.RequestPath, ".")
		}
		if path := f.RequestPathOf(true); path != "" {
			if !r.Replace && !b.file.OmitsUpdateMask() {
				return unsupported("types."+r.Name+".fields."+f.Name+".updateAs", "updateAs needs an Update without a mask")
			}
			updatePath = strings.Split(path, ".")
		}
		if folded := b.folds[f.Name]; folded != nil {
			for _, fold := range folded.folds {
				attrs := model.Attrs{Required: fold.Required}
				create = append(create, &model.Field{Name: fold.Field, Type: fold.Type, Create: &attrs, Attrs: attrs})
			}
		} else if f.Create != nil {
			create = append(create, &model.Field{Name: f.Name, Type: f.Type.CreateType(), Create: f.Create, Attrs: *f.Create, RequestPath: createPath})
		}
		if bodyID && f.Name == r.ID() {
			continue // UpdateRequest copies the status id
		}
		if f.Update != nil {
			update = append(update, &model.Field{Name: f.Name, Type: f.Type.UpdateType(), Update: f.Update, Attrs: *f.Update, RequestPath: updatePath})
			if b.file.Field(r.Name, f.Name).UpdateAs != "" {
				b.used[overrides.Line{Component: r.Name, Field: f.Name}] = true
			}
			if nested[f.Name] {
				paths, arms, err := b.nestedMask(d, f)
				if err != nil {
					return err
				}
				mask = append(mask, paths...)
				d.MaskArms = append(d.MaskArms, arms...)
				used[f.Name] = true
			} else if r.AlwaysExists && !f.Update.Required {
				// A PATCH rejects a mask path that the body does not have, and the API keeps the
				// value of a field that the mask leaves out. So the mask names an optional field
				// only when the spec sets it.
				i := slices.IndexFunc(d.Spec.Fields, func(c *crField) bool { return c.API == f.Name })
				if i < 0 {
					return unsupported("resource."+f.Name, "the spec has no field for the update mask")
				}
				d.MaskArms = append(d.MaskArms, maskArm{Path: f.Name, Check: setCondition(d.Spec.Fields[i], "in.Spec."+d.Spec.Fields[i].GoName)})
			} else {
				mask = append(mask, f.Name)
			}
		}
	}
	if d.Content != nil {
		// The content method builds the whole resource; the spec has no field of it.
		d.CreateConv, d.UpdateConv = "DecodeContent", "DecodeContent"
		b.code = append(b.code, b.contentMethod(d))
	} else if err := b.specConversions(d, create, update); err != nil {
		return err
	}
	if bodyID {
		owner := d.UpdateBody
		if d.BodyIDInWrapper {
			owner = d.UpdateResourceBody
		}
		ref, err := b.ix.field(owner, r.ID())
		if err != nil {
			return err
		}
		if d.IDBits != 0 && strings.TrimPrefix(ref.Want, "*") != d.IDGoType {
			return unsupported("update.id", "the SDK id type %s does not match the API %s", ref.Want, d.IDGoType)
		}
		d.UpdateIDInBody, d.BodyIDField, d.BodyIDValue = true, ref.Name, !strings.HasPrefix(ref.Want, "*")
	}
	if r.Replace || b.file.OmitsUpdateMask() {
		return b.requestValues(r, d)
	}
	for _, name := range slices.Sorted(maps.Keys(nested)) {
		if !used[name] {
			return unsupported("api.updateMask.nested."+name, "the Update request has no field %s", name)
		}
	}
	if r.UpdateMaskPattern != "" {
		valid, _, err := model.UpdateMaskRule(r.UpdateMaskPattern)
		if err != nil {
			return unsupported("update.mask", "%s", err)
		}
		all := slices.Clone(mask)
		for _, arm := range d.MaskArms {
			all = append(all, arm.Path)
		}
		for _, path := range all {
			if !valid(path) {
				return unsupported("update.mask", "the update mask pattern rejects the field %s", path)
			}
		}
	}
	d.Mask = strings.Join(mask, ",")
	return b.requestValues(r, d)
}

func (b *builder) requestValues(r *model.Resource, d *kindData) error {
	if b.file == nil {
		return nil
	}
	for _, request := range []struct {
		name, body string
		values     *[]requestValueData
	}{{"create", d.CreateBody, &d.CreateValues}, {"update", d.UpdateBody, &d.UpdateValues}} {
		for _, name := range sortedKeys(b.file.API.RequestValues[request.name]) {
			location := "api.requestValues." + request.name + "." + name
			if name == r.ID() && request.name == "update" && r.Policy.UpdateIDInBody {
				return unsupported(location, "the status id already sets this field")
			}
			if name == r.UpdateMask && request.name == "update" && r.MaskInBody {
				return unsupported(location, "the update mask already sets this field")
			}
			source := b.file.API.RequestValues[request.name][name]
			if source.Source == "omit" {
				continue // the request leaves the field out
			}
			ref, err := b.ix.field(request.body, name)
			if err != nil {
				return unsupported(location, "%s", err)
			}
			expr, pointer, err := b.requestValueExpression(source, d, location)
			if err != nil {
				return err
			}
			switch ref.Want {
			case "string":
				if pointer {
					expr = "ptr.Deref(" + expr + ", \"\")"
				}
			case "*string":
				if !pointer {
					expr = "ptr.To(" + expr + ")"
				}
			default:
				return unsupported(location, "the sibling field must be a string")
			}
			*request.values = append(*request.values, requestValueData{Field: ref.Name, Expr: expr})
		}
	}
	return nil
}

func (b *builder) requestValueExpression(source overrides.RequestValue, d *kindData, location string) (string, bool, error) {
	switch source.Source {
	case "uuid":
		d.HasUUID = true
		return strconv.Quote(source.Value) + " + gouuid.NewString()", false, nil
	case "literal":
		return strconv.Quote(source.Value), false, nil
	}
	name := strings.TrimPrefix(source.Source, "spec.")
	if source.Source == "field" {
		name = strings.TrimPrefix(location, "api.requestValues.create.")
		name = strings.TrimPrefix(name, "api.requestValues.update.")
	}
	for _, field := range d.Spec.Fields {
		if field.JSON == name && field.typ != nil && field.typ.Kind == model.String {
			return "in.Spec." + field.GoName, field.Optional, nil
		}
	}
	return "", false, unsupported(location, "the spec has no string field %s", name)
}

// conversion writes the method that converts a struct of the custom resource to an SDK request
// type, once per pair, and returns its name.
func (b *builder) conversion(cr *crType, sdkType string, fields []*model.Field, path string) (string, error) {
	return b.conversionAs(cr, sdkType, fields, path, b.conversionSuffix)
}

func (b *builder) conversionAs(cr *crType, sdkType string, fields []*model.Field, path, suffix string) (string, error) {
	key := cr.Name + "->" + sdkType + suffix
	if name, ok := b.convs[key]; ok {
		return name, nil
	}
	name := "to" + sdkType + suffix
	b.convs[key] = name
	previous, previousOut := b.conversionSuffix, b.outVar
	b.conversionSuffix, b.outVar = suffix, ""
	defer func() { b.conversionSuffix, b.outVar = previous, previousOut }()
	var body strings.Builder
	moved := &requestObject{}
	for _, rf := range fields {
		if len(rf.RequestPath) != 0 {
			if err := b.movedField(cr, sdkType, rf, path, moved, &body); err != nil {
				return "", err
			}
			continue
		}
		if children := inlineChildren(cr, rf.Name); len(children) != 0 {
			code, err := b.inlineObject(sdkType, rf, children, path)
			if err != nil {
				return "", err
			}
			body.WriteString(code)
			continue
		}
		if folded := b.folds[rf.Name]; folded != nil && cr == b.specType && rf.Update != nil {
			code, err := b.foldedObject(cr, sdkType, rf, folded, path)
			if err != nil {
				return "", err
			}
			body.WriteString(code)
			continue
		}
		if cr == b.specType && b.setByHook(rf.Name) {
			continue
		}
		field, src, guard := sourceField(cr, rf.Name)
		if field == nil && cr == b.specType {
			if arms := b.file.Field(b.file.Resource, rf.Name).InArms; len(arms) != 0 {
				code, err := b.armValue(cr, sdkType, rf, arms, path)
				if err != nil {
					return "", err
				}
				body.WriteString(code)
				continue
			}
		}
		if field == nil {
			component := cr.schema
			if component == "" && b.file != nil {
				component = b.file.Resource
			}
			if by := b.file.Field(component, rf.Name).RequestValueBy; len(by) != 0 {
				code, err := b.requestValueBy(cr, sdkType, rf, by, path)
				if err != nil {
					return "", err
				}
				b.used[overrides.Line{Component: component, Field: rf.Name}] = true
				body.WriteString(code)
				continue
			}
			if rf.Attrs.Required {
				return "", unsupported(path+"."+rf.Name, "the request requires a field that the custom resource does not have")
			}
			continue
		}
		code, err := b.assignFrom(field, rf.Type, sdkType, path+"."+field.JSON, src)
		if err != nil {
			return "", err
		}
		if guard != "" {
			// The group is optional: a CR that leaves it out sends none of its fields.
			code = "if " + guard + " != nil {\n" + code + "}\n"
		}
		body.WriteString(code)
	}
	body.WriteString(b.requestObjects(moved))
	lookupParam := ""
	if b.hasNameLookup {
		lookupParam = ", lookup *NameLookupContext"
	}
	b.code = append(b.code, fmt.Sprintf(`
// %s converts the %s to the API type %s.
func (in *%s) %s(ctx context.Context, namespace string%s) (*%s.%s, error) {
	out := &%s.%s{}
%s	return out, nil
}
`, name, cr.Name, sdkType, cr.Name, name, lookupParam, b.alias, sdkType, b.alias, sdkType, body.String()))
	return name, nil
}

// requestObject is an object of a request body that holds fields of the CR at a request path with
// dots (requestAs). The fields that share an object share one value, so the second field cannot
// replace the object of the first.
type requestObject struct {
	ref      sdkcheck.Ref     // the field of the parent that holds the object
	typeName string           // the SDK type of the object
	varName  string           // the variable of the generated code
	leaves   []requestLeaf    // fields that the object holds directly
	children []*requestObject // objects inside the object, in order of first use
}

// requestLeaf is the code that sets one field of an object. condition is the test that the CR
// sets the field, or "" when the request always sends it.
type requestLeaf struct {
	code, condition string
}

// conditions returns the tests of every field below the object. ok is false when one of the fields is
// always sent.
func (o *requestObject) conditions() (tests []string, ok bool) {
	var out []string
	for _, leaf := range o.leaves {
		if leaf.condition == "" {
			return nil, false
		}
		out = append(out, leaf.condition)
	}
	for _, c := range o.children {
		inner, ok := c.conditions()
		if !ok {
			return nil, false
		}
		out = append(out, inner...)
	}
	return out, true
}

// setByHook reports whether a hook sets the resource field in the requests (setByHook), and
// marks the line used. The generated conversion leaves the field out.
func (b *builder) setByHook(field string) bool {
	if b.file == nil || !b.file.Field(b.file.Resource, field).SetByHook {
		return false
	}
	b.used[overrides.Line{Component: b.file.Resource, Field: field}] = true
	return true
}

// movedField writes the code of a field that the request holds at another path than the field has
// in Get (requestAs). A path of one segment sets a field of the body. A longer path adds the field
// to the object tree, which requestObjects writes after the last field.
func (b *builder) movedField(cr *crType, sdkType string, rf *model.Field, path string, root *requestObject, body *strings.Builder) error {
	if cr == b.specType && b.setByHook(rf.Name) {
		// The hook sets the field. Its objects are always built, so the hook finds them present.
		if len(rf.RequestPath) < 2 {
			return nil
		}
		node, _, err := b.requestNode(root, sdkType, rf.RequestPath[:len(rf.RequestPath)-1])
		if err != nil {
			return err
		}
		node.leaves = append(node.leaves, requestLeaf{})
		return nil
	}
	field, src, guard := sourceField(cr, rf.Name)
	if field == nil {
		if rf.Attrs.Required {
			return unsupported(path+"."+rf.Name, "the request requires a field that the custom resource does not have")
		}
		return nil
	}
	if guard != "" {
		return unsupported(path+"."+rf.Name, "requestAs does not support a field of an optional group")
	}
	component := cr.schema
	if component == "" {
		component = b.file.Resource
	}
	if b.file.Field(component, rf.Name).RequestAs != "" {
		b.used[overrides.Line{Component: component, Field: rf.Name}] = true
	}
	segments := rf.RequestPath
	node, owner, err := b.requestNode(root, sdkType, segments[:len(segments)-1])
	if err != nil {
		return err
	}
	leaf := *field
	leaf.API = segments[len(segments)-1]
	previous := b.outVar
	defer func() { b.outVar = previous }()
	if node != root {
		b.outVar = node.varName
	}
	code, err := b.assignFrom(&leaf, rf.Type, owner, path+"."+field.JSON, src)
	if err != nil {
		return err
	}
	if node == root {
		body.WriteString(code)
		return nil
	}
	node.leaves = append(node.leaves, requestLeaf{code: code, condition: sentCondition(field, src)})
	return nil
}

// requestNode returns the object of the tree at the request path below root, and its SDK type.
// It adds the objects of the path that the tree does not have yet.
func (b *builder) requestNode(root *requestObject, sdkType string, segments []string) (*requestObject, string, error) {
	node, owner := root, sdkType
	for _, segment := range segments {
		ref, err := b.ix.field(owner, segment)
		if err != nil {
			return nil, "", err
		}
		var next *requestObject
		for _, c := range node.children {
			if c.ref.Name == ref.Name {
				next = c
			}
		}
		if next == nil {
			next = &requestObject{ref: ref, typeName: strings.TrimPrefix(ref.Want, "*"), varName: b.temp("obj")}
			node.children = append(node.children, next)
		}
		node, owner = next, next.typeName
	}
	return node, owner, nil
}

// inlineChildren returns the CR fields of cr that an inline line moved out of the API object field
// api, in order.
func inlineChildren(cr *crType, api string) []*crField {
	var out []*crField
	for _, f := range cr.Fields {
		if f.inlineParent == api {
			out = append(out, f)
		}
	}
	return out
}

// inlineObject writes the code that rebuilds an inlined object from the CR fields that show its
// fields. An optional object is left out when the CR sets none of them.
func (b *builder) inlineObject(sdkType string, rf *model.Field, children []*crField, path string) (string, error) {
	if rf.Type.Kind != model.Object {
		return "", unsupported(path+"."+rf.Name, "an inlined field must be an object in the request")
	}
	ref, err := b.ix.field(sdkType, rf.Name)
	if err != nil {
		return "", err
	}
	objectType := strings.TrimPrefix(ref.Want, "*")
	object := b.temp("obj")
	parent := b.out()
	previous := b.outVar
	b.outVar = object
	defer func() { b.outVar = previous }()
	var inner strings.Builder
	var conditions []string
	sent := map[string]bool{}
	for _, nested := range rf.Type.Fields {
		var crf *crField
		for _, child := range children {
			if child.API == nested.Name {
				crf = child
			}
		}
		if crf == nil {
			if nested.Attrs.Required {
				return "", unsupported(path+"."+rf.Name+"."+nested.Name, "the request requires a field of the inlined object that the custom resource does not have")
			}
			continue
		}
		sent[crf.API] = true
		src := "in." + crf.GoName
		code, err := b.assignFrom(crf, nested.Type, objectType, path+"."+crf.JSON, src)
		if err != nil {
			return "", err
		}
		inner.WriteString(code)
		conditions = append(conditions, sentCondition(crf, src))
	}
	for _, child := range children {
		if !sent[child.API] {
			return "", unsupported(path+"."+child.JSON, "the request object %s has no field %s", objectType, child.API)
		}
	}
	pointer := strings.HasPrefix(ref.Want, "*")
	assign := parent + "." + ref.Name + " = " + object + "\n"
	if !pointer {
		assign = parent + "." + ref.Name + " = *" + object + "\n"
	}
	code := fmt.Sprintf("%s := &%s.%s{}\n%s%s", object, b.alias, objectType, inner.String(), assign)
	if pointer && !slices.Contains(conditions, "") {
		code = "if " + strings.Join(conditions, " || ") + " {\n" + code + "}\n"
	}
	return code, nil
}

// requestObjects writes the code that builds the objects of the tree and sets them in the body.
func (b *builder) requestObjects(root *requestObject) string {
	var out strings.Builder
	for _, c := range root.children {
		out.WriteString(b.requestObject(c, "out"))
	}
	return out.String()
}

// requestObject builds one object, sets it in its parent, and leaves it out when the CR sets none
// of its fields and the SDK type allows it.
func (b *builder) requestObject(o *requestObject, parent string) string {
	var inner strings.Builder
	inner.WriteString(o.varName + " := &" + b.alias + "." + o.typeName + "{}\n")
	for _, leaf := range o.leaves {
		inner.WriteString(leaf.code)
	}
	for _, c := range o.children {
		inner.WriteString(b.requestObject(c, o.varName))
	}
	pointer := strings.HasPrefix(o.ref.Want, "*")
	if pointer {
		inner.WriteString(parent + "." + o.ref.Name + " = " + o.varName + "\n")
	} else {
		inner.WriteString(parent + "." + o.ref.Name + " = *" + o.varName + "\n")
	}
	if conditions, onlyIfSet := o.conditions(); pointer && onlyIfSet && len(conditions) != 0 {
		return "if " + strings.Join(conditions, " || ") + " {\n" + inner.String() + "}\n"
	}
	return inner.String()
}

// setCondition is the test that a CR sets an optional field, for an update mask path. A field
// that the CR reads from a Secret is set also when the CR names only the Secret.
func setCondition(f *crField, src string) string {
	if f.secret != nil {
		return sentCondition(f, src)
	}
	return src + " != nil"
}

// sentCondition is the test that the generated code uses to send a field: "" when it is always sent.
func sentCondition(f *crField, src string) string {
	if f.secret != nil {
		// A CR that names only the Secret sends the field too.
		sibling := strings.TrimSuffix(src, f.GoName) + goName(f.secret.sibling)
		if f.secret.values {
			return "len(" + src + ") != 0 || len(" + sibling + ") != 0"
		}
		return src + " != nil || " + sibling + " != nil"
	}
	switch {
	case f.sendNullAsEmpty:
		return ""
	case f.joinedBy != "":
		if f.Optional {
			return "len(" + src + ") != 0"
		}
	case f.typ.Kind == model.List || f.typ.Kind == model.Set || f.typ.Kind == model.Map:
		if f.Optional || f.omitEmptyInRequest {
			return "len(" + src + ") != 0"
		}
	case f.Optional:
		return src + " != nil"
	}
	return ""
}

// foldedObject writes the Update code of a createAs line: it builds the object that Update nests
// from the flat CR fields. The object is left out when no flat field is set. A writable property
// of the object that no createAs line names has no CR field, so it is an error.
func (b *builder) foldedObject(cr *crType, sdkType string, rf *model.Field, folded *foldedField, path string) (string, error) {
	ref, err := b.ix.field(sdkType, rf.Name)
	if err != nil {
		return "", err
	}
	objectType := strings.TrimPrefix(ref.Want, "*")
	property := map[string]model.CreateFold{}
	for _, fold := range folded.folds {
		property[fold.Property] = fold
	}
	for _, nested := range rf.Type.Fields {
		if _, ok := property[nested.Name]; ok {
			continue
		}
		if nested.Attrs.ReadOnly || (nested.Update != nil && nested.Update.ReadOnly) || nested.Behavior == model.Computed {
			continue // the server sets it
		}
		return "", unsupported(path+"."+rf.Name+"."+nested.Name, "Update can write this field of the object, but no createAs line names it, so the CR cannot set it")
	}
	object := b.temp("obj")
	previous := b.outVar
	b.outVar = object
	defer func() { b.outVar = previous }()
	var inner strings.Builder
	var conditions []string
	for _, fold := range folded.folds {
		crf := crFieldByAPI(cr, fold.Field)
		if crf == nil {
			return "", fmt.Errorf("%s: the CR has no field for the createAs property %s", path, fold.Property)
		}
		if crf.Optional {
			conditions = append(conditions, "in."+crf.GoName+" != nil")
		}
		nested := *crf
		nested.API = fold.Property
		code, err := b.assignFrom(&nested, fold.Type, objectType, path+"."+crf.JSON, "in."+crf.GoName)
		if err != nil {
			return "", err
		}
		inner.WriteString(code)
	}
	if len(conditions) != len(folded.folds) {
		// A required flat field is always set, so the object is always sent.
		conditions = nil
	}
	if len(conditions) != 0 && !strings.HasPrefix(ref.Want, "*") {
		return "", unsupported(path+"."+rf.Name, "Update requires the object, but its createAs fields are optional in the CR")
	}
	assign := "out." + ref.Name + " = " + object + "\n"
	if !strings.HasPrefix(ref.Want, "*") {
		assign = "out." + ref.Name + " = *" + object + "\n"
	}
	code := fmt.Sprintf("%s := &%s.%s{}\n%s%s", object, b.alias, objectType, inner.String(), assign)
	if len(conditions) != 0 {
		code = "if " + strings.Join(conditions, " || ") + " {\n" + code + "}\n"
	}
	return code, nil
}

func crFieldByAPI(cr *crType, api string) *crField {
	for _, f := range cr.Fields {
		if f.API == api {
			return f
		}
	}
	return nil
}

// sourceField finds an API field at its released CR path. guard is the expression of an optional
// group that holds the field: the code must read the field only when the group is set. It is empty
// for any other field.
func sourceField(cr *crType, api string) (field *crField, src, guard string) {
	for _, f := range cr.Fields {
		if f.group {
			for _, child := range f.object.Fields {
				if child.API == api {
					if f.Optional {
						guard = "in." + f.GoName
					}
					return child, "in." + f.GoName + "." + child.GoName, guard
				}
			}
			continue
		}
		if f.API == api {
			return f, "in." + f.GoName, ""
		}
	}
	return nil, "", ""
}

// requestValueBy sends a hidden string or enum value from the selected object arm.
func (b *builder) requestValueBy(cr *crType, sdkType string, field *model.Field, by map[string]string, path string) (string, error) {
	location := path + "." + field.Name
	if field.Type.Kind != model.String && field.Type.Kind != model.Enum {
		return "", unsupported(location, "requestValueBy needs a string or enum API field")
	}
	var choices []model.OneOfGroup
	choices = append(choices, cr.groups...)
	for _, f := range cr.Fields {
		if f.group {
			choices = append(choices, f.object.groups...)
		}
	}
	if !slices.ContainsFunc(choices, func(g model.OneOfGroup) bool {
		if field.Attrs.Required && len(by) != len(g.Arms) {
			return false
		}
		for source := range by {
			if !slices.Contains(g.Arms, source) {
				return false
			}
		}
		return true
	}) {
		return "", unsupported(location, "requestValueBy must name arms of one oneOf group; a required value needs every arm")
	}
	ref, err := b.ix.field(sdkType, field.Name)
	if err != nil {
		return "", err
	}
	var code strings.Builder
	for i, source := range sortedKeys(by) {
		arm, src, guard := sourceField(cr, source)
		if guard != "" {
			return "", unsupported(location, "requestValueBy does not support an arm in an optional group")
		}
		if arm == nil || (arm.typ.Kind != model.Object && arm.typ.Kind != model.OneOf) || !arm.Optional {
			return "", unsupported(location, "requestValueBy source %s must be an optional object arm", source)
		}
		value := by[source]
		expr := strconv.Quote(value)
		if field.Type.Kind == model.Enum {
			if value == field.Type.EnumZero && value != "" {
				component := cr.schema
				if component == "" {
					component = b.file.Resource
				}
				line := overrides.Line{Component: component, Field: field.Name}
				return "", &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".requestValueBy", msg: "the enum zero value cannot represent a selected arm"}
			}
			if !slices.Contains(field.Type.Values, value) {
				return "", unsupported(location, "requestValueBy value %s is not in the API enum", value)
			}
			enumType, err := b.ix.schemaType(field.Type.Schema)
			if err != nil {
				return "", err
			}
			constant, err := b.ix.enumConst(enumType, value)
			if err != nil {
				return "", err
			}
			expr = b.alias + "." + constant
		}
		if i > 0 {
			code.WriteString("else ")
		}
		code.WriteString("if " + src + " != nil {\n")
		code.WriteString(b.set("out."+ref.Name, ref.Want, expr, false))
		code.WriteString("}")
	}
	if field.Attrs.Required {
		code.WriteString(" else {\nreturn nil, fmt.Errorf(" + strconv.Quote(location+": exactly one source arm is required") + ")\n}")
	}
	return code.String() + "\n", nil
}

// assign writes the statements that set one SDK field from the custom resource field.
func (b *builder) assignFrom(f *crField, rt *model.Type, owner, path, src string) (string, error) {
	ref, err := b.ix.field(owner, f.API)
	if err != nil {
		return "", err
	}
	target := b.out() + "." + ref.Name
	if f.secret != nil {
		return b.secretValue(f, ref.Want, target, path, src)
	}
	if f.joinedBy != "" {
		if strings.TrimPrefix(ref.Want, "*") != "string" {
			return "", unsupported(path, "joinedBy needs a string in the SDK, got %s", ref.Want)
		}
		code := b.set(target, ref.Want, fmt.Sprintf("strings.Join(%s, %q)", src, f.joinedBy), false)
		if f.Optional {
			return "if len(" + src + ") != 0 {\n" + code + "}\n", nil
		}
		return code, nil
	}
	if f.refForm == "split" {
		sibling := strings.TrimSuffix(src, f.GoName) + goName(f.refSibling)
		resolved := b.temp("id")
		call := fmt.Sprintf("ResolveResourceID(ctx, namespace, ResourceIDRef{ResourceRef: %s}, %q)", sibling, f.ref)
		if f.refVersion != "" {
			call = fmt.Sprintf("ResolveResourceIDVersion(ctx, namespace, ResourceIDRef{ResourceRef: %s}, %q, %q)", sibling, f.ref, f.refVersion)
		}
		return fmt.Sprintf("if %s != nil {\n%s} else if %s != nil {\n%s, err := %s\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n%s}\n", src, b.set(target, ref.Want, src, true), sibling, resolved, call, path, b.set(target, ref.Want, resolved, false)), nil
	}
	switch {
	case f.typ.Kind == model.List || f.typ.Kind == model.Set || f.typ.Kind == model.Map:
		stmts, expr, isPtr, err := b.value(f, rt, src, ref.Want, path)
		if err != nil {
			return "", err
		}
		// A list that an unwrapped object holds can be a pointer field in the SDK.
		code := stmts + b.set(target, ref.Want, expr, isPtr)
		if f.sendNullAsEmpty {
			// The API needs [] where the CR has none, not null.
			empty := target + " = " + qualify(ref.Want, b.alias) + "{}\n"
			if want := strings.TrimPrefix(ref.Want, "*"); !strings.HasPrefix(want, "[]") && !strings.HasPrefix(want, "map[") {
				// A wrapper that only the request has holds the list: send the wrapper with an
				// empty list, which replaces the stored list.
				emptyStmts, emptyExpr, emptyPtr, err := b.value(f, rt, f.GoType+"{}", ref.Want, path)
				if err != nil {
					return "", err
				}
				empty = emptyStmts + b.set(target, ref.Want, emptyExpr, emptyPtr)
			}
			return "if len(" + src + ") != 0 {\n" + code + "} else {\n" + empty + "}\n", nil
		}
		if f.Optional || f.omitEmptyInRequest {
			return "if len(" + src + ") != 0 {\n" + code + "}\n", nil
		}
		return code, nil
	case f.Optional:
		stmts, expr, isPtr, err := b.value(f, rt, "(*"+src+")", strings.TrimPrefix(ref.Want, "*"), path)
		if err != nil {
			return "", err
		}
		var present string
		if stmts == "" && strings.HasPrefix(ref.Want, "*") && expr == "(*"+src+")" {
			present = target + " = " + src + "\n" // the same pointer type
			if !f.sendNullAsEmpty {
				return present, nil
			}
		} else {
			present = stmts + b.set(target, ref.Want, expr, isPtr)
		}
		if f.sendNullAsEmpty {
			// The API keeps the stored value of an omitted field, so the request clears it.
			return "if " + src + " != nil {\n" + present + "} else {\n" + b.set(target, ref.Want, `""`, false) + "}\n", nil
		}
		return "if " + src + " != nil {\n" + present + "}\n", nil
	}
	stmts, expr, isPtr, err := b.value(f, rt, src, strings.TrimPrefix(ref.Want, "*"), path)
	if err != nil {
		return "", err
	}
	return stmts + b.set(target, ref.Want, expr, isPtr), nil
}

// set writes the assignment of expr to an SDK field of type want. isPtr: expr is a pointer.
func (b *builder) set(target, want, expr string, isPtr bool) string {
	switch {
	case strings.HasPrefix(want, "*") && isPtr:
		return target + " = " + expr + "\n"
	case strings.HasPrefix(want, "*") && token.IsIdentifier(expr):
		return target + " = &" + expr + "\n"
	case strings.HasPrefix(want, "*"):
		v := b.temp("v")
		return v + " := " + expr + "\n" + target + " = &" + v + "\n"
	case isPtr:
		return target + " = *" + expr + "\n"
	}
	return target + " = " + expr + "\n"
}

func (b *builder) temp(prefix string) string {
	b.tmp++
	return prefix + strconv.Itoa(b.tmp)
}

// value writes the statements that convert src, a value of the custom resource, to the SDK value
// type want. It returns the expression of the result and whether that expression is a pointer.
func (b *builder) value(f *crField, rt *model.Type, src, want, path string) (string, string, bool, error) {
	if len(rt.Wrappers) != 0 {
		return b.wrappedValue(f, rt, src, path)
	}
	if f.ref != "" {
		if rt.Kind == model.List || rt.Kind == model.Set {
			elemWant, ok := strings.CutPrefix(want, "[]")
			if !ok || elemWant != referenceWant(f) {
				return "", "", false, unsupported(path, "a reference list needs []%s in the SDK, got %s", referenceWant(f), want)
			}
			out, item := b.temp("v"), b.temp("e")
			statements, value := b.referenceValue(f, item, path+"[]")
			return fmt.Sprintf("%s := make([]%s, 0, len(%s))\nfor _, %s := range %s {\n%s%s = append(%s, %s)\n}\n", out, elemWant, src, item, src, statements, out, out, value), out, false, nil
		}
		// A pointer is set from the value: the caller takes its address.
		if strings.TrimPrefix(want, "*") != referenceWant(f) {
			return "", "", false, unsupported(path, "a reference needs %s in the SDK, got %s", referenceWant(f), want)
		}
		statements, value := b.referenceValue(f, src, path)
		return statements, value, false, nil
	}
	if f.intNames != nil && rt.Kind == model.Integer {
		v := b.temp("v")
		return fmt.Sprintf("%s, err := %s(%s)\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, f.intNames.Func, arg(src), path), v, false, nil
	}
	switch rt.Kind {
	case model.List, model.Set, model.Map:
		elemWant, ok := strings.CutPrefix(want, "[]")
		if rt.Kind == model.Map {
			elemWant, ok = strings.CutPrefix(want, "map[string]")
		}
		if !ok {
			return "", "", false, fmt.Errorf("%s: the SDK type %s is not a collection", path, want)
		}
		elem := &crField{API: f.API, JSON: f.JSON, typ: f.typ.Elem, object: f.object, enum: f.enum, intNames: f.intNames}
		v, e, k := b.temp("v"), b.temp("e"), "_"
		if rt.Kind == model.Map {
			k = b.temp("k")
		}
		item := e
		if value := b.mapValue(f); value != nil {
			item = value.GoType + "(" + e + ")" // the named value type of a map holds the limits
		}
		stmts, expr, isPtr, err := b.value(elem, rt.Elem, item, elemWant, path+"[]")
		if err != nil {
			return "", "", false, err
		}
		if stmts == "" && expr == e && builtinTypes[elemWant] {
			return "", src, false, nil // the same Go type: the value needs no conversion
		}
		if isPtr && !strings.HasPrefix(elemWant, "*") {
			expr = "*" + expr
		} else if !isPtr && strings.HasPrefix(elemWant, "*") {
			expr = "&" + expr
		}
		goType := qualify(want, b.alias)
		if rt.Kind == model.Map {
			return fmt.Sprintf("%s := make(%s, len(%s))\nfor %s, %s := range %s {\n%s%s[%s] = %s\n}\n", v, goType, src, k, e, src, stmts, v, k, expr), v, false, nil
		}
		return fmt.Sprintf("%s := make(%s, 0, len(%s))\nfor %s, %s := range %s {\n%s%s = append(%s, %s)\n}\n", v, goType, src, k, e, src, stmts, v, v, expr), v, false, nil
	case model.Object, model.OneOf:
		if f.object != nil && apiFieldCount(f.object) == 0 && len(rt.Fields) == 0 && want == "map[string]interface{}" {
			return "", "map[string]interface{}{}", false, nil
		}
		fields := rt.Fields
		name, err := b.conversion(f.object, strings.TrimPrefix(want, "*"), fields, path)
		if err != nil {
			return "", "", false, err
		}
		v := b.temp("v")
		args := "ctx, namespace"
		if b.hasNameLookup {
			args += ", lookup"
		}
		return fmt.Sprintf("%s, err := %s.%s(%s)\nif err != nil {\nreturn nil, err\n}\n", v, deref(src), name, args), v, true, nil
	case model.Enum:
		v := b.temp("v")
		return fmt.Sprintf("%s, err := %s(%s)\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, f.enum.Func, arg(src), path), v, false, nil
	case model.Integer:
		if rt.WireString {
			return "", "strconv.FormatInt(" + arg(src) + ", 10)", false, nil
		}
	case model.String:
		if rt.Format == "date-time" {
			if want != "time.Time" {
				return "", "", false, unsupported(path, "the SDK date-time field must be time.Time, got %s", want)
			}
			return "", src + ".Time", false, nil
		}
	case model.Number:
		switch want {
		case "float32":
			return "", "float32(" + src + ".AsApproximateFloat64())", false, nil
		case "float64":
			return "", src + ".AsApproximateFloat64()", false, nil
		}
		return "", "", false, unsupported(path, "the SDK number field must be float32 or float64, got %s", want)
	case model.Bool:
		// A scalar needs no conversion.
	}
	return "", src, false, nil
}

func referenceWant(f *crField) string {
	if f.refIDType != "" {
		return f.refIDType
	}
	return "string"
}

func (b *builder) referenceValue(f *crField, src, path string) (string, string) {
	v := b.temp("v")
	if f.refForm == "named" || f.refForm == "namedBackendOnly" {
		if f.refIDType != "" {
			backend := b.temp("backend")
			name := b.temp("name")
			bits := 64
			if f.refIDType == "int32" {
				bits = 32
			}
			statements := fmt.Sprintf("var %s *int64\nvar %s *string\nif %s.BackendRef != nil {\n%s = %s.BackendRef.Name\nif %s.BackendRef.ID != nil {\nid := int64(*%s.BackendRef.ID)\n%s = &id\n}\n}\n", backend, name, src, name, src, src, src, backend)
			statements += fmt.Sprintf("%s, err := ResolveNamedNumericResourceID(ctx, namespace, %s, %s, %s.ResourceRef, %q, %q, %q, lookup, %s, %d)\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, backend, name, src, f.ref, f.refVersion, f.refLookup, f.refLookup, bits, path)
			return statements, f.refIDType + "(" + v + ")"
		}
		value := arg(src)
		if f.refForm == "namedBackendOnly" {
			if f.refType == "" {
				value = "NamedResourceIDRef{BackendRef: &" + src + ".BackendRef}"
			} else {
				idField := f.refIDField
				if idField == "" {
					idField = "ID"
				}
				value = "NamedResourceIDRef{BackendRef: &NamedBackendRef{ID: " + src + ".BackendRef." + idField + ", Name: " + src + ".BackendRef.Name}}"
			}
		}
		call := fmt.Sprintf("ResolveNamedResourceID(ctx, namespace, %s, %q, %q, %q, lookup, %s)", value, f.ref, f.refVersion, f.refLookup, f.refLookup)
		return fmt.Sprintf("%s, err := %s\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, call, path), v
	}
	if f.refIDType != "" {
		backend := b.temp("backend")
		resource := src + ".ResourceRef"
		statements := "var " + backend + " *int64\n"
		if f.refForm != "resourceRefOnly" {
			statements += fmt.Sprintf("if %s.BackendRef != nil {\nid := int64(%s.BackendRef.ID)\n%s = &id\n}\n", src, src, backend)
		}
		bits := 64
		if f.refIDType == "int32" {
			bits = 32
		}
		statements += fmt.Sprintf("%s, err := ResolveNumericResourceID(ctx, namespace, %s, %s, %q, %q, %d)\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, backend, resource, f.ref, f.refVersion, bits, path)
		return statements, f.refIDType + "(" + v + ")"
	}
	value := arg(src)
	if f.refForm == "resourceRefOnly" {
		value = "ResourceIDRef{ResourceRef: " + src + ".ResourceRef}"
	} else if f.refType != "" {
		value = "ResourceIDRef(" + value + ")"
	}
	call := fmt.Sprintf("ResolveResourceID(ctx, namespace, %s, %q)", value, f.ref)
	if f.refVersion != "" {
		call = fmt.Sprintf("ResolveResourceIDVersion(ctx, namespace, %s, %q, %q)", value, f.ref, f.refVersion)
	}
	return fmt.Sprintf("%s, err := %s\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, call, path), v
}

// wrappedValue restores each one-property API wrapper around the CR value.
func (b *builder) wrappedValue(f *crField, rt *model.Type, src, path string) (string, string, bool, error) {
	types := make([]string, len(rt.Wrappers))
	fields := make([]sdkcheck.Ref, len(rt.Wrappers))
	for i, w := range rt.Wrappers {
		if w.Schema == "" {
			return "", "", false, unsupported(path, "the request wrapper has no component")
		}
		name, err := b.ix.schemaType(w.Schema)
		if err != nil {
			return "", "", false, err
		}
		field, err := b.ix.field(name, w.Field)
		if err != nil {
			return "", "", false, err
		}
		types[i], fields[i] = name, field
	}
	inner := *f
	inner.typ = f.typ.Unwrapped()
	last := len(fields) - 1
	stmts, expr, isPtr, err := b.value(&inner, rt.Unwrapped(), src, fields[last].Want, path)
	if err != nil {
		return "", "", false, err
	}
	for i := last; i >= 0; i-- {
		v := b.temp("w")
		stmts += fmt.Sprintf("%s := %s.%s{}\n", v, b.alias, types[i])
		stmts += b.set(v+"."+fields[i].Name, fields[i].Want, expr, isPtr)
		expr, isPtr = v, false
	}
	return stmts, expr, false, nil
}

// mapValue returns the named type of the values of the map field f, nil when they have none.
func (b *builder) mapValue(f *crField) *crScalar {
	name, ok := strings.CutPrefix(f.GoType, "map[string]")
	if !ok {
		return nil
	}
	for _, s := range b.scalars {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// deref turns "(*x)" into "x" for a method call: Go dereferences the pointer itself.
func deref(src string) string {
	if inner, ok := strings.CutPrefix(src, "(*"); ok {
		return strings.TrimSuffix(inner, ")")
	}
	return src
}

// arg turns "(*x)" into "*x" for a function argument.
func arg(src string) string {
	if inner, ok := strings.CutPrefix(src, "("); ok {
		return strings.TrimSuffix(inner, ")")
	}
	return src
}

// getFromList reads the SDK names of a resource that no operation reads by its id: Get is the List
// (api.getFromList). Only the e2e test calls Get. It reads the List and selects the item with the id,
// so the List must hold every resource in one answer: the generator supports no paging, filter,
// or query parameter of the List yet.
func (b *builder) getFromList(r *model.Resource, d *kindData) error {
	switch {
	case len(r.PageParameters) != 0 || r.NextPage != nil:
		return unsupported("api.getFromList", "the List of %s has paging parameters; the e2e test reads one page", r.Name)
	case len(r.ListFilters) != 0:
		return unsupported("api.getFromList.filters", "the e2e test sends no filter of the List")
	case len(r.GetQuery) != 0:
		return unsupported("api.getFromList.query", "the e2e test sends no query value of the List")
	case !r.Get.Response.Items || d.GetResponseField == "":
		return unsupported("api.getFromList", "Get does not return a list of the resource")
	}
	resource, err := b.ix.typeRef("fields")
	if err != nil {
		return err
	}
	response, err := b.ix.typeRef("get.response")
	if err != nil {
		return err
	}
	field, err := b.ix.field(response.Name, r.Get.Response.Field)
	if err != nil {
		return err
	}
	if field.Want != "[]"+resource.Name {
		return unsupported("api.getFromList", "the List field %s is %s, not a slice of %s", field.Name, field.Want, resource.Name)
	}
	d.GetFromList, d.GetListField, d.GetItemType = true, field.Name, resource.Name
	// The e2e test reads the item that the List holds, not a field of a Get response.
	d.GetResponseField, d.GetResponseValue = "", false
	return nil
}

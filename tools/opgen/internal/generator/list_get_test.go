package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The golden ListThing reads itself from the List (api.getFromList): no operation reads one
// ListThing by its id. Only the e2e test calls Get: it selects the item with the status id, and a
// missing item is a 404.
func TestGetFromListSelectsTheItem(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	e2e := readGenerated(t, root, e2eDir+"/listthing_generated_test.go")
	// Create, update, delete, the minimal spec, and the cleanup backstop read the resource.
	if n := strings.Count(e2e, "listThingGetFromList{ctx: ctx, apiClient: apiClient, id: "); n != 5 {
		t.Errorf("every Get of the e2e test reads the List; found %d:\n%s", n, e2e)
	}
	if n := strings.Count(e2e, "ListThingsServiceGetListThings("); n != 1 {
		t.Errorf("only the helper calls the List; found %d calls", n)
	}
	for _, want := range []string{
		"if ptr.Deref(list.Items[i].Id, \"\") == g.id {",
		"return nil, &http.Response{StatusCode: http.StatusNotFound}, listThingNotInList{}",
		`{"code":404,"message":"Not Found: the ListThing is not in the List"}`,
		// The update step reads the item itself, not a field of a Get response.
		"g.Expect(ptr.Deref(got.Name, \"\")).To(Equal(fullName + \"-updated\"))",
	} {
		if !strings.Contains(e2e, want) {
			t.Errorf("the e2e test lacks %q", want)
		}
	}
	controller := readGenerated(t, root, controllerDir+"/listthing_controller.go")
	if strings.Contains(controller, "GetListThings") {
		t.Error("the controller must not call Get: a resync is one Update")
	}
	if !strings.Contains(controller, "r.Client.ListThingsServiceUpdateListThing(ctx, id)") {
		t.Errorf("Update takes the id in its path:\n%s", controller)
	}
}

// The e2e test reads one page of the List and sends nothing else, so a List with paging, a filter,
// or a query value stops the run.
func TestGetFromListNeedsAListOfOnePage(t *testing.T) {
	head := "resource: ListThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  operations:\n    create: ListThingsService_CreateListThing\n    get: ListThingsService_GetListThings\n    update: ListThingsService_UpdateListThing\n    delete: ListThingsService_DeleteListThing\n  idProperty: id\n  serverSetID: true\n  updateMaskInBody: true\n"
	inline := "types:\n  ListThing:\n    fields:\n      settings: {inline: true, required: true}\n"
	paged := func(s string) string {
		return strings.Replace(s, "      operationId: ListThingsService_GetListThings\n", "      operationId: ListThingsService_GetListThings\n      parameters:\n        - {name: pageSize, in: query, schema: {type: integer, format: int32}}\n        - {name: pageToken, in: query, schema: {type: string}}\n", 1)
	}
	withStatus := func(s string) string {
		return strings.Replace(s, "      operationId: ListThingsService_GetListThings\n", "      operationId: ListThingsService_GetListThings\n      parameters:\n        - {name: status, in: query, schema: {type: string}}\n", 1)
	}
	wantCode(t, checkMask(t, "ListThing", head+"  getFromList:\n    query: {status: active}\n"+inline, withStatus), codeShapeUnsupported)
	wantCode(t, checkMask(t, "ListThing", head+"  getFromList: {}\n"+inline, paged), codeShapeUnsupported)
	if err := checkMask(t, "ListThing", head+"  getFromList: {}\n"+inline, nil); err != nil {
		t.Fatalf("the golden override: %v", err)
	}
}

// inline on the resource takes an object that the contract marks optional only when the line states
// with required: true that the API always returns it. The object can be a $ref with a description
// of its own, a one-item allOf.
func TestInlineOfAnOptionalObject(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contract), "        settings:\n          allOf:\n            - $ref: '#/components/schemas/ListThingSettings'\n          description:") {
		t.Fatal("the golden ListThing no longer nests settings in a one-item allOf")
	}
	head := "resource: ListThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  operations:\n    create: ListThingsService_CreateListThing\n    get: ListThingsService_GetListThings\n    update: ListThingsService_UpdateListThing\n    delete: ListThingsService_DeleteListThing\n  idProperty: id\n  serverSetID: true\n  updateMaskInBody: true\n  getFromList: {}\ntypes:\n  ListThing:\n    fields:\n"
	err = checkMask(t, "ListThing", head+"      settings: {inline: true}\n", nil)
	if err == nil || !strings.Contains(err.Error(), "state that with required: true on the line") {
		t.Fatalf("want the optional object error, got %v", err)
	}
	if err := checkMask(t, "ListThing", head+"      settings: {inline: true, required: true}\n", nil); err != nil {
		t.Fatalf("required states the object: %v", err)
	}
	// A nested inline line shows the fields of an object of a component; required has no meaning there.
	err = checkMask(t, "MaskThing", "resource: MaskThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  MaskThingRule:\n    fields:\n      prefix: {inline: true, required: true}\n", nil)
	if err == nil || !strings.Contains(err.Error(), "required applies only to an object of the resource") {
		t.Fatalf("want the nested required error, got %v", err)
	}
}

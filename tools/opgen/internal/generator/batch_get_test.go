package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// The golden BatchThing reads itself with a batch Get (api.getByIDs): Get has no id in its path,
// takes the id as the one item of its id list, and answers a list. Only the e2e test calls Get.
func TestBatchGetSendsTheIDInItsList(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	e2e := readGenerated(t, root, e2eDir+"/batchthing_generated_test.go")
	// Create, update, delete, the minimal spec, and the cleanup backstop read the resource.
	if n, all := strings.Count(e2e, "apiClient.BatchThingsServiceGetBatchThingsByIds(ctx).Ids([]string{"), strings.Count(e2e, "GetBatchThingsByIds("); n != 5 || n != all {
		t.Errorf("every Get of the e2e test sends the id in its list; found %d of %d:\n%s", n, all, e2e)
	}
	if strings.Contains(e2e, "GetBatchThingsByIds(ctx, ") {
		t.Error("the batch Get takes no id in its path")
	}
	// The update step reads the one item of the list, and checks that it is the resource.
	for _, want := range []string{
		"g.Expect(got.Things).To(HaveLen(1))",
		"g.Expect(got.Things[0].Id).To(Equal(fullID))",
		"g.Expect(got.Things[0].Name).To(Equal(",
	} {
		if !strings.Contains(e2e, want) {
			t.Errorf("the update step lacks %q", want)
		}
	}
	controller := readGenerated(t, root, controllerDir+"/batchthing_controller.go")
	if strings.Contains(controller, "GetBatchThingsByIds") {
		t.Error("the controller must not call Get: a resync is one Replace")
	}
	if !strings.Contains(controller, "r.Client.BatchThingsServiceDeleteBatchThing(ctx, id)") {
		t.Errorf("Delete takes the id in its path:\n%s", controller)
	}
}

func TestGetByIDsNeedsAnExistingKind(t *testing.T) {
	const file = "resource: S\nmode: new\napi:\n  getByIDs: ids\n"
	_, err := overrides.Parse([]byte(file))
	if err == nil || !strings.Contains(err.Error(), "mode new sets no api key") {
		t.Fatalf("want the mode new error, got %v", err)
	}
}

// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package baseline

import (
	"context"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const (
	archiveTargetPath     = "/logs/data-setup/v2"
	archiveActivatePath   = "/dataengine/retention-tags/v1/activate"
	archiveTagsEnabledURL = "/dataengine/retention-tags/v1/enabled"
)

func init() {
	register(kind{
		name:     "archivelogstarget",
		crdFile:  "coralogix.com_archivelogstargets.yaml",
		resource: "archivelogstargets",
		newFake:  newArchiveLogsTargetFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.ArchiveLogsTargetReconciler{
				ArchiveLogsTargetsClient: cs.ArchiveLogs(),
				ArchiveRetentionsClient:  cs.ArchiveRetentions(),
				Interval:                 reconcileInterval,
			}
		},
		scenarios: archiveLogsTargetScenarios,
		admission: archiveLogsTargetAdmission(),
		fidelity: fidelityConfig{
			// The team has one archive target. The cases set it to the bucket that the team has.
			overwritesTeam: true,
			teamState: &teamState{
				readPath: archiveTargetPath,
				restore:  restoreArchiveTarget,
			},
		},
	})
}

// restoreArchiveTarget writes back the target that GET returned: its S3 fields and whether it is
// active.
func restoreArchiveTarget(original any) fidelityStep {
	root, _ := original.(map[string]any)
	target, _ := root["target"].(map[string]any)
	spec, _ := target["archiveSpec"].(map[string]any)
	active, _ := spec["isActive"].(bool)
	return fidelityStep{
		Method: http.MethodPost,
		Path:   archiveTargetPath,
		Body:   map[string]any{"isActive": active, "s3": target["s3"]},
	}
}

// archiveLogsTargetManifest returns an ArchiveLogsTarget CR named "target". spec is YAML,
// indented by two spaces.
func archiveLogsTargetManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: ArchiveLogsTarget\nmetadata:\n  name: target\nspec:\n" + spec
}

func s3Spec(region, bucket string) string {
	return "  s3Target:\n    region: " + region + "\n    bucketName: " + bucket + "\n"
}

var (
	archiveMinimal = archiveLogsTargetManifest(s3Spec(archiveTeamRegion, archiveBucket))
	archiveIBM     = archiveLogsTargetManifest("  ibmCosTarget:\n    bucketCrn: crn:v1:bluemix:public:cloud-object-storage:global:a/0:0:bucket:cxo\n    endpoint: s3.eu-de.cloud-object-storage.appdomain.cloud\n")
)

func s3SpecMap(region, bucket string) map[string]any {
	return map[string]any{"s3Target": map[string]any{"region": region, "bucketName": bucket}}
}

var archiveLogsTargetScenarios = map[string]func(h *harness){
	// The sample has placeholders for the region and the bucket, so the API rejects it.
	"create-sample": func(h *harness) {
		h.CreateFromFile("create the sample", "config/samples/v1alpha1/archivelogstarget/s3-archive-logs-target.yaml")
	},

	"create-minimal": func(h *harness) {
		h.Create("create: set the target, then activate the retentions", archiveMinimal)
		h.Reconcile("reconcile again: the resync sets the target and does not activate")
		h.Reconcile("reconcile again")
	},

	// The operator sends region "" for a CR without a region, and the API rejects it.
	"create-without-region": func(h *harness) {
		h.Create("create without a region: the request sends an empty region", archiveLogsTargetManifest("  s3Target:\n    bucketName: "+archiveBucket+"\n"))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-without-bucket": func(h *harness) {
		h.Create("create without a bucket: the request sends an empty bucket", archiveLogsTargetManifest("  s3Target:\n    region: "+archiveTeamRegion+"\n"))
		h.Delete("delete")
	},

	"create-empty-s3": func(h *harness) {
		h.Create("create with an empty s3Target", archiveLogsTargetManifest("  s3Target: {}\n"))
		h.Delete("delete")
	},

	"create-bucket-missing": func(h *harness) {
		h.Create("create with a bucket that does not exist: the API answers 408", archiveLogsTargetManifest(s3Spec(archiveTeamRegion, "cxo-no-such-bucket")))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-other-region": func(h *harness) {
		h.Create("create with a region other than the one of the team: the API answers 408", archiveLogsTargetManifest(s3Spec("us-east-1", archiveBucket)))
		h.Delete("delete")
	},

	// The operator sends only S3 targets. An IBM COS target fails before any request.
	"create-ibm-cos": func(h *harness) {
		h.Create("create an IBM COS target: no request", archiveIBM)
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-ibm-cos-every-field": func(h *harness) {
		h.Create("create an IBM COS target with every field: no request", archiveLogsTargetManifest(
			"  ibmCosTarget:\n    bucketCrn: crn:v1:bluemix:public:cloud-object-storage:global:a/0:0:bucket:cxo\n"+
				"    endpoint: s3.eu-de.cloud-object-storage.appdomain.cloud\n    serviceCrn: \"crn:v1:bluemix:public:cloud-object-storage:global:a/0:0::\"\n    bucketType: EXTERNAL\n"))
	},

	"create-set-rejected": func(h *harness) {
		archiveTargets(h).answerNextError(archiveTargetPath, http.StatusInternalServerError, "try again")
		h.Create("create: the API answers 500", archiveMinimal)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// The target is set when the activation fails. The CR gets no status id, so the next
	// reconcile sets the target and activates again.
	"create-activate-rejected": func(h *harness) {
		archiveTargets(h).answerNextError(archiveActivatePath, http.StatusInternalServerError, "try again")
		h.Create("create: the activation answers 500", archiveMinimal)
		h.Reconcile("reconcile again: set and activate again")
		h.Reconcile("reconcile again")
	},

	"create-route-not-found": func(h *harness) {
		archiveTargets(h).answerNextRouteNotFound(archiveTargetPath)
		h.Create("create: the API has no such route", archiveMinimal)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-malformed-response": func(h *harness) {
		archiveTargets(h).answerNext(archiveTargetPath, http.StatusOK, "not a target")
		h.Create("create: the answer cannot be decoded", archiveMinimal)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-empty-response": func(h *harness) {
		archiveTargets(h).answerNext(archiveTargetPath, http.StatusOK, map[string]any{})
		h.Create("create: the answer has no target", archiveMinimal)
		h.Reconcile("reconcile again")
	},

	"update-bucket": func(h *harness) {
		h.Create("create", archiveMinimal)
		h.SetSpec("change the bucket", s3SpecMap(archiveTeamRegion, archiveOtherBucket))
		h.Reconcile("reconcile again")
	},

	"update-remove-region": func(h *harness) {
		h.Create("create", archiveMinimal)
		h.SetSpec("remove the region: the request sends an empty region", map[string]any{"s3Target": map[string]any{"bucketName": archiveBucket}})
	},

	"update-to-ibm-cos": func(h *harness) {
		h.Create("create", archiveMinimal)
		h.SetSpec("switch to an IBM COS target: no request", map[string]any{"ibmCosTarget": map[string]any{
			"bucketCrn": "crn:v1:bluemix:public:cloud-object-storage:global:a/0:0:bucket:cxo",
			"endpoint":  "s3.eu-de.cloud-object-storage.appdomain.cloud",
		}})
		h.Delete("delete: no request")
	},

	"update-rejected": func(h *harness) {
		h.Create("create", archiveMinimal)
		archiveTargets(h).answerNextError(archiveTargetPath, http.StatusBadRequest, "the API rejects this request")
		h.SetSpec("change the bucket: the API rejects it", s3SpecMap(archiveTeamRegion, archiveOtherBucket))
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// A 404 that names a resource on update is "not found": the shared reconciler removes the
	// status id, and the next reconcile creates again.
	"update-not-found": func(h *harness) {
		h.Create("create", archiveMinimal)
		archiveTargets(h).answerNextError(archiveTargetPath, http.StatusNotFound, "target not found")
		h.Reconcile("resync: the API answers 404")
		h.Reconcile("reconcile again: create")
	},

	"update-malformed-response": func(h *harness) {
		h.Create("create", archiveMinimal)
		archiveTargets(h).answerNext(archiveTargetPath, http.StatusOK, "not a target")
		h.Reconcile("resync: the answer cannot be decoded")
		h.Reconcile("reconcile again")
	},

	"delete": func(h *harness) {
		h.Create("create", archiveMinimal)
		h.Delete("delete: set the target inactive")
	},

	"delete-without-region": func(h *harness) {
		h.Create("create", archiveMinimal)
		archiveTargets(h).answerNextError(archiveTargetPath, http.StatusBadRequest, "try again")
		h.SetSpec("remove the region: the API rejects the update", map[string]any{"s3Target": map[string]any{"bucketName": archiveBucket}})
		h.Delete("delete: the request sends an empty region, and the API rejects it")
		h.Reconcile("reconcile again")
	},

	"delete-rejected": func(h *harness) {
		h.Create("create", archiveMinimal)
		archiveTargets(h).answerNextError(archiveTargetPath, http.StatusInternalServerError, "try again")
		h.Delete("delete: the API answers 500")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-not-found": func(h *harness) {
		h.Create("create", archiveMinimal)
		archiveTargets(h).answerNextError(archiveTargetPath, http.StatusNotFound, "target not found")
		h.Delete("delete: a 404 that names a resource is success")
	},

	"delete-route-not-found": func(h *harness) {
		h.Create("create", archiveMinimal)
		archiveTargets(h).answerNextRouteNotFound(archiveTargetPath)
		h.Delete("delete: the API has no such route")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// The shared reconciler creates when the status has no id, even while the CR is deleted.
	"delete-without-id": func(h *harness) {
		h.Create("create", archiveMinimal)
		setArchiveStatusID(h, nil)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	// The status id is a fixed text. The operator never sends it, so another id changes nothing.
	"other-status-id": func(h *harness) {
		h.Create("create", archiveMinimal)
		other := "another-id"
		setArchiveStatusID(h, &other)
		h.Reconcile("update with another status id")
		h.Delete("delete with another status id")
	},

	"selector-mismatch": func(h *harness) {
		h.Create("create", archiveMinimal)
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "archive"})})
		h.Reconcile("reconcile after the selector stops matching the CR: set the target inactive")
		h.Reconcile("reconcile again")
	},
}

// setArchiveStatusID writes status.id of the CR, or removes it when id is nil, as a user who
// edits the status could.
func setArchiveStatusID(h *harness, id *string) {
	h.t.Helper()
	obj := h.get()
	if id == nil {
		unstructured.RemoveNestedField(obj.Object, "status", "id")
	} else if err := unstructured.SetNestedField(obj.Object, *id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set ArchiveLogsTarget status id: %v", err)
	}
}

// archiveLogsTargetAdmission lists the CRs that TestAdmission creates. Each case has one reason
// that the CRD accepts or rejects it.
func archiveLogsTargetAdmission() map[string]string {
	spec := func(fields string) string { return archiveLogsTargetManifest(fields) }
	ibm := func(fields string) string {
		return spec("  ibmCosTarget:\n    bucketCrn: crn\n    endpoint: e\n" + fields)
	}
	return map[string]string{
		"s3 target":                        spec(s3Spec(archiveTeamRegion, archiveBucket)),
		"s3 target without a region":       spec("  s3Target:\n    bucketName: b\n"),
		"s3 target without a bucket":       spec("  s3Target:\n    region: r\n"),
		"empty s3 target":                  spec("  s3Target: {}\n"),
		"empty bucket and region":          spec("  s3Target:\n    region: \"\"\n    bucketName: \"\"\n"),
		"bucket of 1 character":            spec(s3Spec("r", "b")),
		"bucket of 300 characters":         spec(s3Spec("r", strings.Repeat("b", 300))),
		"ibm cos target":                   ibm(""),
		"empty ibm cos target":             spec("  ibmCosTarget: {}\n"),
		"ibm cos with a service crn":       ibm("    serviceCrn: crn\n"),
		"ibm cos bucket type EXTERNAL":     ibm("    bucketType: EXTERNAL\n"),
		"ibm cos bucket type INTERNAL":     ibm("    bucketType: INTERNAL\n"),
		"ibm cos bucket type UNSPEC":       ibm("    bucketType: UNSPECIFIED\n"),
		"ibm cos bucket type lowercase":    ibm("    bucketType: external\n"),
		"ibm cos bucket type API name":     ibm("    bucketType: IBM_BUCKET_TYPE_EXTERNAL\n"),
		"s3 and ibm cos":                   spec(s3Spec("r", "b") + "  ibmCosTarget:\n    bucketCrn: crn\n    endpoint: e\n"),
		"empty spec":                       spec(" {}\n"),
		"no spec":                          "apiVersion: coralogix.com/v1alpha1\nkind: ArchiveLogsTarget\nmetadata:\n  name: target\n",
		"unknown field is not an error":    spec(s3Spec("r", "b") + "  format: json\n"),
		"unknown s3 field is not an error": spec("  s3Target:\n    region: r\n    bucketName: b\n    roleArn: arn\n"),
	}
}

// Copyright 2025 Coralogix Ltd.
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

package e2e

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	targetservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/target_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// The team has one archive target, which the generated ArchiveLogsTarget test writes too, so this
// test runs alone.
var _ = Describe("ArchiveLogsTarget", Ordered, Serial, func() {
	var (
		crClient   client.Client
		apiClient  *targetservice.TargetServiceAPIService
		awsRegion  string
		logsBucket string
		targetName = "s3-archivelogs-target"
		key        = types.NamespacedName{Name: targetName, Namespace: testNamespace}
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		apiClient = newOpenAPIClientSet().ArchiveLogs()
		awsRegion = os.Getenv("AWS_REGION")
		logsBucket = os.Getenv("LOGS_BUCKET")
	})

	// waitForSync waits until the operator synced the current generation of the CR, and returns it.
	waitForSync := func(ctx context.Context) *coralogixv1alpha1.ArchiveLogsTarget {
		current := &coralogixv1alpha1.ArchiveLogsTarget{}
		Eventually(func(g Gomega) {
			g.Expect(crClient.Get(ctx, key, current)).To(Succeed())
			synced := meta.FindStatusCondition(current.Status.Conditions, utils.ConditionTypeRemoteSynced)
			g.Expect(synced).NotTo(BeNil())
			g.Expect(synced.Status).To(Equal(metav1.ConditionTrue), "%s: %s", synced.Reason, synced.Message)
			g.Expect(synced.ObservedGeneration).To(Equal(current.Generation))
			g.Expect(current.Status.PrintableStatus).To(Equal("RemoteSynced"))
			g.Expect(current.Status.ID).NotTo(BeNil())
		}, time.Minute, time.Second).Should(Succeed())
		return current
	}

	// getTarget reads the archive target of the team.
	getTarget := func(ctx context.Context, g Gomega) *targetservice.V2Target {
		got, httpResp, err := apiClient.S3TargetServiceGetTarget(ctx).Execute()
		g.Expect(cxsdk.NewAPIError(httpResp, err)).To(Succeed())
		g.Expect(got).NotTo(BeNil())
		g.Expect(got.Target.S3).NotTo(BeNil())
		return &got.Target
	}

	It("Should be set successfully", func(ctx context.Context) {
		By("Setting a storage target in Coralogix backend")
		Expect(crClient.Create(ctx, &coralogixv1alpha1.ArchiveLogsTarget{
			ObjectMeta: metav1.ObjectMeta{Name: targetName, Namespace: testNamespace},
			Spec: coralogixv1alpha1.ArchiveLogsTargetSpec{
				Format: ptr.To("cxDataV1"),
				S3Target: &coralogixv1alpha1.ArchiveLogsTargetS3TargetSpec{
					Region:     ptr.To(awsRegion),
					BucketName: ptr.To(logsBucket),
				},
			},
		})).To(Succeed())

		By("Waiting until the target is synced and the retentions are activated")
		current := waitForSync(ctx)
		Expect(current.Status.RetentionsActivated).To(Equal(ptr.To(true)))

		By("Verifying the storage target is set in Coralogix backend")
		Eventually(func(g Gomega) {
			target := getTarget(ctx, g)
			g.Expect(ptr.Deref(target.ArchiveSpec.IsActive, false)).To(BeTrue())
			g.Expect(target.ArchiveSpec.Format).To(Equal(ptr.To(targetservice.TARGETFORMAT_TARGET_FORMAT_CX_DATA_V1)))
			g.Expect(target.S3.Bucket).To(Equal(logsBucket))
			g.Expect(target.S3.Region).To(Equal(&awsRegion))
		}, accountWideBackendTimeout, time.Second).Should(Succeed())
	})

	// The released kind sent region "" when the spec had none, and the API rejected it. The
	// generated kind leaves region out, and the API keeps the stored one.
	It("Should keep the stored region when the spec removes it", func(ctx context.Context) {
		current := &coralogixv1alpha1.ArchiveLogsTarget{}
		Expect(crClient.Get(ctx, key, current)).To(Succeed())
		updated := current.DeepCopy()
		updated.Spec.S3Target.Region = nil
		Expect(crClient.Patch(ctx, updated, client.MergeFrom(current))).To(Succeed())

		By("Waiting until the operator syncs the spec without region")
		synced := waitForSync(ctx)
		Expect(synced.Spec.S3Target.Region).To(BeNil())
		Expect(synced.Status.RetentionsActivated).To(Equal(ptr.To(true)))

		By("Verifying the backend keeps the stored region")
		Consistently(func(g Gomega) {
			target := getTarget(ctx, g)
			g.Expect(ptr.Deref(target.ArchiveSpec.IsActive, false)).To(BeTrue())
			g.Expect(target.S3.Bucket).To(Equal(logsBucket))
			g.Expect(target.S3.Region).To(Equal(&awsRegion))
		}, 10*time.Second, time.Second).Should(Succeed())
	})

	It("Should be deactivated successfully", func(ctx context.Context) {
		By("Deactivating the archive logs target in Coralogix backend")
		Expect(crClient.Delete(ctx, &coralogixv1alpha1.ArchiveLogsTarget{ObjectMeta: metav1.ObjectMeta{Name: targetName, Namespace: testNamespace}})).To(Succeed())

		By("Verifying target is deactivated in Coralogix backend")
		Eventually(func(g Gomega) {
			target := getTarget(ctx, g)
			g.Expect(ptr.Deref(target.ArchiveSpec.IsActive, true)).To(BeFalse())
		}, accountWideBackendTimeout, time.Second).Should(Succeed(), "Storage target should be deactivated in Coralogix backend")
	})
})

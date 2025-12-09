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
	"fmt"
	"os"
	"time"

	archivemetrics "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/metrics_data_archive_service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("ArchiveMetricsTarget", Ordered, func() {
	var (
		crClient             client.Client
		archiveMetricsClient *cxsdk.ArchiveMetricsClient
		archiveMetricsTarget *coralogixv1alpha1.ArchiveMetricsTarget
		awsRegion            string
		metricsBucket        string
		targetName           = "s3-archivemetrics-target"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		awsRegion = os.Getenv("AWS_REGION")
		metricsBucket = os.Getenv("METRICS_BUCKET")
		archiveMetricsClient = ClientsInstance.GetCoralogixClientSet().ArchiveMetrics()
		archiveMetricsTarget = &coralogixv1alpha1.ArchiveMetricsTarget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      targetName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.ArchiveMetricsTargetSpec{
				S3Target: &coralogixv1alpha1.S3MetricsTarget{
					Region:     archivemetrics.PtrString(awsRegion),
					BucketName: archivemetrics.PtrString(metricsBucket),
				},
				ResolutionPolicy: &coralogixv1alpha1.ResolutionPolicy{
					RawResolution:         archivemetrics.PtrInt64(1),
					FiveMinutesResolution: archivemetrics.PtrInt64(1),
					OneHourResolution:     archivemetrics.PtrInt64(1),
				},
				RetentionDays: archivemetrics.PtrInt64(2),
			},
		}
	})

	It("Should be set successfully", func(ctx context.Context) {

		By("Setting a storage target in Coralogix backend")
		Expect(crClient.Create(ctx, archiveMetricsTarget)).To(Succeed())

		By("Fetching the storage target from the backend")
		fetchedTarget := &coralogixv1alpha1.ArchiveMetricsTarget{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: targetName, Namespace: testNamespace}, fetchedTarget)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedTarget.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedTarget.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetchedTarget.Status.ID != nil {
				return nil
			}
			return fmt.Errorf("archive logs target ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying the storage target is set in Coralogix backend")
		Eventually(func(g Gomega) {
			archiveMetricsTarget, err := archiveMetricsClient.Get(ctx)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(archiveMetricsTarget.TenantConfig.Disabled).To(BeFalse())
			g.Expect(archiveMetricsTarget.TenantConfig.GetS3().Bucket).To(Equal(metricsBucket))
			g.Expect(archiveMetricsTarget.TenantConfig.GetS3().Region).To(Equal(awsRegion))
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deactivated successfully", func(ctx context.Context) {
		By("Deactivating the archive metrics target in Coralogix backend")
		Expect(crClient.Delete(ctx, archiveMetricsTarget)).To(Succeed())

		By("Verifying target is deactivated in Coralogix backend")
		Eventually(func(g Gomega) {
			getTenantConfigResponse, err := archiveMetricsClient.Get(ctx)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getTenantConfigResponse.TenantConfig.Disabled).To(BeTrue())
		}, time.Minute, time.Second).Should(Succeed(), "Storage target should be deactivated in Coralogix backend")
	})

})

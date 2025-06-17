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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var _ = Describe("ArchiveLogsTarget", Ordered, func() {
	var (
		crClient          client.Client
		archiveLogsClient *cxsdk.ArchiveLogsClient
		archiveLogsTarget *coralogixv1alpha1.ArchiveLogsTarget
		awsRegion         string
		logsBucket        string
		targetName        = "s3-archivelogs-target"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		awsRegion = os.Getenv("AWS_REGION")
		logsBucket = os.Getenv("LOGS_BUCKET")
		archiveLogsClient = (*cxsdk.ArchiveLogsClient)(ClientsInstance.GetCoralogixClientSet().ArchiveLogs())
		archiveLogsTarget = &coralogixv1alpha1.ArchiveLogsTarget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      targetName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.ArchiveLogsTargetSpec{
				S3Target: &coralogixv1alpha1.S3Target{
					Region: awsRegion,
					Bucket: logsBucket,
				},
			},
		}
	})

	It("Should be set successfully", func(ctx context.Context) {

		By("Setting a storage target in Coralogix backend")
		Expect(crClient.Create(ctx, archiveLogsTarget)).To(Succeed())

		By("Fetching the storage target from the backend")
		fetchedTarget := &coralogixv1alpha1.ArchiveLogsTarget{}
		Eventually(func(g Gomega) error {
			ok := g.Expect(crClient.Get(ctx, types.NamespacedName{Name: targetName, Namespace: testNamespace}, fetchedTarget)).To(Succeed())
			if !ok {
				return fmt.Errorf("error fetching target")
			}
			for _, condition := range fetchedTarget.Status.Conditions {
				if condition.Type == "Failed" && condition.Status == "True" {
					return fmt.Errorf("archive logs target creation failed: %s", condition.Message)
				}
			}
			if fetchedTarget.Status.ID != nil {
				return nil
			}
			return fmt.Errorf("archive logs target ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying the storage target is set in Coralogix backend")
		Eventually(func() error {
			archiveLogsTarget, err := archiveLogsClient.Get(ctx)
			if err != nil {
				return fmt.Errorf("error fetching storage target: %w", err)
			}
			if !archiveLogsTarget.Target.GetArchiveSpec().IsActive {
				return fmt.Errorf("archive logs target is not active in Coralogix backend")
			}
			if archiveLogsTarget.Target.GetS3().Bucket == logsBucket &&
				*archiveLogsTarget.Target.GetS3().Region == awsRegion {
				return nil
			}
			return fmt.Errorf("archive logs target not found in Coralogix backend with expected values")
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deactivated successfully", func(ctx context.Context) {
		By("Deactivating the archive logs target in Coralogix backend")
		Expect(crClient.Delete(ctx, archiveLogsTarget)).To(Succeed())

		By("Verifying target is deactivated in Coralogix backend")
		Eventually(func() error {
			storageTarget, err := archiveLogsClient.Get(ctx)
			if err != nil {
				return fmt.Errorf("error fetching storage target: %w", err)
			}
			if storageTarget.Target.GetArchiveSpec().IsActive {
				return fmt.Errorf("archive logs target is still active in Coralogix backend")
			}
			return nil
		}, time.Minute, time.Second).Should(Succeed(), "Storage target should be deactivated in Coralogix backend")
	})

})

// Copyright 2024 Coralogix Ltd.
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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

const dashboardFolderName = "dashboard-folder-sample"

var _ = Describe("DashboardsFolder", Ordered, func() {
	var (
		crClient                client.Client
		dashboardsFoldersClient *cxsdk.DashboardsFoldersClient
		dashboardsFolder        *coralogixv1alpha1.DashboardsFolder
		dashboardFolderID       string
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		dashboardsFoldersClient = ClientsInstance.GetCoralogixClientSet().DashboardsFolders()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating DashboardsFolder")
		dashboardsFolder = getSampleDashboardsFolder("Test Dashboard Folder")

		Expect(crClient.Create(ctx, dashboardsFolder)).To(Succeed())

		By("Fetching the DashboardsFolder ID")
		fetchedDashboardsFolder := &coralogixv1alpha1.DashboardsFolder{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: dashboardFolderName, Namespace: testNamespace}, fetchedDashboardsFolder)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedDashboardsFolder.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedDashboardsFolder.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetchedDashboardsFolder.Status.ID != nil {
				dashboardFolderID = *fetchedDashboardsFolder.Status.ID
				return nil
			}
			return fmt.Errorf("DashboardsFolder ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying DashboardsFolder exists in Coralogix backend")
		Eventually(func() error {
			_, err := dashboardsFoldersClient.Get(ctx, &cxsdk.GetDashboardFolderRequest{FolderId: wrapperspb.String(dashboardFolderID)})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the DashboardsFolder")
		newName := "Test Updated Dashboard"
		modifiedDashboardsFolder := dashboardsFolder.DeepCopy()
		modifiedDashboardsFolder.Spec.Name = newName
		Expect(crClient.Patch(ctx, modifiedDashboardsFolder, client.MergeFrom(dashboardsFolder))).To(Succeed())

		By("Verifying DashboardsFolder is updated in Coralogix backend")
		Eventually(func() string {
			getDashboardRes, err := dashboardsFoldersClient.Get(ctx, &cxsdk.GetDashboardFolderRequest{FolderId: wrapperspb.String(dashboardFolderID)})
			Expect(err).ToNot(HaveOccurred())
			return getDashboardRes.GetFolder().GetName().GetValue()
		}, time.Minute, time.Second).Should(Equal(newName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the DashboardsFolder")
		Expect(crClient.Delete(ctx, dashboardsFolder)).To(Succeed())

		By("Verifying DashboardsFolder is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := dashboardsFoldersClient.Get(ctx, &cxsdk.GetDashboardFolderRequest{FolderId: wrapperspb.String(dashboardFolderID)})
			return cxsdk.Code(err)
		}).Should(Equal(codes.NotFound))
	})
})

func getSampleDashboardsFolder(name string) *coralogixv1alpha1.DashboardsFolder {
	return &coralogixv1alpha1.DashboardsFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboardFolderName,
			Namespace: testNamespace,
		},
		Spec: coralogixv1alpha1.DashboardsFolderSpec{
			Name: name,
		},
	}
}

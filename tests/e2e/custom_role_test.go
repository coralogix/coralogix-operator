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
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("CustomRole", Ordered, func() {
	var (
		crClient       client.Client
		rolesClient    *cxsdk.RolesClient
		customRoleID   uint32
		customRole     *coralogixv1alpha1.CustomRole
		customRoleName = fmt.Sprintf("custom-role-sample-%d", time.Now().Unix())
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		rolesClient = ClientsInstance.GetCoralogixClientSet().Roles()
		customRole = getSampleCustomRole(customRoleName, testNamespace)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating CustomRole")
		Expect(crClient.Create(ctx, customRole)).To(Succeed())

		By("Fetching the CustomRole ID")
		fetchedCustomRole := &coralogixv1alpha1.CustomRole{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: customRoleName, Namespace: testNamespace}, fetchedCustomRole)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedCustomRole.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedCustomRole.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetchedCustomRole.Status.ID != nil {
				id, err := strconv.Atoi(*fetchedCustomRole.Status.ID)
				Expect(err).ToNot(HaveOccurred())
				customRoleID = uint32(id)
				return nil
			}
			return fmt.Errorf("CustomRole ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying CustomRole exists in Coralogix backend")
		Eventually(func() error {
			_, err := rolesClient.Get(ctx, &cxsdk.GetCustomRoleRequest{RoleId: customRoleID})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the CustomRole")
		newCustomRoleName := "custom-role-updated"
		modifiedCustomRole := customRole.DeepCopy()
		modifiedCustomRole.Spec.Name = newCustomRoleName
		Expect(crClient.Patch(ctx, modifiedCustomRole, client.MergeFrom(customRole))).To(Succeed())

		By("Verifying CustomRole is updated in Coralogix backend")
		Eventually(func() string {
			getCustomRoleRes, err := rolesClient.Get(ctx, &cxsdk.GetCustomRoleRequest{RoleId: customRoleID})
			Expect(err).ToNot(HaveOccurred())
			return getCustomRoleRes.Role.Name
		}, time.Minute, time.Second).Should(Equal(newCustomRoleName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the CustomRole")
		Expect(crClient.Delete(ctx, customRole)).To(Succeed())

		By("Verifying CustomRole is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := rolesClient.Get(ctx, &cxsdk.GetCustomRoleRequest{RoleId: customRoleID})
			return cxsdk.Code(err)
		}).Should(Equal(codes.NotFound))
	})
})

func getSampleCustomRole(name, namespace string) *coralogixv1alpha1.CustomRole {
	return &coralogixv1alpha1.CustomRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.CustomRoleSpec{
			Name:           name,
			Description:    "This is a sample custom role",
			ParentRoleName: "Standard User",
			Permissions: []string{
				"team-actions:UpdateConfig",
				"TEAM-CUSTOM-API-KEYS:READCONFIG",
			},
		},
	}
}

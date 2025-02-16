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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var _ = Describe("Group", Ordered, func() {
	var (
		crClient       client.Client
		groupsClient   *cxsdk.GroupsClient
		scope          *coralogixv1alpha1.Scope
		customRole     *coralogixv1alpha1.CustomRole
		group          *coralogixv1alpha1.Group
		groupID        uint32
		groupName      = fmt.Sprintf("group-sample-%d", time.Now().Unix())
		scopeName      = "scope-for-group"
		customRoleName = "custom-role-for-group"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		groupsClient = ClientsInstance.GetCoralogixClientSet().Groups()
		scope = getSampleScope(scopeName, testNamespace)
		customRole = getSampleCustomRole(customRoleName, testNamespace)
		group = &coralogixv1alpha1.Group{
			ObjectMeta: metav1.ObjectMeta{
				Name:      groupName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.GroupSpec{
				Name:        groupName,
				Description: ptr.To("This is a sample group"),
				Members: []coralogixv1alpha1.Member{
					{UserName: "example@coralogix.com"},
					{UserName: "example2@coralogix.com"},
				},
				Scope: &coralogixv1alpha1.GroupScope{
					ResourceRef: &coralogixv1alpha1.ResourceRef{
						Name: scopeName,
					},
				},
				CustomRoles: []coralogixv1alpha1.GroupCustomRole{
					{
						ResourceRef: &coralogixv1alpha1.ResourceRef{
							Name: customRoleName,
						},
					},
				},
			},
		}
	})

	It("Should create dependent resources successfully", func(ctx context.Context) {
		By("Creating Scope and CustomRole")
		Expect(crClient.Create(ctx, scope)).To(Succeed())
		Expect(crClient.Create(ctx, customRole)).To(Succeed())
	})

	It("Should create Group successfully", func(ctx context.Context) {
		By("Creating Group")
		Expect(crClient.Create(ctx, group)).To(Succeed())

		By("Fetching the Group ID")
		fetchedGroup := &coralogixv1alpha1.Group{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: groupName, Namespace: testNamespace}, fetchedGroup)).To(Succeed())
			if fetchedGroup.Status.ID != nil {
				id, err := strconv.Atoi(*fetchedGroup.Status.ID)
				Expect(err).ToNot(HaveOccurred())
				groupID = uint32(id)
				return nil
			}
			return fmt.Errorf("group ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying Group exists in Coralogix backend")
		Eventually(func() error {
			_, err := groupsClient.Get(ctx, &cxsdk.GetTeamGroupRequest{
				GroupId: &cxsdk.TeamGroupID{Id: groupID},
			})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Updating the Group")
		newGroupName := "group-sample-updated"
		modifiedGroup := group.DeepCopy()
		modifiedGroup.Spec.Name = newGroupName
		Expect(crClient.Patch(ctx, modifiedGroup, client.MergeFrom(group))).To(Succeed())

		By("Verifying Group is updated in Coralogix backend")
		Eventually(func() string {
			getGroupRes, err := groupsClient.Get(ctx, &cxsdk.GetTeamGroupRequest{
				GroupId: &cxsdk.TeamGroupID{Id: groupID},
			})
			Expect(err).ToNot(HaveOccurred())
			return getGroupRes.Group.Name
		}, time.Minute, time.Second).Should(Equal(newGroupName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Group")
		Expect(crClient.Delete(ctx, group)).To(Succeed())

		By("Verifying Group is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := groupsClient.Get(ctx, &cxsdk.GetTeamGroupRequest{
				GroupId: &cxsdk.TeamGroupID{Id: uint32(groupID)},
			})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})

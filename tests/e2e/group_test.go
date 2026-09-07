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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("Group", Ordered, func() {
	var (
		crClient       client.Client
		scope          *coralogixv1alpha1.Scope
		customRole     *coralogixv1alpha1.CustomRole
		group          *coralogixv1alpha1.Group
		groupID        int64
		groupName      = uniqueName("group-sample")
		scopeName      = uniqueName("scope-for-group")
		customRoleName = uniqueName("custom-role-for-group")
		groupKey       types.NamespacedName
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		groupKey = types.NamespacedName{Name: groupName, Namespace: testNamespace}
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
				GroupType:   ptr.To("open"),
				Members: []coralogixv1alpha1.Member{
					{UserName: groupFixtureUserA},
					{UserName: groupFixtureUserB},
				},
				Scope: &coralogixv1alpha1.GroupScope{
					ResourceRef: coralogixv1alpha1.ResourceRef{
						Name: scopeName,
					},
				},
				CustomRole: &coralogixv1alpha1.GroupCustomRole{
					ResourceRef: coralogixv1alpha1.ResourceRef{
						Name: customRoleName,
					},
				},
			},
		}
	})

	It("Should create dependent resources successfully", func(ctx context.Context) {
		By("Creating Scope and CustomRole")
		Expect(crClient.Create(ctx, scope)).To(Succeed())
		Expect(crClient.Create(ctx, customRole)).To(Succeed())

		By("Waiting for Scope to be RemoteSynced")
		Eventually(func(g Gomega) {
			fetchedScope := &coralogixv1alpha1.Scope{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: scopeName, Namespace: testNamespace}, fetchedScope)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedScope.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedScope.Status.ID).ToNot(BeNil())
		}, time.Minute, time.Second).Should(Succeed())

		By("Waiting for CustomRole to be RemoteSynced")
		Eventually(func(g Gomega) {
			fetchedRole := &coralogixv1alpha1.CustomRole{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: customRoleName, Namespace: testNamespace}, fetchedRole)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedRole.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedRole.Status.ID).ToNot(BeNil())
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should create Group successfully", func(ctx context.Context) {
		By("Creating Group")
		Expect(crClient.Create(ctx, group)).To(Succeed())

		fetchedGroup := waitForGroupRemoteSynced(ctx, crClient, groupKey)
		groupID = parseGroupID(*fetchedGroup.Status.ID)

		By("Verifying Group members were resolved through Users OpenAPI")
		expectGroupMembers(ctx, groupID, groupFixtureUserA, groupFixtureUserB)
		expectRemoteGroupName(ctx, groupID, groupName)
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Updating the Group")
		newGroupName := uniqueName("group-sample-updated")
		modifiedGroup := group.DeepCopy()
		modifiedGroup.Spec.Name = newGroupName
		Expect(crClient.Patch(ctx, modifiedGroup, client.MergeFrom(group))).To(Succeed())

		By("Verifying Group is updated in Coralogix backend")
		expectRemoteGroupName(ctx, groupID, newGroupName)
		expectGroupMembers(ctx, groupID, groupFixtureUserA, groupFixtureUserB)
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Group")
		Expect(crClient.Delete(ctx, group)).To(Succeed())

		By("Verifying Group is deleted from Coralogix backend")
		expectRemoteGroupGone(ctx, groupID)
	})
})

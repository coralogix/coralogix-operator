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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("Group SCIM to OpenAPI migration", Serial, Ordered, Label("scim-migration"), func() {
	var (
		crClient       client.Client
		scope          *coralogixv1alpha1.Scope
		customRole     *coralogixv1alpha1.CustomRole
		group          *coralogixv1alpha1.Group
		groupID        int64
		groupName      = uniqueName("group-scim-migration")
		scopeName      = uniqueName("scope-for-group-migration")
		customRoleName = uniqueName("custom-role-for-group-migration")
		groupKey       types.NamespacedName
		remoteIDBefore string
	)

	BeforeEach(func() {
		if os.Getenv("E2E_GROUP_SCIM_MIGRATION") == "" {
			Skip("set E2E_GROUP_SCIM_MIGRATION=1 to run the SCIM-to-OpenAPI Group upgrade spec")
		}
		if os.Getenv("E2E_UPGRADE_IMAGE") == "" {
			Skip("set E2E_UPGRADE_IMAGE to the PR operator image")
		}
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
				Description: ptr.To("Group created by a SCIM-backed operator"),
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

	It("creates the Group with the released SCIM operator", func(ctx context.Context) {
		Expect(crClient.Create(ctx, scope)).To(Succeed())
		Expect(crClient.Create(ctx, customRole)).To(Succeed())

		Eventually(func(g Gomega) {
			fetchedScope := &coralogixv1alpha1.Scope{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: scopeName, Namespace: testNamespace}, fetchedScope)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedScope.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
		}, time.Minute, time.Second).Should(Succeed())
		Eventually(func(g Gomega) {
			fetchedRole := &coralogixv1alpha1.CustomRole{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: customRoleName, Namespace: testNamespace}, fetchedRole)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedRole.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
		}, time.Minute, time.Second).Should(Succeed())

		Expect(crClient.Create(ctx, group)).To(Succeed())
		fetched := waitForGroupRemoteSynced(ctx, crClient, groupKey)
		remoteIDBefore = *fetched.Status.ID
		groupID = parseGroupID(remoteIDBefore)
		expectGroupMembers(ctx, groupID, groupFixtureUserA, groupFixtureUserB)
	})

	It("keeps the Group synced after upgrade to the OpenAPI operator", func(ctx context.Context) {
		upgradeOperatorImage(ctx, os.Getenv("E2E_UPGRADE_IMAGE"))

		Consistently(func(g Gomega) {
			fetched := &coralogixv1alpha1.Group{}
			g.Expect(crClient.Get(ctx, groupKey, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.ID).ToNot(BeNil())
			g.Expect(*fetched.Status.ID).To(Equal(remoteIDBefore))
		}, 20*time.Second, time.Second).Should(Succeed())

		expectGroupMembers(ctx, groupID, groupFixtureUserA, groupFixtureUserB)
	})

	It("updates the Group with the OpenAPI operator", func(ctx context.Context) {
		newGroupName := uniqueName("group-scim-migration-updated")
		current := &coralogixv1alpha1.Group{}
		Expect(crClient.Get(ctx, groupKey, current)).To(Succeed())
		modified := current.DeepCopy()
		modified.Spec.Name = newGroupName
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(current))).To(Succeed())

		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.Group{}
			g.Expect(crClient.Get(ctx, groupKey, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.ID).ToNot(BeNil())
			g.Expect(*fetched.Status.ID).To(Equal(remoteIDBefore))
			cond := meta.FindStatusCondition(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)
			g.Expect(cond).ToNot(BeNil())
			g.Expect(cond.ObservedGeneration).To(Equal(fetched.Generation))
		}, time.Minute, time.Second).Should(Succeed())

		expectRemoteGroupName(ctx, groupID, newGroupName)
		expectGroupMembers(ctx, groupID, groupFixtureUserA, groupFixtureUserB)
	})

	It("deletes the Group after the upgrade", func(ctx context.Context) {
		Expect(crClient.Delete(ctx, group)).To(Succeed())
		expectRemoteGroupGone(ctx, groupID)
	})
})

func upgradeOperatorImage(ctx context.Context, image string) {
	k8sClient := ClientsInstance.GetK8sClient()
	var depName string
	Eventually(func(g Gomega) {
		depList, err := k8sClient.AppsV1().
			Deployments("coralogix-operator-system").
			List(ctx, metav1.ListOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(depList.Items).ToNot(BeEmpty())
		depName = depList.Items[0].Name
	}, time.Minute, time.Second).Should(Succeed())

	By("Rolling the operator to " + image)
	Eventually(func(g Gomega) {
		dep, err := k8sClient.AppsV1().
			Deployments("coralogix-operator-system").
			Get(ctx, depName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(dep.Spec.Template.Spec.Containers).ToNot(BeEmpty())
		dep.Spec.Template.Spec.Containers[0].Image = image
		dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
		_, err = k8sClient.AppsV1().
			Deployments("coralogix-operator-system").
			Update(ctx, dep, metav1.UpdateOptions{})
		g.Expect(err).ToNot(HaveOccurred())
	}, time.Minute, time.Second).Should(Succeed())

	Eventually(func(g Gomega) {
		updated, err := k8sClient.AppsV1().
			Deployments("coralogix-operator-system").
			Get(ctx, depName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated.Spec.Template.Spec.Containers[0].Image).To(Equal(image))
		g.Expect(updated.Spec.Replicas).ToNot(BeNil())
		g.Expect(updated.Status.UpdatedReplicas).To(Equal(*updated.Spec.Replicas))
		g.Expect(updated.Status.ReadyReplicas).To(Equal(*updated.Spec.Replicas))
		g.Expect(updated.Status.UnavailableReplicas).To(BeZero())
		available := false
		for _, condition := range updated.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
				available = true
			}
		}
		g.Expect(available).To(BeTrue())
	}, 2*time.Minute, time.Second).Should(Succeed())
}

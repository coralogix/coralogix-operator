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
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

const operatorNamespace = "coralogix-operator-system"

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

		// Helm chart 1.0 still requires spec.customRoles. The current typed Group
		// sends spec.customRole and is rejected by that CRD.
		Expect(crClient.Create(ctx, legacyHelmGroup(groupName, scopeName, customRoleName))).To(Succeed())
		fetched := waitForGroupRemoteSynced(ctx, crClient, groupKey)
		remoteIDBefore = *fetched.Status.ID
		groupID = parseGroupID(remoteIDBefore)
		expectGroupMembers(ctx, groupID, groupFixtureUserA, groupFixtureUserB)
	})

	It("keeps the Group synced after upgrade to the OpenAPI operator", func(ctx context.Context) {
		upgradeToOpenAPIOperator(ctx, crClient, groupKey, customRoleName, os.Getenv("E2E_UPGRADE_IMAGE"))

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

func legacyHelmGroup(name, scopeName, customRoleName string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "coralogix.com/v1alpha1",
		"kind":       "Group",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": testNamespace,
		},
		"spec": map[string]interface{}{
			"name":        name,
			"description": "Group created by a SCIM-backed operator",
			"members": []interface{}{
				map[string]interface{}{"userName": groupFixtureUserA},
				map[string]interface{}{"userName": groupFixtureUserB},
			},
			"customRoles": []interface{}{
				map[string]interface{}{
					"resourceRef": map[string]interface{}{"name": customRoleName},
				},
			},
			"scope": map[string]interface{}{
				"resourceRef": map[string]interface{}{"name": scopeName},
			},
		},
	}}
	return u
}

func upgradeToOpenAPIOperator(
	ctx context.Context,
	crClient client.Client,
	groupKey types.NamespacedName,
	customRoleName string,
	image string,
) {
	depName := operatorDeploymentName(ctx)

	// Stop the SCIM operator before the Group CRD rename. Otherwise it can
	// reconcile a Group that no longer has spec.customRoles and clear the role.
	By("Scaling the operator to 0")
	scaleOperator(ctx, depName, 0)
	waitForOperatorPods(ctx, depName, 0, "")

	By("Applying the current Group CRD")
	applyCurrentGroupCRD(ctx)
	rewriteGroupCustomRole(ctx, crClient, groupKey, customRoleName)

	By("Rolling the operator to " + image)
	Eventually(func(g Gomega) {
		dep, err := ClientsInstance.GetK8sClient().AppsV1().
			Deployments(operatorNamespace).
			Get(ctx, depName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(dep.Spec.Template.Spec.Containers).ToNot(BeEmpty())
		dep.Spec.Replicas = ptr.To(int32(1))
		dep.Spec.Template.Spec.Containers[0].Image = image
		dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
		_, err = ClientsInstance.GetK8sClient().AppsV1().
			Deployments(operatorNamespace).
			Update(ctx, dep, metav1.UpdateOptions{})
		g.Expect(err).ToNot(HaveOccurred())
	}, time.Minute, time.Second).Should(Succeed())

	waitForOperatorRollout(ctx, depName, image)
}

func operatorDeploymentName(ctx context.Context) string {
	var depName string
	Eventually(func(g Gomega) {
		depList, err := ClientsInstance.GetK8sClient().AppsV1().
			Deployments(operatorNamespace).
			List(ctx, metav1.ListOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(depList.Items).ToNot(BeEmpty())
		depName = depList.Items[0].Name
	}, time.Minute, time.Second).Should(Succeed())
	return depName
}

func scaleOperator(ctx context.Context, depName string, replicas int32) {
	Eventually(func(g Gomega) {
		dep, err := ClientsInstance.GetK8sClient().AppsV1().
			Deployments(operatorNamespace).
			Get(ctx, depName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		dep.Spec.Replicas = ptr.To(replicas)
		_, err = ClientsInstance.GetK8sClient().AppsV1().
			Deployments(operatorNamespace).
			Update(ctx, dep, metav1.UpdateOptions{})
		g.Expect(err).ToNot(HaveOccurred())
	}, time.Minute, time.Second).Should(Succeed())
}

func applyCurrentGroupCRD(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "config/crd/bases/coralogix.com_groups.yaml")
	out, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred(), string(out))
}

func rewriteGroupCustomRole(
	ctx context.Context,
	crClient client.Client,
	groupKey types.NamespacedName,
	customRoleName string,
) {
	By("Rewriting the Group to spec.customRole")
	Eventually(func(g Gomega) {
		current := &coralogixv1alpha1.Group{}
		g.Expect(crClient.Get(ctx, groupKey, current)).To(Succeed())
		modified := current.DeepCopy()
		modified.Spec.CustomRole = &coralogixv1alpha1.GroupCustomRole{
			ResourceRef: coralogixv1alpha1.ResourceRef{Name: customRoleName},
		}
		g.Expect(crClient.Patch(ctx, modified, client.MergeFrom(current))).To(Succeed())
		fetched := &coralogixv1alpha1.Group{}
		g.Expect(crClient.Get(ctx, groupKey, fetched)).To(Succeed())
		g.Expect(fetched.Spec.CustomRole).ToNot(BeNil())
		g.Expect(fetched.Spec.CustomRole.ResourceRef.Name).To(Equal(customRoleName))
	}, time.Minute, time.Second).Should(Succeed())
}

func waitForOperatorRollout(ctx context.Context, depName, image string) {
	Eventually(func(g Gomega) {
		updated, err := ClientsInstance.GetK8sClient().AppsV1().
			Deployments(operatorNamespace).
			Get(ctx, depName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated.Spec.Template.Spec.Containers[0].Image).To(Equal(image))
		g.Expect(updated.Spec.Replicas).ToNot(BeNil())
		want := *updated.Spec.Replicas
		g.Expect(updated.Status.UpdatedReplicas).To(Equal(want))
		g.Expect(updated.Status.Replicas).To(Equal(updated.Status.UpdatedReplicas))
		g.Expect(updated.Status.AvailableReplicas).To(Equal(updated.Status.UpdatedReplicas))
		g.Expect(updated.Status.ReadyReplicas).To(Equal(want))
		g.Expect(updated.Status.UnavailableReplicas).To(BeZero())
		available := false
		for _, condition := range updated.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
				available = true
			}
		}
		g.Expect(available).To(BeTrue())
		assertOperatorPodsAndReplicaSets(ctx, g, updated, image)
	}, 2*time.Minute, time.Second).Should(Succeed())
}

func waitForOperatorPods(ctx context.Context, depName string, want int, image string) {
	Eventually(func(g Gomega) {
		dep, err := ClientsInstance.GetK8sClient().AppsV1().
			Deployments(operatorNamespace).
			Get(ctx, depName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(dep.Spec.Replicas).ToNot(BeNil())
		g.Expect(*dep.Spec.Replicas).To(Equal(int32(want)))
		g.Expect(dep.Status.Replicas).To(Equal(int32(want)))
		if want == 0 {
			assertOperatorPodsAndReplicaSets(ctx, g, dep, "")
			return
		}
		g.Expect(dep.Status.UpdatedReplicas).To(Equal(int32(want)))
		g.Expect(dep.Status.ReadyReplicas).To(Equal(int32(want)))
		assertOperatorPodsAndReplicaSets(ctx, g, dep, image)
	}, 2*time.Minute, time.Second).Should(Succeed())
}

func assertOperatorPodsAndReplicaSets(ctx context.Context, g Gomega, dep *appsv1.Deployment, image string) {
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	g.Expect(err).ToNot(HaveOccurred())

	pods, err := ClientsInstance.GetK8sClient().CoreV1().
		Pods(operatorNamespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	g.Expect(err).ToNot(HaveOccurred())

	live := make([]corev1.Pod, 0, len(pods.Items))
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		live = append(live, pod)
	}
	want := 0
	if dep.Spec.Replicas != nil {
		want = int(*dep.Spec.Replicas)
	}
	g.Expect(live).To(HaveLen(want))
	for _, pod := range live {
		g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
		g.Expect(pod.Spec.Containers).ToNot(BeEmpty())
		if image != "" {
			g.Expect(pod.Spec.Containers[0].Image).To(Equal(image))
		}
	}

	replicaSets, err := ClientsInstance.GetK8sClient().AppsV1().
		ReplicaSets(operatorNamespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	g.Expect(err).ToNot(HaveOccurred())
	active := 0
	for _, rs := range replicaSets.Items {
		if !metav1.IsControlledBy(&rs, dep) {
			continue
		}
		if rsReplicas(rs) == 0 && rs.Status.Replicas == 0 {
			continue
		}
		active++
		if image != "" {
			g.Expect(rs.Spec.Template.Spec.Containers).ToNot(BeEmpty())
			g.Expect(rs.Spec.Template.Spec.Containers[0].Image).To(Equal(image))
		}
	}
	if want == 0 {
		g.Expect(active).To(BeZero())
		return
	}
	g.Expect(active).To(Equal(1))
}

func rsReplicas(rs appsv1.ReplicaSet) int32 {
	if rs.Spec.Replicas == nil {
		return 0
	}
	return *rs.Spec.Replicas
}

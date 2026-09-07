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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
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
		roleIDBefore   int64
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
		fetchedRole := &coralogixv1alpha1.CustomRole{}
		Eventually(func(g Gomega) {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: customRoleName, Namespace: testNamespace}, fetchedRole)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedRole.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedRole.Status.ID).ToNot(BeNil())
		}, time.Minute, time.Second).Should(Succeed())
		parsedRoleID, err := strconv.ParseInt(*fetchedRole.Status.ID, 10, 64)
		Expect(err).ToNot(HaveOccurred())
		roleIDBefore = parsedRoleID

		// Helm chart 1.0 still requires spec.customRoles. The current typed Group
		// sends spec.customRole and is rejected by that CRD.
		Expect(crClient.Create(ctx, legacyHelmGroup(groupName, scopeName, customRoleName))).To(Succeed())
		fetched := waitForReleasedGroupRemoteSynced(ctx, crClient, groupKey)
		remoteIDBefore = *fetched.Status.ID
		groupID = parseGroupID(remoteIDBefore)
		expectGroupMembers(ctx, groupID, groupFixtureUserA, groupFixtureUserB)
		expectRemoteGroupRole(ctx, groupID, roleIDBefore)
	})

	It("keeps the Group synced after helm upgrade to the OpenAPI operator", func(ctx context.Context) {
		helmUpgradeToOpenAPIOperator(ctx, os.Getenv("E2E_UPGRADE_IMAGE"))

		Consistently(func(g Gomega) {
			fetched := &coralogixv1alpha1.Group{}
			g.Expect(crClient.Get(ctx, groupKey, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.ID).ToNot(BeNil())
			g.Expect(*fetched.Status.ID).To(Equal(remoteIDBefore))
		}, 20*time.Second, time.Second).Should(Succeed())

		expectGroupMembers(ctx, groupID, groupFixtureUserA, groupFixtureUserB)
		expectRemoteGroupRole(ctx, groupID, roleIDBefore)
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
		expectRemoteGroupRole(ctx, groupID, roleIDBefore)
	})

	It("deletes the Group after the upgrade", func(ctx context.Context) {
		Expect(crClient.Delete(ctx, group)).To(Succeed())
		expectRemoteGroupGone(ctx, groupID)
	})
})

func legacyHelmGroup(name, scopeName, customRoleName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
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
}

func helmUpgradeToOpenAPIOperator(ctx context.Context, image string) {
	repo, tag := helmImageRepoAndTag(image)
	apiKey := os.Getenv("CORALOGIX_API_KEY")
	region := os.Getenv("CORALOGIX_REGION")
	Expect(apiKey).ToNot(BeEmpty())
	Expect(region).ToNot(BeEmpty())

	// Helm upgrade updates CRDs and RBAC with the image. Patching only the
	// image leaves Helm 1.0 RBAC, and this build exits on startup because it
	// cannot get prometheusrules.monitoring.coreos.com.
	By("Helm upgrading the operator to " + image)
	cmd := exec.CommandContext(ctx, "helm", "upgrade", "coralogix-operator", operatorChartPath(),
		"--namespace", operatorNamespace,
		"--wait",
		"--timeout", "3m",
		"--set", "secret.data.apiKey="+apiKey,
		"--set", "coralogixOperator.image.repository="+repo,
		"--set", "coralogixOperator.image.tag="+tag,
		"--set", "coralogixOperator.image.pullPolicy=IfNotPresent",
		"--set", "coralogixOperator.region="+region,
	)
	out, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred(), string(out))

	waitForOperatorRollout(ctx, operatorDeploymentName(ctx), image)
}

func helmImageRepoAndTag(image string) (string, string) {
	repo, tag, found := strings.Cut(image, ":")
	Expect(found).To(BeTrue(), "E2E_UPGRADE_IMAGE must be repository:tag")
	Expect(repo).ToNot(BeEmpty())
	Expect(tag).ToNot(BeEmpty())
	return repo, strings.TrimPrefix(tag, "v")
}

func operatorChartPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	Expect(ok).To(BeTrue())
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "charts", "coralogix-operator")
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

func waitForOperatorRollout(ctx context.Context, depName, image string) {
	Eventually(func(g Gomega) {
		updated, err := ClientsInstance.GetK8sClient().AppsV1().
			Deployments(operatorNamespace).
			Get(ctx, depName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(updated.Spec.Template.Spec.Containers[0].Image).To(Equal(image))
		g.Expect(updated.Spec.Replicas).ToNot(BeNil())
		want := *updated.Spec.Replicas
		g.Expect(updated.Status.UpdatedReplicas).To(Equal(want), describeOperatorPods(ctx, updated))
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

func describeOperatorPods(ctx context.Context, dep *appsv1.Deployment) string {
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return err.Error()
	}
	pods, err := ClientsInstance.GetK8sClient().CoreV1().
		Pods(operatorNamespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err.Error()
	}
	lines := []string{
		fmt.Sprintf("replicas ready=%d updated=%d available=%d unavailable=%d",
			dep.Status.ReadyReplicas, dep.Status.UpdatedReplicas,
			dep.Status.AvailableReplicas, dep.Status.UnavailableReplicas),
	}
	for _, pod := range pods.Items {
		reason := string(pod.Status.Phase)
		if pod.Status.Reason != "" {
			reason = pod.Status.Reason
		}
		image := ""
		if len(pod.Spec.Containers) > 0 {
			image = pod.Spec.Containers[0].Image
		}
		lines = append(lines, fmt.Sprintf("pod %s phase=%s reason=%s image=%s",
			pod.Name, pod.Status.Phase, reason, image))
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				lines = append(lines, fmt.Sprintf("  waiting %s: %s", cs.State.Waiting.Reason, cs.State.Waiting.Message))
			}
			if cs.State.Terminated != nil {
				lines = append(lines, fmt.Sprintf("  terminated %s: %s", cs.State.Terminated.Reason, cs.State.Terminated.Message))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func rsReplicas(rs appsv1.ReplicaSet) int32 {
	if rs.Spec.Replicas == nil {
		return 0
	}
	return *rs.Spec.Replicas
}

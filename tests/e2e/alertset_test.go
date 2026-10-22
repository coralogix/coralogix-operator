// Copyright 2026 Coralogix Ltd.
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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("AlertSet", Ordered, func() {
	var (
		crClient       client.Client
		alertsClient   *cxsdk.AlertsClient
		alertSet       *coralogixv1alpha1.AlertSet
		firstID        string
		secondID       string
		replacementID  string
		alertSetSuffix string
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		alertsClient = ClientsInstance.GetCoralogixClientSet().Alerts()
		alertSetSuffix = uuid.NewString()[:8]
		alertSet = &coralogixv1alpha1.AlertSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "alert-set-" + alertSetSuffix,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{
				newE2EAlertSetItem("first-alert", "AlertSet first "+alertSetSuffix),
				newE2EAlertSetItem("second-alert", "AlertSet second "+alertSetSuffix),
			}},
		}
	})

	It("creates all alerts and stores their IDs", func(ctx context.Context) {
		Expect(crClient.Create(ctx, alertSet)).To(Succeed())

		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.AlertSet{}
			g.Expect(crClient.Get(ctx, client.ObjectKeyFromObject(alertSet), fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.Alerts).To(HaveLen(2))
			statuses := alertSetStatusesByKey(fetched.Status.Alerts)
			g.Expect(statuses["first-alert"].ID).NotTo(BeNil())
			g.Expect(statuses["second-alert"].ID).NotTo(BeNil())
			firstID = *statuses["first-alert"].ID
			secondID = *statuses["second-alert"].ID
			alertSet = fetched
		}, time.Minute, time.Second).Should(Succeed())

		verifyRemoteAlertName(ctx, alertsClient, firstID, "AlertSet first "+alertSetSuffix)
		verifyRemoteAlertName(ctx, alertsClient, secondID, "AlertSet second "+alertSetSuffix)
	})

	It("uses keys as identity when items are reordered and updated", func(ctx context.Context) {
		updated := alertSet.DeepCopy()
		updated.Spec.Alerts[0].Spec.Name = "AlertSet first updated " + alertSetSuffix
		updated.Spec.Alerts[0], updated.Spec.Alerts[1] = updated.Spec.Alerts[1], updated.Spec.Alerts[0]
		Expect(crClient.Patch(ctx, updated, client.MergeFrom(alertSet))).To(Succeed())

		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.AlertSet{}
			g.Expect(crClient.Get(ctx, client.ObjectKeyFromObject(alertSet), fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			statuses := alertSetStatusesByKey(fetched.Status.Alerts)
			g.Expect(*statuses["first-alert"].ID).To(Equal(firstID))
			g.Expect(*statuses["second-alert"].ID).To(Equal(secondID))
			alertSet = fetched
		}, time.Minute, time.Second).Should(Succeed())

		verifyRemoteAlertName(ctx, alertsClient, firstID, "AlertSet first updated "+alertSetSuffix)
	})

	It("deletes and creates an alert when its key changes", func(ctx context.Context) {
		updated := alertSet.DeepCopy()
		for index := range updated.Spec.Alerts {
			if updated.Spec.Alerts[index].Key == "second-alert" {
				updated.Spec.Alerts[index].Key = "replacement-alert"
				updated.Spec.Alerts[index].Spec.Name = "AlertSet replacement " + alertSetSuffix
			}
		}
		Expect(crClient.Patch(ctx, updated, client.MergeFrom(alertSet))).To(Succeed())

		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.AlertSet{}
			g.Expect(crClient.Get(ctx, client.ObjectKeyFromObject(alertSet), fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			statuses := alertSetStatusesByKey(fetched.Status.Alerts)
			g.Expect(statuses).NotTo(HaveKey("second-alert"))
			g.Expect(statuses["replacement-alert"].ID).NotTo(BeNil())
			replacementID = *statuses["replacement-alert"].ID
			g.Expect(replacementID).NotTo(Equal(secondID))
			alertSet = fetched
		}, time.Minute, time.Second).Should(Succeed())

		verifyRemoteAlertDeleted(ctx, alertsClient, secondID)
		verifyRemoteAlertName(ctx, alertsClient, replacementID, "AlertSet replacement "+alertSetSuffix)
	})

	It("removes an item and then deletes the AlertSet", func(ctx context.Context) {
		updated := alertSet.DeepCopy()
		for index := range updated.Spec.Alerts {
			if updated.Spec.Alerts[index].Key == "first-alert" {
				updated.Spec.Alerts = append(updated.Spec.Alerts[:index], updated.Spec.Alerts[index+1:]...)
				break
			}
		}
		Expect(crClient.Patch(ctx, updated, client.MergeFrom(alertSet))).To(Succeed())

		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.AlertSet{}
			g.Expect(crClient.Get(ctx, client.ObjectKeyFromObject(alertSet), fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.Alerts).To(HaveLen(1))
			g.Expect(fetched.Status.Alerts[0].Key).To(Equal("replacement-alert"))
			alertSet = fetched
		}, time.Minute, time.Second).Should(Succeed())
		verifyRemoteAlertDeleted(ctx, alertsClient, firstID)

		Expect(crClient.Delete(ctx, alertSet)).To(Succeed())
		Eventually(func() bool {
			fetched := &coralogixv1alpha1.AlertSet{}
			err := crClient.Get(ctx, types.NamespacedName{Name: alertSet.Name, Namespace: alertSet.Namespace}, fetched)
			return client.IgnoreNotFound(err) == nil && err != nil
		}, time.Minute, time.Second).Should(BeTrue())
		verifyRemoteAlertDeleted(ctx, alertsClient, replacementID)
	})
})

func newE2EAlertSetItem(key, name string) coralogixv1alpha1.AlertSetItem {
	query := fmt.Sprintf("applicationName:%s", key)
	return coralogixv1alpha1.AlertSetItem{
		Key: key,
		Spec: coralogixv1beta1.AlertSpec{
			Name:     name,
			Priority: coralogixv1beta1.AlertPriorityP5,
			TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
				LogsImmediate: &coralogixv1beta1.LogsImmediate{
					LogsFilter: &coralogixv1beta1.LogsFilter{
						SimpleFilter: coralogixv1beta1.LogsSimpleFilter{LuceneQuery: &query},
					},
				},
			},
		},
	}
}

func alertSetStatusesByKey(statuses []coralogixv1alpha1.AlertSetItemStatus) map[string]coralogixv1alpha1.AlertSetItemStatus {
	result := make(map[string]coralogixv1alpha1.AlertSetItemStatus, len(statuses))
	for _, status := range statuses {
		result[status.Key] = status
	}
	return result
}

func verifyRemoteAlertName(ctx context.Context, alertsClient *cxsdk.AlertsClient, id, expectedName string) {
	Eventually(func(g Gomega) string {
		response, err := alertsClient.Get(ctx, &cxsdk.GetAlertDefRequest{Id: wrapperspb.String(id)})
		g.Expect(err).NotTo(HaveOccurred())
		return response.GetAlertDef().GetAlertDefProperties().GetName().GetValue()
	}, time.Minute, time.Second).Should(Equal(expectedName))
}

func verifyRemoteAlertDeleted(ctx context.Context, alertsClient *cxsdk.AlertsClient, id string) {
	Eventually(func() codes.Code {
		_, err := alertsClient.Get(ctx, &cxsdk.GetAlertDefRequest{Id: wrapperspb.String(id)})
		return cxsdk.Code(err)
	}, time.Minute, time.Second).Should(Equal(codes.NotFound))
}

var _ = Describe("AlertSet access policy", Ordered, func() {
	var (
		crClient      client.Client
		alertsClient  *cxsdk.AlertsClient
		openAPIAlerts *alerts.AlertDefinitionsServiceAPIService
		alertSetName  string
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		alertsClient = ClientsInstance.GetCoralogixClientSet().Alerts()
		openAPIAlerts = newOpenAPIClientSet().Alerts()
	})

	patchItems := func(ctx context.Context, mutate func(items []coralogixv1alpha1.AlertSetItem)) *coralogixv1alpha1.AlertSet {
		current := &coralogixv1alpha1.AlertSet{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, current)).To(Succeed())
		updated := current.DeepCopy()
		mutate(updated.Spec.Alerts)
		Expect(crClient.Patch(ctx, updated, client.MergeFrom(current))).To(Succeed())
		refreshed := &coralogixv1alpha1.AlertSet{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, refreshed)).To(Succeed())
		return waitAlertSetReconciled(ctx, crClient, alertSetName, refreshed.Generation)
	}

	setItemPolicy := func(items []coralogixv1alpha1.AlertSetItem, key string, policy *string) {
		for i := range items {
			if items[i].Key == key {
				items[i].Spec.AccessPolicy = policy
				return
			}
		}
		Fail("alertset item not found: " + key)
	}

	bumpAlertSetAnnotations := func(ctx context.Context) {
		current := &coralogixv1alpha1.AlertSet{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, current)).To(Succeed())
		updated := current.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		updated.Annotations["coralogix-operator-e2e-probe"] = uuid.NewString()
		Expect(crClient.Patch(ctx, updated, client.MergeFrom(current))).To(Succeed())
	}

	remoteName := func(ctx context.Context, id string) string {
		resp, httpResp, err := openAPIAlerts.AlertDefsServiceGetAlertDef(ctx, id).Execute()
		Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
		alertDef, ok := resp.GetAlertDefOk()
		Expect(ok).To(BeTrue())
		return alertDef.AlertDefProperties.GetName()
	}

	It("Should manage access policies across a bulk replace", func(ctx context.Context) {
		By("Creating an AlertSet with P1, absent and P1 policies")
		alertSetName = uniqueName("alert-set-access-policy")
		// Registered before creation so a mid-spec failure still deletes the
		// AlertSet and sweeps the remote alerts. The spec ends with A back on
		// P1, which grants Delete, so the operator can delete; any other
		// outcome is collected and reported.
		DeferCleanup(func(cleanupCtx context.Context) {
			var cleanupErrs []error
			key := types.NamespacedName{Name: alertSetName, Namespace: testNamespace}
			ids := map[string]struct{}{}
			fresh := &coralogixv1alpha1.AlertSet{}
			if err := crClient.Get(cleanupCtx, key, fresh); err != nil {
				if client.IgnoreNotFound(err) != nil {
					cleanupErrs = append(cleanupErrs, err)
				}
			} else {
				// Recover IDs from the current status, not only captured ones.
				for _, item := range fresh.Status.Alerts {
					if item.ID != nil {
						ids[*item.ID] = struct{}{}
					}
				}
				var delErr error
				for i := 0; i < 3; i++ {
					f := &coralogixv1alpha1.AlertSet{}
					if err := crClient.Get(cleanupCtx, key, f); err != nil {
						delErr = client.IgnoreNotFound(err)
						break
					}
					err := crClient.Delete(cleanupCtx, f)
					if client.IgnoreNotFound(err) == nil {
						delErr = nil
						break
					}
					delErr = err
				}
				if delErr != nil {
					cleanupErrs = append(cleanupErrs, delErr)
				} else if err := waitObjectGone(cleanupCtx, crClient, key, &coralogixv1alpha1.AlertSet{}); err != nil {
					cleanupErrs = append(cleanupErrs, err)
				}
			}
			for id := range ids {
				if err := waitRemoteAlertGone(cleanupCtx, openAPIAlerts, id); err != nil {
					cleanupErrs = append(cleanupErrs, err)
				}
			}
			reportCleanupErrors(cleanupErrs)
		})
		itemA := newE2EAlertSetItem("item-a", "AlertSet access policy A")
		itemA.Spec.AccessPolicy = ptr.To(alertAccessPolicyP1)
		itemB := newE2EAlertSetItem("item-b", "AlertSet access policy B")
		itemC := newE2EAlertSetItem("item-c", "AlertSet access policy C")
		itemC.Spec.AccessPolicy = ptr.To(alertAccessPolicyP1)
		alertSet := &coralogixv1alpha1.AlertSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertSetName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.AlertSetSpec{Alerts: []coralogixv1alpha1.AlertSetItem{itemA, itemB, itemC}},
		}
		Expect(crClient.Create(ctx, alertSet)).To(Succeed())

		created := &coralogixv1alpha1.AlertSet{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, created)).To(Succeed())
		fetched := waitAlertSetReconciled(ctx, crClient, alertSetName, created.Generation)
		statuses := alertSetStatusesByKey(fetched.Status.Alerts)
		Expect(statuses["item-a"].ID).ToNot(BeNil())
		Expect(statuses["item-b"].ID).ToNot(BeNil())
		Expect(statuses["item-c"].ID).ToNot(BeNil())
		idA, idB, idC := *statuses["item-a"].ID, *statuses["item-b"].ID, *statuses["item-c"].ID
		Expect(statuses["item-a"].AccessPolicyManaged).To(BeTrue())
		Expect(statuses["item-b"].AccessPolicyManaged).To(BeFalse())
		Expect(statuses["item-c"].AccessPolicyManaged).To(BeTrue())
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idA, ptr.To(alertAccessPolicyP1))
			expectAlertPolicy(ctx, g, openAPIAlerts, idB, nil)
			expectAlertPolicy(ctx, g, openAPIAlerts, idC, ptr.To(alertAccessPolicyP1))
		}, time.Minute, time.Second).Should(Succeed())

		By("Changing only the policy on A")
		fetched = patchItems(ctx, func(items []coralogixv1alpha1.AlertSetItem) {
			setItemPolicy(items, "item-a", ptr.To(alertAccessPolicyP2))
		})
		statuses = alertSetStatusesByKey(fetched.Status.Alerts)
		Expect(*statuses["item-a"].ID).To(Equal(idA))
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idA, ptr.To(alertAccessPolicyP2))
		}, time.Minute, time.Second).Should(Succeed())

		By("Omitting the policy on A while renaming it")
		renamedA := "AlertSet access policy A renamed"
		fetched = patchItems(ctx, func(items []coralogixv1alpha1.AlertSetItem) {
			for i := range items {
				if items[i].Key == "item-a" {
					items[i].Spec.AccessPolicy = nil
					items[i].Spec.Name = renamedA
				}
			}
		})

		By("Waiting for the rename to reach the backend before asserting the policy survived")
		Eventually(func(g Gomega) {
			g.Expect(remoteName(ctx, idA)).To(Equal(renamedA))
		}, time.Minute, time.Second).Should(Succeed())
		statuses = alertSetStatusesByKey(fetched.Status.Alerts)
		Expect(statuses["item-a"].AccessPolicyManaged).To(BeTrue())
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idA, ptr.To(alertAccessPolicyP2))
		}, time.Minute, time.Second).Should(Succeed())

		By("Removing C's policy with no property change (real bulk delete)")
		fetched = patchItems(ctx, func(items []coralogixv1alpha1.AlertSetItem) {
			setItemPolicy(items, "item-c", ptr.To(""))
		})
		statuses = alertSetStatusesByKey(fetched.Status.Alerts)
		Expect(*statuses["item-c"].ID).To(Equal(idC))
		Expect(statuses["item-c"].AccessPolicyManaged).To(BeFalse())
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idC, nil)
		}, time.Minute, time.Second).Should(Succeed())

		By("Changing C while empty is held (repeated delete)")
		patchItems(ctx, func(items []coralogixv1alpha1.AlertSetItem) {
			for i := range items {
				if items[i].Key == "item-c" {
					items[i].Spec.Description = "AlertSet access policy C second delete"
				}
			}
		})
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idC, nil)
		}, time.Minute, time.Second).Should(Succeed())

		By("Restoring C's policy externally, then reconciling it away")
		setPolicyExternally(ctx, openAPIAlerts, idC, &alerts.AccessPolicyAction{Upsert: ptr.To(alertAccessPolicyP1)})
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idC, ptr.To(alertAccessPolicyP1))
		}, time.Minute, time.Second).Should(Succeed())
		bumpAlertSetAnnotations(ctx)
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idC, nil)
		}, time.Minute, time.Second).Should(Succeed())

		By("Re-setting A's policy to P2")
		fetched = patchItems(ctx, func(items []coralogixv1alpha1.AlertSetItem) {
			setItemPolicy(items, "item-a", ptr.To(alertAccessPolicyP2))
		})
		statuses = alertSetStatusesByKey(fetched.Status.Alerts)
		Expect(*statuses["item-a"].ID).To(Equal(idA))
		Expect(statuses["item-a"].AccessPolicyManaged).To(BeTrue())
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idA, ptr.To(alertAccessPolicyP2))
		}, time.Minute, time.Second).Should(Succeed())

		By("Drifting A's managed policy externally, then reconciling it back")
		setPolicyExternally(ctx, openAPIAlerts, idA, &alerts.AccessPolicyAction{Upsert: ptr.To(alertAccessPolicyP1)})
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idA, ptr.To(alertAccessPolicyP1))
		}, time.Minute, time.Second).Should(Succeed())
		bumpAlertSetAnnotations(ctx)
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idA, ptr.To(alertAccessPolicyP2))
			fresh := &coralogixv1alpha1.AlertSet{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, fresh)).To(Succeed())
			g.Expect(alertSetStatusesByKey(fresh.Status.Alerts)["item-a"].State).To(Equal(coralogixv1alpha1.AlertSetItemStateSynced))
		}, time.Minute, time.Second).Should(Succeed())

		By("Applying an invalid policy to B while changing A's description")
		updatedADescription := "AlertSet access policy A invalid round"
		current := &coralogixv1alpha1.AlertSet{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, current)).To(Succeed())
		updated := current.DeepCopy()
		setItemPolicy(updated.Spec.Alerts, "item-b", ptr.To(alertAccessPolicyInvalid))
		for i := range updated.Spec.Alerts {
			if updated.Spec.Alerts[i].Key == "item-a" {
				updated.Spec.Alerts[i].Spec.Description = updatedADescription
			}
		}
		Expect(crClient.Patch(ctx, updated, client.MergeFrom(current))).To(Succeed())
		invalidGeneration := updated.Generation
		if invalidGeneration == 0 {
			Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, current)).To(Succeed())
			invalidGeneration = current.Generation
		}
		Eventually(func(g Gomega) {
			fresh := &coralogixv1alpha1.AlertSet{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, fresh)).To(Succeed())
			cond := meta.FindStatusCondition(fresh.Status.Conditions, utils.ConditionTypeRemoteSynced)
			g.Expect(cond).ToNot(BeNil())
			g.Expect(cond.ObservedGeneration).To(Equal(invalidGeneration))
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			st := alertSetStatusesByKey(fresh.Status.Alerts)
			g.Expect(st["item-b"].State).To(Equal(coralogixv1alpha1.AlertSetItemStateFailed))
			g.Expect(st["item-b"].Message).ToNot(BeEmpty())
			g.Expect(st["item-a"].State).To(Equal(coralogixv1alpha1.AlertSetItemStateSynced))
		}, time.Minute, time.Second).Should(Succeed())
		Eventually(func(g Gomega) {
			resp, httpResp, err := openAPIAlerts.AlertDefsServiceGetAlertDef(ctx, idA).Execute()
			g.Expect(oapicxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
			alertDef, ok := resp.GetAlertDefOk()
			g.Expect(ok).To(BeTrue())
			g.Expect(alertDef.AlertDefProperties.GetDescription()).To(Equal(updatedADescription))
		}, time.Minute, time.Second).Should(Succeed())

		By("Adding an item with an invalid policy (create path)")
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, current)).To(Succeed())
		withInvalidCreate := current.DeepCopy()
		setItemPolicy(withInvalidCreate.Spec.Alerts, "item-b", nil)
		itemD := newE2EAlertSetItem("item-d", "AlertSet access policy D")
		itemD.Spec.AccessPolicy = ptr.To(alertAccessPolicyInvalid)
		withInvalidCreate.Spec.Alerts = append(withInvalidCreate.Spec.Alerts, itemD)
		Expect(crClient.Patch(ctx, withInvalidCreate, client.MergeFrom(current))).To(Succeed())
		createGeneration := withInvalidCreate.Generation
		if createGeneration == 0 {
			Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, current)).To(Succeed())
			createGeneration = current.Generation
		}
		Eventually(func(g Gomega) {
			fresh := &coralogixv1alpha1.AlertSet{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, fresh)).To(Succeed())
			cond := meta.FindStatusCondition(fresh.Status.Conditions, utils.ConditionTypeRemoteSynced)
			g.Expect(cond).ToNot(BeNil())
			g.Expect(cond.ObservedGeneration).To(Equal(createGeneration))
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			st := alertSetStatusesByKey(fresh.Status.Alerts)
			g.Expect(st["item-d"].State).To(Equal(coralogixv1alpha1.AlertSetItemStateFailed))
			g.Expect(st["item-d"].Message).ToNot(BeEmpty())
			g.Expect(st["item-a"].State).To(Equal(coralogixv1alpha1.AlertSetItemStateSynced))
		}, time.Minute, time.Second).Should(Succeed())

		By("Restoring a deletable policy on A before teardown")
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, current)).To(Succeed())
		restored := current.DeepCopy()
		setItemPolicy(restored.Spec.Alerts, "item-a", ptr.To(alertAccessPolicyP1))
		for i := range restored.Spec.Alerts {
			if restored.Spec.Alerts[i].Key == "item-d" {
				restored.Spec.Alerts = append(restored.Spec.Alerts[:i], restored.Spec.Alerts[i+1:]...)
				break
			}
		}
		Expect(crClient.Patch(ctx, restored, client.MergeFrom(current))).To(Succeed())
		refreshed := &coralogixv1alpha1.AlertSet{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, refreshed)).To(Succeed())
		fetched = waitAlertSetReconciled(ctx, crClient, alertSetName, refreshed.Generation)
		statuses = alertSetStatusesByKey(fetched.Status.Alerts)
		Expect(statuses).NotTo(HaveKey("item-d"))
		Expect(statuses["item-a"].AccessPolicyManaged).To(BeTrue())
		Eventually(func(g Gomega) {
			expectAlertPolicy(ctx, g, openAPIAlerts, idA, ptr.To(alertAccessPolicyP1))
		}, time.Minute, time.Second).Should(Succeed())

		By("Deleting the AlertSet")
		fresh := &coralogixv1alpha1.AlertSet{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, fresh)).To(Succeed())
		Expect(crClient.Delete(ctx, fresh)).To(Succeed())
		Eventually(func() bool {
			fetched := &coralogixv1alpha1.AlertSet{}
			err := crClient.Get(ctx, types.NamespacedName{Name: alertSetName, Namespace: testNamespace}, fetched)
			return client.IgnoreNotFound(err) == nil && err != nil
		}, time.Minute, time.Second).Should(BeTrue())
		verifyRemoteAlertDeleted(ctx, alertsClient, idA)
		verifyRemoteAlertDeleted(ctx, alertsClient, idB)
		verifyRemoteAlertDeleted(ctx, alertsClient, idC)
	})
})


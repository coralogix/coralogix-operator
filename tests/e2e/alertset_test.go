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
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

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

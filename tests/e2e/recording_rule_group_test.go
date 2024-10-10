/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"

	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
)

var _ = Describe("RecordingRuleGroupSet", Ordered, func() {
	var (
		crClient                     client.Client
		recordingRuleGroupSetsClient *cxsdk.RecordingRuleGroupSetsClient
		recordingRuleGroupSetID      string
		recordingRuleGroupSet        *coralogixv1alpha1.RecordingRuleGroupSet
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		recordingRuleGroupSetsClient = ClientsInstance.GetCoralogixClientSet().RecordingRuleGroups()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating RecordingRuleGroupSet")
		recordingRuleGroupSetName := "recording-rule-group-set"
		recordingRuleGroupSet = &coralogixv1alpha1.RecordingRuleGroupSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      recordingRuleGroupSetName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.RecordingRuleGroupSetSpec{
				Groups: []coralogixv1alpha1.RecordingRuleGroup{
					{
						Name:            "rules",
						IntervalSeconds: 60,
						Rules: []coralogixv1alpha1.RecordingRule{
							{
								Expr:   "vector(1)",
								Record: "ExampleRecord",
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, recordingRuleGroupSet)).To(Succeed())

		By("Fetching the RecordingRuleGroupSet ID")
		fetchedRecordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx,
				types.NamespacedName{Name: recordingRuleGroupSetName, Namespace: testNamespace},
				fetchedRecordingRuleGroupSet)).To(Succeed())
			if fetchedRecordingRuleGroupSet.Status.ID != nil {
				recordingRuleGroupSetID = *fetchedRecordingRuleGroupSet.Status.ID
				return nil
			}
			return fmt.Errorf("RecordingRuleGroupSet ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying RecordingRuleGroupSet exists in Coralogix backend")
		Eventually(func() error {
			_, err := recordingRuleGroupSetsClient.Get(ctx, &cxsdk.GetRuleGroupSetRequest{Id: recordingRuleGroupSetID})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the RecordingRuleGroupSet")
		newRuleName := "rules-updated"
		modifiedRecordingRuleGroupSet := recordingRuleGroupSet.DeepCopy()
		modifiedRecordingRuleGroupSet.Spec.Groups[0].Name = newRuleName
		Expect(crClient.Patch(ctx, modifiedRecordingRuleGroupSet, client.MergeFrom(recordingRuleGroupSet))).To(Succeed())

		By("Verifying RecordingRuleGroupSet is updated in Coralogix backend")
		Eventually(func() string {
			getRuleGroupRes, err := recordingRuleGroupSetsClient.Get(ctx, &cxsdk.GetRuleGroupSetRequest{Id: recordingRuleGroupSetID})
			Expect(err).ToNot(HaveOccurred())
			return getRuleGroupRes.GetGroups()[0].GetName()
		}, time.Minute, time.Second).Should(Equal(newRuleName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the RecordingRuleGroupSet")
		Expect(crClient.Delete(ctx, recordingRuleGroupSet)).To(Succeed())

		By("Verifying RecordingRuleGroupSet is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := recordingRuleGroupSetsClient.Get(ctx, &cxsdk.GetRuleGroupSetRequest{Id: recordingRuleGroupSetID})
			return status.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})

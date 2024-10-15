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

var _ = Describe("RuleGroup", Ordered, func() {
	var (
		crClient         client.Client
		ruleGroupsClient *cxsdk.RuleGroupsClient
		ruleGroupID      string
		ruleGroup        *coralogixv1alpha1.RuleGroup
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		ruleGroupsClient = ClientsInstance.GetCoralogixClientSet().RuleGroups()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating RuleGroup")
		ruleGroupName := "json-extract-rule"
		ruleGroup = &coralogixv1alpha1.RuleGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ruleGroupName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.RuleGroupSpec{
				Name:        ruleGroupName,
				Description: "rule-group from k8s operator",
				RuleSubgroups: []coralogixv1alpha1.RuleSubGroup{
					{
						Rules: []coralogixv1alpha1.Rule{
							{
								Name:        "Worker to category",
								Description: "Extracts value from 'worker' and populates 'Category'",
								JsonExtract: &coralogixv1alpha1.JsonExtract{
									DestinationField: "Category",
									JsonKey:          "worker",
								},
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, ruleGroup)).To(Succeed())

		By("Fetching the RuleGroup ID")
		fetchedRuleGroup := &coralogixv1alpha1.RuleGroup{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: ruleGroupName, Namespace: testNamespace}, fetchedRuleGroup)).To(Succeed())
			if fetchedRuleGroup.Status.ID != nil {
				ruleGroupID = *fetchedRuleGroup.Status.ID
				return nil
			}

			return fmt.Errorf("RuleGroup ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying RuleGroup exists in Coralogix backend")
		Eventually(func() error {
			_, err := ruleGroupsClient.Get(ctx, &cxsdk.GetRuleGroupRequest{GroupId: ruleGroupID})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the RuleGroup")
		newRuleGroupName := "json-extract-rule-updated"
		modifiedRuleGroup := ruleGroup.DeepCopy()
		modifiedRuleGroup.Spec.Name = newRuleGroupName
		Expect(crClient.Patch(ctx, modifiedRuleGroup, client.MergeFrom(ruleGroup))).To(Succeed())

		By("Verifying RuleGroup is updated in Coralogix backend")
		Eventually(func() string {
			getRuleGroupRes, err := ruleGroupsClient.Get(ctx, &cxsdk.GetRuleGroupRequest{GroupId: ruleGroupID})
			Expect(err).ToNot(HaveOccurred())
			return getRuleGroupRes.GetRuleGroup().GetName().GetValue()
		}, time.Minute, time.Second).Should(Equal(newRuleGroupName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the RuleGroup")
		Expect(crClient.Delete(ctx, ruleGroup)).To(Succeed())

		By("Verifying RuleGroup is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := ruleGroupsClient.Get(ctx, &cxsdk.GetRuleGroupRequest{GroupId: ruleGroupID})
			return status.Code(err)
		}).Should(Equal(codes.NotFound))
	})
})

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
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	quotas "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/quota_allocation_rule_set_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("QuotaAllocationRuleSet", Ordered, func() {
	var (
		crClient     client.Client
		quotasClient *quotas.QuotaAllocationRuleSetServiceAPIService
		ruleSet      *coralogixv1alpha1.QuotaAllocationRuleSet
		snapshot     *quotas.QuotaAllocationEntityTypeRuleSet
		snapshotSet  bool
		crName       = "quota-allocation-rule-set-sample"
	)

	BeforeAll(func(ctx context.Context) {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		cfg := cxsdk.NewConfigBuilder().WithAPIKeyEnv().WithRegionEnv().Build()
		quotasClient = cxsdk.NewClientSet(cfg).Quotas()
		var err error
		snapshot, err = getQuotaAllocationRuleSet(ctx, quotasClient)
		if cxsdk.Code(err) == http.StatusForbidden {
			Skip("quota allocation rule set API is not enabled for the configured API key")
		}
		Expect(err).ToNot(HaveOccurred())
		err = replaceQuotaAllocationRuleSet(ctx, quotasClient, snapshot)
		if cxsdk.Code(err) == http.StatusForbidden {
			Skip("quota allocation rule set API manage permission is not enabled for the configured API key")
		}
		Expect(err).ToNot(HaveOccurred())
		snapshotSet = true
	})

	BeforeEach(func() {
		ruleSet = &coralogixv1alpha1.QuotaAllocationRuleSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.QuotaAllocationRuleSetSpec{
				Rules: []coralogixv1alpha1.QuotaAllocationRule{
					{
						EntityType:     "logs",
						Allocation:     resource.MustParse("60"),
						AllocationType: quotaAllocationTypePtr(coralogixv1alpha1.QuotaAllocationTypePercentage),
						Enabled:        true,
						CanOverflow:    true,
					},
					{
						EntityType:     "metrics",
						Allocation:     resource.MustParse("40"),
						AllocationType: quotaAllocationTypePtr(coralogixv1alpha1.QuotaAllocationTypePercentage),
						Enabled:        true,
						CanOverflow:    false,
					},
				},
			},
		}
	})

	AfterAll(func(ctx context.Context) {
		if snapshotSet {
			defer restoreQuotaAllocationRuleSet(ctx, quotasClient, snapshot)
		}

		if ruleSet == nil {
			return
		}

		_ = crClient.Delete(ctx, ruleSet)
		Eventually(func() bool {
			fetched := &coralogixv1alpha1.QuotaAllocationRuleSet{}
			err := crClient.Get(ctx, client.ObjectKey{Name: crName, Namespace: testNamespace}, fetched)
			return apierrors.IsNotFound(err)
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("Should create and delete QuotaAllocationRuleSet successfully", func(ctx context.Context) {
		By("Creating QuotaAllocationRuleSet")
		Expect(crClient.Create(ctx, ruleSet)).To(Succeed())

		By("Verifying QuotaAllocationRuleSet is synced")
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.QuotaAllocationRuleSet{}
			g.Expect(crClient.Get(ctx, client.ObjectKey{Name: crName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying quota allocation rules exist in Coralogix backend")
		Eventually(func(g Gomega) {
			backendRuleSet, err := getQuotaAllocationRuleSet(ctx, quotasClient)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(findUserQuotaAllocationRule(backendRuleSet.Rules, "logs")).ToNot(BeNil())
			g.Expect(findUserQuotaAllocationRule(backendRuleSet.Rules, "metrics")).ToNot(BeNil())

			logsRule := findUserQuotaAllocationRule(backendRuleSet.Rules, "logs")
			g.Expect(logsRule.Allocation).To(Equal(float32(60)))
			g.Expect(logsRule.GetAllocationType()).To(Equal(quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_PERCENTAGE))
			g.Expect(logsRule.Enabled).To(BeTrue())
			g.Expect(logsRule.CanOverflow).To(BeTrue())

			metricsRule := findUserQuotaAllocationRule(backendRuleSet.Rules, "metrics")
			g.Expect(metricsRule.Allocation).To(Equal(float32(40)))
			g.Expect(metricsRule.GetAllocationType()).To(Equal(quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_PERCENTAGE))
			g.Expect(metricsRule.Enabled).To(BeTrue())
			g.Expect(metricsRule.CanOverflow).To(BeFalse())
		}, time.Minute, time.Second).Should(Succeed())

		By("Deleting the QuotaAllocationRuleSet")
		Expect(crClient.Delete(ctx, ruleSet)).To(Succeed())

		By("Verifying user-managed quota allocation rules were deleted in Coralogix backend")
		Eventually(func(g Gomega) {
			backendRuleSet, err := getQuotaAllocationRuleSet(ctx, quotasClient)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(findUserQuotaAllocationRule(backendRuleSet.Rules, "logs")).To(BeNil())
			g.Expect(findUserQuotaAllocationRule(backendRuleSet.Rules, "metrics")).To(BeNil())
		}, time.Minute, time.Second).Should(Succeed())
	})
})

func quotaAllocationTypePtr(allocationType coralogixv1alpha1.QuotaAllocationType) *coralogixv1alpha1.QuotaAllocationType {
	return &allocationType
}

func getQuotaAllocationRuleSet(ctx context.Context, quotasClient *quotas.QuotaAllocationRuleSetServiceAPIService) (*quotas.QuotaAllocationEntityTypeRuleSet, error) {
	response, httpResp, err := quotasClient.
		QuotaAllocationRuleSetServiceGetQuotaAllocationRuleSet(ctx).
		Id("quota-allocation-rule-set").
		Execute()
	if err != nil {
		apiErr := cxsdk.NewAPIError(httpResp, err)
		if cxsdk.IsNotFound(apiErr) {
			return &quotas.QuotaAllocationEntityTypeRuleSet{}, nil
		}
		return nil, apiErr
	}
	if response == nil || response.RuleSet == nil {
		return &quotas.QuotaAllocationEntityTypeRuleSet{}, nil
	}
	return response.RuleSet, nil
}

func restoreQuotaAllocationRuleSet(ctx context.Context, quotasClient *quotas.QuotaAllocationRuleSetServiceAPIService, snapshot *quotas.QuotaAllocationEntityTypeRuleSet) {
	if snapshot == nil || len(snapshot.Rules) == 0 {
		_, httpResp, err := quotasClient.QuotaAllocationRuleSetServiceDeleteQuotaAllocationRuleSet(ctx).Execute()
		if err != nil {
			apiErr := cxsdk.NewAPIError(httpResp, err)
			Expect(cxsdk.IsNotFound(apiErr)).To(BeTrue(), "unexpected quota allocation rule set delete error: %v", err)
		}
		return
	}

	Expect(replaceQuotaAllocationRuleSet(ctx, quotasClient, snapshot)).To(Succeed())
}

func replaceQuotaAllocationRuleSet(ctx context.Context, quotasClient *quotas.QuotaAllocationRuleSetServiceAPIService, ruleSet *quotas.QuotaAllocationEntityTypeRuleSet) error {
	restoreRules := []quotas.QuotaAllocationEntityTypeRule{}
	if ruleSet != nil {
		restoreRules = make([]quotas.QuotaAllocationEntityTypeRule, 0, len(ruleSet.Rules))
		for _, rule := range ruleSet.Rules {
			rule.CxManaged = nil
			restoreRules = append(restoreRules, rule)
		}
	}

	_, httpResp, err := quotasClient.
		QuotaAllocationRuleSetServiceReplaceQuotaAllocationRuleSet(ctx).
		ReplaceQuotaAllocationRuleSetRequest(quotas.ReplaceQuotaAllocationRuleSetRequest{
			RuleSet: quotas.QuotaAllocationEntityTypeRuleSet{Rules: restoreRules},
		}).
		Execute()
	return cxsdk.NewAPIError(httpResp, err)
}

func findUserQuotaAllocationRule(rules []quotas.QuotaAllocationEntityTypeRule, entityType string) *quotas.QuotaAllocationEntityTypeRule {
	for i := range rules {
		if rules[i].EntityType == entityType && !rules[i].GetCxManaged() {
			return &rules[i]
		}
	}

	return nil
}

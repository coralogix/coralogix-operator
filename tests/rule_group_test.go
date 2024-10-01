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

package tests

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
)

var _ = Describe("Rule Group", func() {
	var (
		crClient        = ClientsInstance.CrClient
		ruleGroupClient = ClientsInstance.CXClientSet.RuleGroups()
	)

	It("should be created successfully", func(ctx context.Context) {
		By("Defining a RuleGroup resource")
		ruleGroup := &coralogixv1alpha1.RuleGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "parsing-rule",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.RuleGroupSpec{
				Name:         "parsing-rule",
				Description:  "rule-group from k8s operator tests",
				Applications: []string{"application-name"},
				Subsystems:   []string{"subsystems-name"},
				Severities:   []coralogixv1alpha1.RuleSeverity{"Warning", "Info"},
				RuleSubgroups: []coralogixv1alpha1.RuleSubGroup{
					{
						Active: true,
						Rules: []coralogixv1alpha1.Rule{
							{
								Active:      true,
								Name:        "HttpRequestParser2",
								Description: "Parse the fields of the HTTP request - will be applied after HttpRequestParser1",
								Parse: &coralogixv1alpha1.Parse{
									DestinationField: "text",
									Regex:            `(?P<remote_addr>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\s*-\s*(?P<user>[^\s]+)\s*\[(?P<timestemp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{1,6}Z)\]\s*\"(?P<method>[A-z]+)\s[\/\\]+(?P<request>[^\s]+)\s*(?P<protocol>[A-z0-9\/\.]+)\"\s*(?P<status>\d+)\s*(?P<body_bytes_sent>\d+)?\s*\"(?P<http_referer>[^\"]+)\"\s*\"(?P<http_user_agent>[^\"]+)\"\s(?P<request_time>\d{1,6})\s*(?P<response_time>\d{1,6})`,
									SourceField:      "text",
								},
							},
						},
					},
				},
			},
		}

		By("Creating the RuleGroup resource in the cluster")
		err := crClient.Create(ctx, ruleGroup)
		Expect(err).NotTo(HaveOccurred())

		By("Fetching the RuleGroup ID")
		var ruleGroupID string
		fetchedRuleGroup := &coralogixv1alpha1.RuleGroup{}
		Eventually(func() error {
			err := crClient.Get(ctx, types.NamespacedName{Name: "parsing-rule", Namespace: "default"}, fetchedRuleGroup)
			if err != nil {
				return err
			}

			if fetchedRuleGroup.Status.ID == nil {
				return fmt.Errorf("RuleGroup ID is not set")
			}

			ruleGroupID = *fetchedRuleGroup.Status.ID
			return nil
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying RuleGroup exists in Coralogix backend")
		Eventually(func() error {
			_, err = ruleGroupClient.Get(ctx, &cxsdk.GetRuleGroupRequest{GroupId: ruleGroupID})
			return err
		}, time.Minute, time.Second).Should(Succeed())

		By("Deleting the RuleGroup resource")
		err = crClient.Delete(ctx, ruleGroup)
		Expect(err).NotTo(HaveOccurred())

		By("Verifying RuleGroup is deleted from Coralogix backend")
		Eventually(func() error {
			_, err = ruleGroupClient.Get(ctx, &cxsdk.GetRuleGroupRequest{GroupId: ruleGroupID})
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return nil
				}
				return err
			}
			return fmt.Errorf("RuleGroup still exists in Coralogix backend")
		}).Should(Succeed())
	})
})

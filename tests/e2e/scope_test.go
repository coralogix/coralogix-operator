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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var _ = Describe("Scope", Ordered, func() {
	var (
		crClient     client.Client
		scopesClient *cxsdk.ScopesClient
		scopeID      string
		scope        *coralogixv1alpha1.Scope
		scopeName    = "scope-sample"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		scopesClient = ClientsInstance.GetCoralogixClientSet().Scopes()
		scope = &coralogixv1alpha1.Scope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      scopeName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.ScopeSpec{
				Name:        scopeName,
				Description: ptr.To("This is a sample scope"),
				Filters: []coralogixv1alpha1.ScopeFilter{
					{
						EntityType: "logs",
						Expression: "<v1>(subsystemName == 'purchases') || (subsystemName == 'signups')",
					},
					{
						EntityType: "spans",
						Expression: "<v1>(subsystemName == 'clothing') || (subsystemName == 'electronics')",
					},
				},
				DefaultExpression: "<v1>true",
			},
		}
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Scope")
		Expect(crClient.Create(ctx, scope)).To(Succeed())

		By("Fetching the Scope ID")
		fetchedScope := &coralogixv1alpha1.Scope{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: scopeName, Namespace: testNamespace}, fetchedScope)).To(Succeed())
			if fetchedScope.Status.ID != nil {
				scopeID = *fetchedScope.Status.ID
				return nil
			}
			return fmt.Errorf("scope ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying Scope exists in Coralogix backend")
		Eventually(func() error {
			_, err := scopesClient.Get(ctx, &cxsdk.GetTeamScopesByIDsRequest{
				Ids: []string{scopeID},
			})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the Scope")
		newScopeName := "scope-sample-updated"
		modifiedScope := scope.DeepCopy()
		modifiedScope.Spec.Name = newScopeName
		Expect(crClient.Patch(ctx, modifiedScope, client.MergeFrom(scope))).To(Succeed())

		By("Verifying Scope is updated in Coralogix backend")
		Eventually(func() string {
			getScopeRes, err := scopesClient.Get(ctx, &cxsdk.GetTeamScopesByIDsRequest{
				Ids: []string{scopeID},
			})
			Expect(err).ToNot(HaveOccurred())
			return getScopeRes.Scopes[0].DisplayName
		}, time.Minute, time.Second).Should(Equal(newScopeName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Scope")
		Expect(crClient.Delete(ctx, scope)).To(Succeed())

		By("Verifying Scope is deleted from Coralogix backend")
		Eventually(func() int {
			getScopeRes, err := scopesClient.Get(ctx, &cxsdk.GetTeamScopesByIDsRequest{
				Ids: []string{scopeID},
			})
			Expect(err).ToNot(HaveOccurred())
			return len(getScopeRes.Scopes)
		}, time.Minute, time.Second).Should(Equal(0))
	})
})

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
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
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
		scope = getSampleScope(scopeName, testNamespace)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Scope")
		Expect(crClient.Create(ctx, scope)).To(Succeed())

		By("Fetching the Scope ID")
		fetchedScope := &coralogixv1alpha1.Scope{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: scopeName, Namespace: testNamespace}, fetchedScope)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedScope.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedScope.Status.PrintableStatus).To(Equal("RemoteSynced"))
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

	It("After deleted from Coralogix backend directly, it should be recreated based on configured interval",
		func(ctx context.Context) {
			By("Deleting the Scope from Coralogix backend")
			_, err := scopesClient.Delete(ctx, &cxsdk.DeleteScopeRequest{Id: scopeID})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying Scope is populated with a new ID after configured interval")
			var newScopeID string
			Eventually(func() bool {
				fetchedScope := &coralogixv1alpha1.Scope{}
				Expect(crClient.Get(ctx, types.NamespacedName{Name: scopeName, Namespace: testNamespace}, fetchedScope)).To(Succeed())
				if fetchedScope.Status.ID == nil {
					return false
				}
				newScopeID = *fetchedScope.Status.ID
				return newScopeID != scopeID
			}, 2*time.Minute, time.Second).Should(BeTrue())

			By("Verifying Scope with new the ID exists in Coralogix backend")
			Eventually(func() error {
				_, err := scopesClient.Get(ctx, &cxsdk.GetTeamScopesByIDsRequest{
					Ids: []string{newScopeID},
				})
				return err
			}, time.Minute, time.Second).Should(Succeed())

		})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Scope")
		Expect(crClient.Delete(ctx, scope)).To(Succeed())

		By("Verifying Scope is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := scopesClient.Get(ctx, &cxsdk.GetTeamScopesByIDsRequest{
				Ids: []string{scopeID},
			})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})

func getSampleScope(name, namespace string) *coralogixv1alpha1.Scope {
	return &coralogixv1alpha1.Scope{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.ScopeSpec{
			Name:        name,
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
}

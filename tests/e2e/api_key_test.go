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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("ApiKey", Ordered, func() {
	var (
		crClient      client.Client
		ApiKeysClient *cxsdk.ApikeysClient
		apiKeyID      string
		apiKeyName    = "team-key-sample"
		apiKey        *coralogixv1alpha1.ApiKey
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		ApiKeysClient = ClientsInstance.GetCoralogixClientSet().APIKeys()
		apiKey = &coralogixv1alpha1.ApiKey{
			ObjectMeta: metav1.ObjectMeta{
				Name:      apiKeyName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.ApiKeySpec{
				Name:   apiKeyName,
				Active: true,
				Owner: coralogixv1alpha1.ApiKeyOwner{
					TeamId: ptr.To(uint32(4013254)),
				},
				Presets:     []string{"APM"},
				Permissions: []string{"ALERTS-MAP:READ"},
			},
		}
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating ApiKey")
		Expect(crClient.Create(ctx, apiKey)).To(Succeed())

		By("Verifying secret was created")
		secret := &corev1.Secret{}
		Eventually(func() error {
			return crClient.Get(ctx, types.NamespacedName{Name: apiKeyName + "-secret", Namespace: testNamespace}, secret)
		}, time.Minute, time.Second).Should(Succeed())

		By("Fetching the ApiKey ID")
		fetchedApiKey := &coralogixv1alpha1.ApiKey{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: apiKeyName, Namespace: testNamespace}, fetchedApiKey)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedApiKey.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedApiKey.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetchedApiKey.Status.Id != nil {
				apiKeyID = *fetchedApiKey.Status.Id
				return nil
			}
			return fmt.Errorf("ApiKey ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying ApiKey exists in Coralogix backend")
		Eventually(func() error {
			_, err := ApiKeysClient.Get(ctx, &cxsdk.GetAPIKeyRequest{KeyId: apiKeyID})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the ApiKey")
		newApiKeyName := "team-key-sample-updated"
		modifiedApiKey := apiKey.DeepCopy()
		modifiedApiKey.Spec.Name = newApiKeyName
		Expect(crClient.Patch(ctx, modifiedApiKey, client.MergeFrom(apiKey))).To(Succeed())

		By("Verifying ApiKey is updated in Coralogix backend")
		Eventually(func() string {
			getApiKeyRes, err := ApiKeysClient.Get(ctx, &cxsdk.GetAPIKeyRequest{KeyId: apiKeyID})
			Expect(err).ToNot(HaveOccurred())
			return getApiKeyRes.GetKeyInfo().GetName()
		}, time.Minute, time.Second).Should(Equal(newApiKeyName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the ApiKey")
		Expect(crClient.Delete(ctx, apiKey)).To(Succeed())

		By("Verifying secret was deleted")
		secret := &corev1.Secret{}
		Eventually(func() bool {
			err := crClient.Get(ctx, types.NamespacedName{Name: apiKeyName + "-secret", Namespace: testNamespace}, secret)
			return errors.IsNotFound(err)
		}, time.Minute, time.Second).Should(BeTrue())

		By("Verifying ApiKey is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := ApiKeysClient.Get(ctx, &cxsdk.GetAPIKeyRequest{KeyId: apiKeyID})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})

	It("should deny creation of ApiKey with both userId and teamId", func(ctx context.Context) {
		apiKey.Spec.Owner.UserId = ptr.To("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
		apiKey.Spec.Owner.TeamId = ptr.To(uint32(12345678))
		err := crClient.Create(ctx, apiKey)
		Expect(err.Error()).To(ContainSubstring("Exactly one of userId or teamId must be set"))
	})

	It("should deny creation of ApiKey without presets and permissions", func(ctx context.Context) {
		apiKey.Spec.Presets = nil
		apiKey.Spec.Permissions = nil
		err := crClient.Create(ctx, apiKey)
		Expect(err.Error()).To(ContainSubstring("At least one of presets or permissions must be set"))
	})

	//TODO: Adding validation for creation of inactive ApiKey
	//It("should deny creation of inactive ApiKey", func(ctx context.Context) {
	//	apiKey.Spec.Active = false
	//	err := crClient.Create(ctx, apiKey)
	//	Expect(err.Error()).To(ContainSubstring("ApiKey must be activated on creation"))
	//})
})

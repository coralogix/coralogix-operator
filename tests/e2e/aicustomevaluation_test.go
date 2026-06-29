// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	aiapplications "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_applications_service"
	aievaluations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_evaluations_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("AICustomEvaluation", Ordered, func() {
	var (
		crClient               client.Client
		aiApplications         *aiapplications.AIApplicationsServiceAPIService
		aiEvaluations          *aievaluations.AIEvaluationsServiceAPIService
		aiCustomEvaluationID   string
		aiCustomEvaluation     *coralogixv1alpha1.AICustomEvaluation
		application            aiCustomEvaluationApplicationRef
		aiCustomEvaluationName = fmt.Sprintf("ai-custom-evaluation-%d", time.Now().Unix())
	)

	BeforeAll(func(ctx context.Context) {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		clientSet := newAIEvaluationOpenAPIClientSet()
		aiApplications = clientSet.AIApplications()
		aiEvaluations = clientSet.AIEvaluations()

		var err error
		application, err = firstAvailableAICustomEvaluationApplication(ctx, aiApplications)
		Expect(err).ToNot(HaveOccurred())

		aiCustomEvaluation = newAICustomEvaluation(
			aiCustomEvaluationName,
			"competitor-policy",
			coralogixv1alpha1.AICustomEvaluationPolicyTypeQuality,
			"Flags competitor references in assistant responses.",
			"Score whether {response} mentions competitor products.\nTreat each assistant answer independently.",
			[]aiCustomEvaluationApplicationRef{application},
			newAICustomEvaluationCreateCriteria(),
		)
	})

	AfterAll(func(ctx context.Context) {
		deleteAICustomEvaluationAndWait(ctx, crClient, aiCustomEvaluationName)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating AICustomEvaluation")
		Expect(crClient.Create(ctx, aiCustomEvaluation)).To(Succeed())

		By("Fetching the AICustomEvaluation ID")
		fetched := &coralogixv1alpha1.AICustomEvaluation{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: aiCustomEvaluationName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
			g.Expect(fetched.Status.ApplicationIds).To(ConsistOf(application.id))
			g.Expect(fetched.Status.Applications).To(ConsistOf(coralogixv1alpha1.AICustomEvaluationApplicationStatus{
				Id:          application.id,
				Application: application.application,
				Subsystem:   application.subsystem,
			}))
			if fetched.Status.Id != nil {
				aiCustomEvaluationID = *fetched.Status.Id
				return nil
			}
			return fmt.Errorf("AI custom evaluation ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying AICustomEvaluation exists in Coralogix backend")
		Eventually(func(g Gomega) {
			customEvaluation := getRemoteAICustomEvaluation(ctx, g, aiEvaluations, aiCustomEvaluationID)
			expectRemoteAICustomEvaluation(g, customEvaluation, aiCustomEvaluation.Spec, []string{application.id})
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the AICustomEvaluation")
		current := &coralogixv1alpha1.AICustomEvaluation{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: aiCustomEvaluationName, Namespace: testNamespace}, current)).To(Succeed())
		modified := current.DeepCopy()
		modified.Spec.Name = "competitor-policy-updated"
		modified.Spec.PolicyType = coralogixv1alpha1.AICustomEvaluationPolicyTypeSecurity
		modified.Spec.Description = "Flags responses that recommend competitor tools."
		modified.Spec.Instructions = "Score whether {response} recommends competitor products.\nOnly evaluate the final assistant response."
		modified.Spec.ShouldIncludeSystemPrompt = ptr.To(true)
		modified.Spec.Criteria = newAICustomEvaluationUpdateCriteria()
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(current))).To(Succeed())
		aiCustomEvaluation = modified

		By("Verifying AICustomEvaluation is updated in Coralogix backend")
		Eventually(func(g Gomega) {
			customEvaluation := getRemoteAICustomEvaluation(ctx, g, aiEvaluations, aiCustomEvaluationID)
			expectRemoteAICustomEvaluation(g, customEvaluation, modified.Spec, []string{application.id})
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should unlink all applications successfully", func(ctx context.Context) {
		By("Patching the AICustomEvaluation applications")
		current := &coralogixv1alpha1.AICustomEvaluation{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: aiCustomEvaluationName, Namespace: testNamespace}, current)).To(Succeed())
		modified := current.DeepCopy()
		modified.Spec.Applications = nil
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(current))).To(Succeed())
		aiCustomEvaluation = modified

		By("Verifying AICustomEvaluation links are removed")
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.AICustomEvaluation{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: aiCustomEvaluationName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.ApplicationIds).To(BeEmpty())
			g.Expect(fetched.Status.Applications).To(BeEmpty())

			customEvaluation := getRemoteAICustomEvaluation(ctx, g, aiEvaluations, aiCustomEvaluationID)
			g.Expect(customEvaluation.GetApplicationIds()).To(BeEmpty())
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the AICustomEvaluation")
		Expect(crClient.Delete(ctx, aiCustomEvaluation)).To(Succeed())

		By("Verifying AICustomEvaluation is deleted from Coralogix backend")
		Eventually(func() bool {
			_, found := remoteAICustomEvaluationByID(ctx, aiEvaluations, aiCustomEvaluationID)
			return found
		}, time.Minute, time.Second).Should(BeFalse())
	})
})

var _ = Describe("AICustomEvaluation minimal", Ordered, func() {
	var (
		crClient             client.Client
		aiEvaluations        *aievaluations.AIEvaluationsServiceAPIService
		aiCustomEvaluationID string
		aiCustomEvaluation   *coralogixv1alpha1.AICustomEvaluation
		resourceName         = fmt.Sprintf("ai-custom-evaluation-minimal-%d", time.Now().Unix())
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		aiEvaluations = newAIEvaluationOpenAPIClientSet().AIEvaluations()
		aiCustomEvaluation = newAICustomEvaluation(
			resourceName,
			"minimal-policy",
			coralogixv1alpha1.AICustomEvaluationPolicyTypeQuality,
			"",
			"Score whether {response} matches the policy.",
			nil,
			nil,
		)
	})

	AfterAll(func(ctx context.Context) {
		deleteAICustomEvaluationAndWait(ctx, crClient, resourceName)
	})

	It("Should be created and deleted successfully", func(ctx context.Context) {
		By("Creating minimal AICustomEvaluation")
		Expect(crClient.Create(ctx, aiCustomEvaluation)).To(Succeed())

		By("Fetching the AICustomEvaluation ID")
		Eventually(func(g Gomega) error {
			fetched := &coralogixv1alpha1.AICustomEvaluation{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.ApplicationIds).To(BeEmpty())
			if fetched.Status.Id != nil {
				aiCustomEvaluationID = *fetched.Status.Id
				return nil
			}
			return fmt.Errorf("AI custom evaluation ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying minimal AICustomEvaluation defaults in Coralogix backend")
		Eventually(func(g Gomega) {
			customEvaluation := getRemoteAICustomEvaluation(ctx, g, aiEvaluations, aiCustomEvaluationID)
			expectRemoteAICustomEvaluation(g, customEvaluation, aiCustomEvaluation.Spec, nil)
		}, time.Minute, time.Second).Should(Succeed())

		By("Deleting minimal AICustomEvaluation")
		Expect(crClient.Delete(ctx, aiCustomEvaluation)).To(Succeed())
		Eventually(func() bool {
			_, found := remoteAICustomEvaluationByID(ctx, aiEvaluations, aiCustomEvaluationID)
			return found
		}, time.Minute, time.Second).Should(BeFalse())
	})
})

var _ = Describe("AICustomEvaluation application resolution", Ordered, func() {
	var (
		crClient       client.Client
		aiApplications *aiapplications.AIApplicationsServiceAPIService
		application    aiCustomEvaluationApplicationRef
		resourceName   = fmt.Sprintf("ai-custom-evaluation-missing-app-update-%d", time.Now().Unix())
	)

	BeforeAll(func(ctx context.Context) {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		clientSet := newAIEvaluationOpenAPIClientSet()
		aiApplications = clientSet.AIApplications()

		var err error
		application, err = firstAvailableAICustomEvaluationApplication(ctx, aiApplications)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func(ctx context.Context) {
		deleteAICustomEvaluationAndWait(ctx, crClient, resourceName)
	})

	It("Should report unsynced when application is missing on create", func(ctx context.Context) {
		missingApplication := aiCustomEvaluationApplicationRef{
			application: fmt.Sprintf("missing-ai-application-%d", time.Now().UnixNano()),
			subsystem:   fmt.Sprintf("missing-ai-subsystem-%d", time.Now().UnixNano()),
		}
		resourceName := fmt.Sprintf("ai-custom-evaluation-missing-app-create-%d", time.Now().Unix())
		aiCustomEvaluation := newAICustomEvaluation(
			resourceName,
			"missing-application-policy",
			coralogixv1alpha1.AICustomEvaluationPolicyTypeQuality,
			"",
			"Score whether {response} matches the policy.",
			[]aiCustomEvaluationApplicationRef{missingApplication},
			nil,
		)

		By("Creating AICustomEvaluation with a missing application")
		Expect(crClient.Create(ctx, aiCustomEvaluation)).To(Succeed())
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.AICustomEvaluation{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: testNamespace}, fetched)).To(Succeed())
			expectAICustomEvaluationRemoteUnsynced(g, fetched)
		}, time.Minute, time.Second).Should(Succeed())

		deleteAICustomEvaluationAndWait(ctx, crClient, resourceName)
	})

	It("Should report unsynced when application is missing on update", func(ctx context.Context) {
		aiCustomEvaluation := newAICustomEvaluation(
			resourceName,
			"missing-application-update-policy",
			coralogixv1alpha1.AICustomEvaluationPolicyTypeQuality,
			"",
			"Score whether {response} matches the policy.",
			[]aiCustomEvaluationApplicationRef{application},
			nil,
		)

		By("Creating AICustomEvaluation with a valid application")
		Expect(crClient.Create(ctx, aiCustomEvaluation)).To(Succeed())
		Eventually(func(g Gomega) error {
			fetched := &coralogixv1alpha1.AICustomEvaluation{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			if fetched.Status.Id != nil {
				return nil
			}
			return fmt.Errorf("AI custom evaluation ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Patching the AICustomEvaluation to a missing application")
		current := &coralogixv1alpha1.AICustomEvaluation{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: testNamespace}, current)).To(Succeed())
		modified := current.DeepCopy()
		modified.Spec.Applications = []coralogixv1alpha1.AICustomEvaluationApplicationSelector{
			{
				Application: fmt.Sprintf("missing-ai-application-%d", time.Now().UnixNano()),
				Subsystem:   fmt.Sprintf("missing-ai-subsystem-%d", time.Now().UnixNano()),
			},
		}
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(current))).To(Succeed())

		By("Verifying AICustomEvaluation reports the missing application")
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.AICustomEvaluation{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: testNamespace}, fetched)).To(Succeed())
			expectAICustomEvaluationRemoteUnsynced(g, fetched)
		}, time.Minute, time.Second).Should(Succeed())
	})
})

var _ = Describe("AICustomEvaluation schema validation", func() {
	var crClient client.Client

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
	})

	It("Should reject invalid instructions", func(ctx context.Context) {
		aiCustomEvaluation := newAICustomEvaluation(
			fmt.Sprintf("ai-custom-evaluation-invalid-instructions-%d", time.Now().Unix()),
			"invalid-instructions-policy",
			coralogixv1alpha1.AICustomEvaluationPolicyTypeQuality,
			"",
			"Score whether the response matches the policy.",
			nil,
			nil,
		)

		Expect(crClient.Create(ctx, aiCustomEvaluation)).ToNot(Succeed())
	})

	It("Should reject invalid policy type", func(ctx context.Context) {
		aiCustomEvaluation := newAICustomEvaluation(
			fmt.Sprintf("ai-custom-evaluation-invalid-policy-%d", time.Now().Unix()),
			"invalid-policy",
			"other",
			"",
			"Score whether {response} matches the policy.",
			nil,
			nil,
		)

		Expect(crClient.Create(ctx, aiCustomEvaluation)).ToNot(Succeed())
	})
})

type aiCustomEvaluationApplicationRef struct {
	id          string
	application string
	subsystem   string
}

func firstAvailableAICustomEvaluationApplication(
	ctx context.Context,
	applicationsClient *aiapplications.AIApplicationsServiceAPIService,
) (aiCustomEvaluationApplicationRef, error) {
	const pageSize = int32(200)
	for pageOffset := int64(0); ; pageOffset++ {
		result, httpResp, err := applicationsClient.
			AiApplicationsServiceListAiApplications(ctx).
			PageSize(pageSize).
			PageOffset(pageOffset).
			Execute()
		if err != nil {
			return aiCustomEvaluationApplicationRef{}, cxsdk.NewAPIError(httpResp, err)
		}

		page := result.GetAiApplications()
		for _, application := range page {
			applicationID := application.GetId()
			applicationName := application.GetApplication()
			subsystem := application.GetSubsystem()
			if applicationID == "" || applicationName == "" || subsystem == "" {
				continue
			}

			return aiCustomEvaluationApplicationRef{
				id:          applicationID,
				application: applicationName,
				subsystem:   subsystem,
			}, nil
		}

		if len(page) < int(pageSize) {
			break
		}
	}

	return aiCustomEvaluationApplicationRef{}, fmt.Errorf("no AI applications with a subsystem found")
}

func newAICustomEvaluation(
	resourceName string,
	name string,
	policyType string,
	description string,
	instructions string,
	applications []aiCustomEvaluationApplicationRef,
	criteria *coralogixv1alpha1.AICustomEvaluationCriteria,
) *coralogixv1alpha1.AICustomEvaluation {
	selectors := make([]coralogixv1alpha1.AICustomEvaluationApplicationSelector, 0, len(applications))
	for _, application := range applications {
		selectors = append(selectors, coralogixv1alpha1.AICustomEvaluationApplicationSelector{
			Application: application.application,
			Subsystem:   application.subsystem,
		})
	}

	return &coralogixv1alpha1.AICustomEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: testNamespace,
		},
		Spec: coralogixv1alpha1.AICustomEvaluationSpec{
			Name:                      name,
			PolicyType:                policyType,
			Description:               description,
			Instructions:              instructions,
			ShouldIncludeSystemPrompt: ptr.To(false),
			Applications:              selectors,
			Criteria:                  criteria,
		},
	}
}

func newAICustomEvaluationCreateCriteria() *coralogixv1alpha1.AICustomEvaluationCriteria {
	return &coralogixv1alpha1.AICustomEvaluationCriteria{
		Acceptable: &coralogixv1alpha1.AICustomEvaluationCriterion{
			Flags: "Does not mention competitor products.\nAnswer stays focused on our product.",
			Examples: []string{
				"User: which tool should I use?\nAssistant: Our product is a strong fit.",
			},
		},
		Prohibited: &coralogixv1alpha1.AICustomEvaluationCriterion{
			Flags: "Mentions a competitor product.\nNames another vendor as the recommended option.",
			Examples: []string{
				"User: which tool should I use?\nAssistant: CompetitorX is a strong fit.",
			},
		},
	}
}

func newAICustomEvaluationUpdateCriteria() *coralogixv1alpha1.AICustomEvaluationCriteria {
	return &coralogixv1alpha1.AICustomEvaluationCriteria{
		Acceptable: &coralogixv1alpha1.AICustomEvaluationCriterion{
			Flags: "Does not recommend competitor products.\nMentions only our product or neutral guidance.",
			Examples: []string{
				"User: what should I buy?\nAssistant: Our product covers that workflow.",
			},
		},
		Prohibited: &coralogixv1alpha1.AICustomEvaluationCriterion{
			Flags: "Recommends a competitor product.\nNames a competitor as the best choice.",
		},
	}
}

func getRemoteAICustomEvaluation(
	ctx context.Context,
	g Gomega,
	evaluationsClient *aievaluations.AIEvaluationsServiceAPIService,
	id string,
) aievaluations.CustomEvaluation {
	customEvaluation, found := remoteAICustomEvaluationByID(ctx, evaluationsClient, id)
	g.Expect(found).To(BeTrue())
	return customEvaluation
}

func remoteAICustomEvaluationByID(
	ctx context.Context,
	evaluationsClient *aievaluations.AIEvaluationsServiceAPIService,
	id string,
) (aievaluations.CustomEvaluation, bool) {
	result, httpResp, err := evaluationsClient.
		AiEvaluationsServiceGetCustomEvaluations(ctx).
		Execute()
	Expect(cxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())

	for _, customEvaluation := range result.GetItems() {
		if customEvaluation.GetId() == id {
			return customEvaluation, true
		}
	}

	return aievaluations.CustomEvaluation{}, false
}

func expectRemoteAICustomEvaluation(
	g Gomega,
	customEvaluation aievaluations.CustomEvaluation,
	spec coralogixv1alpha1.AICustomEvaluationSpec,
	expectedApplicationIDs []string,
) {
	g.Expect(customEvaluation.GetName()).To(Equal(spec.Name))
	g.Expect(customEvaluation.GetDescription()).To(Equal(spec.Description))
	g.Expect(customEvaluation.GetApplicationIds()).To(ConsistOf(expectedStrings(expectedApplicationIDs)...))

	config := customEvaluation.GetConfig()
	g.Expect(config.GetPolicyType()).To(Equal(spec.PolicyType))
	g.Expect(config.GetInstructions()).To(Equal(spec.Instructions))
	g.Expect(config.GetShouldIncludeSystemPrompt()).To(Equal(spec.ShouldIncludeSystemPromptValue()))

	acceptable := coralogixv1alpha1.AICustomEvaluationCriterion{}
	prohibited := coralogixv1alpha1.AICustomEvaluationCriterion{}
	if spec.Criteria != nil {
		if spec.Criteria.Acceptable != nil {
			acceptable = *spec.Criteria.Acceptable
		}
		if spec.Criteria.Prohibited != nil {
			prohibited = *spec.Criteria.Prohibited
		}
	}

	g.Expect(config.GetSafe()).To(Equal(acceptable.Flags))
	g.Expect(config.GetViolates()).To(Equal(prohibited.Flags))
	expectRemoteAICustomEvaluationExamples(g, config.GetExamples(), acceptable.Examples, prohibited.Examples)
}

func expectRemoteAICustomEvaluationExamples(
	g Gomega,
	examples []aievaluations.CustomEvaluationExample,
	expectedAcceptable []string,
	expectedProhibited []string,
) {
	acceptable := make([]string, 0)
	prohibited := make([]string, 0)
	for _, example := range examples {
		switch example.GetScore() {
		case "1":
			acceptable = append(acceptable, example.GetConversation())
		case "0":
			prohibited = append(prohibited, example.GetConversation())
		default:
			g.Expect(example.GetScore()).To(BeElementOf("0", "1"))
		}
	}

	g.Expect(acceptable).To(ConsistOf(expectedStrings(expectedAcceptable)...))
	g.Expect(prohibited).To(ConsistOf(expectedStrings(expectedProhibited)...))
}

func expectAICustomEvaluationRemoteUnsynced(g Gomega, aiCustomEvaluation *coralogixv1alpha1.AICustomEvaluation) {
	condition := meta.FindStatusCondition(aiCustomEvaluation.Status.Conditions, utils.ConditionTypeRemoteSynced)
	g.Expect(condition).ToNot(BeNil())
	g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
	g.Expect(condition.Message).To(ContainSubstring("AI application not found"))
	g.Expect(aiCustomEvaluation.Status.PrintableStatus).To(Equal("RemoteUnsynced"))
}

func deleteAICustomEvaluationAndWait(ctx context.Context, crClient client.Client, name string) {
	aiCustomEvaluation := &coralogixv1alpha1.AICustomEvaluation{}
	err := crClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNamespace}, aiCustomEvaluation)
	if err == nil {
		err = crClient.Delete(ctx, aiCustomEvaluation)
		if err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	} else if !apierrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	Eventually(func() bool {
		fetched := &coralogixv1alpha1.AICustomEvaluation{}
		err := crClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNamespace}, fetched)
		return apierrors.IsNotFound(err)
	}, time.Minute, time.Second).Should(BeTrue())
}

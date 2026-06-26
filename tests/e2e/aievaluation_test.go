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
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
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

var _ = Describe("AIEvaluation PII", Ordered, func() {
	var (
		crClient             client.Client
		aiApplications       *aiapplications.AIApplicationsServiceAPIService
		aiEvaluations        *aievaluations.AIEvaluationsServiceAPIService
		aiEvaluationID       string
		aiEvaluation         *coralogixv1alpha1.AIEvaluation
		application          aiEvaluationApplicationRef
		target               string
		aiEvaluationCRName   = fmt.Sprintf("ai-evaluation-pii-%d", time.Now().Unix())
		createdPIICategories = []coralogixv1alpha1.AIEvaluationPIICategory{
			coralogixv1alpha1.AIEvaluationPIICategoryEmailAddress,
			coralogixv1alpha1.AIEvaluationPIICategoryCreditCard,
		}
		updatedPIICategories = []coralogixv1alpha1.AIEvaluationPIICategory{
			coralogixv1alpha1.AIEvaluationPIICategoryPhoneNumber,
			coralogixv1alpha1.AIEvaluationPIICategoryUSSSN,
		}
	)

	BeforeAll(func(ctx context.Context) {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		clientSet := newAIEvaluationOpenAPIClientSet()
		aiApplications = clientSet.AIApplications()
		aiEvaluations = clientSet.AIEvaluations()

		var err error
		application, target, err = firstAvailableAIEvaluationApplication(ctx, aiApplications, aiEvaluations, aievaluations.EVALUATIONTYPE_PII)
		Expect(err).ToNot(HaveOccurred())

		aiEvaluation = newPIIAIEvaluation(
			aiEvaluationCRName,
			testNamespace,
			application,
			target,
			resource.MustParse("0.8"),
			true,
			createdPIICategories,
		)
	})

	AfterAll(func(ctx context.Context) {
		if aiEvaluation == nil {
			return
		}

		err := crClient.Delete(ctx, aiEvaluation)
		if err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		Eventually(func() bool {
			fetched := &coralogixv1alpha1.AIEvaluation{}
			err := crClient.Get(ctx, client.ObjectKey{Name: aiEvaluationCRName, Namespace: testNamespace}, fetched)
			return apierrors.IsNotFound(err)
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating AIEvaluation")
		Expect(crClient.Create(ctx, aiEvaluation)).To(Succeed())

		By("Fetching the AIEvaluation ID")
		fetched := &coralogixv1alpha1.AIEvaluation{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: aiEvaluationCRName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetched.Status.Id != nil {
				aiEvaluationID = *fetched.Status.Id
				return nil
			}
			return fmt.Errorf("AI evaluation ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying AIEvaluation exists in Coralogix backend")
		Eventually(func(g Gomega) {
			evaluation := getRemoteAIEvaluation(ctx, g, aiEvaluations, aiEvaluationID)
			g.Expect(evaluation.GetApplication()).To(Equal(application.application))
			g.Expect(evaluation.GetSubsystem()).To(Equal(application.subsystem))
			g.Expect(strings.ToLower(string(evaluation.GetTarget()))).To(Equal(target))
			g.Expect(evaluation.GetThreshold()).To(BeNumerically("~", 0.8, 0.00001))
			g.Expect(evaluation.GetIsEnabled()).To(BeTrue())
			expectRemoteAIEvaluationPIICategories(g, evaluation, createdPIICategories)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the AIEvaluation")
		modified := aiEvaluation.DeepCopy()
		modified.Spec.Threshold = resource.MustParse("0.9")
		modified.Spec.IsEnabled = ptr.To(false)
		modified.Spec.Config.PII.Categories = updatedPIICategories
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(aiEvaluation))).To(Succeed())

		By("Verifying AIEvaluation is updated in Coralogix backend")
		Eventually(func(g Gomega) {
			evaluation := getRemoteAIEvaluation(ctx, g, aiEvaluations, aiEvaluationID)
			g.Expect(evaluation.GetThreshold()).To(BeNumerically("~", 0.9, 0.00001))
			g.Expect(evaluation.GetIsEnabled()).To(BeFalse())
			expectRemoteAIEvaluationPIICategories(g, evaluation, updatedPIICategories)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the AIEvaluation")
		Expect(crClient.Delete(ctx, aiEvaluation)).To(Succeed())

		By("Verifying AIEvaluation is deleted from Coralogix backend")
		Eventually(func() int {
			_, httpResp, err := aiEvaluations.
				AiEvaluationsServiceGetAiEvaluation(ctx, aiEvaluationID).
				Execute()
			return cxsdk.Code(cxsdk.NewAPIError(httpResp, err))
		}, time.Minute, time.Second).Should(Equal(http.StatusNotFound))
	})
})

var _ = Describe("AIEvaluation Allowed Topics", Ordered, func() {
	var (
		crClient             client.Client
		aiApplications       *aiapplications.AIApplicationsServiceAPIService
		aiEvaluations        *aievaluations.AIEvaluationsServiceAPIService
		aiEvaluationID       string
		aiEvaluation         *coralogixv1alpha1.AIEvaluation
		application          aiEvaluationApplicationRef
		target               string
		aiEvaluationCRName   = fmt.Sprintf("ai-evaluation-allowed-topics-%d", time.Now().Unix())
		createdAllowedTopics = []string{"billing", "account settings"}
		updatedAllowedTopics = []string{"observability", "incident response"}
	)

	BeforeAll(func(ctx context.Context) {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		clientSet := newAIEvaluationOpenAPIClientSet()
		aiApplications = clientSet.AIApplications()
		aiEvaluations = clientSet.AIEvaluations()

		var err error
		application, target, err = firstAvailableAIEvaluationApplication(ctx, aiApplications, aiEvaluations, aievaluations.EVALUATIONTYPE_ALLOWED_TOPICS)
		Expect(err).ToNot(HaveOccurred())

		aiEvaluation = newAllowedTopicsAIEvaluation(
			aiEvaluationCRName,
			testNamespace,
			application,
			target,
			resource.MustParse("0.8"),
			true,
			createdAllowedTopics,
		)
	})

	AfterAll(func(ctx context.Context) {
		if aiEvaluation == nil {
			return
		}

		err := crClient.Delete(ctx, aiEvaluation)
		if err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		Eventually(func() bool {
			fetched := &coralogixv1alpha1.AIEvaluation{}
			err := crClient.Get(ctx, client.ObjectKey{Name: aiEvaluationCRName, Namespace: testNamespace}, fetched)
			return apierrors.IsNotFound(err)
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating AIEvaluation")
		Expect(crClient.Create(ctx, aiEvaluation)).To(Succeed())

		By("Fetching the AIEvaluation ID")
		fetched := &coralogixv1alpha1.AIEvaluation{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: aiEvaluationCRName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetched.Status.Id != nil {
				aiEvaluationID = *fetched.Status.Id
				return nil
			}
			return fmt.Errorf("AI evaluation ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying AIEvaluation exists in Coralogix backend")
		Eventually(func(g Gomega) {
			evaluation := getRemoteAIEvaluation(ctx, g, aiEvaluations, aiEvaluationID)
			g.Expect(evaluation.GetApplication()).To(Equal(application.application))
			g.Expect(evaluation.GetSubsystem()).To(Equal(application.subsystem))
			g.Expect(strings.ToLower(string(evaluation.GetTarget()))).To(Equal(target))
			g.Expect(evaluation.GetThreshold()).To(BeNumerically("~", 0.8, 0.00001))
			g.Expect(evaluation.GetIsEnabled()).To(BeTrue())
			expectRemoteAIEvaluationAllowedTopics(g, evaluation, createdAllowedTopics)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the AIEvaluation")
		modified := aiEvaluation.DeepCopy()
		modified.Spec.Threshold = resource.MustParse("0.9")
		modified.Spec.IsEnabled = ptr.To(false)
		modified.Spec.Config.AllowedTopics.Topics = updatedAllowedTopics
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(aiEvaluation))).To(Succeed())

		By("Verifying AIEvaluation is updated in Coralogix backend")
		Eventually(func(g Gomega) {
			evaluation := getRemoteAIEvaluation(ctx, g, aiEvaluations, aiEvaluationID)
			g.Expect(evaluation.GetThreshold()).To(BeNumerically("~", 0.9, 0.00001))
			g.Expect(evaluation.GetIsEnabled()).To(BeFalse())
			expectRemoteAIEvaluationAllowedTopics(g, evaluation, updatedAllowedTopics)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the AIEvaluation")
		Expect(crClient.Delete(ctx, aiEvaluation)).To(Succeed())

		By("Verifying AIEvaluation is deleted from Coralogix backend")
		Eventually(func() int {
			_, httpResp, err := aiEvaluations.
				AiEvaluationsServiceGetAiEvaluation(ctx, aiEvaluationID).
				Execute()
			return cxsdk.Code(cxsdk.NewAPIError(httpResp, err))
		}, time.Minute, time.Second).Should(Equal(http.StatusNotFound))
	})
})

var _ = Describe("AIEvaluation Toxicity", Ordered, func() {
	var (
		crClient           client.Client
		aiApplications     *aiapplications.AIApplicationsServiceAPIService
		aiEvaluations      *aievaluations.AIEvaluationsServiceAPIService
		aiEvaluationID     string
		aiEvaluation       *coralogixv1alpha1.AIEvaluation
		application        aiEvaluationApplicationRef
		target             string
		aiEvaluationCRName = fmt.Sprintf("ai-evaluation-toxicity-%d", time.Now().Unix())
	)

	BeforeAll(func(ctx context.Context) {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		clientSet := newAIEvaluationOpenAPIClientSet()
		aiApplications = clientSet.AIApplications()
		aiEvaluations = clientSet.AIEvaluations()

		var err error
		application, target, err = firstAvailableAIEvaluationApplication(ctx, aiApplications, aiEvaluations, aievaluations.EVALUATIONTYPE_TOXICITY)
		Expect(err).ToNot(HaveOccurred())

		aiEvaluation = newToxicityAIEvaluation(
			aiEvaluationCRName,
			testNamespace,
			application,
			target,
			resource.MustParse("0.8"),
			true,
		)
	})

	AfterAll(func(ctx context.Context) {
		if aiEvaluation == nil {
			return
		}

		err := crClient.Delete(ctx, aiEvaluation)
		if err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		Eventually(func() bool {
			fetched := &coralogixv1alpha1.AIEvaluation{}
			err := crClient.Get(ctx, client.ObjectKey{Name: aiEvaluationCRName, Namespace: testNamespace}, fetched)
			return apierrors.IsNotFound(err)
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating AIEvaluation")
		Expect(crClient.Create(ctx, aiEvaluation)).To(Succeed())

		By("Fetching the AIEvaluation ID")
		fetched := &coralogixv1alpha1.AIEvaluation{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: aiEvaluationCRName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetched.Status.Id != nil {
				aiEvaluationID = *fetched.Status.Id
				return nil
			}
			return fmt.Errorf("AI evaluation ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying AIEvaluation exists in Coralogix backend")
		Eventually(func(g Gomega) {
			evaluation := getRemoteAIEvaluation(ctx, g, aiEvaluations, aiEvaluationID)
			g.Expect(evaluation.GetApplication()).To(Equal(application.application))
			g.Expect(evaluation.GetSubsystem()).To(Equal(application.subsystem))
			g.Expect(strings.ToLower(string(evaluation.GetTarget()))).To(Equal(target))
			g.Expect(evaluation.GetThreshold()).To(BeNumerically("~", 0.8, 0.00001))
			g.Expect(evaluation.GetIsEnabled()).To(BeTrue())
			expectRemoteAIEvaluationToxicity(g, evaluation)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the AIEvaluation")
		modified := aiEvaluation.DeepCopy()
		modified.Spec.Threshold = resource.MustParse("0.9")
		modified.Spec.IsEnabled = ptr.To(false)
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(aiEvaluation))).To(Succeed())

		By("Verifying AIEvaluation is updated in Coralogix backend")
		Eventually(func(g Gomega) {
			evaluation := getRemoteAIEvaluation(ctx, g, aiEvaluations, aiEvaluationID)
			g.Expect(evaluation.GetThreshold()).To(BeNumerically("~", 0.9, 0.00001))
			g.Expect(evaluation.GetIsEnabled()).To(BeFalse())
			expectRemoteAIEvaluationToxicity(g, evaluation)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the AIEvaluation")
		Expect(crClient.Delete(ctx, aiEvaluation)).To(Succeed())

		By("Verifying AIEvaluation is deleted from Coralogix backend")
		Eventually(func() int {
			_, httpResp, err := aiEvaluations.
				AiEvaluationsServiceGetAiEvaluation(ctx, aiEvaluationID).
				Execute()
			return cxsdk.Code(cxsdk.NewAPIError(httpResp, err))
		}, time.Minute, time.Second).Should(Equal(http.StatusNotFound))
	})
})

type aiEvaluationApplicationRef struct {
	application string
	subsystem   string
}

func newAIEvaluationOpenAPIClientSet() *cxsdk.ClientSet {
	builder := cxsdk.NewConfigBuilder().WithAPIKeyEnv()
	if domain := os.Getenv("CORALOGIX_DOMAIN"); domain != "" {
		builder = builder.WithDomain(domain)
	} else {
		builder = builder.WithRegionEnv()
	}
	return cxsdk.NewClientSet(builder.Build())
}

func firstAvailableAIEvaluationApplication(
	ctx context.Context,
	applicationsClient *aiapplications.AIApplicationsServiceAPIService,
	evaluationsClient *aievaluations.AIEvaluationsServiceAPIService,
	evaluationType aievaluations.EvaluationType,
) (aiEvaluationApplicationRef, string, error) {
	result, httpResp, err := applicationsClient.
		AiApplicationsServiceListAiApplications(ctx).
		PageSize(200).
		PageOffset(0).
		Execute()
	if err != nil {
		return aiEvaluationApplicationRef{}, "", cxsdk.NewAPIError(httpResp, err)
	}

	for _, application := range result.GetAiApplications() {
		applicationName := application.GetApplication()
		subsystem := application.GetSubsystem()
		if applicationName == "" || subsystem == "" {
			continue
		}

		target, available, err := availableAIEvaluationTarget(ctx, evaluationsClient, applicationName, subsystem, evaluationType)
		if err != nil {
			return aiEvaluationApplicationRef{}, "", err
		}
		if available {
			return aiEvaluationApplicationRef{
				application: applicationName,
				subsystem:   subsystem,
			}, target, nil
		}
	}

	return aiEvaluationApplicationRef{}, "", fmt.Errorf("no AI application with a subsystem has an available %s evaluation target", evaluationType)
}

func availableAIEvaluationTarget(
	ctx context.Context,
	evaluationsClient *aievaluations.AIEvaluationsServiceAPIService,
	application string,
	subsystem string,
	evaluationType aievaluations.EvaluationType,
) (string, bool, error) {
	result, httpResp, err := evaluationsClient.
		AiEvaluationsServiceListAiEvaluations(ctx).
		Application(application).
		Subsystem(subsystem).
		EvaluationType(evaluationType).
		PageSize(200).
		PageOffset(0).
		Execute()
	if err != nil {
		return "", false, cxsdk.NewAPIError(httpResp, err)
	}

	usedTargets := make(map[string]struct{}, len(result.GetAiEvaluations()))
	for _, evaluation := range result.GetAiEvaluations() {
		target := strings.ToLower(string(evaluation.GetTarget()))
		if target != "" {
			usedTargets[target] = struct{}{}
		}
	}

	for _, candidate := range []string{coralogixv1alpha1.AIEvaluationTargetResponse, coralogixv1alpha1.AIEvaluationTargetPrompt} {
		if _, used := usedTargets[candidate]; !used {
			return candidate, true, nil
		}
	}

	return "", false, nil
}

func newPIIAIEvaluation(
	name string,
	namespace string,
	application aiEvaluationApplicationRef,
	target string,
	threshold resource.Quantity,
	isEnabled bool,
	categories []coralogixv1alpha1.AIEvaluationPIICategory,
) *coralogixv1alpha1.AIEvaluation {
	return &coralogixv1alpha1.AIEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.AIEvaluationSpec{
			Application: application.application,
			Subsystem:   application.subsystem,
			Target:      target,
			Threshold:   threshold,
			IsEnabled:   ptr.To(isEnabled),
			Config: coralogixv1alpha1.AIEvaluationConfig{
				PII: &coralogixv1alpha1.AIEvaluationPIIConfig{
					Categories: categories,
				},
			},
		},
	}
}

func newAllowedTopicsAIEvaluation(
	name string,
	namespace string,
	application aiEvaluationApplicationRef,
	target string,
	threshold resource.Quantity,
	isEnabled bool,
	topics []string,
) *coralogixv1alpha1.AIEvaluation {
	return &coralogixv1alpha1.AIEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.AIEvaluationSpec{
			Application: application.application,
			Subsystem:   application.subsystem,
			Target:      target,
			Threshold:   threshold,
			IsEnabled:   ptr.To(isEnabled),
			Config: coralogixv1alpha1.AIEvaluationConfig{
				AllowedTopics: &coralogixv1alpha1.AIEvaluationAllowedTopicsConfig{
					Topics: topics,
				},
			},
		},
	}
}

func newToxicityAIEvaluation(
	name string,
	namespace string,
	application aiEvaluationApplicationRef,
	target string,
	threshold resource.Quantity,
	isEnabled bool,
) *coralogixv1alpha1.AIEvaluation {
	return &coralogixv1alpha1.AIEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.AIEvaluationSpec{
			Application: application.application,
			Subsystem:   application.subsystem,
			Target:      target,
			Threshold:   threshold,
			IsEnabled:   ptr.To(isEnabled),
			Config: coralogixv1alpha1.AIEvaluationConfig{
				Toxicity: coralogixv1alpha1.NewAIEvaluationToxicityConfig(),
			},
		},
	}
}

func getRemoteAIEvaluation(
	ctx context.Context,
	g Gomega,
	evaluationsClient *aievaluations.AIEvaluationsServiceAPIService,
	id string,
) aievaluations.AiEvaluation {
	result, httpResp, err := evaluationsClient.
		AiEvaluationsServiceGetAiEvaluation(ctx, id).
		Execute()
	g.Expect(cxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
	return result.GetAiEvaluation()
}

func expectRemoteAIEvaluationPIICategories(
	g Gomega,
	evaluation aievaluations.AiEvaluation,
	expected []coralogixv1alpha1.AIEvaluationPIICategory,
) {
	config := evaluation.GetConfig()
	g.Expect(config.EvaluationConfigPii).ToNot(BeNil())
	piiConfig := config.EvaluationConfigPii.GetPii()
	g.Expect(schemaPIICategories(piiConfig.GetCategories())).To(ConsistOf(expectedPIICategories(expected)...))
}

func expectRemoteAIEvaluationAllowedTopics(
	g Gomega,
	evaluation aievaluations.AiEvaluation,
	expected []string,
) {
	config := evaluation.GetConfig()
	g.Expect(config.EvaluationConfigAllowedTopics).ToNot(BeNil())
	allowedTopicsConfig := config.EvaluationConfigAllowedTopics.GetAllowedTopics()
	g.Expect(allowedTopicsConfig.GetTopics()).To(ConsistOf(expectedStrings(expected)...))
}

func expectRemoteAIEvaluationToxicity(
	g Gomega,
	evaluation aievaluations.AiEvaluation,
) {
	config := evaluation.GetConfig()
	g.Expect(config.EvaluationConfigToxicity).ToNot(BeNil())
	g.Expect(config.EvaluationConfigToxicity.GetToxicity()).To(BeEmpty())
}

func schemaPIICategories(categories []aievaluations.PiiCategory) []string {
	result := make([]string, 0, len(categories))
	for _, category := range categories {
		result = append(result, string(category))
	}
	return result
}

func expectedPIICategories(categories []coralogixv1alpha1.AIEvaluationPIICategory) []interface{} {
	result := make([]interface{}, 0, len(categories))
	for _, category := range categories {
		result = append(result, string(category))
	}
	return result
}

func expectedStrings(values []string) []interface{} {
	result := make([]interface{}, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

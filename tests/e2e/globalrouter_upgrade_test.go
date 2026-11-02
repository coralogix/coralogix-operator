package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// This spec runs only in the upgrade job. The released operator creates the router first.
var _ = Describe("GlobalRouter upgrade", Serial, Label("globalrouter-upgrade"), func() {
	It("keeps the remote ID, updates the remote router, and deletes it", func(ctx context.Context) {
		if os.Getenv("E2E_GLOBALROUTER_UPGRADE") != "1" {
			Skip("set E2E_GLOBALROUTER_UPGRADE=1 in the upgrade job")
		}

		idPath := filepath.Join(os.Getenv("RUNNER_TEMP"), "upgrade-test-global-router.id")
		savedID, err := os.ReadFile(idPath)
		Expect(err).NotTo(HaveOccurred())
		id := strings.TrimSpace(string(savedID))
		Expect(id).NotTo(BeEmpty())

		crClient := ClientsInstance.GetControllerRuntimeClient()
		apiClient := newOpenAPIClientSet().GlobalRouters()
		key := types.NamespacedName{Name: "upgrade-test-global-router", Namespace: "default"}
		DeferCleanup(func(ctx context.Context) {
			cr := &coralogixv1alpha1.GlobalRouter{}
			if err := crClient.Get(ctx, key, cr); err == nil {
				_ = crClient.Delete(ctx, cr)
			}
		})

		synced := func(g Gomega) *coralogixv1alpha1.GlobalRouter {
			cr := &coralogixv1alpha1.GlobalRouter{}
			g.Expect(crClient.Get(ctx, key, cr)).To(Succeed())
			g.Expect(cr.Status.ID).NotTo(BeNil())
			g.Expect(*cr.Status.ID).To(Equal(id), "the remote ID changed")
			condition := meta.FindStatusCondition(cr.Status.Conditions, utils.ConditionTypeRemoteSynced)
			g.Expect(condition).NotTo(BeNil())
			g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(condition.ObservedGeneration).To(Equal(cr.Generation))
			return cr
		}

		By("Waiting for the new operator to sync the released router")
		var before *coralogixv1alpha1.GlobalRouter
		Eventually(func(g Gomega) {
			before = synced(g)
		}, 3*time.Minute, 2*time.Second).Should(Succeed())

		By("Checking that the remote router still has the same ID")
		Eventually(func(g Gomega) {
			got, httpResp, err := apiClient.GlobalRoutersServiceGetGlobalRouter(ctx, id).Execute()
			g.Expect(cxsdk.NewAPIError(httpResp, err)).To(Succeed())
			g.Expect(got).NotTo(BeNil())
			g.Expect(got.Router).NotTo(BeNil())
			g.Expect(ptr.Deref(got.Router.Id, "")).To(Equal(id), "the remote ID changed")
		}, time.Minute, time.Second).Should(Succeed())

		By("Updating the router after the upgrade")
		updated := before.DeepCopy()
		updated.Spec.Description = "Router updated after the operator upgrade"
		Expect(crClient.Patch(ctx, updated, client.MergeFrom(before))).To(Succeed())
		Eventually(func(g Gomega) {
			synced(g)
			got, httpResp, err := apiClient.GlobalRoutersServiceGetGlobalRouter(ctx, id).Execute()
			g.Expect(cxsdk.NewAPIError(httpResp, err)).To(Succeed())
			g.Expect(got).NotTo(BeNil())
			g.Expect(got.Router).NotTo(BeNil())
			g.Expect(ptr.Deref(got.Router.Description, "")).To(Equal(updated.Spec.Description))
		}, 3*time.Minute, 2*time.Second).Should(Succeed())

		By("Deleting the router after the upgrade")
		Expect(crClient.Delete(ctx, updated)).To(Succeed())
		Eventually(func() bool {
			cr := &coralogixv1alpha1.GlobalRouter{}
			return apierrors.IsNotFound(crClient.Get(ctx, key, cr))
		}, 3*time.Minute, 2*time.Second).Should(BeTrue())
		Eventually(func() bool {
			_, httpResp, err := apiClient.GlobalRoutersServiceGetGlobalRouter(ctx, id).Execute()
			return cxsdk.IsNotFound(cxsdk.NewAPIError(httpResp, err))
		}, time.Minute, time.Second).Should(BeTrue())
	})
})

package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// setupGlobalRouterGeneratedE2E creates the reference used by the generated full case.
func setupGlobalRouterGeneratedE2E(ctx context.Context, crClient client.Client, fullName string) {
	connectorName := fullName + "-connector"
	connector := getSampleGenericHttpsConnectorForRouterGaps(connectorName, testNamespace)
	Expect(crClient.Create(ctx, connector)).To(Succeed())
	DeferCleanup(func(ctx context.Context) {
		_ = crClient.Delete(ctx, connector)
	})
	Eventually(func(g Gomega) {
		fetched := &coralogixv1alpha1.Connector{}
		g.Expect(crClient.Get(ctx, types.NamespacedName{Name: connectorName, Namespace: testNamespace}, fetched)).To(Succeed())
		g.Expect(fetched.Status.Id).NotTo(BeNil())
	}, time.Minute, time.Second).Should(Succeed())
}

// prepareGlobalRouterGeneratedE2E makes the generated spec usable in a live account.
func prepareGlobalRouterGeneratedE2E(cr *coralogixv1alpha1.GlobalRouter) {
	cr.Spec.Id = nil
	cr.Spec.RoutingLabels = &coralogixv1alpha1.GlobalRouterRoutingLabels{
		Environment: ptr.To(cr.Name),
	}
	if len(cr.Spec.Rules) == 0 {
		return
	}
	// The API accepts one fallback form at a time. The full case uses fallbackTargets.
	cr.Spec.Fallback = nil
	cr.Spec.Rules[0].Condition = `alertDef.priority == "P1"`
	connector := coralogixv1alpha1.NCRef{ResourceRef: &coralogixv1alpha1.ResourceRef{Name: cr.Name + "-connector"}}
	cr.Spec.Rules[0].Targets[0].Connector = connector
	cr.Spec.Rules[0].Targets[0].Preset = &coralogixv1alpha1.NCRef{
		BackendRef: &coralogixv1alpha1.NCBackendRef{ID: "preset_system_generic_https_alerts_empty"},
	}
	cr.Spec.FallbackTargets[0].Target.Connector = connector
	cr.Spec.FallbackTargets[0].Target.Preset = nil
}

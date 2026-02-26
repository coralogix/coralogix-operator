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

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	prometheusv1alpha "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	controllers "github.com/coralogix/coralogix-operator/v2/internal/controller"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
	v1beta1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/monitoring"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
	//+kubebuilder:scaffold:imports
)

const OperatorVersion = "2.1.0"

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(prometheus.AddToScheme(scheme))

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	utilruntime.Must(v1beta1.AddToScheme(scheme))

	utilruntime.Must(prometheusv1alpha.AddToScheme(scheme))

	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	config.InitScheme(scheme)
	cfg := config.InitConfig(setupLog)

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	var tlsOpts []func(*tls.Config)
	if !cfg.EnableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	metricsServerOptions := metricsserver.Options{
		BindAddress:   cfg.MetricsAddr,
		SecureServing: cfg.SecureMetrics,
		TLSOpts:       tlsOpts,
	}

	if cfg.SecureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgrOpts := ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsServerOptions,
		Client: client.Options{
			Cache: &client.CacheOptions{
				Unstructured: true,
			},
		},
		HealthProbeBindAddress: cfg.ProbeAddr,
		LeaderElection:         cfg.EnableLeaderElection,
		LeaderElectionID:       cfg.LeaderElectionID,
		PprofBindAddress:       "0.0.0.0:8888",
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientSet := cxsdk.NewClientSet(cxsdk.NewSDKCallPropertiesCreatorOperator(
		strings.ToLower(cfg.CoralogixGrpcUrl),
		cxsdk.NewAuthContext(cfg.CoralogixApiKey, cfg.CoralogixApiKey),
		OperatorVersion))

	oapiClientSet := openapicxsdk.NewClientSet(openapicxsdk.NewConfigBuilder().
		WithURL(cfg.CoralogixOpenApiUrl).
		WithAPIKey(cfg.CoralogixApiKey).
		WithOperatorVersion(OperatorVersion).
		Build())

	config.InitClient(mgr.GetClient())

	if err = (&v1alpha1controllers.RuleGroupReconciler{
		RuleGroupClient: oapiClientSet.RuleGroups(),
		Interval:        cfg.ReconcileIntervals[utils.RuleGroupKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RuleGroup")
		os.Exit(1)
	}
	if err = (&v1beta1controllers.AlertReconciler{
		ClientSet: oapiClientSet,
		Interval:  cfg.ReconcileIntervals[utils.AlertKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Alert")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.RecordingRuleGroupSetReconciler{
		RecordingRulesClient:        oapiClientSet.RecordingRules(),
		Interval:                    cfg.ReconcileIntervals[utils.RecordingRuleGroupSetKind],
		RecordingRuleGroupSetSuffix: cfg.RecordingRuleGroupSetSuffix,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RecordingRuleGroupSet")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.OutboundWebhookReconciler{
		OutboundWebhooksClient: oapiClientSet.Webhooks(),
		Interval:               cfg.ReconcileIntervals[utils.OutboundWebhookKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OutboundWebhook")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ApiKeyReconciler{
		ApiKeysClient: oapiClientSet.APIKeys(),
		Interval:      cfg.ReconcileIntervals[utils.ApiKeyKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApiKey")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.CustomRoleReconciler{
		CustomRolesClient: oapiClientSet.CustomRoles(),
		Interval:          cfg.ReconcileIntervals[utils.CustomRoleKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CustomRole")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.ScopeReconciler{
		ScopesClient: oapiClientSet.Scopes(),
		Interval:     cfg.ReconcileIntervals[utils.ScopeKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Scope")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.GroupReconciler{
		GroupsClient: oapiClientSet.Groups(),
		CXClientSet:  clientSet,
		Interval:     cfg.ReconcileIntervals[utils.GroupKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Group")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.TCOLogsPoliciesReconciler{
		TCOPoliciesClient:       oapiClientSet.TCOPolicies(),
		ArchiveRetentionsClient: oapiClientSet.ArchiveRetentions(),
		Interval:                cfg.ReconcileIntervals[utils.TCOLogsPoliciesKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TCOLogsPolicies")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.TCOTracesPoliciesReconciler{
		TCOPoliciesClient:       oapiClientSet.TCOPolicies(),
		ArchiveRetentionsClient: oapiClientSet.ArchiveRetentions(),
		Interval:                cfg.ReconcileIntervals[utils.TCOTracesPoliciesKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TCOTracesPolicies")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.IntegrationReconciler{
		IntegrationsClient: oapiClientSet.Integrations(),
		Interval:           cfg.ReconcileIntervals[utils.IntegrationKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Integration")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.AlertSchedulerReconciler{
		AlertSchedulerClient: clientSet.AlertSchedulers(),
		Interval:             cfg.ReconcileIntervals[utils.AlertSchedulerKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AlertScheduler")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.DashboardReconciler{
		DashboardsClient: clientSet.Dashboards(),
		Interval:         cfg.ReconcileIntervals[utils.DashboardKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Dashboard")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.DashboardsFolderReconciler{
		DashboardsFoldersClient: oapiClientSet.DashboardFolders(),
		Interval:                cfg.ReconcileIntervals[utils.DashboardsFolderKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DashboardsFolder")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ViewReconciler{
		ViewsClient: oapiClientSet.Views(),
		Interval:    cfg.ReconcileIntervals[utils.ViewKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "View")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ViewFolderReconciler{
		ViewFoldersClient: oapiClientSet.ViewsFolders(),
		Interval:          cfg.ReconcileIntervals[utils.ViewFolderKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ViewFolder")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ConnectorReconciler{
		ConnectorsClient: oapiClientSet.Connectors(),
		Interval:         cfg.ReconcileIntervals[utils.ConnectorKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Connector")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.PresetReconciler{
		PresetsClient: oapiClientSet.Presets(),
		Interval:      cfg.ReconcileIntervals[utils.PresetKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Preset")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.GlobalRouterReconciler{
		GlobalRoutersClient: oapiClientSet.GlobalRouters(),
		Interval:            cfg.ReconcileIntervals[utils.GlobalRouterKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GlobalRouter")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.ArchiveLogsTargetReconciler{
		ArchiveLogsTargetsClient: oapiClientSet.ArchiveLogs(),
		ArchiveRetentionsClient:  clientSet.ArchiveRetentions(),
		Interval:                 cfg.ReconcileIntervals[utils.ArchiveLogsTargetKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ArchiveLogsTarget")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.ArchiveMetricsTargetReconciler{
		ArchiveMetricsTargetsClient: oapiClientSet.ArchiveMetrics(),
		Interval:                    cfg.ReconcileIntervals[utils.ArchiveMetricsTargetKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ArchiveMetricsTarget")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.SLOReconciler{
		SLOsClient: oapiClientSet.SLOs(),
		Interval:   cfg.ReconcileIntervals[utils.SLOKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SLO")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.Events2MetricReconciler{
		E2MClient: oapiClientSet.Events2Metrics(),
		Interval:  cfg.ReconcileIntervals[utils.Events2MetricKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Events2Metric")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.IPAccessReconciler{
		IPAccesssClient: oapiClientSet.IPAccess(),
		Interval:        cfg.ReconcileIntervals[utils.IPAccess],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IPAccess")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.CustomEnrichmentReconciler{
		CustomEnrichmentsClient: oapiClientSet.CustomEnrichments(),
		Interval:                cfg.ReconcileIntervals[utils.CustomEnrichmentKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CustomEnrichment")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.EnrichmentReconciler{
		EnrichmentsClient: oapiClientSet.Enrichments(),
		Interval:          cfg.ReconcileIntervals[utils.EnrichmentKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Enrichment")
		os.Exit(1)
	}

	enablePromRuleController, err := shouldEnablePromRuleController(
		context.Background(),
		setupLog,
		cfg,
		mgr.GetAPIReader(),
	)
	if err != nil {
		setupLog.Error(err, "unable to determine whether to enable PrometheusRule controller")
		os.Exit(1)
	}
	if enablePromRuleController {
		if err = (&controllers.PrometheusRuleReconciler{
			Interval: cfg.ReconcileIntervals[utils.PrometheusRuleKind],
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "PrometheusRule")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := monitoring.RegisterMetrics(); err != nil {
		setupLog.Error(err, "unable to set up metrics")
		os.Exit(1)
	}

	monitoring.SetOperatorInfoMetric(
		runtime.Version(),
		OperatorVersion,
		cxsdk.CoralogixGrpcEndpointFromRegion(strings.ToLower(cfg.CoralogixGrpcUrl)),
	)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func shouldEnablePromRuleController(ctx context.Context,
	log logr.Logger,
	cfg *config.Config,
	c client.Reader,
) (bool, error) {
	if !cfg.PrometheusRuleController {
		log.Info("PrometheusRule controller disabled via configuration")
		return false, nil
	}

	exists, err := prometheusRuleCRDExists(ctx, c)
	if err != nil {
		return false, fmt.Errorf("failed to check PrometheusRule CRD existence: %w", err)
	}

	if !exists {
		log.Info(
			"PrometheusRule controller requested but CRD not found; controller will be disabled")
		return false, nil
	}

	log.Info("Enabling PrometheusRule controller")
	return true, nil
}

func prometheusRuleCRDExists(ctx context.Context, c client.Reader) (bool, error) {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := c.Get(ctx, types.NamespacedName{Name: "prometheusrules.monitoring.coreos.com"}, crd)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

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
	"crypto/tls"
	"os"
	"runtime"
	"strings"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	prometheusv1alpha "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/config"
	controllers "github.com/coralogix/coralogix-operator/internal/controller"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/internal/controller/coralogix/v1alpha1"
	v1beta1controllers "github.com/coralogix/coralogix-operator/internal/controller/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	"github.com/coralogix/coralogix-operator/internal/utils"
	//+kubebuilder:scaffold:imports
)

const OperatorVersion = "0.5.0"

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
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
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

	clientSet := cxsdk.NewClientSet(cxsdk.NewCallPropertiesCreatorOperator(
		strings.ToLower(cfg.CoralogixUrl),
		cxsdk.NewAuthContext(cfg.CoralogixApiKey, cfg.CoralogixApiKey),
		OperatorVersion))

	config.InitClient(mgr.GetClient())

	if err = (&v1alpha1controllers.RuleGroupReconciler{
		RuleGroupClient: clientSet.RuleGroups(),
		Interval:        cfg.ReconcileIntervals[utils.RuleGroupKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RuleGroup")
		os.Exit(1)
	}
	if err = (&v1beta1controllers.AlertReconciler{
		CoralogixClientSet: clientSet,
		Interval:           cfg.ReconcileIntervals[utils.AlertKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Alert")
		os.Exit(1)
	}
	if cfg.PrometheusRuleController {
		if err = (&controllers.PrometheusRuleReconciler{
			Interval: cfg.ReconcileIntervals[utils.PrometheusRuleKind],
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "PrometheusRule")
			os.Exit(1)
		}
	}
	if err = (&v1alpha1controllers.RecordingRuleGroupSetReconciler{
		RecordingRuleGroupSetClient: clientSet.RecordingRuleGroups(),
		Interval:                    cfg.ReconcileIntervals[utils.RecordingRuleGroupSetKind],
		RecordingRuleGroupSetSuffix: cfg.RecordingRuleGroupSetSuffix,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RecordingRuleGroupSet")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.OutboundWebhookReconciler{
		OutboundWebhooksClient: clientSet.Webhooks(),
		Interval:               cfg.ReconcileIntervals[utils.OutboundWebhookKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OutboundWebhook")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ApiKeyReconciler{
		ApiKeysClient: clientSet.APIKeys(),
		Interval:      cfg.ReconcileIntervals[utils.ApiKeyKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApiKey")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.CustomRoleReconciler{
		CustomRolesClient: clientSet.Roles(),
		Interval:          cfg.ReconcileIntervals[utils.CustomRoleKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CustomRole")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.ScopeReconciler{
		ScopesClient: clientSet.Scopes(),
		Interval:     cfg.ReconcileIntervals[utils.ScopeKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Scope")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.GroupReconciler{
		CXClientSet: clientSet,
		Interval:    cfg.ReconcileIntervals[utils.GroupKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Group")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.TCOLogsPoliciesReconciler{
		CoralogixClientSet: clientSet,
		Interval:           cfg.ReconcileIntervals[utils.TCOLogsPoliciesKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TCOLogsPolicies")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.TCOTracesPoliciesReconciler{
		CoralogixClientSet: clientSet,
		Interval:           cfg.ReconcileIntervals[utils.TCOTracesPoliciesKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TCOTracesPolicies")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.IntegrationReconciler{
		IntegrationsClient: clientSet.Integrations(),
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
		DashboardsFoldersClient: clientSet.DashboardsFolders(),
		Interval:                cfg.ReconcileIntervals[utils.DashboardsFolderKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DashboardsFolder")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ViewReconciler{
		ViewsClient: clientSet.Views(),
		Interval:    cfg.ReconcileIntervals[utils.ViewKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "View")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ViewFolderReconciler{
		ViewFoldersClient: clientSet.ViewFolders(),
		Interval:          cfg.ReconcileIntervals[utils.ViewFolderKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ViewFolder")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ConnectorReconciler{
		NotificationsClient: clientSet.Notifications(),
		Interval:            cfg.ReconcileIntervals[utils.ConnectorKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Connector")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.PresetReconciler{
		NotificationsClient: clientSet.Notifications(),
		Interval:            cfg.ReconcileIntervals[utils.PresetKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Preset")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.GlobalRouterReconciler{
		NotificationsClient: clientSet.Notifications(),
		Interval:            cfg.ReconcileIntervals[utils.GlobalRouterKind],
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GlobalRouter")
		os.Exit(1)
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

	if err := monitoring.SetupMetrics(); err != nil {
		setupLog.Error(err, "unable to set up metrics")
		os.Exit(1)
	}

	monitoring.SetOperatorInfoMetric(
		runtime.Version(),
		OperatorVersion,
		cxsdk.CoralogixGrpcEndpointFromRegion(strings.ToLower(cfg.CoralogixUrl)),
	)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

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
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	prometheusv1alpha "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/api/coralogix"
	"github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/config"
	controllers "github.com/coralogix/coralogix-operator/internal/controller"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/internal/controller/coralogix/v1alpha1"
	v1beta1controllers "github.com/coralogix/coralogix-operator/internal/controller/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	webhookcoralogixv1alpha1 "github.com/coralogix/coralogix-operator/internal/webhook/coralogix/v1alpha1"
	webhookcoralogixv1beta1 "github.com/coralogix/coralogix-operator/internal/webhook/coralogix/v1beta1"
	//+kubebuilder:scaffold:imports
)

const OperatorVersion = "0.3.3"

var (
	scheme                    = k8sruntime.NewScheme()
	setupLog                  = ctrl.Log.WithName("setup")
	operatorRegionToSdkRegion = map[string]string{
		"APAC1":   "AP1",
		"AP1":     "AP1",
		"APAC2":   "AP2",
		"AP2":     "AP2",
		"APAC3":   "AP3",
		"AP3":     "AP3",
		"EUROPE1": "EU1",
		"EU1":     "EU1",
		"EUROPE2": "EU2",
		"EU2":     "EU2",
		"USA1":    "US1",
		"US1":     "US1",
		"USA2":    "US2",
		"US2":     "US2",
	}
	validRegions = coralogix.GetKeys(operatorRegionToSdkRegion)
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
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	region := os.Getenv("CORALOGIX_REGION")
	flag.StringVar(&region, "region", region, fmt.Sprintf("The region of your Coralogix cluster. Can be one of %q. Conflicts with 'domain'.", validRegions))

	domain := os.Getenv("CORALOGIX_DOMAIN")
	flag.StringVar(&domain, "domain", domain, "The domain of your Coralogix cluster. Conflicts with 'region'.")

	apiKey := os.Getenv("CORALOGIX_API_KEY")
	flag.StringVar(&apiKey, "api-key", apiKey, "The proper api-key based on your Coralogix cluster's region.")

	labelSelector := os.Getenv("LABEL_SELECTOR")
	flag.StringVar(&labelSelector, "label-selector", labelSelector, "A comma-separated list of key=value labels to filter custom resources.")

	namespaceSelector := os.Getenv("NAMESPACE_SELECTOR")
	flag.StringVar(&namespaceSelector, "namespace-selector", namespaceSelector, "A list of namespaces to filter custom resources.")

	reconcileInterval := os.Getenv("RECONCILE_INTERVAL_SECONDS")
	flag.StringVar(&reconcileInterval, "reconcile-interval-seconds", reconcileInterval, "The interval in seconds between reconciliations.")

	enableWebhooks := os.Getenv("ENABLE_WEBHOOKS")
	flag.StringVar(&enableWebhooks, "enable-webhooks", enableWebhooks, "Enable webhooks for the operator. Default is true.")
	enableWebhooks = strings.ToLower(enableWebhooks)

	var prometheusRuleController bool
	flag.BoolVar(&prometheusRuleController, "prometheus-rule-controller", true, "Determine if the prometheus rule controller should be started. Default is true.")

	var recordingRuleGroupSetSuffix string
	flag.StringVar(&recordingRuleGroupSetSuffix, "recording-rule-group-set-suffix", "", "Suffix to be added to the RecordingRuleGroupSet")

	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if region != "" && domain != "" {
		err := fmt.Errorf("region and domain flags are mutually exclusive")
		setupLog.Error(err, "invalid arguments for running operator")
		os.Exit(1)
	}

	if region == "" && domain == "" {
		err := fmt.Errorf("region or domain must be set")
		setupLog.Error(err, "invalid arguments for running operator")
		os.Exit(1)
	}

	var targetUrl string
	if region != "" {
		if !slices.Contains(validRegions, region) {
			err := fmt.Errorf("region value is '%s', but can be one of %q", region, validRegions)
			setupLog.Error(err, "invalid arguments for running operator")
			os.Exit(1)
		}
		targetUrl = operatorRegionToSdkRegion[region]
	} else if domain != "" {
		targetUrl = domain
	}

	if apiKey == "" {
		err := fmt.Errorf("api-key can not be empty")
		setupLog.Error(err, "invalid arguments for running operator")
		os.Exit(1)
	}

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

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgrOpts := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9e1892e3.coralogix",
		PprofBindAddress:       "0.0.0.0:8888",
	}

	// Check if webhooks are enabled before setting up the webhook server
	if enableWebhooks != "false" {
		mgrOpts.WebhookServer = webhook.NewServer(webhook.Options{
			TLSOpts: tlsOpts,
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientSet := cxsdk.NewClientSet(cxsdk.NewCallPropertiesCreatorOperator(
		strings.ToLower(targetUrl),
		cxsdk.NewAuthContext(apiKey, apiKey),
		OperatorVersion))

	err = config.InitConfig(mgr.GetClient(), mgr.GetScheme(), labelSelector, namespaceSelector, reconcileInterval)
	if err != nil {
		setupLog.Error(err, "unable to initialize config")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.RuleGroupReconciler{
		RuleGroupClient: clientSet.RuleGroups(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RuleGroup")
		os.Exit(1)
	}
	if err = (&v1beta1controllers.AlertReconciler{
		CoralogixClientSet: clientSet,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Alert")
		os.Exit(1)
	}
	if prometheusRuleController {
		if err = (&controllers.PrometheusRuleReconciler{}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "RecordingRuleGroup")
			os.Exit(1)
		}
	}
	if err = (&v1alpha1controllers.RecordingRuleGroupSetReconciler{
		RecordingRuleGroupSetClient: clientSet.RecordingRuleGroups(),
		RecordingRuleGroupSetSuffix: recordingRuleGroupSetSuffix,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RecordingRuleGroupSet")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.OutboundWebhookReconciler{
		OutboundWebhooksClient: clientSet.Webhooks(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OutboundWebhook")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ApiKeyReconciler{
		ApiKeysClient: clientSet.APIKeys(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApiKey")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.CustomRoleReconciler{
		CustomRolesClient: clientSet.Roles(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CustomRole")
		os.Exit(1)
	}

	if err = (&v1alpha1controllers.ScopeReconciler{
		ScopesClient: clientSet.Scopes(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Scope")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.GroupReconciler{
		CXClientSet: clientSet,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Group")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.TCOLogsPoliciesReconciler{
		CoralogixClientSet: clientSet,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TCOLogsPolicies")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.TCOTracesPoliciesReconciler{
		CoralogixClientSet: clientSet,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TCOTracesPolicies")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.IntegrationReconciler{
		IntegrationsClient: clientSet.Integrations(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Integration")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.ConnectorReconciler{
		NotificationsClient: clientSet.Notifications(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Connector")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.PresetReconciler{
		NotificationsClient: clientSet.Notifications(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Preset")
		os.Exit(1)
	}
	if err = (&v1alpha1controllers.GlobalRouterReconciler{
		NotificationsClient: clientSet.Notifications(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GlobalRouter")
		os.Exit(1)
	}

	if enableWebhooks != "false" {
		if err = webhookcoralogixv1alpha1.SetupOutboundWebhookWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OutboundWebhook")
			os.Exit(1)
		}

		if err = webhookcoralogixv1alpha1.SetupRuleGroupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "RuleGroup")
			os.Exit(1)
		}

		if err = webhookcoralogixv1alpha1.SetupApiKeyWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ApiKey")
			os.Exit(1)
		}

		if err = webhookcoralogixv1beta1.SetupAlertWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Alert")
			os.Exit(1)
		}
		if err = webhookcoralogixv1alpha1.SetupConnectorWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Connector")
			os.Exit(1)
		}
		if err = webhookcoralogixv1alpha1.SetupPresetWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Preset")
			os.Exit(1)
		}
	} else {
		setupLog.Info("Webhooks are disabled")
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
		cxsdk.CoralogixGrpcEndpointFromRegion(strings.ToLower(targetUrl)),
	)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

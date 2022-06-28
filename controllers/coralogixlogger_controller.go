/*
Copyright 2020 Coralogix Ltd..

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	loggersv1 "github.com/coralogix/coralogix-operator/api/v1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CoralogixLoggerReconciler reconciles a CoralogixLogger object
type CoralogixLoggerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=loggers.coralogix.com,resources=coralogixloggers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=loggers.coralogix.com,resources=coralogixloggers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=loggers.coralogix.com,resources=coralogixloggers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts;secrets;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces;pods,verbs=get;list;watch

func (r *CoralogixLoggerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	ctx := context.TODO()
	log := r.Log.WithValues("coralogixlogger", req.NamespacedName)

	// Fetch the CoralogixLogger instance status
	instanceStatus := loggersv1.CoralogixLoggerStatus{}

	// Fetch the CoralogixLogger instance
	instance := &loggersv1.CoralogixLogger{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Check instance state
	if instance.Status.State == "RUNNING" {
		log.Info("Skip: CoralogixLogger is already running", "CoralogixLogger.Namespace", instance.Namespace, "CoralogixLogger.Name", instance.Name)
		return ctrl.Result{}, nil
	} else {
		instanceStatus.State = "PENDING"
		instanceStatus.Phase = "Initializing Provision"
		instanceStatus.Reason = fmt.Sprintf("Provisioning of CoralogixLogger '%s' in namespace '%s'...", instance.Name, instance.Namespace)
		r.UpdateStatus(instance, instanceStatus)
	}

	// Check if this ServiceAccount already exists
	serviceAccount := newServiceAccount(instance)
	if err := controllerutil.SetControllerReference(instance, serviceAccount, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundServiceAccount := &corev1.ServiceAccount{}
	err = r.Get(ctx, types.NamespacedName{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace}, foundServiceAccount)
	if err != nil && errors.IsNotFound(err) {
		instanceStatus.State = "PROVISIONING"
		instanceStatus.Phase = "ServiceAccount"
		instanceStatus.Reason = "Provisioning of ServiceAccount..."
		r.UpdateStatus(instance, instanceStatus)

		log.Info("Creating a new ServiceAccount", "ServiceAccount.Namespace", serviceAccount.Namespace, "ServiceAccount.Name", serviceAccount.Name)
		err = r.Create(ctx, serviceAccount)
		if err != nil {
			instanceStatus.State = "FAILED"
			instanceStatus.Reason = "Provisioning of ServiceAccount failed."
			r.UpdateStatus(instance, instanceStatus)
			return ctrl.Result{}, err
		}
		instanceStatus.ServiceAccount = serviceAccount.Name
		instanceStatus.Reason = "Provisioning of ServiceAccount successful."
	} else if err != nil {
		instanceStatus.State = "FAILED"
		instanceStatus.Phase = "ServiceAccount"
		instanceStatus.Reason = "Provisioning of ServiceAccount failed."
		r.UpdateStatus(instance, instanceStatus)
		return ctrl.Result{}, err
	} else {
		instanceStatus.ServiceAccount = foundServiceAccount.Name
		log.Info("Skip: ServiceAccount already exists", "ServiceAccount.Namespace", foundServiceAccount.Namespace, "ServiceAccount.Name", foundServiceAccount.Name)
	}
	r.UpdateStatus(instance, instanceStatus)

	// Check if this ClusterRole already exists
	clusterRole := newClusterRole(instance)
	if err := controllerutil.SetControllerReference(instance, clusterRole, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundClusterRole := &rbacv1.ClusterRole{}
	err = r.Get(ctx, types.NamespacedName{Name: clusterRole.Name, Namespace: ""}, foundClusterRole)
	if err != nil && errors.IsNotFound(err) {
		instanceStatus.State = "PROVISIONING"
		instanceStatus.Phase = "ClusterRole"
		instanceStatus.Reason = "Provisioning of ClusterRole..."
		r.UpdateStatus(instance, instanceStatus)

		log.Info("Creating a new ClusterRole", "ClusterRole.Namespace", clusterRole.Namespace, "ClusterRole.Name", clusterRole.Name)
		err = r.Create(ctx, clusterRole)
		if err != nil {
			instanceStatus.State = "FAILED"
			instanceStatus.Reason = "Provisioning of ClusterRole failed."
			r.UpdateStatus(instance, instanceStatus)
			return ctrl.Result{}, err
		}
		instanceStatus.ClusterRole = clusterRole.Name
		instanceStatus.Reason = "Provisioning of ClusterRole successful."
	} else if err != nil {
		instanceStatus.State = "FAILED"
		instanceStatus.Phase = "ClusterRole"
		instanceStatus.Reason = "Provisioning of ClusterRole failed."
		r.UpdateStatus(instance, instanceStatus)
		return ctrl.Result{}, err
	} else {
		instanceStatus.ClusterRole = foundClusterRole.Name
		log.Info("Skip: ClusterRole already exists", "ClusterRole.Namespace", foundClusterRole.Namespace, "ClusterRole.Name", foundClusterRole.Name)
	}
	r.UpdateStatus(instance, instanceStatus)

	// Check if this ClusterRoleBinding already exists
	clusterRoleBinding := newClusterRoleBinding(instance)
	if err := controllerutil.SetControllerReference(instance, clusterRoleBinding, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = r.Get(ctx, types.NamespacedName{Name: clusterRoleBinding.Name, Namespace: ""}, foundClusterRoleBinding)
	if err != nil && errors.IsNotFound(err) {
		instanceStatus.State = "PROVISIONING"
		instanceStatus.Phase = "ClusterRoleBinding"
		instanceStatus.Reason = "Provisioning of ClusterRoleBinding..."
		r.UpdateStatus(instance, instanceStatus)

		log.Info("Creating a new ClusterRoleBinding", "ClusterRoleBinding.Namespace", clusterRoleBinding.Namespace, "ClusterRoleBinding.Name", clusterRoleBinding.Name)
		err = r.Create(ctx, clusterRoleBinding)
		if err != nil {
			instanceStatus.State = "FAILED"
			instanceStatus.Reason = "Provisioning of ClusterRoleBinding failed."
			r.UpdateStatus(instance, instanceStatus)
			return ctrl.Result{}, err
		}
		instanceStatus.ClusterRoleBinding = clusterRoleBinding.Name
		instanceStatus.Reason = "Provisioning of ClusterRoleBinding successful."
	} else if err != nil {
		instanceStatus.State = "FAILED"
		instanceStatus.Phase = "ClusterRoleBinding"
		instanceStatus.Reason = "Provisioning of ClusterRoleBinding failed."
		r.UpdateStatus(instance, instanceStatus)
		return ctrl.Result{}, err
	} else {
		instanceStatus.ClusterRoleBinding = foundClusterRoleBinding.Name
		log.Info("Skip: ClusterRoleBinding already exists", "ClusterRoleBinding.Namespace", foundClusterRoleBinding.Namespace, "ClusterRoleBinding.Name", foundClusterRoleBinding.Name)
	}
	r.UpdateStatus(instance, instanceStatus)

	// Check if this DaemonSet already exists
	daemonSet := newDaemonSet(instance)
	if err := controllerutil.SetControllerReference(instance, daemonSet, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundDaemonSet := &appsv1.DaemonSet{}
	err = r.Get(ctx, types.NamespacedName{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, foundDaemonSet)
	if err != nil && errors.IsNotFound(err) {
		instanceStatus.State = "PROVISIONING"
		instanceStatus.Phase = "DaemonSet"
		instanceStatus.Reason = "Provisioning of DaemonSet..."
		r.UpdateStatus(instance, instanceStatus)

		log.Info("Creating a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		err = r.Create(ctx, daemonSet)
		if err != nil {
			instanceStatus.State = "FAILED"
			instanceStatus.Reason = "Provisioning of DaemonSet failed."
			r.UpdateStatus(instance, instanceStatus)
			return ctrl.Result{}, err
		}
		instanceStatus.DaemonSet = daemonSet.Name
		instanceStatus.Reason = "Provisioning of DaemonSet successful."
	} else if err != nil {
		instanceStatus.State = "FAILED"
		instanceStatus.Phase = "DaemonSet"
		instanceStatus.Reason = "Provisioning of DaemonSet failed."
		r.UpdateStatus(instance, instanceStatus)
		return ctrl.Result{}, err
	} else {
		instanceStatus.DaemonSet = foundDaemonSet.Name
		log.Info("Skip: DaemonSet already exists", "DaemonSet.Namespace", foundDaemonSet.Namespace, "DaemonSet.Name", foundDaemonSet.Name)
	}
	r.UpdateStatus(instance, instanceStatus)

	instanceStatus.State = "RUNNING"
	instanceStatus.Phase = "Provisioning Succeeded"
	instanceStatus.Reason = fmt.Sprintf("CoralogixLogger '%s' successfully provisioned in namespace '%s'.", instance.Name, instance.Namespace)
	r.UpdateStatus(instance, instanceStatus)

	return ctrl.Result{}, nil
}

// Setup manager
func (r *CoralogixLoggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&loggersv1.CoralogixLogger{}).
		Complete(r)
}

// Update instance status
func (r *CoralogixLoggerReconciler) UpdateStatus(cr *loggersv1.CoralogixLogger, status loggersv1.CoralogixLoggerStatus) error {
	if !reflect.DeepEqual(cr.Status, status) {
		cr.Status = status
		err := r.Status().Update(context.TODO(), cr)
		if err != nil {
			return err
		}
	}
	return nil
}

// newServiceAccount returns a ServiceAccount with the same namespace as the cr
func newServiceAccount(cr *loggersv1.CoralogixLogger) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-coralogix-service-account",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"k8s-app": "fluentd-coralogix-" + cr.Name,
			},
		},
	}
}

// newClusterRole returns a ClusterRole with the same namespace as the cr
func newClusterRole(cr *loggersv1.CoralogixLogger) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-coralogix-role",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"k8s-app": "fluentd-coralogix-" + cr.Name,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{
					"namespaces",
					"pods",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
		},
	}
}

// newClusterRoleBinding returns a ClusterRoleBinding with the same namespace as the cr
func newClusterRoleBinding(cr *loggersv1.CoralogixLogger) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-coralogix-role-binding",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"k8s-app": "fluentd-coralogix-" + cr.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "fluentd-coralogix-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "fluentd-coralogix-service-account",
				Namespace: cr.Namespace,
			},
		},
	}
}

// newDaemonSet returns a DaemonSet with the same name/namespace as the cr
func newDaemonSet(cr *loggersv1.CoralogixLogger) *appsv1.DaemonSet {
	var uid int64 = 0
	var privileged bool = true
	coralogixEndpoints := map[string]string{
		"Europe":    "api.coralogix.com",
		"Europe2":   "api.eu2.coralogix.com",
		"India":     "api.app.coralogix.in",
		"Singapore": "api.coralogixsg.com",
		"US":        "api.coralogix.us",
	}
	coralogixURL, ok := coralogixEndpoints[cr.Spec.Region]
	if !ok {
		coralogixURL = "api.coralogix.com"
	}
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-coralogix-" + cr.Name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"k8s-app":                       "fluentd-coralogix-" + cr.Name,
				"kubernetes.io/cluster-service": "true",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"k8s-app": "fluentd-coralogix-" + cr.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app":                       "fluentd-coralogix-" + cr.Name,
						"kubernetes.io/cluster-service": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "fluentd",
						Image:           "registry.connect.redhat.com/coralogix/coralogix-fluentd:1.0.1",
						ImagePullPolicy: corev1.PullAlways,
						SecurityContext: &corev1.SecurityContext{
							RunAsUser:  &uid,
							Privileged: &privileged,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "CORALOGIX_ENDPOINT",
								Value: coralogixURL,
							},
							{
								Name:  "CORALOGIX_PRIVATE_KEY",
								Value: cr.Spec.PrivateKey,
							},
							{
								Name:  "CLUSTER_NAME",
								Value: cr.Spec.ClusterName,
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("400Mi"),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "varlog",
								MountPath: "/var/log",
							},
							{
								Name:      "varlibdockercontainers",
								MountPath: "/var/lib/docker/containers",
								ReadOnly:  true,
							},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "varlog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log",
								},
							},
						},
						{
							Name: "varlibdockercontainers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/docker/containers",
								},
							},
						},
					},
					ServiceAccountName: "fluentd-coralogix-service-account",
				},
			},
		},
	}
}

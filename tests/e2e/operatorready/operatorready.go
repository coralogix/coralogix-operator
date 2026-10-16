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

// Package operatorready decides whether the operator that e2e specs run against is the one
// the Deployment currently describes.
//
// A Deployment is Available while the pod from before a Helm upgrade still runs. That old pod
// also keeps the leader lease until the lease expires, so it reconciles every CR created in the
// meantime. Specs that start in that window test the old operator, not the new one.
package operatorready

import (
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	revisionAnnotation = "deployment.kubernetes.io/revision"
	defaultLeaseName   = "coralogix-operator"
)

// LeaderLease returns the lease name the operator container uses, and whether leader election
// is on. It reads the same flags, with the same defaults, as internal/config.
func LeaderLease(container corev1.Container) (name string, enabled bool) {
	name, enabled = defaultLeaseName, true
	for _, arg := range container.Args {
		flagName, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		switch flagName {
		case "leader-elect":
			if !hasValue {
				enabled = true
				continue
			}
			parsed, err := strconv.ParseBool(value)
			enabled = err == nil && parsed
		case "leader-election-id":
			name = value
		}
	}
	return name, enabled
}

// Check returns nil once the rollout of dep is complete, no pod from an older ReplicaSet is
// left, and, when leader election is on, a live pod of the current ReplicaSet holds the lease.
// Otherwise it returns the reason to keep waiting. lease is nil when it does not exist yet.
func Check(dep *appsv1.Deployment, replicaSets []appsv1.ReplicaSet, pods []corev1.Pod, lease *coordinationv1.Lease) error {
	if dep.Status.ObservedGeneration < dep.Generation {
		return fmt.Errorf("deployment %s generation %d is not observed yet (observed %d)",
			dep.Name, dep.Generation, dep.Status.ObservedGeneration)
	}
	if len(dep.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("deployment %s has no containers", dep.Name)
	}

	current, owned := currentReplicaSet(dep, replicaSets)
	if current == "" {
		return fmt.Errorf("deployment %s has no ReplicaSet for revision %q",
			dep.Name, dep.Annotations[revisionAnnotation])
	}

	livePods := map[string]bool{}
	for _, pod := range pods {
		owner := metav1.GetControllerOf(&pod)
		if owner == nil || owner.Kind != "ReplicaSet" || !owned[owner.Name] {
			continue
		}
		if owner.Name != current {
			return fmt.Errorf("pod %s from older ReplicaSet %s still exists", pod.Name, owner.Name)
		}
		if pod.DeletionTimestamp == nil && isReady(pod) {
			livePods[pod.Name] = true
		}
	}
	if len(livePods) == 0 {
		return fmt.Errorf("no ready pod of current ReplicaSet %s", current)
	}

	leaseName, enabled := LeaderLease(dep.Spec.Template.Spec.Containers[0])
	if !enabled {
		return nil
	}
	if lease == nil || lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity == "" {
		return fmt.Errorf("lease %s has no holder yet", leaseName)
	}
	// controller-runtime names the holder "<hostname>_<uuid>", and a pod's hostname is its name.
	holder := *lease.Spec.HolderIdentity
	holderPod, _, _ := strings.Cut(holder, "_")
	if !livePods[holderPod] {
		return fmt.Errorf("lease %s is held by %q, not by a ready pod of current ReplicaSet %s",
			leaseName, holder, current)
	}
	return nil
}

// currentReplicaSet returns the name of the ReplicaSet for the Deployment's current revision,
// and the names of every ReplicaSet the Deployment controls.
func currentReplicaSet(dep *appsv1.Deployment, replicaSets []appsv1.ReplicaSet) (string, map[string]bool) {
	revision := dep.Annotations[revisionAnnotation]
	current := ""
	owned := map[string]bool{}
	for _, rs := range replicaSets {
		if !metav1.IsControlledBy(&rs, dep) {
			continue
		}
		owned[rs.Name] = true
		if revision != "" && rs.Annotations[revisionAnnotation] == revision {
			current = rs.Name
		}
	}
	return current, owned
}

func isReady(pod corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

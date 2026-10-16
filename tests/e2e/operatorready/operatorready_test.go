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

package operatorready

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

const (
	oldRS  = "coralogix-operator-6b9f7c8d4"
	newRS  = "coralogix-operator-d795f98c6"
	oldPod = oldRS + "-x2k9p"
	newPod = newRS + "-jtd6w"
)

// TestCheckHelmUpgradeTimeline replays a Helm upgrade from an old chart. Only the last step,
// where the new pod holds the lease and the old pod is gone, may let specs start.
func TestCheckHelmUpgradeTimeline(t *testing.T) {
	steps := []struct {
		name      string
		dep       *appsv1.Deployment
		rs        []appsv1.ReplicaSet
		pods      []corev1.Pod
		lease     *coordinationv1.Lease
		wantError string
	}{
		{
			name:      "helm upgrade returned, the deployment controller has not seen it",
			dep:       deployment(1, "1", chartArgs()),
			rs:        []appsv1.ReplicaSet{replicaSet(oldRS, "1")},
			pods:      []corev1.Pod{pod(oldPod, oldRS, true, false)},
			lease:     lease(oldPod),
			wantError: "generation 2 is not observed yet",
		},
		{
			name:      "new pod starting, old pod still runs and holds the lease",
			dep:       deployment(2, "2", chartArgs()),
			rs:        []appsv1.ReplicaSet{replicaSet(oldRS, "1"), replicaSet(newRS, "2")},
			pods:      []corev1.Pod{pod(oldPod, oldRS, true, false), pod(newPod, newRS, false, false)},
			lease:     lease(oldPod),
			wantError: "pod " + oldPod + " from older ReplicaSet " + oldRS + " still exists",
		},
		{
			name:      "new pod ready, old pod terminating and still holds the lease",
			dep:       deployment(2, "2", chartArgs()),
			rs:        []appsv1.ReplicaSet{replicaSet(oldRS, "1"), replicaSet(newRS, "2")},
			pods:      []corev1.Pod{pod(oldPod, oldRS, true, true), pod(newPod, newRS, true, false)},
			lease:     lease(oldPod),
			wantError: "from older ReplicaSet",
		},
		{
			// This is the window that broke the dashboard import spec: the rollout is
			// complete, but nobody runs the new code until the old lease expires.
			name:      "old pod gone, its lease has not expired yet",
			dep:       deployment(2, "2", chartArgs()),
			rs:        []appsv1.ReplicaSet{replicaSet(oldRS, "1"), replicaSet(newRS, "2")},
			pods:      []corev1.Pod{pod(newPod, newRS, true, false)},
			lease:     lease(oldPod),
			wantError: "lease coralogix-operator is held by \"" + oldPod + "_",
		},
		{
			name:  "new pod acquired the lease",
			dep:   deployment(2, "2", chartArgs()),
			rs:    []appsv1.ReplicaSet{replicaSet(oldRS, "1"), replicaSet(newRS, "2")},
			pods:  []corev1.Pod{pod(newPod, newRS, true, false)},
			lease: lease(newPod),
		},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			err := Check(step.dep, step.rs, step.pods, step.lease)
			if step.wantError == "" {
				assert.NoError(t, err)
				return
			}
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), step.wantError)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	settledRS := []appsv1.ReplicaSet{replicaSet(oldRS, "1"), replicaSet(newRS, "2")}
	settledPods := []corev1.Pod{pod(newPod, newRS, true, false)}

	tests := []struct {
		name      string
		dep       *appsv1.Deployment
		rs        []appsv1.ReplicaSet
		pods      []corev1.Pod
		lease     *coordinationv1.Lease
		wantError string
	}{
		{
			name:      "lease does not exist yet",
			dep:       deployment(2, "2", chartArgs()),
			rs:        settledRS,
			pods:      settledPods,
			wantError: "lease coralogix-operator has no holder yet",
		},
		{
			name:      "lease has an empty holder",
			dep:       deployment(2, "2", chartArgs()),
			rs:        settledRS,
			pods:      settledPods,
			lease:     &coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{HolderIdentity: ptr.To("")}},
			wantError: "has no holder yet",
		},
		{
			name:      "new pod holds the lease but is not ready",
			dep:       deployment(2, "2", chartArgs()),
			rs:        settledRS,
			pods:      []corev1.Pod{pod(newPod, newRS, false, false)},
			lease:     lease(newPod),
			wantError: "no ready pod of current ReplicaSet " + newRS,
		},
		{
			name:      "no ReplicaSet for the current revision",
			dep:       deployment(2, "3", chartArgs()),
			rs:        settledRS,
			pods:      settledPods,
			lease:     lease(newPod),
			wantError: "has no ReplicaSet for revision \"3\"",
		},
		{
			name:  "leader election off does not need a lease",
			dep:   deployment(2, "2", []string{"-leader-elect=false"}),
			rs:    settledRS,
			pods:  settledPods,
			lease: nil,
		},
		{
			name: "pods of another deployment are ignored",
			dep:  deployment(2, "2", chartArgs()),
			rs:   settledRS,
			pods: append([]corev1.Pod{pod("other-7f8d9-abcde", "other-7f8d9", true, false)},
				settledPods...),
			lease: lease(newPod),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Check(tt.dep, tt.rs, tt.pods, tt.lease)
			if tt.wantError == "" {
				assert.NoError(t, err)
				return
			}
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantError)
			}
		})
	}
}

func TestLeaderLease(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantName    string
		wantEnabled bool
	}{
		{name: "helm chart", args: chartArgs(), wantName: "coralogix-operator", wantEnabled: true},
		{name: "custom release name", args: []string{"-leader-elect=true", "-leader-election-id=my-release"}, wantName: "my-release", wantEnabled: true},
		{name: "kustomize bare flag", args: []string{"--leader-elect", "--zap-log-level=debug"}, wantName: "coralogix-operator", wantEnabled: true},
		{name: "flags absent use the operator defaults", args: nil, wantName: "coralogix-operator", wantEnabled: true},
		{name: "disabled", args: []string{"-leader-elect=false"}, wantName: "coralogix-operator", wantEnabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, enabled := LeaderLease(corev1.Container{Args: tt.args})
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantEnabled, enabled)
		})
	}
}

func chartArgs() []string {
	return []string{"-leader-elect=true", "-leader-election-id=coralogix-operator"}
}

// deployment returns a Deployment at generation 2, the generation right after one Helm upgrade.
func deployment(observedGeneration int64, revision string, args []string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "coralogix-operator",
			UID:         types.UID("deployment-uid"),
			Generation:  2,
			Annotations: map[string]string{revisionAnnotation: revision},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "coralogix-operator", Args: args}}},
			},
		},
		Status: appsv1.DeploymentStatus{ObservedGeneration: observedGeneration},
	}
}

func replicaSet(name, revision string) appsv1.ReplicaSet {
	return appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			UID:             types.UID(name + "-uid"),
			Annotations:     map[string]string{revisionAnnotation: revision},
			OwnerReferences: []metav1.OwnerReference{controllerRef("Deployment", "coralogix-operator", "deployment-uid")},
		},
	}
}

func pod(name, replicaSet string, ready, terminating bool) corev1.Pod {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			OwnerReferences: []metav1.OwnerReference{controllerRef("ReplicaSet", replicaSet, replicaSet+"-uid")},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	readyStatus := corev1.ConditionFalse
	if ready {
		readyStatus = corev1.ConditionTrue
	}
	p.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: readyStatus}}
	if terminating {
		p.DeletionTimestamp = ptr.To(metav1.Now())
	}
	return p
}

func lease(holderPod string) *coordinationv1.Lease {
	return &coordinationv1.Lease{
		Spec: coordinationv1.LeaseSpec{HolderIdentity: ptr.To(holderPod + "_6f1c2a4e-1b7d-4c3e-9a51-2d8f0e7b3c19")},
	}
}

func controllerRef(kind, name, uid string) metav1.OwnerReference {
	return metav1.OwnerReference{Kind: kind, Name: name, UID: types.UID(uid), Controller: ptr.To(true)}
}

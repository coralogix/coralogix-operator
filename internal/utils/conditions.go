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

package utils

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReasonRemoteSyncedSuccessfully = "RemoteSyncedSuccessfully"
	ReasonRemoteCreationFailed     = "RemoteCreationFailed"
	ReasonRemoteUpdateFailed       = "RemoteUpdateFailed"
	ReasonRemoteDeletionFailed     = "RemoteDeletionFailed"
	ReasonRemoteResourceNotFound   = "RemoteResourceNotFound"
	ReasonInternalK8sError         = "InternalK8sError"

	ConditionTypeRemoteSynced = "RemoteSynced"
)

// SetSyncedConditionFalse sets the RemoteSynced condition to False. returns true if the conditions are changed by this call.
func SetSyncedConditionFalse(conditions *[]metav1.Condition, observedGeneration int64, reason, message string) bool {
	return meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               ConditionTypeRemoteSynced,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: observedGeneration,
	})
}

// SetSyncedConditionTrue sets the RemoteSynced condition to True. returns true if the conditions are changed by this call.
func SetSyncedConditionTrue(conditions *[]metav1.Condition, observedGeneration int64, reason string) bool {
	return meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               ConditionTypeRemoteSynced,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            "Remote resource synced",
		ObservedGeneration: observedGeneration,
	})
}

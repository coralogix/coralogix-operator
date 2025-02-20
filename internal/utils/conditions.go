package utils

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReasonRemoteCreatedSuccessfully = "RemoteCreatedSuccessfully"
	ReasonRemoteCreationFailed      = "RemoteCreationFailed"
	ReasonRemoteUpdatedSuccessfully = "RemoteUpdatedSuccessfully"
	ReasonRemoteUpdateFailed        = "RemoteUpdateFailed"
	ReasonRemoteDeletionFailed      = "RemoteDeletionFailed"
	ReasonRemoteDeletedSuccessfully = "RemoteDeletedSuccessfully"
	ReasonRemoteResourceNotFound    = "RemoteResourceNotFound"
	ReasonInternalK8sError          = "InternalK8sError"

	ConditionTypeError        = "Error"
	ConditionTypeRemoteSynced = "RemoteSynced"
)

// ConditionsObj represents a CRD type that has been enabled with metav1.Conditions, it can then benefit of a series of utility methods.
// +k8s:deepcopy-gen=false
type ConditionsObj interface {
	GetConditions() []metav1.Condition
	SetConditions(conditions []metav1.Condition)
}

// SetSyncedConditionFalse sets the RemoteSynced condition to False with the provided reason and message only if it is not already False.
// returns true if the conditions are changed by this call
func SetSyncedConditionFalse(conditions *[]metav1.Condition, observedGeneration int64, reason, message string) bool {
	if !meta.IsStatusConditionFalse(*conditions, ConditionTypeRemoteSynced) {
		syncingCondition := metav1.Condition{
			Type:               ConditionTypeRemoteSynced,
			Status:             metav1.ConditionFalse,
			Reason:             reason,
			Message:            message,
			ObservedGeneration: observedGeneration,
		}
		return meta.SetStatusCondition(conditions, syncingCondition)
	}

	return false
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

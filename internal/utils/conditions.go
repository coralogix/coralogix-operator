package utils

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	RemoteCreatedSuccessfully = "RemoteCreatedSuccessfully"
	RemoteCreationFailed      = "RemoteCreationFailed"
	RemoteUpdatedSuccessfully = "RemoteUpdatedSuccessfully"
	RemoteUpdateFailed        = "RemoteUpdateFailed"
	RemoteDeletionFailed      = "RemoteDeletionFailed"
	RemoteDeletedSuccessfully = "RemoteDeletedSuccessfully"
	RemoteResourceNotFound    = "RemoteResourceNotFound"
	InternalK8sError          = "InternalK8sError"

	ConditionTypeError        = "Error"
	ConditionTypeRemoteSynced = "RemoteSynced"
)

// ConditionsObj represents a CRD type that has been enabled with metav1.Conditions, it can then benefit of a series of utility methods.
type ConditionsObj interface {
	GetConditions() []metav1.Condition
	SetConditions(conditions []metav1.Condition)
}

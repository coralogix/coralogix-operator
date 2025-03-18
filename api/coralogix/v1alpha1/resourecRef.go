package v1alpha1

type ResourceRef struct {
	Name string `json:"name"`

	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

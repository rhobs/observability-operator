package v1alpha1

// TroubleShootingSpec allows to enable and configure troubleshooting features
type TroubleShootingSpec struct {
	// Enabled indicates whether the troubleshooting capabilities are enabled.
	// By default, it is set to false.
	// +optional
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`
}

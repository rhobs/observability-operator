package v1alpha1

// TracingSpec defines the desired state of the tracing capability.
type TracingSpec struct {
	CommonCapabilitiesSpec CommonCapabilitiesSpec `json:",inline"`
}

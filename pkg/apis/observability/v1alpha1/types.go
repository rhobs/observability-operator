// +groupName=observability.openshift.io
// +kubebuilder:rbac:groups=observability.openshift.io,resources=clusterobservability,verbs=list;get;watch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=clusterobservability/status;clusterobservability/finalizers,verbs=get;update
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterObservability defines the desired state of the observability stack.
//
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=clobs;clobs
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +operator-sdk:csv:customresourcedefinitions:displayName="Cluster Observability"
type ClusterObservability struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the cluster observability.
	Spec ClusterObservabilitySpec `json:"spec,omitempty"`

	// Status of the signal manager.
	Status ClusterObservabilityStatus `json:"status,omitempty"`
}

// +kubebuilder:resource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterObservabilityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterObservability `json:"items"`
}

type ClusterObservabilitySpec struct {
	// Storage defines the storage for the capabilities that require a storage.
	Storage StorageSpec `json:"storage,omitempty"`

	// Capabilities defines the observability capabilities.
	// Each capability has to be enabled explicitly.
	Capabilities *CapabilitiesSpec `json:"capabilities,omitempty"`
}

// ClusterObservabilityStatus defines the observed state of ClusterObservability.
type ClusterObservabilityStatus struct{}

// StorageSpec defines the storage.
type StorageSpec struct {
	Secret SecretSpec `json:"secret,omitempty"`
}

// SecretSpec defines the secret for the storage.
type SecretSpec struct {
	// Name is the name of the secret for the storage.
	Name string `json:"name,omitempty"`
}

// CapabilitiesSpec defines the observability capabilities.
type CapabilitiesSpec struct {

	// Tracing defines the tracing capabilities.
	// +optional
	// +kubebuilder:validation:Optional
	Tracing TracingSpec `json:"tracing,omitempty"`

	// OpenTelemetry defines the OpenTelemetry capabilities.
	// +optional
	// +kubebuilder:validation:Optional
	OpenTelemetry OpenTelemetrySpec `json:"opentelemetry,omitempty"`
}

// CommonCapabilitiesSpec defines the common capabilities.
type CommonCapabilitiesSpec struct {
	// Enabled indicates whether the capability is enabled and it operator should deploy an instance.
	// By default, it is set to false.
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
	// OLM indicates whether the operators used by the capability should be deployed via OLM.
	// When the capability is enabled, the OLM is set to true, otherwise it is set to false.
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	OLM bool `json:"olm,omitempty"`
}

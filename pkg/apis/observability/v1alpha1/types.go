// +groupName=observability.openshift.io
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
// +kubebuilder:printcolumn:name="OpenTelemetry",type="string",JSONPath=".status.opentelemetry"
// +kubebuilder:printcolumn:name="Tempo",type="string",JSONPath=".status.tempo"
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
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Capabilities",order=2
	Capabilities *CapabilitiesSpec `json:"capabilities,omitempty"`
}

// ClusterObservabilityStatus defines the observed state of ClusterObservability.
type ClusterObservabilityStatus struct {
	// OpenTelemetry defines the status of the OpenTelemetry capability.
	// The value is in the form of instance namespace/name (version)
	// +optional
	OpenTelemetry string `json:"opentelemetry,omitempty"`
	// Tempo defines the status of the Tempo capability.
	// The value is in the form of instance namespace/name (version)
	// +optional
	Tempo string `json:"tempo,omitempty"`

	// Conditions provide status information about the instance.
	// +listType=atomic
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type StorageSecretType string

const (
	S3 StorageSecretType = "s3"
)

// StorageSpec defines the storage.
type StorageSpec struct {
	Secret SecretSpec `json:"secret,omitempty"`
}

// SecretSpec defines the secret for the storage.
type SecretSpec struct {
	// Name is the name of the secret for the storage.
	Name string `json:"name,omitempty"`

	// Type is the type of the secret for the storage.
	Type StorageSecretType `json:"type,omitempty"`
}

// CapabilitiesSpec defines the observability capabilities.
type CapabilitiesSpec struct {

	// Tracing defines the tracing capabilities.
	// +optional
	// +kubebuilder:validation:Optional
	Tracing TracingSpec `json:"tracing,omitempty"`
}

// CommonCapabilitiesSpec defines the common capabilities.
type CommonCapabilitiesSpec struct {
	// Enabled indicates whether the capability is enabled and whether the operator should deploy an instance.
	// By default, it is set to false.
	// +optional
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`

	// Operators defines the operators installation for the capability.
	// +optional
	// +kubebuilder:validation:Optional
	Operators OperatorsSpec `json:"operators,omitempty"`
}

// OperatorsSpec defines the operators installation.
type OperatorsSpec struct {
	// Install indicates whether the operator(s) used by the capability should be installed via OLM.
	// When the capability is enabled, the install is set to true, otherwise it is set to false.
	// +optional
	// +kubebuilder:validation:Optional
	Install *bool `json:"install,omitempty"`
}

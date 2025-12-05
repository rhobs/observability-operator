// +groupName=observability.openshift.io
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityInstaller defines the desired state of the observability stack.
//
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=obsinstall;obsinst
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="OpenTelemetry",type="string",JSONPath=".status.opentelemetry"
// +kubebuilder:printcolumn:name="Tempo",type="string",JSONPath=".status.tempo"
// +operator-sdk:csv:customresourcedefinitions:displayName="Observability Installer"
// +operator-sdk:csv:customresourcedefinitions:description="Provides end-to-end observability capabilities with minimal configuration. Simplifies deployment and management of observability components such as tracing."
// +kubebuilder:metadata:annotations="observability.openshift.io/api-support=TechPreview"
type ObservabilityInstaller struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the observability installer.
	Spec ObservabilityInstallerSpec `json:"spec,omitempty"`

	// Status of the signal manager.
	Status ObservabilityInstallerStatus `json:"status,omitempty"`
}

// +kubebuilder:resource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ObservabilityInstallerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ObservabilityInstaller `json:"items"`
}

type ObservabilityInstallerSpec struct {
	// Capabilities defines the observability capabilities.
	// Each capability has to be enabled explicitly.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Capabilities",order=2
	Capabilities *CapabilitiesSpec `json:"capabilities,omitempty"`
}

// ObservabilityInstallerStatus defines the observed state of ObservabilityInstaller.
type ObservabilityInstallerStatus struct {
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

// TLSSpec is the TLS configuration.
// +kubebuilder:validation:XValidation:rule="(has(self.keySecret) && has(self.certSecret)) || (!has(self.keySecret) && !has(self.certSecret))",message="KeySecret and CertSecret must be set together"
type TLSSpec struct {
	// CAConfigMap is the name of a ConfigMap containing a CA certificate (e.g. service-ca.crt).
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:ConfigMap",displayName="CA ConfigMap"
	CAConfigMap *ConfigMapKeySelector `json:"caConfigMap,omitempty"`

	// CertSecret is the name of a Secret containing a certificate (e.g. tls.crt).
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret",displayName="Certificate Secret"
	CertSecret *SecretKeySelector `json:"certSecret,omitempty"`

	// KeySecret is the name of a Secret containing a private key (e.g. tls.key).
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret",displayName="Certificate Secret"
	KeySecret *SecretKeySelector `json:"keySecret,omitempty"`

	// MinVersion defines the minimum acceptable TLS version.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Min TLS Version"
	MinVersion string `json:"minVersion,omitempty"`
}

// CapabilitiesSpec defines the observability capabilities.
type CapabilitiesSpec struct {

	// Tracing defines the tracing capabilities.
	// The tracing capability install an OpenTelemetry Operator instance and a Tempo instance.
	// The Tempo instance is configured with a single tenant called application.
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
	// This field can be used to install the operator(s) without installing any operands.
	// +optional
	// +kubebuilder:validation:Optional
	Install *bool `json:"install,omitempty"`
}

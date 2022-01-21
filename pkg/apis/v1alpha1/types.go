// +groupName=monitoring.rhobs
// +kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=list;get;watch
// +kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks/status;monitoringstacks/finalizers,verbs=get;update

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitoringStack is the Schema for the monitoringstacks API
// +k8s:openapi-gen=true
// +kubebuilder:resource
// +kubebuilder:subresource:status
type MonitoringStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MonitoringStackSpec   `json:"spec,omitempty"`
	Status MonitoringStackStatus `json:"status,omitempty"`
}

// MonitoringStackList contains a list of MonitoringStack
// +kubebuilder:resource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MonitoringStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MonitoringStack `json:"items"`
}

// Loglevel set log levels of configured components
// +kubebuilder:validation:Enum=debug;info;warning
type LogLevel string

const (
	// Debug Log level
	Debug LogLevel = "debug"

	// Info Log level
	Info LogLevel = "info"

	// Warning Log level
	Warning LogLevel = "warning"
)

// MonitoringStackSpec is the specification for desired Monitoring Stack
type MonitoringStackSpec struct {
	// +optional
	// +kubebuilder:default="info"
	LogLevel LogLevel `json:"logLevel,omitempty"`

	// Label selector for Monitoring Stack Resources.
	// +optional
	ResourceSelector *metav1.LabelSelector `json:"resourceSelector,omitempty"`

	// Time duration to retain data for. Default is '120h',
	// and must match the regular expression `[0-9]+(ms|s|m|h|d|w|y)` (milliseconds seconds minutes hours days weeks years).
	// +kubebuilder:validation:Pattern="^[0-9]+(ms|s|m|h|d|w|y)$"
	// +kubebuilder:default="120h"
	Retention string `json:"retention,omitempty"`

	// Define resources requests and limits for Monitoring Stack Pods.
	// +optional
	// +kubebuilder:default={requests:{cpu: "100m", memory: "256M"}, limits:{memory: "512M", cpu: "500m"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Define prometheus config
	// +optional
	PrometheusConfig *PrometheusConfig `json:"prometheusConfig,omitempty"`
}

// MonitoringStackStatus defines the observed state of MonitoringStack.
// It should always be reconstructable from the state of the cluster and/or outside world.
type MonitoringStackStatus struct {
	// TODO(sthaha): INSERT ADDITIONAL STATUS FIELDS -- observed state of prometheus
	// ??
}

type PrometheusConfig struct {
	// Define persistent volume claim for prometheus
	// +optional
	PersistentVolumeClaim corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaim,omitempty"`
}

// NamespaceSelector is a selector for selecting either all namespaces or a
// list of namespaces.
// +k8s:openapi-gen=true
type NamespaceSelector struct {
	// Boolean describing whether all namespaces are selected in contrast to a
	// list restricting them.
	Any bool `json:"any,omitempty"`
	// List of namespace names.
	MatchNames []string `json:"matchNames,omitempty"`
}

// ThanosQuerier outlines the Thanos querier components, managed by this stack
// +k8s:openapi-gen=true
// +kubebuilder:resource
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ThanosQuerier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ThanosQuerierSpec   `json:"spec,omitempty"`
	Status ThanosQuerierStatus `json:"status,omitempty"`
}

// ThanosQuerierList contains a list of ThanosQuerier
// +kubebuilder:resource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ThanosQuerierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ThanosQuerier `json:"items"`
}

// ThanosQuerierSpec defines a single Thanos Querier instance. This means a
// label selector by which Monitoring Stack instances to query are selected, and
// an optional namespace selector and a list of replica labels by which to
// deduplicate.
type ThanosQuerierSpec struct {
	// Selector to select Monitoring stacks to unify
	Selector metav1.LabelSelector `json:"selector"`
	// Selector to select which namespaces the Monitoring Stack objects are discovered from.
	NamespaceSelector NamespaceSelector `json:"namespaceSelector,omitempty"`
	ReplicaLabels     []string          `json:"replicaLabels,omitempty"`
}

// ThanosQuerierStatus defines the observed state of ThanosQuerier.
// It should always be reconstructable from the state of the cluster and/or outside world.
type ThanosQuerierStatus struct {
}

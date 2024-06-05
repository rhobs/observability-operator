// +groupName=monitoring.rhobs
// +kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=list;get;watch
// +kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks/status;monitoringstacks/finalizers,verbs=get;update

package v1alpha1

import (
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rhobs/observability-operator/pkg/apis/shared"
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

func (m *MonitoringStack) Conditions() []shared.Condition {
	return m.Status.Conditions
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
// +kubebuilder:validation:Enum=debug;info;warn;error
type LogLevel string

const (
	// Debug Log level
	Debug LogLevel = "debug"

	// Info Log level
	Info LogLevel = "info"

	// Warn Log level
	Warn LogLevel = "warn"

	// Error Log level
	Error LogLevel = "error"
)

// MonitoringStackSpec is the specification for desired Monitoring Stack
type MonitoringStackSpec struct {
	// +optional
	// +kubebuilder:default="info"
	LogLevel LogLevel `json:"logLevel,omitempty"`

	// Label selector for Monitoring Stack Resources.
	// To monitor everything, set to empty map selector. E.g. resourceSelector: {}.
	// To disable service discovery, set to null. E.g. resourceSelector:.
	// +optional
	// +nullable
	ResourceSelector *metav1.LabelSelector `json:"resourceSelector"`

	// Namespace selector for Monitoring Stack Resources.
	// To monitor everything, set to empty map selector. E.g. namespaceSelector: {}.
	// To monitor resources in the namespace where Monitoring Stack was created in, set to null. E.g. namespaceSelector:.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Time duration to retain data for. Default is '120h',
	// and must match the regular expression `[0-9]+(ms|s|m|h|d|w|y)` (milliseconds seconds minutes hours days weeks years).
	// +kubebuilder:default="120h"
	Retention monv1.Duration `json:"retention,omitempty"`

	// Define resources requests and limits for Monitoring Stack Pods.
	// +optional
	// +kubebuilder:default={requests:{cpu: "100m", memory: "256Mi"}, limits:{memory: "512Mi", cpu: "500m"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Define prometheus config
	// +optional
	// +kubebuilder:default={replicas: 2}
	PrometheusConfig *PrometheusConfig `json:"prometheusConfig,omitempty"`

	// Define Alertmanager config
	// +optional
	// +kubebuilder:default={disabled: false}
	AlertmanagerConfig AlertmanagerConfig `json:"alertmanagerConfig,omitempty"`
}

// MonitoringStackStatus defines the observed state of MonitoringStack.
// It should always be reconstructable from the state of the cluster and/or outside world.
type MonitoringStackStatus struct {
	// Conditions provide status information about the MonitoringStack
	// +listType=atomic
	Conditions []shared.Condition `json:"conditions"`
}

type PrometheusConfig struct {
	// Number of replicas/pods to deploy for a Prometheus deployment.
	// +optional
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`

	// Define remote write for prometheus
	// +optional
	RemoteWrite []monv1.RemoteWriteSpec `json:"remoteWrite,omitempty"`
	// Define persistent volume claim for prometheus
	// +optional
	PersistentVolumeClaim *corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaim,omitempty"`
	// Define ExternalLabels for prometheus
	// +optional
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`
	// Enable Prometheus to be used as a receiver for the Prometheus remote write protocol. Defaults to the value of `false`.
	// +optional
	EnableRemoteWriteReceiver bool `json:"enableRemoteWriteReceiver,omitempty"`
	// Enable Prometheus to accept OpenTelemetry Metrics via the otlp/http protocol.
	// Defaults to the value of `false`.
	// The resulting endpoint is /api/v1/otlp/v1/metrics.
	// +optional
	EnableOtlpHttpReceiver *bool `json:"enableOtlpHttpReceiver,omitempty"`
	// Default interval between scrapes.
	// +optional
	ScrapeInterval *monv1.Duration `json:"scrapeInterval,omitempty"`
}

type AlertmanagerConfig struct {
	// Disables the deployment of Alertmanager.
	// +optional
	// +kubebuilder:default=false
	Disabled bool `json:"disabled,omitempty"`
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

func (t ThanosQuerier) MatchesNamespace(namespace string) bool {
	namespaceSelector := t.Spec.NamespaceSelector
	if namespaceSelector.Any {
		return true
	}

	if len(namespaceSelector.MatchNames) == 0 {
		return t.Namespace == namespace
	}

	for _, ns := range namespaceSelector.MatchNames {
		if ns == namespace {
			return true
		}
	}

	return false
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

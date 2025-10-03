// +groupName=monitoring.rhobs
// +kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=list;get;watch
// +kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks/status;monitoringstacks/finalizers,verbs=get;update

package v1alpha1

import (
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitoringStack is the Schema for the monitoringstacks API
// +k8s:openapi-gen=true
// +kubebuilder:resource
// +kubebuilder:subresource:status
// +kubebuilder:metadata:annotations="observability.openshift.io/api-support=TechPreview"
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

// +kubebuilder:validation:Enum=CreateClusterRoleBindings;NoClusterRoleBindings
type ClusterRoleBindingPolicy string

const (
	// CreateClusterRoleBindings instructs the MonitoringStack to create the
	// default ClusterRoleBindings if a NamespaceSelector is present. Note that
	// this allows user who can access the Prometheus or Alertmanager
	// ServiceAccounts to possibly elevate their priviledges.
	CreateClusterRoleBindings ClusterRoleBindingPolicy = "CreateClusterRoleBindings"

	// NoClusterRoleBindings instructs the MonitoringStack controller to _not_
	// create any ClusterRoleBindings. If the MonitoringStack is configured with
	// a NamespaceSelector, admin users will have to create the appropriate
	// RoleBindings to allow access to the desired namespaces.
	NoClusterRoleBindings ClusterRoleBindingPolicy = "NoClusterRoleBindings"
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

	// CreateClusterRoleBindings for a Monitoring Stack Resource
	// If a NamespaceSelector is given, the controller can create a
	// ClusterRoleBinding for the Prometheus and Alertmanager
	// ServiceAccounts. This allows the
	// ServiceAccount access to all namespaces, allowing the
	// ServiceDiscovery to work across namespaces out of the
	// box. However by impersonating this ServiceAccount a user could elevate
	// their access in unintended ways.
	// To avoid this set CreateClusterRoleBindings to NoClusterRoleBindings. Note
	// that admins must create the needed namespaced RoleBindings manually
	// so that endpoint discovery works as expected.
	// +kubebuilder:default="CreateClusterRoleBindings"
	// +optional
	CreateClusterRoleBindings ClusterRoleBindingPolicy `json:"createClusterRoleBindings,omitempty"`

	// Time duration to retain data for. Default is '120h',
	// and must match the regular expression `[0-9]+(ms|s|m|h|d|w|y)` (milliseconds seconds minutes hours days weeks years).
	// +kubebuilder:default="120h"
	Retention monv1.Duration `json:"retention,omitempty"`

	// Define resources requests and limits for Monitoring Stack Pods.
	// +optional
	// +kubebuilder:default={requests:{cpu: "100m", memory: "256Mi"}, limits:{memory: "512Mi", cpu: "500m"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Define tolerations for Monitoring Stack Pods.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Define node selector for Monitoring Stack Pods.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

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
	Conditions []Condition `json:"conditions"`
}

type ConditionStatus string

// +required
// +kubebuilder:validation:Required
// +kubebuilder:validation:Pattern=`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$`
// +kubebuilder:validation:MaxLength=316
type ConditionType string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"

	ReconciledCondition        ConditionType = "Reconciled"
	AvailableCondition         ConditionType = "Available"
	ResourceDiscoveryCondition ConditionType = "ResourceDiscovery"
)

type Condition struct {
	// type of condition in CamelCase or in foo.example.com/CamelCase.
	// The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt)
	Type ConditionType `json:"type"`
	// observedGeneration represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// lastTransitionTime is the last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// reason contains a programmatic identifier indicating the reason for the condition's last transition.
	// Producers of specific condition types may define expected values and meanings for this field,
	// and whether the values are considered a guaranteed API.
	// The value should be a CamelCase string.
	// This field may not be empty.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$`
	Reason string `json:"reason"`
	// message is a human readable message indicating details about the transition.
	// This may be an empty string.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message"`
	// status of the condition
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=True;False;Unknown;Degraded
	Status ConditionStatus `json:"status"`
}

func (c Condition) Equal(n Condition) bool {
	if c.Reason == n.Reason && c.Status == n.Status && c.Message == n.Message && c.ObservedGeneration == n.ObservedGeneration {
		return true
	}
	return false
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
	// Configure TLS options for the Prometheus web server.
	// +optional
	WebTLSConfig *WebTLSConfig `json:"webTLSConfig,omitempty"`
}

type AlertmanagerConfig struct {
	// Disables the deployment of Alertmanager.
	// +optional
	// +kubebuilder:default=false
	Disabled bool `json:"disabled,omitempty"`
	// Configure TLS options for the Alertmanager web server.
	// +optional
	WebTLSConfig *WebTLSConfig `json:"webTLSConfig,omitempty"`
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
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource
// +kubebuilder:subresource:status
// +kubebuilder:metadata:annotations="observability.openshift.io/api-support=TechPreview"
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
	// selector is the label selector used to select MonitoringStack
	// resources.
	//
	// By default, all resources are matched.
	Selector metav1.LabelSelector `json:"selector"`

	// namespaceSelector defines in which namespaces the MonitoringStack
	// resources are discovered from.
	//
	// By default, resources are only discovered in the current namespace.
	NamespaceSelector NamespaceSelector `json:"namespaceSelector,omitempty"`

	// replicaLabels is the list of labels used to deduplicate the data between
	// highly-available replicas.
	//
	// Thanos Querier is always configured with `prometheus_replica` as replica
	// label.
	// +optional
	ReplicaLabels []string `json:"replicaLabels,omitempty"`

	// webTLSConfig configures the TLS options for the Thanos web server.
	// +optional
	WebTLSConfig *WebTLSConfig `json:"webTLSConfig,omitempty"`
}

// ThanosQuerierStatus defines the observed state of ThanosQuerier.
// It should always be reconstructable from the state of the cluster and/or outside world.
type ThanosQuerierStatus struct{}

// SecretKeySelector selects a key of a secret.
type SecretKeySelector struct {
	// The name of the secret in the object's namespace to select from.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// The key of the secret to select from.  Must be a valid secret key.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// WebTLSConfig contains configuration to enable TLS on web endpoints.
type WebTLSConfig struct {
	// Reference to the TLS private key for the web server.
	// +kubebuilder:validation:Required
	PrivateKey SecretKeySelector `json:"privateKey"`
	// Reference to the TLS public certificate for the web server.
	// +kubebuilder:validation:Required
	Certificate SecretKeySelector `json:"certificate"`
	// Reference to the root Certificate Authority used to verify the web server's certificate.
	// +kubebuilder:validation:Required
	CertificateAuthority SecretKeySelector `json:"certificateAuthority"`
}

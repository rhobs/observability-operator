// +groupName=observability.openshift.io
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins,verbs=list;get;watch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins/status;uiplugins/finalizers,verbs=get;update

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UIPlugin defines an observability console plugin.
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:validation:XValidation:rule="self.spec.type != 'Logging' || self.metadata.name == 'logging'",message="UIPlugin name must be 'logging' if type is Logging"
// +kubebuilder:validation:XValidation:rule="self.spec.type != 'TroubleshootingPanel' || self.metadata.name == 'troubleshooting-panel'",message="UIPlugin name must be 'troubleshooting-panel' if type is TroubleshootingPanel"
// +kubebuilder:validation:XValidation:rule="self.spec.type != 'DistributedTracing' || self.metadata.name == 'distributed-tracing'",message="UIPlugin name must be 'distributed-tracing' if type is DistributedTracing"
// +kubebuilder:validation:XValidation:rule="self.spec.type != 'Dashboards' || self.metadata.name == 'dashboards'",message="UIPlugin name must be 'dashboards' if type is Dashboards"
// +kubebuilder:validation:XValidation:rule="self.spec.type != 'Monitoring' || self.metadata.name == 'monitoring'",message="UIPlugin name must be 'monitoring' if type is Monitoring"
type UIPlugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UIPluginSpec   `json:"spec,omitempty"`
	Status UIPluginStatus `json:"status,omitempty"`
}

// UIPluginList contains a list of UIPlugin
// +kubebuilder:resource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UIPluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UIPlugin `json:"items"`
}

// +kubebuilder:validation:Enum=Dashboards;TroubleshootingPanel;DistributedTracing;Logging;Monitoring
type UIPluginType string

const (
	// TypeDashboards deploys the Dashboards Dynamic Plugin for OpenShift Console.
	TypeDashboards UIPluginType = "Dashboards"
	// DistributedTracing deploys the Distributed Tracing Dynamic Plugin for the OpenShift Console
	TypeDistributedTracing UIPluginType = "DistributedTracing"
	// TroubleshootingPanel deploys the Troubleshooting Panel Dynamic Plugin for the OpenShift Console
	TypeTroubleshootingPanel UIPluginType = "TroubleshootingPanel"
	// Monitoring deploys the Monitoring Plugin for the OpenShift Console
	TypeMonitoring UIPluginType = "Monitoring"

	// TypeLogging deploys the Logging View Plugin for OpenShift Console.
	TypeLogging UIPluginType = "Logging"
)

// DeploymentConfig contains options allowing the customization of the deployment hosting the UI Plugin.
type DeploymentConfig struct {
	// Define a label-selector for nodes which the Pods should be scheduled on.
	//
	// When no selector is specified it will default to a value only selecting Linux nodes ("kubernetes.io/os=linux").
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:nodeSelector"}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Define the tolerations used for the deployment.
	//
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Pod Tolerations",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Toleration"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// TroubleshootingPanelConfig contains options for configuring the Troubleshooting Panel  plugin
type TroubleshootingPanelConfig struct {
	// Timeout is the maximum duration before a query timeout.
	//
	// The value is expected to be a sequence of digits followed by a unit suffix, which can be 's' (seconds)
	// or 'm' (minutes).
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OCP Console Query Timeout",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:ocpConsoleTimeout"}
	// +kubebuilder:validation:Pattern:="^([0-9]+)([sm]{1})$"
	Timeout string `json:"timeout,omitempty"`
}

// DistributedTracingConfig contains options for configuring the Distributed Tracing plugin
type DistributedTracingConfig struct {
	// Timeout is the maximum duration before a query timeout.
	//
	// The value is expected to be a sequence of digits followed by a unit suffix, which can be 's' (seconds)
	// or 'm' (minutes).
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OCP Console Query Timeout",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:ocpConsoleTimeout"}
	// +kubebuilder:validation:Pattern:="^([0-9]+)([sm]{1})$"
	Timeout string `json:"timeout,omitempty"`
}

// LoggingConfig contains options for configuring the logging console plugin.
type LoggingConfig struct {
	// LokiStack points to the LokiStack instance of which logs should be displayed.
	// It always references a LokiStack in the "openshift-logging" namespace.
	//
	// +kubebuilder:validation:Required
	LokiStack LokiStackReference `json:"lokiStack"`

	// LogsLimit is the max number of entries returned for a query.
	//
	// +kubebuilder:validation:Minimum=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OCP Console Log Limit",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:ocpConsoleLogLimit"}
	LogsLimit int32 `json:"logsLimit,omitempty"`

	// Timeout is the maximum duration before a query timeout.
	//
	// The value is expected to be a sequence of digits followed by an optional unit suffix, which can be 's' (seconds)
	// or 'm' (minutes). If the unit is omitted, it defaults to seconds.
	//
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OCP Console Query Timeout",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:ocpConsoleTimeout"}
	// +kubebuilder:validation:Pattern:="^([0-9]+)([sm]{0,1})$"
	Timeout string `json:"timeout,omitempty"`
}

// LokiStackReference is used to configure a reference to a LokiStack that should be used
// by the Logging console plugin.
//
// Currently, always points to a LokiStack resource in the "openshift-logging" namespace.
//
// +structType=atomic
type LokiStackReference struct {
	// Name of the LokiStack resource.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength:=1
	Name string `json:"name"`
}

// MonitoringConfig contains options for configuring the monitoring console plugin.
type MonitoringConfig struct {
	// Alertmanager points to the alertmanager instance of which it should create a proxy to.
	//
	// +kubebuilder:validation:Optional
	Alertmanager AlertmanagerReference `json:"alertmanager,omitempty"`

	// ThanosQuerier points to the thanos-querier service of which it should create a proxy to.
	//
	// +kubebuilder:validation:Optional
	ThanosQuerier ThanosQuerierReference `json:"thanosQuerier,omitempty"`

	// Perses points to the perses instance service of which it should create a proxy to.
	//
	// +kubebuilder:validation:Optional
	Perses PersesReference `json:"perses,omitempty"`
}

// Alertmanager is used to configure a reference to a alertmanage that should be used
// by the monitoring console plugin.
//
// +structType=atomic
type AlertmanagerReference struct {
	// Url of the Alertmanager to proxy to.
	//
	// +kubebuilder:validation:Optional
	Url string `json:"url,omitempty"`
}

// ThanosQuerier is used to configure a reference to a thanos-querier service that should be used
// by the monitoring console plugin.
//
// +structType=atomic
type ThanosQuerierReference struct {
	// Url of the ThanosQuerier to proxy to.
	//
	// +kubebuilder:validation:Optional
	Url string `json:"url,omitempty"`
}

// Perses is used to configure a reference to a perses service that should be used
// by the monitoring console plugin.
//
// +structType=atomic
type PersesReference struct {
	// Name of the Perses Service to proxy to.
	//
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// Namespace of the Perses Service to proxy to.
	//
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

// UIPluginSpec is the specification for desired state of UIPlugin.
//
// +kubebuilder:validation:XValidation:rule="self.type == 'TroubleshootingPanel' || !has(self.troubleshootingPanel)", message="Troubleshooting Panel configuration is only supported with the TroubleshootingPanel type"
// +kubebuilder:validation:XValidation:rule="self.type == 'DistributedTracing' || !has(self.distributedTracing)", message="Distributed Tracing configuration is only supported with the DistributedTracing type"
// +kubebuilder:validation:XValidation:rule="self.type != 'Logging' || has(self.logging)", message="Logging configuration is required if type is Logging"
type UIPluginSpec struct {
	// Type defines the UI plugin.
	// +required
	// +kubebuilder:validation:Required
	Type UIPluginType `json:"type"`

	// Deployment allows customizing aspects of the generated deployment hosting the UI Plugin.
	//
	// +kubebuilder:validation:Optional
	Deployment *DeploymentConfig `json:"deployment,omitempty"`

	// TroubleshootingPanel contains configuration for the troubleshooting console plugin.
	//
	// +kubebuilder:validation:Optional
	TroubleshootingPanel *TroubleshootingPanelConfig `json:"troubleshootingPanel,omitempty"`

	// DistributedTracing contains configuration for the distributed tracing console plugin.
	//
	// +kubebuilder:validation:Optional
	DistributedTracing *DistributedTracingConfig `json:"distributedTracing,omitempty"`

	// Logging contains configuration for the logging console plugin.
	//
	// It only applies to UIPlugin Type: Logging.
	//
	// +kubebuilder:validation:Optional
	Logging *LoggingConfig `json:"logging,omitempty"`

	// Monitoring contains configuration for the monitoring console plugin.
	//
	// +kubebuilder:validation:Optional
	Monitoring *MonitoringConfig `json:"monitoring,omitempty"`
}

// UIPluginStatus defines the observed state of UIPlugin.
// It should always be reconstructable from the state of the cluster and/or outside world.
type UIPluginStatus struct {
	// Conditions provide status information about the plugin.
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

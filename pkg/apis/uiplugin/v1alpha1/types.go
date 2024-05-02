// +groupName=observability.openshift.io
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins,verbs=list;get;watch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins/status;uiplugins/finalizers,verbs=get;update

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rhobs/observability-operator/pkg/apis/shared"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UIPlugin defines an observability console plugin.
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
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

// +kubebuilder:validation:Enum=Dashboards
type UIPluginType string

const (
	// Dashboards deploys the Dashboards Dynamic Plugin for OpenShift Console.
	TypeDashboards UIPluginType = "Dashboards"
)

// UIPluginSpec is the specification for desired state of UIPlugin.
type UIPluginSpec struct {
	// Type defines the UI plugin.
	// +required
	// +kubebuilder:validation:Required
	Type UIPluginType `json:"type"`
}

// UIPluginStatus defines the observed state of UIPlugin.
// It should always be reconstructable from the state of the cluster and/or outside world.
type UIPluginStatus struct {
	// Conditions provide status information about the plugin.
	// +listType=atomic
	Conditions []shared.Condition `json:"conditions"`
}

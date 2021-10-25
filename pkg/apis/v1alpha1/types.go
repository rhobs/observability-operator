// +groupName=monitoring.rhobs
// +kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=list;get;watch
// +kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks/status;monitoringstacks/finalizers,verbs=get;update

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

type MonitoringStackSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS -- desired state of cluster
}

// MonitoringStackStatus defines the observed state of MonitoringStack.
// It should always be reconstructable from the state of the cluster and/or outside world.
type MonitoringStackStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS -- observed state of cluster
}

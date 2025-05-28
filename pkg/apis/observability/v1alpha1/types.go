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
}

// ClusterObservabilityStatus defines the observed state of ClusterObservability.
type ClusterObservabilityStatus struct{}

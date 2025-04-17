// +kubebuilder:object:generate=true
// +groupName=observability.openshift.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "observability.openshift.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion, &ClusterObservability{}, &ClusterObservabilityList{})
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}

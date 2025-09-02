/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package v1alpha1 contains API Schema definitions for the rhobs v1alpha1 API group
//
// The observability-operator API module uses semantic versioning for version tags,
// but does not guarantee backward compatibility, even for versions v1.0.0 and above.
// Breaking changes may occur without major version bumps.
//
// +kubebuilder:object:generate=true
// +groupName=monitoring.rhobs
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "monitoring.rhobs", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion, &MonitoringStack{}, &MonitoringStackList{}, &ThanosQuerier{}, &ThanosQuerierList{})
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}

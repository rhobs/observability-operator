//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapabilitiesSpec) DeepCopyInto(out *CapabilitiesSpec) {
	*out = *in
	out.Tracing = in.Tracing
	in.OpenTelemetry.DeepCopyInto(&out.OpenTelemetry)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapabilitiesSpec.
func (in *CapabilitiesSpec) DeepCopy() *CapabilitiesSpec {
	if in == nil {
		return nil
	}
	out := new(CapabilitiesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterObservability) DeepCopyInto(out *ClusterObservability) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterObservability.
func (in *ClusterObservability) DeepCopy() *ClusterObservability {
	if in == nil {
		return nil
	}
	out := new(ClusterObservability)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterObservability) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterObservabilityList) DeepCopyInto(out *ClusterObservabilityList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterObservability, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterObservabilityList.
func (in *ClusterObservabilityList) DeepCopy() *ClusterObservabilityList {
	if in == nil {
		return nil
	}
	out := new(ClusterObservabilityList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterObservabilityList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterObservabilitySpec) DeepCopyInto(out *ClusterObservabilitySpec) {
	*out = *in
	out.Storage = in.Storage
	if in.Capabilities != nil {
		in, out := &in.Capabilities, &out.Capabilities
		*out = new(CapabilitiesSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterObservabilitySpec.
func (in *ClusterObservabilitySpec) DeepCopy() *ClusterObservabilitySpec {
	if in == nil {
		return nil
	}
	out := new(ClusterObservabilitySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterObservabilityStatus) DeepCopyInto(out *ClusterObservabilityStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterObservabilityStatus.
func (in *ClusterObservabilityStatus) DeepCopy() *ClusterObservabilityStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterObservabilityStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CommonCapabilitiesSpec) DeepCopyInto(out *CommonCapabilitiesSpec) {
	*out = *in
	out.Operators = in.Operators
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CommonCapabilitiesSpec.
func (in *CommonCapabilitiesSpec) DeepCopy() *CommonCapabilitiesSpec {
	if in == nil {
		return nil
	}
	out := new(CommonCapabilitiesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OTLPExporter) DeepCopyInto(out *OTLPExporter) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OTLPExporter.
func (in *OTLPExporter) DeepCopy() *OTLPExporter {
	if in == nil {
		return nil
	}
	out := new(OTLPExporter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenTelemetrySpec) DeepCopyInto(out *OpenTelemetrySpec) {
	*out = *in
	out.CommonCapabilitiesSpec = in.CommonCapabilitiesSpec
	if in.Exporter != nil {
		in, out := &in.Exporter, &out.Exporter
		*out = new(OTLPExporter)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenTelemetrySpec.
func (in *OpenTelemetrySpec) DeepCopy() *OpenTelemetrySpec {
	if in == nil {
		return nil
	}
	out := new(OpenTelemetrySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorsSpec) DeepCopyInto(out *OperatorsSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorsSpec.
func (in *OperatorsSpec) DeepCopy() *OperatorsSpec {
	if in == nil {
		return nil
	}
	out := new(OperatorsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretSpec) DeepCopyInto(out *SecretSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretSpec.
func (in *SecretSpec) DeepCopy() *SecretSpec {
	if in == nil {
		return nil
	}
	out := new(SecretSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageSpec) DeepCopyInto(out *StorageSpec) {
	*out = *in
	out.Secret = in.Secret
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageSpec.
func (in *StorageSpec) DeepCopy() *StorageSpec {
	if in == nil {
		return nil
	}
	out := new(StorageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TracingSpec) DeepCopyInto(out *TracingSpec) {
	*out = *in
	out.CommonCapabilitiesSpec = in.CommonCapabilitiesSpec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TracingSpec.
func (in *TracingSpec) DeepCopy() *TracingSpec {
	if in == nil {
		return nil
	}
	out := new(TracingSpec)
	in.DeepCopyInto(out)
	return out
}

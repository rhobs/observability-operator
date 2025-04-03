// +groupName=observability.openshift.io
// +kubebuilder:rbac:groups=observability.openshift.io,resources=signalmanagers,verbs=list;get;watch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=signalmanagers/status;signalmanagers/finalizers,verbs=get;update
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FIXME review naming: SignalManager, Pattern
// FIXME: review cluster scope, validation

// SignalManager is a custom resource to enable observability in the cluster.
//
// Each type of observability signal (logs, metrics, network events, ...) requires operators to be
// installed and resources created to configure collection, processing, and storage of signal data.
//
// The SignalManager automatically installs the operators, custom resource definitions, and
// resources to enable all the desired observability signals in a cluster with default
// configurations.
//
// This means you can get observability up and running quickly and easily,
// but still customize the details if and when you need to.
//
// ## Pattern
//
// A "Pattern" is a named set of configurations for each of the observability signals.
// Choosing a pattern automatically installs required operators (if needed) _and_ creates
// working resources so you have complete, working, observability stacks.
//
// The following patterns are always available, others may be made available.
//
//   - Default:
//     Installs operators and resources suitable for the most common use cases.
//     The operator owns and manages the resources, and keeps them in the default state.
//   - Custom:
//     Installs operators, but does not create any live resources.
//     The user can create customized resources, they will not be modified by this operator.
//   - Disabled: Do not install any operators, resource definitions, or resources.
//
// Custom patterns can be defined in `spec.patterns`.
//
// ## Examples
//
// Enable all observability components with default settings.
//
//	kind: SignalManager
//	spec:
//	  pattern: Default
//
// Disable all observability components except for logging.
//
//	kind: SignalManager
//	spec:
//	  pattern: Disabled
//	  signals:
//	    name: Log
//	    pattern: Default
//
// Enable most components with defaults, install the logging operators,
// but use custom logging resources (created separately)
//
//	kind: SignalManager
//	spec:
//	  pattern: Default
//	  signals:
//	    name: log
//	    pattern: Custom
//
// ## Lifecycle and ownership
//
// Ownership of resources depends on the pattern:
//
//   - None: No operators installed, no resources created or reconciled.
//   - Custom: Operators installed but no resources created. User is free to create resources
//     they are not owned or reconciled by this operator.
//   - Default, or any other defined configuration:
//     This operator creates, owns, and reconciles resources to keep them consistent with the chosen
//     pattern.
//
// FIXME: Operator may reconcile only part of the resource and allow user to tweak other parts.
// Needs consideration. COO already uses server-side-apply to do this in some cases.
//
// FIXME: Patterns may need to be "parameterized" e.g. with sizing data.
// How to include such parameters without duplicating existing CRs?
//
// FIXME: Define behavior on spec changes: deleting, re-creating, updating resources.
// Change to Custom should leave resources in place so user can eddit.
// What to do on change _from_ Custom?
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
type SignalManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Lists signals and the pattern to deploy them.
	Spec SignalManagerSpec `json:"spec,omitempty"`

	// Status of the signal manager.
	Status SignalManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:resource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SignalManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SignalManager `json:"items"`
}

type SignalManagerSpec struct {

	// The default pattern for signals that are not listed or have no `pattern` field.
	Pattern Pattern `json:"pattern"`

	// Signals is a list of signal types with the desired pattern.
	Signals []SignalSpec `json:"signals,omitempty"`

	// Patterns is a list of custom pattern definitions.
	Patterns []PatternSpec `json:"patterns,omitempty"`
}

type SignalSpec struct {
	// Signal name
	Name Signal `json:"name"`

	// Pattern for this signal. Optional, if absent use the 'Default' pattern.
	Pattern Pattern `json:"pattern,omitempty"`

	// Namespace to install to.
	//
	// Optional, each signal type has a default namespace.
	Namespace string `json:"namespace,omitempty"`
}

// Signal is the name of a signal type.
// TODO complete the list
type Signal string

const (
	SignalLog     Signal = "Log"
	SignalTrace          = "Trace"
	SignalMetric         = "Metric"
	SignalNetflow        = "Netflow"
)

// Pattern is the name of a deployment pattern. The following patterns are always available:
//   - Default: Install operators and custom resources to enable the signal with a default configuration
//     intended to be suitable for most single-cluster use cases.
//     The operator owns and manages the resources, and keeps them in the default state.
//   - Custom: Install operators and custom resource definitions, but do not create any resources.
//     The user can create customized resources, they will not be managed or modified by this operator.
//   - Disabled: Do not install operators, resource definitions, or resources.
//
// Custom patterns can also be defined in spec.patterns.
// The operator owns and manages the resources, and keeps them in the state specified by the pattern.
type Pattern string

const (
	PatternDefault          = "Default"
	PatternCustom           = "Custom"
	PatternDisabled Pattern = "Disabled"
)

// PatternSpec defines a custom pattern.
//
// TODO a pattern is a bundle of YAML resources. Need to define how they are stored
// on the cluster. Simplest format is a flat YAML file, but we may need more structure
// to store kustomize scripts, multi-stage deployments, health checks, metadata etc....
// Possible storage formats: ConfigMap, PersistentVolume, container image...
//
// Patterns should also be usable directly, without depending on this API.
// Preferably using only kubectl and kustomize.
type PatternSpec struct {
	// Name of the pattern.
	Name Pattern `json:"pattern"`
}

// SignalManagerStatus TODO
// - Conditions to track status of each pattern - pull condition info from signal operators?
type SignalManagerStatus struct{}

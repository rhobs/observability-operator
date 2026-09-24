// Package capability defines the internal contract and lifecycle helpers for
// ObservabilityInstaller capabilities.
package capability

import (
	"context"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

// Definition defines how to reconcile a single capability on one installer.
type Definition struct {
	// Name identifies the capability in diagnostics, e.g. "logging" or "tracing".
	Name string
	// Plan returns owned and desired resources for this capability.
	Plan func(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) (Plan, error)
	// Shared returns all resources that could be shared with other installers.
	// For example, Tempo remains installed as long as any installer has tracing enabled.
	// It must return its resources when called on an empty or disabled installer,
	// and must not read API or credentials.
	//
	Shared func(*obsv1alpha1.ObservabilityInstaller) Plan
	// UpdateStatus returns updated status and requests retries when needed.
	UpdateStatus func(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) StatusResult
	// WatchTypes lists operand types that trigger reconciliation.
	WatchTypes []client.Object
}

// Plan describes operator and operand lifecycle for a capability.
type Plan struct {
	// Operators lists the capability's OLM requirements.
	Operators []OperatorRequirement
	// Desired contains resources to reconcile. It is a subset of Owned.
	Desired []client.Object
	// Owned is the complete cleanup inventory. An owned resource absent from
	// Desired, matched by GVK, namespace, and name, is deleted.
	// This allows disabled capabilities to remove stale resources.
	Owned []client.Object
}

// OperatorRequirement describes an OLM operator used by a capability.
type OperatorRequirement struct {
	// Name identifies the requirement in diagnostics, e.g. "loki" or "cluster-logging"
	Name string
	// Subscription is the subscription COO creates when the operator is needed.
	Subscription *olmv1alpha1.Subscription
	// Desired reports whether any installer currently needs the operator.
	Desired bool
	// EquivalentPackages lists alternative OLM package names that satisfy the same operator
	// requirement. If one is present, COO will use it instead of installing the operator.
	EquivalentPackages []string
}

// NewOperatorRequirement builds an operator requirement from config.
func NewOperatorRequirement(name string, config OperatorConfig) OperatorRequirement {
	return OperatorRequirement{
		Name:               name,
		Subscription:       Subscription(config),
		EquivalentPackages: config.EquivalentPackages,
	}
}

// StatusResult reports the outcome of status evaluation.
type StatusResult struct {
	// RequeueAfter requests another reconciliation after this interval.
	RequeueAfter time.Duration
	// Err reports a status evaluation failure.
	Err error
}

// DesiredOperators sets the desired state of each operator requirement.
func DesiredOperators(desired bool, operators ...OperatorRequirement) []OperatorRequirement {
	for i := range operators {
		operators[i].Desired = desired
	}
	return operators
}

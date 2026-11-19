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

// Capability defines how the ObservabilityInstaller controller reconciles one
// observability capability, such as tracing or logging.
//
// A capability describes resources; the controller decides what to apply and
// what to delete. Each kind of resource comes as a pair: an inventory of
// everything the capability may create, and the desired subset. Inventory
// entries that are not desired are deleted, matched by GVK, namespace and name,
// so disabling a capability or a feature of it removes stale resources. The
// controller reports an error if a desired resource is not in its inventory.
// An invalid shared plan aborts reconciliation; a failed per-installer plan
// defers cleanup for that capability and all shared resources.
//
// Reconciling one installer, the "current" installer, proceeds as follows:
//
//  1. Shared state is aggregated over all installers that are not being
//     deleted, including the current one: [Capability.SharedInventory],
//     [Capability.SharedDesired], [Capability.Operators] and
//     [Capability.OperatorsDesired].
//  2. Per-installer state is computed for the current installer only:
//     [Capability.Inventory], and [Capability.Desired] unless the installer is
//     being deleted.
//  3. Desired operators are subscribed, unless an equivalent package is
//     already installed. Desired resources are applied with server-side apply,
//     owned by the current installer where Kubernetes allows. Undesired
//     inventory entries are deleted, as are COO-managed subscriptions and CSVs
//     for operators that no installer needs.
//  4. Watches start for [Capability.WatchTypes] whose API group is served.
//  5. The finalizer is removed if the installer is being deleted.
//  6. [Capability.UpdateStatus] runs for each capability and the status is
//     written.
//
// Every reconcile of every installer re-evaluates the shared state, so all
// methods are called often and must be cheap and deterministic. Only Desired
// and UpdateStatus may read the API.
type Capability interface {
	// Name identifies the capability in errors and logs, e.g. "logging" or
	// "tracing". It must be unique among capabilities.
	Name() string

	// Inventory lists every per-installer resource the capability may create
	// for the given installer, whatever its spec. Resources here but not in
	// Desired are deleted, so it is the cleanup list for a disabled
	// capability, a disabled feature, and a deleted installer.
	//
	// It is called on every reconcile, including while the installer is being
	// deleted. Only identity (GVK, namespace and name) is used for matching;
	// the spec is ignored, so objects may omit it. It takes no reader so
	// that cleanup never depends on referenced resources, such as credentials,
	// that may be missing.
	Inventory(*obsv1alpha1.ObservabilityInstaller) []client.Object

	// Desired returns the per-installer resources to create or update for
	// the given installer's current spec. It returns nothing when the
	// capability is disabled. Each resource must be in Inventory.
	//
	// It is called on every reconcile, but not while the installer is being
	// deleted. The reader reads directly from the API server, so referenced
	// resources such as credentials need not be cached. An error prevents
	// applies and deletes for this capability's per-installer inventory, but
	// other capabilities continue. The error is reported and retried with
	// backoff.
	Desired(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) ([]client.Object, error)

	// SharedInventory lists every resource the capability may share between
	// installers, such as a cluster-scoped UIPlugin. Shared resources not
	// desired by any installer are deleted only when all per-installer plans
	// succeed. It must not depend on any installer, so the list is complete
	// even when no installer exists. Only identity is used.
	SharedInventory() []client.Object

	// SharedDesired returns the shared resources the given installer needs.
	// Each must be in SharedInventory.
	//
	// It is called for every installer that is not being deleted, on every
	// reconcile of any installer, so it must not read the API. A shared
	// resource is desired while any installer requests it; when several do,
	// the first one listed provides its spec, so requests should agree.
	SharedDesired(*obsv1alpha1.ObservabilityInstaller) []client.Object

	// Operators lists every OLM operator the capability may install. It must
	// not depend on any installer. Requirements with the same subscription
	// in different capabilities are merged: the operator is installed while
	// any capability needs it.
	Operators() []OperatorRequirement

	// OperatorsDesired reports whether the given installer needs all of the
	// capability's operators. It is true when the capability is enabled, and
	// may also be true when only operator installation is requested.
	//
	// It is called for every installer that is not being deleted, on every
	// reconcile of any installer, so it must not read the API. Operators are
	// installed while any installer needs them; when none does, COO removes
	// the subscriptions and CSVs it created.
	OperatorsDesired(*obsv1alpha1.ObservabilityInstaller) bool

	// UpdateStatus sets the capability's fields in the given installer's
	// status from the state of its operands, and clears them when the
	// capability is disabled. It is called on every reconcile after
	// resources are applied, including when applying failed; the controller
	// writes the status afterwards. It must not modify status fields owned by
	// the controller, such as Conditions, or by other capabilities.
	//
	// The reader is the controller's cache, so it can read WatchTypes.
	// Missing operands are normal while operators install: use
	// [StatusReadError] to request a retry. The controller requeues after the
	// shortest RequeueAfter of all capabilities, and the first Err sets the
	// Reconciled condition to false.
	UpdateStatus(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) StatusResult

	// WatchTypes lists operand types whose changes trigger reconciliation,
	// typically those read by UpdateStatus. It is read once at startup. A
	// watch starts when its API group is first served, usually after the
	// capability's operator installs its CRDs. A change to any watched object
	// reconciles every installer.
	WatchTypes() []client.Object
}

// OperatorRequirement describes an OLM operator used by a capability.
type OperatorRequirement struct {
	// Name identifies the requirement in diagnostics, e.g. "loki" or "cluster-logging"
	Name string
	// Subscription is the subscription COO creates when the operator is needed.
	Subscription *olmv1alpha1.Subscription
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

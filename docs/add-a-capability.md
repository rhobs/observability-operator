# Add an ObservabilityInstaller capability

This is an internal pattern for the `ObservabilityInstaller` controller, not a
plugin system for other COO controllers. Use
[`logging/`](../pkg/controllers/observability/logging/) for a multi-operand
example and [`tracing/`](../pkg/controllers/observability/tracing/) for Tempo
and OpenTelemetry.

## Review map

Read the typed API in `pkg/apis/observability/v1alpha1/`, then
[`capability/`](../pkg/controllers/observability/capability/) (the contract),
[`capabilities.go`](../pkg/controllers/observability/capabilities.go)
(cross-installer aggregation), and
[`reconcilers.go`](../pkg/controllers/observability/reconcilers.go) (lifecycle).
[`watch_registry.go`](../pkg/controllers/observability/watch_registry.go)
registers watches when operand CRDs become available. Review `tracing/` for
the migration and `logging/` plus
[`storage/`](../pkg/controllers/observability/storage/) for new behavior.
CRDs, deepcopy code, bundle manifests, and `docs/api.md` are generated; review
their sources first and verify with `make generate`.

The tracing migration qualifies two ClusterRole and ClusterRoleBinding names
with the installer namespace. Their previous names remain in tracing's cleanup
inventory; the tracing plan tests cover this behavior.

## Lifecycle contract

Each implementation lives in `pkg/controllers/observability/<component>/` and
exposes `New(Options) capability.Definition`. Keep typed operand builders in
that package. The parent imports implementations, never the reverse. Register
the constructor in `capabilities.go`; avoid capability-specific branches in
central reconciliation.

| Definition field | Purpose |
| --- | --- |
| `Name` | Stable name for diagnostics |
| `Plan(ctx, installer, reader)` | Per-installer resources and cleanup inventory |
| `Shared(installer)` | Cross-installer requirements; no API or credential reads |
| `UpdateStatus(ctx, installer, reader)` | Capability-specific status and retry |
| `WatchTypes` | Operand kinds to watch when available |

A `Plan` has two distinct resource sets:

- `Owned`: complete cleanup inventory, even if disabled or deleting. Anything
  absent from `Desired` is deleted.
- `Desired`: resources applied while this installer is enabled and not deleting.

| Installer state | Per-installer `Desired` | `Owned` | Shared operators | UIPlugin (`Shared`) |
| --- | --- | --- | --- | --- |
| Enabled | Operands | Cleanup inventory | Requested | Applied |
| Disabled or absent | Empty | Cleanup inventory | Only if operators-only requested | Deleted if no other installer wants it |
| Deleting | Empty | Cleanup inventory | Not requested by this installer | Not requested by this installer |

`aggregateShared` combines requests from non-deleting installers: operators and
shared operands remain desired while **any** installer needs them. When none
do, COO deletes its unused subscriptions and current CSVs, and its UIPlugins.
Aggregation deduplicates resources by GVK, namespace, and name, keeping the
first; it cannot merge conflicting configurations for one shared identity.
A cluster-scoped plugin configured per installer, such as logging's reference
to a LokiStack, therefore follows the first installer in list order. All
installers compute the same aggregate, so they apply the same plugin instead
of overwriting each other.

## Implement

1. **Extend the typed API** in `pkg/apis/observability/v1alpha1/`. Embed
   `CommonCapabilitiesSpec`, add a pointer to `CapabilitiesSpec` with a
   nil-safe getter, define validation/defaults and any status fields. Reuse
   `ObjectStorageSpec` where appropriate. The API is a separate Go module.
   Add API tests, then run `make generate`; do not hand-edit generated files.

2. **Build the per-installer plan.** Always return `Owned` for cleanup; only
   populate `Desired` when enabled and not deleting. Include TypeMeta and
   correct namespace/name: identity is GVK + namespace + name. Namespace-qualify
   installer-specific cluster-scoped resource names to avoid collisions. If
   renaming an existing resource, include its old identity in `Owned` but not
   `Desired` (as tracing does for collector RBAC).

   Inventory must not depend on source credentials. For storage, call
   `storage.Inventory` unconditionally and `storage.Materialize` only when
   active; append `Resources.Objects()` to `Desired`. Keep provider-specific
   mappings in typed builders.

3. **Declare shared requirements.** `Shared` must work with an empty installer
   (used to obtain cleanup inventory when there are no users), and must not
   read the API or credentials. Build OLM requirements with
   `capability.NewOperatorRequirement`; use
   `capability.DesiredOperators(enabled || installOperators, operators...)`
   for `Plan.Operators`. `EquivalentPackages` identify external packages that
   also satisfy a requirement. A deleting installer requests nothing. If there
   are shared operands, put their inventory in `Owned` and put them in
   `Desired` only while enabled. Both current capabilities do this with their
   UIPlugin. A shared object is applied by every installer that wants it, so
   label it as part of the operator rather than of an installer, see
   `tracing.newUIPlugin`.

4. **Implement status and watches.** Clear capability status fields first so
   disabling cannot leave stale values. Return an empty `StatusResult` when
   absent, disabled, or deleting. Use `StatusReadError` for operand reads:
   NotFound requests a short retry without an error condition; other failures
   also return an error. List typed operand objects in `WatchTypes`; the parent
   registers available kinds and retries failed registrations. Do not add
   per-capability watch flags to the parent.

   Status must reflect what is actually checked: logging's `Status.LokiStack`
   means the LokiStack exists, **not** that it is ready; `Status.Logging`
   reflects ClusterLogForwarder's Ready condition. Tracing reports operand
   versions, not readiness.

5. **Wire the capability.** Register its constructor in `capabilities.go`,
   passing only its operator configuration. For new operators, add options to
   the parent and configure them in `pkg/operator/operator.go`. Register
   operand API types in `pkg/operator/scheme.go` and add required kubebuilder
   RBAC markers to `observability_controller.go`. ObservabilityInstaller stays
   behind the OpenShift feature gate. For a new console plugin or image,
   update the UIPlugin compatibility matrix.

The generic reconciler handles creation, apply, and deletion; don't add a
capability-specific lifecycle branch. It matches subscriptions by exact
`Subscription.Spec.Package`, not name prefix. An externally managed
subscription to the configured package or an equivalent package prevents COO
from installing its own; COO deletes only its own unused subscriptions.

## Test and regenerate

Cover enabled, disabled, absent, operators-only, and deleting installers;
inventory without credential reads; provider mappings and TLS; names across
namespaces; empty-installer shared inventory; status and watches. Test
cross-installer aggregation, external subscriptions, and finalization in the
parent package. Add E2E coverage where practical.

```bash
make generate
make operator
make test-unit
make lint
(cd pkg/apis && go test ./...)
```

Commit generated API, CRD, RBAC, and documentation changes. E2E tests require
a cluster; see [developer documentation](developer.md).

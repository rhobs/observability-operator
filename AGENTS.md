# AGENTS.md

AI agent guidance for the Cluster Observability Operator (COO).

## What this is

A Kubernetes operator for OpenShift managing observability stacks via CRDs,
built on controller-runtime, distributed via OLM.

CRDs (all `v1alpha1`):
- **MonitoringStack** / **ThanosQuerier** (`monitoring.rhobs`, `pkg/apis/monitoring/`) — Prometheus, Alertmanager, Thanos
- **UIPlugin** (`observability.openshift.io`, `pkg/apis/uiplugin/`) — OpenShift Console dynamic plugins
- **ObservabilityInstaller** (`observability.openshift.io`, `pkg/apis/observability/`) — installs Tempo/OTEL operators via OLM

API types live in `pkg/apis/` — a **separate Go module** (`pkg/apis/go.mod`).

Prometheus types come from the Red Hat fork `github.com/rhobs/obo-prometheus-operator`
(API group `monitoring.rhobs`, not upstream `monitoring.coreos.com`).

## Commands

```
make test-unit          # go test -cover ./cmd/... ./pkg/...
make lint-golang        # golangci-lint (includes --fix)
make generate           # CRDs, deepcopy, RBAC, kustomize, docs
make operator           # generate + build
make bundle             # OLM bundle
go test ./pkg/controllers/uiplugin/...   # single-package tests
```

Always run `make test-unit && make lint-golang` after changes.
Run `make generate` after touching `pkg/apis/`, RBAC markers, or kustomize overlays — CI fails on dirty tree.

## Layout

- `cmd/operator/` — entrypoint
- `pkg/apis/` — CRD types (separate Go module)
- `pkg/controllers/` — one dir per CRD group, plus `util/` (shared helpers) and `operator/` (manager startup)
- `pkg/reconciler/` — reconcile abstraction (`Updater`/`Deleter`/`CreateOrUpdater`)
- `pkg/generator/` — offline manifest generator sharing reconciler code
- `config/` — embedded kustomize bases and components for UIPlugin and ObservabilityInstaller (see `config/README.md`)
- `pkg/overlay/` — `Overlay` type: in-memory kustomize runner (`krusty`), returns `[]client.Object`
- `deploy/` — kustomize bases/overlays for the operator itself (`crds/`, `operator/`, `olm/`, `perses/`, `package-operator/`, etc.)
- `test/e2e/` — end-to-end tests (require a cluster)
- `bundle/` — OLM bundle (generated, committed)
- `tmp/` — gitignored build output; `tmp/bin/` holds dev tools (`make tools`)

## Code conventions

**Imports** (enforced by gci): stdlib → third-party → `github.com/rhobs/observability-operator/...`, blank line between groups.

**Forbidden imports** (depguard): `github.com/pkg/errors`, `golang.org/x/exp/slices`.

**Server-side apply**: all managed resources use `client.Apply` with `ForceOwnership` and `FieldOwner("observability-operator")`.

**Labels**: every managed resource needs `app.kubernetes.io/managed-by: observability-operator` via `util.AddCommonLabels` — the manager cache filters on these.

**Reconcile pattern**: compose `Updater`/`Deleter`/`NewOptional(Unmanaged)Updater` from `pkg/reconciler/`. Don't hand-roll owner ref management.

**Error handling**: wrap with `fmt.Errorf("...: %w", err)`. Either log **or** return — never both. Use `client.IgnoreNotFound`, `apierrors.IsNotFound/IsAlreadyExists/IsConflict`.

**RBAC**: `+kubebuilder:rbac` markers on controller files → `make generate` regenerates the cluster role.

**OpenShift gate**: `--openshift.enabled` flag gates UIPlugin/ObservabilityInstaller/operator controllers. See `FeatureGates.OpenShift.Enabled` in `pkg/operator/operator.go`.

**Kustomize overlays**: UIPlugin and ObservabilityInstaller operands are defined as embedded kustomize bases in `config/` and assembled at runtime via `pkg/overlay/overlay.go`. Hard-coded values live in base manifests; dynamic fields from the CR spec are applied as overlay patches. Each controller has a `BuildOverlay()` function (`pkg/controllers/uiplugin/overlay.go`, `pkg/controllers/observability/overlay.go`). Exceptions that remain as Go builders: Perses/Accelerator/APM dashboards, korrel8r config (Go template), Tempo secrets. See `config/README.md` for full details.

## Testing

- Unit tests: `*_test.go` alongside code, table-driven, using `gotest.tools/v3` (`assert`, `golden`) and `testify`.
- Golden files: `pkg/controllers/**/testdata`; update with `-test.update` flag.
- E2E: `test/e2e/` with custom `framework.Framework`; needs a cluster (`hack/setup-e2e-env.sh`).

## Generated files — do not hand-edit

`zz_generated.deepcopy.go`, `deploy/crds/`, `deploy/operator/*-cluster-role.yaml`, `deploy/olm/`, `bundle/`, `docs/api.md`.

## Commits

Conventional Commits enforced by commitlint: `feat:`, `fix:`, `chore:`, `build:`, `test:`, `docs:`, etc.
Sign commits: `git commit -s`.

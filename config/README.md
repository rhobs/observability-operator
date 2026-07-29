# Kustomize Config

Base manifests and kustomize components used by both the operator and the CLI generator.
The operator reconciles live resources; the generator writes equivalent YAML to stdout.

## Directory Layout

```
observabilityinstaller/
  base/                          Base kustomization
  components/tracing/
    collector/
      resources/                 OpenTelemetryCollector, RBAC
      operator/                  OLM Subscription
    store/
      resources/                 TempoStack
      operator/                  OLM Subscription

uiplugins/components/
  consoleplugin/                 ConsolePlugin CRD variants
    v1/                          v4.19+
    rhobs-v1/                    v4.17–v4.19
    v1alpha1/                    pre-v4.17
  dashboards/resources/          Dashboard plugin operands
  distributed-tracing/resources/ Distributed tracing plugin operands
  logging/resources/             Logging plugin operands
  monitoring/
    resources/                   Monitoring plugin operands
    health-analyzer/             Cluster health analyzer operands
    perses/                      Perses RBAC
  troubleshooting-panel/
    resources/                   Troubleshooting panel operands
    korrel8r/                    Korrel8r operands

fs.go                            Embeds the above via //go:embed
```

Each plugin component has a `resources/` sub-directory with its core operands.
`operator/` sub-directories (installer only) contain OLM Subscriptions.

## Namespace Conventions

- **Instance namespace** (from the ObservabilityInstaller CR's `metadata.namespace`):
  all ObservabilityInstaller operands under `resources/` directories.
- **COO namespace** (where the observability-operator runs):
  UIPlugin operands (UIPlugin is cluster-scoped) and all Subscriptions under `operator/` directories.

Resources without an explicit namespace in YAML get the instance namespace via `SetNamespace`.
Subscriptions have a placeholder namespace replaced with the COO namespace at build time.

## Implementation

An in-memory kustomize overlay is built from embedded base manifests plus dynamically generated
patches and replacements, using `sigs.k8s.io/kustomize/api/krusty`.

Key files:
- `pkg/overlay/overlay.go` -- `Overlay` type: populates an in-memory filesystem, runs kustomize, returns `[]client.Object`.
- `pkg/controllers/observability/overlay.go` -- `BuildOverlay()` for ObservabilityInstaller.
- `pkg/controllers/observability/tracing_overlay.go` -- tracing components (OTEL collector, TempoStack, Subscriptions).
- `pkg/controllers/uiplugin/overlay.go` -- `BuildOverlay()` for UIPlugin operands.

Hard-coded values live in base manifests; the overlay patches dynamic fields from the CR spec.
Secrets are the exception -- assembled in Go from cluster reads or input files.

## Scope

These manifests and builder paths cover **UIPlugin** operands and the
**Tracing** capability of `ObservabilityInstaller`. The following remain Go
builders by design (see `docs/design/kustomize-overlay-refactor.md`, "Scope
exceptions (step 4, documented)") and are *not* generated from `config/`:

- Perses dashboards (Perses SDK builders in `pkg/controllers/uiplugin`).
- Accelerator and APM dashboards.
- The korrel8r config (`pkg/controllers/uiplugin/config/korrel8r.yaml`, a Go
  template because Loki/Tempo service names vary per cluster).
- Tempo secrets (`pkg/controllers/observability/tempo_components.go`).

`MonitoringStack`/`ThanosQuerier` controllers are untouched by this refactor.

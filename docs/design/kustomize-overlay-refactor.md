# Kustomize Overlay Refactor

## Goal

Replace Go-struct resource builders with a kustomize overlay built in memory
from static manifests under `config/`. The operator reconciles the overlay
output; a CLI generator emits the same YAML offline. Both share the same build
code so they cannot drift.

## Architecture

```
config/ (embedded static manifests)
      + components  (kustomize components selected per CR)
      + patches     (dynamic fields from the CR spec)
      + resources   (inline Go-built edge cases)
      + replacements (placeholders -> CR-derived names)
      -> kustomize build (krusty, in-memory FS)
      -> namespace transform
      -> common labels
      -> []client.Object / YAML
```

### Key files

- `pkg/overlay/overlay.go` -- shared `Overlay` type: `AddComponent`,
  `AddPatch`, `AddResource`, `ReplaceValue`, in-memory kustomize build,
  namespace transform (`UnsetOnly`), common-label injection, `Build()` ->
  `[]client.Object`.
- `config/` -- embedded manifest trees (see `config/README.md`).
- `pkg/controllers/observability/overlay.go` -- `BuildOverlay` for
  ObservabilityInstaller; `ResolveInstallerObjects` shared by operator and
  generator.
- `pkg/controllers/observability/tracing_overlay.go` -- OTel collector,
  TempoStack and Subscription patches.
- `pkg/controllers/uiplugin/overlay.go` -- `BuildOverlay` for
  UIPlugin operands; `ResolvePluginObjects` shared by operator and generator.
- `pkg/generator/` -- offline CLI that reads CRs from YAML and produces the
  same objects via the shared resolve functions.
- `pkg/images/images.go` -- default image map shared by operator and
  generator.

### Label policy

Base YAML in `config/` carries only labels needed to anchor kustomize
replacements (`app.kubernetes.io/instance`). Common labels (`managed-by`,
`part-of`, `name`) are applied by `addCommonLabels` at build time.

## Scope

This refactor covers **UIPlugin** operands and the **Tracing** capability of
`ObservabilityInstaller`. `MonitoringStack`/`ThanosQuerier` controllers are
unchanged.

### Scope exceptions

The following stay as Go builders by design:

- **Perses/Accelerator/APM dashboards** -- built from the Perses SDK; no
  static equivalent without a code generator.
- **Korrel8r config** (`config/korrel8r.yaml`) -- a Go template because
  Loki/Tempo service names vary per cluster.
- **Tempo secrets** (`tempo_components.go`) -- assembled from source secrets
  read at reconcile time.

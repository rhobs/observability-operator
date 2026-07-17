# Kustomization for operator and generator

This project provides a traditional `operator` and a command line `generator`.

The generator takes operator CRs and produces operands as YAML files instead of live resources,
using the same reconcile code as the reconcile operator.
Generator output can be applied with `kubectl` without using the operator.

Both operator and generator use kustomize overlays to edit base manifests and create operand resources.
See [Implementation](#implementation).

You can deploy the generated YAML directly in situations where you don't want the operator.
- Generate once, apply many times to  multiple clusters.
- Apply to constrained edge nodes that don't run the operator.
- Build one-time install wizards, assisted installers that don't use an operator.
- Use other YAML-based tools to further manipulate and deploy the YAML.

Domain experts can contribute to the resources without understanding the controller code.

Users can create their own modified kustomization overlays to be used with or without the operator.

## Directory layout 

- `observabilityinstaller/`: Base operands for the `ObservabilityInstaller`.
  - `base/`: Base kustomization
  - `components/`: Optional components in the observability stack, with their operators.
    - `collectors/`: Resources that collect observability data
      - `tracing/`: OpenTelemetryCollector, RBAC
    - `stores/`: Resources that store observability data
      - `tempostack/`: TempoStack, RBAC
- `uiplugins/`: Base operands for the `UIPlugin`.
  - `components/`: Output resources for each `UIPlugin.type`
    - `dashboards/`: Dashboard console plugin operands
    - `distributed-tracing/`: Distributed tracing console plugin operands
    - `logging/`: Logging console plugin operands
    - `troubleshooting-panel/`: Troubleshooting panel console plugin operands (includes korrel8r)

- `fs.go`: Embeds observabilityinstaller and uiplugins directories

**NOTE**: Each component directory has sub-directories:
- `resources/` (required) — Operands (e.g. TempoStack, ClusterLogForwarder, RBAC, secrets, configmaps).
- `operator/` (optional) — Subscription for OLM to install the required operator for operands in `resources/`.

## Placeholder values

Some special placeholder values are replaced automatically during reconcile.

- `name: placeholder-observability-installer-name`: Name derived from the ObservabilityInstaller name.
- `namespace: placeholder-observability-installer-namespace`: The ObservabilityInstaller namespace.

Resources with explicit (non-placeholder) names and namespaces are not modified.

## Implementation

The Go controller and generator build an in-memory kustomize overlay using `sigs.k8s.io/kustomize/api/krusty`.
The Go controller applies the overlay to reconcile live resources, the generator writes the resources to stdout.

Key files:

- `config/fs.go` — embeds observabilityinstaller and uiplugins kustomize directories using `//go:embed`.
- `pkg/overlay/overlay.go` — `Overlay` type that populates an
  in-memory `filesys.FileSystem` with both embedded filesystems plus generated patches
  and resources, then runs `krusty.MakeKustomizer().Run()` to produce `[]client.Object`.
- `pkg/controllers/observability/build_overlay.go` — `BuildOverlay()` entry point that
  selectively adds components based on the `ObservabilityInstaller` spec.
- `pkg/controllers/observability/tracing_overlay.go` — adds tracing components
  (OTEL collector, TempoStack, Subscriptions) with dynamic patches.

The generated overlay is equivalent to:

```
generated/
├── kustomization.yaml               # base + components + patches
├── patches/
│   ├── tempostack.yaml              # dynamic storage config (type, credentials, TLS)
│   ├── opentelemetrycollector.yaml   # dynamic tempo endpoint
│   └── subscription-*.yaml          # startingCSV/channel patches (conditional)
```

Hard-coded values live in the base component manifests.
The overlay patches dynamic fields with values from the `ObservabilityInstaller` spec.
Placeholder values are replaced with real values after the kustomize build.

Secrets are the exception — input secrets are read from the cluster (or from file in the generator)
and output secrets are constructed from the values in Go code.

UIPlugin resources (Deployment, Service, etc.) are not part of the observability overlay.
The observability overlay creates a `UIPlugin` CR, and the UIPlugin controller
(or the generator's `resolveUIPlugins`) handles the associated resources separately.

### Status

Tracing resources are built via kustomize overlay:
- TempoStack, OpenTelemetryCollector, RBAC, UIPlugin CR, Subscriptions
- Operator subscriptions are patched for `startingCSV` and `channel` via overlay
- Secrets remain as Go code (need cluster reads for credential assembly)

The generator (`cmd/generator`) supports the same capabilities as the controller,
including `-k` for kustomize input and `--opentelemetry-csv` / `--tempo-csv` flags for subscription versioning.

## To be done

- For standalone (non-controller) use, add example storage secrets or a
  documented manual step for providing S3/Azure/GCS credentials.
- Scripts to apply with kubectl in 2 phases, operators first, wait for CSVs, then operands.
- Support for logging and network observability
- Common S3 storage configuration for tempostack and lokistack


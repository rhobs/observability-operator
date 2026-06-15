# Declarative kustomize manifests that deploy the observability stack.

The Go controller in `pkg/controllers/observability/` builds a kustomize overlay
at runtime, using these manifests as base components and applying patches for
dynamic values (storage config, TLS, secrets). See [Generated overlay](#generated-overlay).

## Why declarative resources instead of programmed controller structs?

- Kustomize resources can be applied direct by `kubectl` without using the COO. Examples:
  - Generate once, apply many times in a multi-cluster.
  - Apply to constrained edge nodes that don't run COO.
  - One-time install wizards, assisted installers that don't use COO.
- Other teams can contribute resource edits more easily that controller code edits.
- Users can create their own modified versions of the stack and deploy with COO.
  
## Directory layout

```
stack/
├── base/                        # Shared base: target namespace
├── components/
│   ├── collectors/
│   │   ├── logging/             # ClusterLogForwarder + Cluster Logging operator
│   │   ├── network/             # FlowCollector + Network Observability operator
│   │   └── tracing/             # OpenTelemetryCollector + RBAC + Red Hat build of OpenTelemetry operator
│   ├── stores/
│   │   ├── lokistack/           # LokiStack + Loki operator
│   │   └── tempostack/          # TempoStack + RBAC + Tempo operator
│   ├── console/                 # UIPlugin (OpenShift console plugins)
│   ├── korrel8r/                # Korrel8r correlation engine
│   └── s3-storage/              # S3 object storage (minio, ODF, or local)
├── overlays/
│   ├── single-cluster/          # All collectors + stores + minio on one cluster
│   ├── hub/                     # Stores only (receives signals from edge clusters)
│   └── edge/                    # Collectors only (forwards signals to a hub)
```

Each component has two sub-directories:

- `operator/` (optional) — Namespace, OperatorGroup, and Subscription to install the operator via OLM.
- `resources/` — Operand CRs and supporting resources (RBAC, secrets, configmaps).

Overlays compose components and apply namespace transformers and patches.

## Generated overlay

The Go controller builds an in-memory kustomize overlay using
`sigs.k8s.io/kustomize/api/krusty`. The key files:

- `stack/fs.go` — embeds `base/` and `components/` into the binary via `//go:embed`.
- `pkg/controllers/observability/overlay.go` — `Overlay` type that populates an
  in-memory `filesys.FileSystem` with the embedded manifests plus generated patches
  and resources, then runs `krusty.MakeKustomizer().Run()` to produce `[]client.Object`.
- `pkg/controllers/observability/*_components.go` — refactored to add patches to an overlay,
  then generate objects, rather than constructing objects directly.

The generated overlay is equivalent to:

```
generated/
├── kustomization.yaml           # components + patches + resources
├── namespace-transformer.yaml   # sets namespace from instance
├── patches/
│   └── tempostack.yaml          # dynamic storage config (type, credentials, TLS)
└── resources/
    ├── tempo-secret.yaml         # assembled storage credentials
    ├── tempo-tls-secret.yaml     # TLS cert (conditional)
    └── tempo-ca-configmap.yaml   # CA bundle (conditional)
```

Hard-coded values (tenants, gateway, query frontend) live in the base component manifests.
The overlay patches only dynamic fields derived from the `ObservabilityInstaller` spec.

### Status

- **Done**: TempoStack, storage secrets, UIPlugin (via overlay)
- **Not yet converted**: OpenTelemetryCollector, OTEL RBAC (still built as Go objects)
- **Not yet converted**: Operator Subscriptions (still built as Go objects)

## To be done

- Convert OpenTelemetryCollector and its RBAC to the overlay approach
  (currently still built as Go objects in `otel_components.go`).
- Convert operator Subscriptions to the overlay approach.
- For standalone (non-controller) use, add example storage secrets or a
  documented manual step for providing S3/Azure/GCS credentials.
- LokiStack and logging/network components have not been compared against Go code.

## Open questions

- The Go controller installs operators via `Subscription` objects it constructs
  at runtime (with version info from `Options`). The manifest `operator/`
  directories hardcode channel/source. How should operator versions be managed?
- tempostack.yaml has hard coded tenant IDs, this looks incorrect.

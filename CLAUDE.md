# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The Observability Operator is a Kubernetes operator that manages Monitoring/Alerting stacks through Custom Resource Definitions (CRDs). Built on [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime), it relies heavily on a forked version of Prometheus Operator ([rhobs/obo-prometheus-operator](https://github.com/rhobs/obo-prometheus-operator)) for core functionality.

**Language**: Go 1.25.5
**Deployment**: OLM (Operator Lifecycle Manager) based, supports OpenShift and Kubernetes

## Development Setup

```bash
# Install project-specific tools to tmp/bin
make tools

# Set up local Kind cluster with OLM and local registry
./hack/setup-e2e-env.sh

# Delete the Kind cluster when done
kind delete cluster --name obs-operator
```

The `./hack/setup-e2e-env.sh` script is the unified environment setup used by both local development and CI. See `--help` for advanced options.

## Common Commands

```bash
# Build operator binary (also runs 'make generate')
make operator

# Build only (no code generation)
make build

# Run unit tests
make test-unit

# Run E2E tests (requires Kind cluster setup)
./test/run-e2e.sh
# Use --help to see options for rerunning specific tests

# Run specific E2E test
./test/run-e2e.sh --run TestMonitoringStack

# Linting
make lint                  # both Go and shell
make lint-golang          # Go only
make lint-shell           # shell scripts only

# Code generation (CRDs, deepcopy, RBAC, etc.)
make generate

# Generate only CRDs
make generate-crds

# Generate API documentation
make docs
```

## Running the Operator Locally

The typical workflow is to deploy the operator bundle (which includes both observability-operator and prometheus-operator), then scale down the in-cluster deployment and run the operator locally:

```bash
# Build and push operator and bundle to local registry
make operator-image bundle-image operator-push bundle-push \
    IMG_BASE="local-registry:30000/observability-operator" \
    VERSION=0.0.0-dev \
    PUSH_OPTIONS=--tls-verify=false

# Deploy the bundle
./tmp/bin/operator-sdk run bundle \
    local-registry:30000/observability-operator-bundle:0.0.0-dev \
    --install-mode AllNamespaces \
    --namespace operators --skip-tls

# Scale down in-cluster operator
kubectl scale --replicas=0 -n operators deployment/observability-operator

# Run operator locally
go run ./cmd/operator/... --zap-devel --zap-log-level=100 --kubeconfig ~/.kube/config 2>&1 | tee tmp/operator.log
```

## Architecture

### Core Custom Resources

- **MonitoringStack** (`monitoring.rhobs/v1alpha1`): Main resource for creating a monitoring stack (Prometheus + Alertmanager)
- **ThanosQuerier** (`monitoring.rhobs/v1alpha1`): Manages Thanos Querier deployments
- **UIPlugin** (`uiplugin.observability.openshift.io/v1alpha1`): OpenShift Console UI plugins for observability
- **Observability** (`observability.core.rhobs/v1alpha1`): Manages observability infrastructure (OpenTelemetry, Tempo)

### Code Structure

```
cmd/operator/                   # Operator entry point
pkg/
├── apis/                       # API type definitions (Go types)
│   ├── monitoring/v1alpha1/    # MonitoringStack, ThanosQuerier
│   ├── observability/v1alpha1/ # Observability resource
│   └── uiplugin/v1alpha1/      # UIPlugin resource
├── controllers/                # Controller implementations
│   ├── monitoring/
│   │   ├── monitoring-stack/   # MonitoringStack controller
│   │   └── thanos-querier/     # ThanosQuerier controller
│   ├── observability/          # Observability installer controller
│   ├── uiplugin/               # UIPlugin controller
│   ├── operator/               # Meta-controller for operator lifecycle
│   └── util/                   # Shared controller utilities
├── operator/                   # Operator setup and configuration
├── reconciler/                 # Reusable reconciler patterns
└── assets/                     # Embedded static resources

deploy/
├── crds/                       # Generated CRD manifests
├── operator/                   # Operator RBAC and deployment
├── dependencies/               # Prometheus Operator and admission webhooks
└── samples/                    # Example CRs
```

### Controller Pattern

Controllers use a reconciler pattern with:
- **Reconcile loop**: Watches CRs and reconciles desired state
- **Components**: Each controller breaks down resources into logical components
- **Create/Update reconcilers**: Generic reconcilers in `pkg/reconciler/` handle resource creation and updates

### Forked Prometheus Operator

The operator uses `github.com/rhobs/obo-prometheus-operator` instead of upstream prometheus-operator. This allows it to run alongside the upstream operator without conflicts. When updating:
1. Update version in `go.mod` and `deploy/dependencies/kustomization.yaml`
2. Run `make generate` to regenerate CRDs
3. See [docs/developer.md](docs/developer.md) for full update process

## Code Generation

The project uses [controller-gen](https://github.com/kubernetes-sigs/controller-tools) with kubebuilder markers in Go types to generate:
- Kubernetes CRDs
- ClusterRole RBAC manifests
- DeepCopy methods
- API documentation

**Important**: When modifying API types in `pkg/apis/`, always run `make generate` to regenerate manifests. The markers (e.g., `+kubebuilder:validation:...`, `+kubebuilder:rbac:...`) control what gets generated.

See [kubebuilder marker documentation](https://book.kubebuilder.io/reference/markers.html) for available markers.

## Testing

- **Unit tests**: Located alongside source files with `_test.go` suffix
- **E2E tests**: In `test/e2e/`, test full operator functionality against a Kind cluster
- **Test framework**: Uses standard Go testing with `testify` for assertions

Run single unit test:
```bash
go test -v ./pkg/controllers/monitoring/monitoring-stack -run TestSpecificFunction
```

## Commit Conventions

Follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` - new feature (minor version bump)
- `fix:` - bug fix (patch version bump)
- `BREAKING CHANGE:` - breaking change (major version bump)
- Other types: `build:`, `chore:`, `ci:`, `docs:`, `style:`, `refactor:`, `perf:`, `test:`

## Release Process

Releases follow [SemVer 2.0.0](https://semver.org/) and are automated based on conventional commits:

```bash
git checkout main
git pull
git checkout -b cut-new-release
make initiate-release       # Auto-generates version from commits
# Or force a specific version:
make initiate-release-as RELEASE_VERSION=1.4.0
```

This updates `CHANGELOG.md` and `VERSION` files. After PR approval and merge, CI creates a pre-release. The final manual step is unchecking "Set as a pre-release" in GitHub to trigger stable channel publication.

## Container Runtime

The build system auto-detects container runtime (prefers `podman`, falls back to `docker`). Override with:
```bash
CONTAINER_RUNTIME=docker make operator-image
```

## OpenShift Features

The operator has OpenShift-specific features controlled by feature gates:
- UIPlugin controller (OpenShift Console plugins)
- Serving certificate integration
- Additional OpenShift-specific resources

These are toggled via `FeatureGates.OpenShift.Enabled` in the operator configuration.

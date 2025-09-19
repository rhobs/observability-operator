# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Development
- `make tools` - Install all required project dependencies to `tmp/bin`
- `make test-unit` - Run unit tests for cmd/ and pkg/ directories
- `make lint` - Run both golang and shell linting
- `make lint-golang` - Run golangci-lint with auto-fix
- `make generate` - Generate CRDs, deepcopy, kustomize configs, and docs

### Building
- `make operator` - Generate manifests and build operator binary (default target)
- `make build` - Build operator binary to `./tmp/operator`
- `make operator-image` - Build operator container image
- `go run ./cmd/operator/... --zap-devel --zap-log-level=100 --kubeconfig ~/.kube/config` - Run operator locally

### Testing
- `make test-e2e` - Run end-to-end tests
- `./test/run-e2e.sh` - Run E2E tests against local kind cluster
- `./hack/setup-e2e-env.sh` - Setup local development environment with kind cluster

### Environment Setup
1. `make tools` - Install project tools
2. `./hack/setup-e2e-env.sh` - Setup complete development environment

## Architecture Overview

The Observability Operator is a Kubernetes operator that manages monitoring/alerting stacks through CRDs, built on controller-runtime.

### Key Components

**Core APIs** (pkg/apis/):
- `monitoring.rhobs/v1alpha1` - MonitoringStack CRD for complete monitoring stacks
- `observability.rhobs/v1alpha1` - Core observability APIs including OpenTelemetry/tracing
- `uiplugin.rhobs/v1alpha1` - OpenShift console UI plugin integration

**Controllers** (pkg/controllers/):
- `monitoring/monitoring-stack/` - Manages Prometheus, Alertmanager, and monitoring components
- `monitoring/thanos-querier/` - Handles Thanos querier deployments
- `uiplugin/` - Manages OpenShift console plugins and UI components
- `operator/` - Core operator lifecycle management

**Key Dependencies**:
- Uses forked prometheus-operator (`github.com/rhobs/obo-prometheus-operator`) for compatibility
- Integrates with OpenShift APIs for console UI plugins
- Built on controller-runtime v0.21.0

### Deployment Structure

- `deploy/crds/` - CRD manifests (common + kubernetes-specific)
- `deploy/dependencies/` - Required dependency resources
- `deploy/operator/` - Operator deployment manifests
- `deploy/olm/` - OLM bundle configuration

## Development Notes

- Requires OLM (Operator Lifecycle Manager) for proper operation
- Generate manifests with `make generate` after modifying Go types in `pkg/apis/`
- Use conventional commits for automatic changelog/release management
- The operator can run locally while dependencies (prometheus-operator) run in-cluster

## Testing Specific Files

To run tests for a specific package:
```bash
go test ./pkg/controllers/monitoring/monitoring-stack/...
go test ./pkg/controllers/uiplugin/...
```
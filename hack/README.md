# E2E Test Environment Setup

This directory contains scripts for setting up the end-to-end (e2e) test environment.

## Unified Setup Script

**`setup-e2e-env.sh`** - The main script for setting up e2e test environments. This script unifies the setup process used by both local development and CI environments to prevent config drift. 

**Prerequisites**: Run `make tools` first to install project-specific tools (operator-sdk, oc, etc.).

### Key Features

- **Unified Setup**: Same script used locally and in GitHub Actions
- **Flexible Configuration**: Options to control what gets installed/configured
- **Tool Management**: Can install kind, kubectl, and any system packages via package managers
- **Environment Validation**: Checks prerequisites before proceeding
- **CI-Friendly**: Special options for CI environments (skip /etc/hosts checks, etc.)

### Usage Examples

```bash
# First, install project tools
make tools

# Full setup with defaults (local development)
./hack/setup-e2e-env.sh

# CI-friendly setup (skip host checks, use specific versions)
./hack/setup-e2e-env.sh --skip-host-check --kind-version v0.23.0

# Only validate prerequisites, don't install anything
./hack/setup-e2e-env.sh --validate-only

# Install tools but don't create cluster (useful for rebuilding)
./hack/setup-e2e-env.sh --no-cluster

# Install additional packages (any system packages)
./hack/setup-e2e-env.sh curl jq tree htop

# Custom cluster configuration
./hack/setup-e2e-env.sh --cluster-name my-test --kind-image kindest/node:v1.25.0
```

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--help` | Show usage information | |
| `--validate-only` | Only validate prerequisites | |
| `--no-kind` | Skip kind installation | |
| `--no-kubectl` | Skip kubectl installation | |
| `--no-cluster` | Skip cluster setup | |
| `--no-olm` | Skip OLM installation | |
| `--no-registry` | Skip local registry setup | |
| `--skip-host-check` | Skip /etc/hosts validation (for CI) | |
| `--cluster-name NAME` | Kind cluster name | `obs-operator` |
| `--kind-version VERSION` | Kind version to install | `v0.23.0` |
| `--kind-image IMAGE` | Kind node image | `kindest/node:v1.24.0` |
| `--kubeconfig PATH` | Kubeconfig file path | `~/.kube/kind/obs-operator` |


### What Gets Set Up

The script sets up a complete e2e test environment including:

1. **Tool Installation** (if needed):
   - kind (Kubernetes in Docker)
   - kubectl (Kubernetes CLI)
   - Any additional system packages via package managers (apt-get, dnf, yum, zypper, pacman, brew, apk)
   - **Note**: Project tools (operator-sdk, oc, etc.) must be installed via `make tools`

2. **Kind Cluster**:
   - Creates cluster with configuration from `hack/kind/config.yaml`
   - Labels control-plane as infra node
   - Waits for cluster to be ready

3. **Cluster Components**:
   - OLM (Operator Lifecycle Manager) v0.28.0
   - Local Docker registry for testing
   - Monitoring CRDs

4. **Validation**:
   - Prerequisite checks (go, git, curl)
   - Host configuration validation
   - Cluster health verification

## Backward Compatibility

The old `hack/kind/setup.sh` script is now deprecated but still works - it forwards to the new unified script with appropriate options.

## CI Integration

GitHub Actions use this same script via `.github/e2e-tests-olm/action.yaml`. Note that CI environments install project tools via the tools-cache action before running the setup script:

```yaml
- name: Install required tools using unified setup
  uses: ./.github/tools-cache

- name: Set up e2e environment
  shell: bash
  run: |
    ./hack/setup-e2e-env.sh \
      --skip-host-check \
      --kind-version ${{ inputs.kind-version }} \
      --kind-image ${{ inputs.kind-image }}
```

This ensures both local development and CI use identical setup procedures, preventing config drift and test result differences. 
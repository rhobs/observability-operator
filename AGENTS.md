# Agent Guidelines — Observability Operator

Guidelines for AI coding assistants working on this project. This file
deliberately avoids information that ages (versions, directory listings, image
tags). Consult the authoritative sources referenced below instead.

## Quick reference

| What                 | Where / command                                     |
| -------------------- | --------------------------------------------------- |
| Go version           | `go.mod` (`go` directive)                           |
| Build                | `make operator`                                     |
| Unit tests           | `make test-unit`                                    |
| E2E tests            | `make test-e2e` (requires a running cluster)        |
| Lint                 | `make lint` (Go + shell)                            |
| Code generation      | `make generate`                                     |
| Install dev tools    | `make tools` (installs to `tmp/bin/`)               |
| Container runtime    | `podman` preferred, `docker` fallback               |
| All Makefile targets | `make help` or read `Makefile` and `Makefile.tools` |

## Project overview

The Cluster Observability Operator (COO) is a Kubernetes operator for
OpenShift that manages observability components. It is built with
controller-runtime and distributed via OLM.

Key custom resources (all `v1alpha1`):

- **MonitoringStack** (`monitoring.rhobs`) — deploys Prometheus,
  Alertmanager, and Thanos sidecar.
- **ThanosQuerier** (`monitoring.rhobs`) — federates queries across
  MonitoringStacks.
- **UIPlugin** (`observability.openshift.io`, cluster-scoped) — deploys
  OpenShift Console dynamic plugins (logging, tracing,
  troubleshooting panel, monitoring).
- **ObservabilityInstaller** (`observability.openshift.io`) — installs
  Tempo and OpenTelemetry operators via OLM subscriptions.

API types live in `pkg/apis/` which is a **separate Go module**
(`pkg/apis/go.mod`) so downstream consumers can import types without the
full operator dependency tree.

## Forked prometheus-operator

The operator depends on `github.com/rhobs/obo-prometheus-operator`, a
Red Hat fork of prometheus-operator. CRDs from this fork use the
`monitoring.rhobs` API group (not upstream `monitoring.coreos.com`).
When working with Prometheus, Alertmanager, or ServiceMonitor types,
import from `github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1`.

## Code conventions

### Imports

Imports are grouped and ordered by `gci` (enforced via `.golangci.yml`):

1. Standard library
2. Third-party
3. `github.com/rhobs/observability-operator` (project-local)

### Error handling

Either log the error **or** return it — never both. This was explicitly
cleaned up in PR #1148. The project treats "log and return" as a bug.

### Reconciliation pattern

All controllers use **server-side apply** via controller-runtime's
`client.Apply`:

```go
client.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("observability-operator"))
```

Managed resources carry the label
`app.kubernetes.io/managed-by: observability-operator`.

The generic reconciler abstraction lives in `pkg/reconciler/`. Use the
existing `Updater`, `Deleter`, and `CreateOrUpdater` types rather than
hand-rolling owner reference management.

### RBAC

RBAC is generated from `+kubebuilder:rbac` markers on controller files.
After modifying markers, run `make generate` to regenerate
`deploy/operator/observability-operator-cluster-role.yaml`.

### OpenShift feature gate

Functionality that requires OpenShift APIs (UIPlugins,
ObservabilityInstaller, operator self-monitoring, TLS via service CA) is
gated behind the `--openshift.enabled` CLI flag. The corresponding
struct path is `FeatureGates.OpenShift.Enabled` in
`pkg/operator/operator.go`. When adding OpenShift-only controllers or
features, gate them the same way.

### UIPlugin compatibility matrix

UIPlugin images and support levels are managed through the compatibility
matrix in `pkg/controllers/uiplugin/compatibility_matrix.go`. When
adding or updating a console plugin, update the matrix entries. Tests
for the matrix are in `compatibility_matrix_test.go`.

## Commit conventions

Commits follow [Conventional Commits](https://www.conventionalcommits.org/)
enforced by `commitlint` (config: `commitlint.config.mjs`). CI rejects
non-conforming messages.

Common prefixes observed in this project:

| Prefix   | Use for                           |
| -------- | --------------------------------- |
| `feat:`  | New features or capabilities      |
| `fix:`   | Bug fixes                         |
| `chore:` | Maintenance, dep bumps, refactors |
| `test:`  | Test-only changes                 |
| `docs:`  | Documentation                     |
| `build:` | Build system, CI                  |
| `api:`   | API type changes                  |

PR titles often carry a Jira ticket prefix, e.g.
`COO-1234: fix: description` or `NO-JIRA: chore: description`.
Use `NO-JIRA:` or `NO-ISSUE:` when there is no associated ticket.

Sign commits: `git commit -s`.

## Testing

### Unit tests

- Located alongside production code (`_test.go` in the same package).
- Table-driven tests with `tt := []struct{...}{...}` are the norm.
- Assertion libraries: `gotest.tools/v3/assert` (primary),
  `gotest.tools/v3/golden` for golden-file tests, and
  `github.com/stretchr/testify` in some places.
- `errcheck` is disabled for test files (see `.golangci.yml`).

### E2E tests

- Located in `test/e2e/`, package `e2e`.
- Use a custom `framework.Framework` (in `test/e2e/framework/`) wrapping
  a controller-runtime client.
- Test functions follow the pattern: a top-level `TestXxxController`
  function with subtests driven by `testCase` structs.
- Polling-based assertions (e.g. `AssertResourceEventuallyExists`,
  `AssertStatefulsetReady`, `AssertPromQLResult`) with configurable
  timeouts.
- `DumpOnFailure` captures diagnostics only when a test fails; register
  it **before** `CleanUp` so resources still exist when dumping.
- E2E requires a running cluster. For local development use
  `hack/setup-e2e-env.sh` to create a kind cluster with OLM, then
  `test/run-e2e.sh` to build, deploy, and test.
- OpenShift-specific E2E: `test/run-e2e-ocp.sh`.

### Generated code verification

CI runs `make --always-make generate bundle && git diff --exit-code` to
ensure generated files are committed. Always run `make generate` after
modifying API types, RBAC markers, or kustomize overlays.

## CI

GitHub Actions (see `.github/workflows/`). PR checks include:

1. Commit message lint (conventional commits)
2. GitHub Actions YAML lint
3. Unit tests + Go lint + shell lint
4. Generated code verification
5. Bundle image build
6. E2E tests on a kind cluster with OLM

Merging requires the `lgtm` and `approved` labels. The project uses
Prow-style OWNERS files for review assignment (see `OWNERS`,
`OWNERS_ALIASES`, and per-directory `OWNERS` files under `pkg/`).

## Code ownership

Review and approval routing is handled by OWNERS files. Key areas:

- `pkg/controllers/uiplugin/` — `ui` team
- `pkg/controllers/monitoring/`, `pkg/apis/monitoring/` — `mon` team
- `pkg/apis/observability/` — `cluster-obs` team
- Root `OWNERS` — `maintainers` alias

See `OWNERS_ALIASES` for team membership.

## Release and backport process

- Development targets `main`.
- Releases are cut to `release-X.Y` branches.
- Backports to release branches use PRs with titles prefixed
  `[release-X.Y]`, e.g. `[release-1.5] COO-1234: fix: description`.
- Release images (operator, bundle, catalog) are published to Quay via
  GitHub Actions workflows (`release.yaml`, `release-branch.yaml`).
- The current version is tracked in the `VERSION` file at the repo root.

## Adding a new feature — typical workflow

1. Define or extend API types in `pkg/apis/<group>/v1alpha1/types.go`.
2. Run `make generate` to regenerate deepcopy, CRDs, RBAC, docs.
3. Implement the controller logic in `pkg/controllers/`.
4. Add unit tests (table-driven, same package).
5. Add E2E tests in `test/e2e/`.
6. Run `make test-unit && make lint`.
7. Commit with a conventional commit message, signed off.
8. Open a PR — CI runs the full suite.

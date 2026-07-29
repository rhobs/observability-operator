# AGENTS.md

Guidance for AI agents (and humans) working in this repository.
Read `README.md` and `docs/developer.md` for project-level docs; this file
focuses on the conventions and constraints that matter when editing code.

## What this project is

`observability-operator` (ObO) is a Kubernetes operator that manages
Monitoring/Alerting stacks through CRDs. It is built on
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) and
delegates Prometheus/Alertmanager handling to a **forked** version of
Prometheus Operator (`github.com/rhobs/obo-prometheus-operator`).

The operator reconciles these user-facing CRDs:
- `MonitoringStack` and `ThanosQuerier` (`pkg/apis/monitoring/v1alpha1`, api-group `monitoring.rhobs`)
- `UIPlugin` (`pkg/apis/uiplugin/v1alpha1`)
- `ObservabilityInstaller` (`pkg/apis/observability/v1alpha1`, OpenShift only)

## Repository layout

- `cmd/operator/` – the operator entrypoint (`main.go`)
- `pkg/apis/` – CRD Go types; **separate nested Go module** (`go.mod` is its own)
- `pkg/controllers/` – controllers, one dir per CRD
- `pkg/controllers/util/` – shared helpers (common labels, etc.)
- `pkg/reconciler/` – the core reconcile-abstraction (`Updater`/`Deleter`/optional variants)
- `pkg/operator/` – manager/controller registration and startup
- `pkg/assets/` – embedded YAML assets
- `deploy/` – kustomize bases/overlays: `crds/`, `dependencies/`, `operator/`, `olm/`, `perses/`, `package-operator/`
- `test/e2e/` – end-to-end tests (run against a cluster, not `go test` alone)
- `docs/` – `developer.md`, `api.md` (generated), `design/`, `user-guides/`
- `hack/` – dev/e2e environment scripts
- `bundle/` – OLM bundle (generated)
- `tmp/` – build output and tool binaries (gitignored); `tmp/bin` holds dev tools

## Commands

All dev tools are installed to `tmp/bin` (see `Makefile.tools`). Install them with:

```sh
make tools
```

- Build the operator: `make build` (default goal is `operator` = `generate build`)
- Run unit tests: `make test-unit` (`go test -cover ./cmd/... ./pkg/...`)
- Run a single package's tests: `go test ./pkg/controllers/uiplugin/...`
- Lint Go: `make lint-golang` (uses `tmp/bin/golangci-lint`, includes `--fix`)
- Lint shell: `make lint-shell`
- Regenerate manifests/code: `make generate` (CRDs, deepcopy, RBAC, kustomize, package resources, docs)
- Generate the OLM bundle: `make bundle`
- E2E (needs a kind/OCP cluster): `./test/run-e2e.sh` (env setup: `./hack/setup-e2e-env.sh`)
- Run the operator locally: `go run ./cmd/operator/... --zap-devel --kubeconfig ~/.kube/config`

Always run `make test-unit` and `make lint-golang` after making changes. Also run
`make generate` and check `git diff` when you touch `pkg/apis` or controller RBAC
markers, since CI fails on a dirty tree (`make generate bundle && git diff --exit-code`).

## Code conventions

- **Import grouping** (enforced by gci): standard → third-party → `github.com/rhobs/observability-operator/...`, with a blank line between groups. `gofmt` sorts within groups. Run `make lint-golang` to auto-fix.
- **Forbidden imports** (via depguard): `github.com/pkg/errors` (use stdlib `errors`/`fmt`), `golang.org/x/exp/slices` (use `slices`).
- **RBAC** is declared with kubebuilder markers (`//+kubebuilder:rbac:...`) next to the controller that needs it; run `make generate` to regenerate `deploy/operator/observability-operator-cluster-role.yaml`.
- **Labels**: every managed resource must carry common labels (`app.kubernetes.io/managed-by: observability-operator`, etc.) via `util.AddCommonLabels`. The manager's cache uses a default label selector on these labels — controllers that create resources without them silently break caching. Exceptions to the label selector are declared explicitly in `pkg/operator/operator.go`.
- **Resources are applied server-side**: the reconciler `Updater` patches with `client.Apply`, `ForceOwnership`, and `FieldOwner("observability-operator")`. Don't switch these to plain `Create/Update`.
- **Reconcile abstraction**: component reconcilers implement the `Reconciler` interface in `pkg/reconciler` and are composed into a list; the controller iterates them. Prefer composing `Updater`/`Deleter`/`NewOptional(Unmanaged)Updater` over ad-hoc per-component logic. `NewUnmanagedUpdater` skips the controller-owner reference (e.g. when another operator owns the object, like Perses).
- **Error handling**: wrap errors with `fmt.Errorf("...: %w", err)`, include the resource's `namespace/name (GVK)` for upstream errors (see `pkg/reconciler`). Use `client.IgnoreNotFound`, `apierrors.IsNotFound/IsAlreadyExists/IsConflict` rather than raw string matches.
- License headers follow `hack/boilerplate.go.txt` and are applied by codegen; new files should include the Apache-2.0 header for consistency.

## Testing

- Unit tests live next to the code (`*_test.go`) and use `gotest.tools/v3` (`assert`, `golden`).
- Golden-file tests under `pkg/controllers/**/testdata` may need `gotest.tools` golden flag (`-test.update`) when expected output intentionally changes.
- CRD/controller tests use a fake client where possible; heavier controller logic is covered by e2e tests in `test/e2e/`.
- E2E tests need a cluster and are gated on environment setup; don't rely on `go test ./...` at the root to pass without a cluster.

## Generated files — do not hand-edit

Regenerate, don't edit:
- `pkg/apis/**/zz_generated.deepcopy.go`
- `deploy/crds/**` CRD YAML (from Go types + controller-gen)
- `deploy/operator/observability-operator-cluster-role.yaml`
- `deploy/olm/**` kustomize image references (`make generate-kustomize`)
- `bundle/` and `docs/api.md`

If a change touches CRD types, RBAC markers, or kustomize image tags, run
`make generate` (and `make bundle` if bundle files changed) and commit the result.

## Updating the forked prometheus-operator

Bumping the OBO Prometheus Operator version touches **both**:
- `go.mod` (root module)
- `deploy/dependencies/kustomization.yaml`

Then run `make generate`. See `docs/developer.md` for the full procedure.

## Git / commits

- Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/) (validated by commitlint `@commitlint/config-conventional`): `fix:`, `feat:`, `build:`, `chore:`, `ci:`, `docs:`, `style:`, `refactor:`, `perf:`, `test:`.
- These drive `CHANGELOG.md`/`VERSION` via `standard-version` in release automation, so scope and type matter.
- Release versioning is SemVer; `VERSION` is a release artifact, not bumped manually.

## Environment notes

- `CONTAINER_RUNTIME` defaults to `podman` if present, else `docker`.
- Image builds require the local registry running in the kind cluster (see `docs/developer.md`).
- The `VELCOME`/tmp artifacts: `tmp/` is gitignored; `bundle/` is generated and committed.
- OpenShift is a feature gate (`--openshift.enabled`); the default non-OpenShift build skips UIPlugin/ObservabilityInstaller/operator controllers.

## See also

- `docs/developer.md` – detailed dev env, e2e, and release process
- `docs/api.md` – CRD API reference (generated)
- `.golangci.yml` – exact lint rules
- `Makefile.tools` – pinned tool versions table
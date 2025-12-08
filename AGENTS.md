## AGENTS GUIDE

This document is for **AI/code assistants and other automation** contributing changes to this repository.

The primary goal is to keep generated assets, CRDs, OLM bundles, and deployment manifests **consistent and reproducible** while making safe, reviewable edits.

---

## Project Overview

- **Name**: Observability Operator (`observability-operator`)
- **Type**: Kubernetes operator (Go, controller-runtime) plus Kustomize/OLM packaging
- **Purpose**: Manage monitoring/alerting stacks (and related observability components) via CRDs.

Key entry points:

- `cmd/operator/main.go` – operator binary entrypoint.
- `pkg/apis/...` – API types and CRDs definitions.
- `pkg/controllers/...` – reconcilers/controllers.
- `deploy/...` – Kustomize bases for CRDs, operator deployment, OLM, package-operator, etc.
- `bundle/...` – Generated OLM bundle (do not hand-edit).

For human-focused docs, see `README.md` and `docs/developer.md`.

---

## Conventions & Tooling

- **Language**: Go (see Go version in root `go.mod`).
- **Build & generation**: GNU Make, controller-gen, kustomize, operator-sdk.
- **Commit messages**: Follow **Conventional Commits** (see `docs/developer.md`).
- **Versioning**: SemVer, automated via `make initiate-release`.

Important make targets:

- `make generate` – regenerate CRDs, RBAC, deepcopy, Kustomize outputs, and API docs.
- `make test-unit` – run Go unit tests.
- `make test-e2e` – run Go-based e2e tests (alternative to `./test/run-e2e.sh`).
- `make operator` / `make build` – build the operator binary into `tmp/operator`.

Only **invoke** these commands in suggestions; do **not assume** they have been run.

---

## Repository Layout (for Agents)

- `cmd/` – CLI/entrypoint code.
- `pkg/apis/` – custom resource APIs; changes here require `make generate`.
- `pkg/controllers/` – controller logic; generally safe for targeted edits.
- `deploy/`:
  - `crds/` – generated CRDs (kubernetes + common).
  - `dependencies/` – Kustomize config for dependent operators (e.g. obo-prometheus-operator).
  - `monitoring/`, `operator/`, `olm/`, `package-operator/` – operator deployment and packaging.
- `bundle/` – **generated** OLM bundle content.
- `jsonnet/`, `dashboards/`, `must-gather/` – ancillary assets.
- `tmp/` – build/test artifacts; never commit changes from here.

Prefer editing **sources** (Go, Kustomize bases, templates) rather than generated artifacts.

---

## Safe vs Unsafe Edits

**Prefer to edit**

- Go code in:
  - `pkg/controllers/...`
  - `pkg/operator/...`
  - `pkg/reconciler/...`
  - `cmd/operator/...`
- API types in `pkg/apis/...` (but remember to run `make generate` afterwards).
- Kustomize bases under `deploy/...`:
  - `deploy/dependencies/...`
  - `deploy/monitoring/...`
  - `deploy/olm/...`
  - `deploy/package-operator/...`
- Documentation in `docs/` and `README.md`.

**Avoid hand-editing**

- `bundle/...` – OLM bundle content is generated via `make bundle`.
- Generated CRDs/RBAC in:
  - `deploy/crds/common/...`
  - `deploy/crds/kubernetes/...`
  - `deploy/operator/observability-operator-cluster-role.yaml`
- Files under `tmp/`.

When in doubt, look for a related **Makefile target** or a comment indicating files are generated.

---

## Common Change Patterns

### 1. Updating or Adding API Fields / CRDs

1. Modify Go API types under `pkg/apis/...`.
2. Update any validation/defaulting or controller logic in `pkg/controllers/...` as needed.
3. Regenerate artifacts:

   ```sh
   make generate
   ```

4. Ensure CRDs and RBAC changes are committed alongside the Go changes.

### 2. Bumping the forked Prometheus Operator (obo-prometheus-operator)

As described in `docs/developer.md`:

1. Update the dependency version in:
   - `go.mod`
   - `deploy/dependencies/kustomization.yaml`
2. Regenerate manifests:

   ```sh
   make generate
   ```

3. Commit Go module + dependency + generated manifest changes together.

### 3. Adjusting Operator / OLM Manifests

- For operator deployment changes (env vars, args, resources, etc.):
  - Prefer editing the relevant **Kustomize bases** under `deploy/operator/`, `deploy/monitoring/`, or `deploy/dependencies/`.
- For OLM bundle / CSV adjustments:
  - Edit Kustomize configs in `deploy/olm/` rather than direct edits in `bundle/`.
  - Recreate the bundle as needed with:

    ```sh
    make bundle
    ```

  - Be aware that `make bundle` may reset uncommitted changes in `bundle/` (see `Makefile`).

### 4. Tests & Linting

To keep suggestions consistent with local workflows, assume the following are used:

- Unit tests:

  ```sh
  make test-unit
  ```

- End-to-end tests:

  ```sh
  ./test/run-e2e.sh
  ```

- Linting:

  ```sh
  make lint
  ```

Proposed changes should be structured so they can pass these commands.

---

## Style & Quality Notes for Agents

- Follow existing **Go style** and `controller-runtime` patterns; avoid introducing new frameworks.
- Keep reconciliation logic **idempotent** and resilient to partial failures.
- When editing YAML, preserve:
  - Resource kinds, API versions, and labels/annotations used by operators/OLM.
  - Existing indentation and ordering where practical.
- For breaking or behavioural changes, ensure commit messages (authored by humans) can use
  `feat:`, `fix:`, or `BREAKING CHANGE:` consistently.

If a change appears to impact release automation, image tags, or OLM packaging, prefer to
suggest **small, focused diffs** and explicitly call out the potential impact in comments
or PR descriptions for human reviewers.




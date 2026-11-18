# ART COO integration tests

One pipeline tests one ART FBC Snapshot on one OpenShift minor version.
`IntegrationTestScenario` resources define the **OCP version × test type**
matrix. Adding a version or upgrade baseline does not require copying pipeline
logic or writing another test runner.

The implementation is ordinary Tekton YAML and uses the existing COO e2e suite.

## Structure

`coo-fbc-pipeline.yaml` contains orchestration and parameters. Implementation is
split into named, reusable Tasks in `tasks/`, so the pipeline graph remains
readable as the matrix grows:

```text
Common setup:
  resolve-snapshot -> resolve-release -> prepare-cluster-env
    -> provision-cluster -> configure-cluster

e2e path:
  install-candidate

upgrade path:
  install-released -> prepare-upgrade-fixture -> upgrade-to-candidate

Common verification:
  verify-candidate -> verify-upgrade-fixture [upgrade only] -> run-e2e-tests

Finally:
  cleanup
```

Both paths reuse `olm-operator.yaml` for Subscription/InstallPlan handling and
`run-e2e.yaml` for the existing `test/e2e` Go suite. The upgrade fixture uses the
same Task before and after the upgrade. No product-image rebuild or upstream
development catalog is involved.

Upgrade baseline state is kept on the test cluster (a Subscription annotation
and an immutable fixture ConfigMap). The common path has ordering dependencies
on the conditional branches, not result dependencies on skipped upgrade Tasks.
This matters in Tekton: a missing result from a skipped branch can also skip
its consumers, accidentally preventing the fresh-install e2e suite from running.

## Matrix policy

The matrix belongs in `releng/konflux-release-data`:

```text
tenants-config/cluster/kflux-ocp-p01/tenants/art-coo-tenant/coo-integration-tests.yaml
```

Each ITS row supplies:

| Setting | Meaning |
|---|---|
| `spec.application` | The y-stream's FBC application, currently `fbc-coo-1-5` |
| `ocp_version` | OpenShift minor version for this row |
| `test_type` | `e2e` or `upgrade` |
| `tests_ref` | Release-matching public test source; a commit pins reproducible retests |
| `upgrade_from_csv` | Optional upgrade baseline; otherwise the released stable head |
| `cluster_workflow` / `cluster_profile` | Provisioning policy, if defaults are unsuitable |
| `cluster_releases` | Optional ci-operator releases JSON, if a stable payload is unsuitable |
| `provision_timeout` | Optional provisioning timeout, at most the pipeline Task's two-hour budget |

The current platform is linux/amd64. Architecture is not an enabled matrix axis;
adding it requires reviewing the Go image, worker type, and release selector.

Use a separate ITS for each version/test-type/baseline combination that should
be covered. Its name should identify the case. Each case gets its own result
and isolated cluster. There is deliberately no nested Tekton matrix, shared
cluster, or parallel fresh-install/upgrade race within a PipelineRun.

ART already produces an FBC Snapshot per target OCP version. A row inspects the
image's OCP label and reports **Not tested** without provisioning when the
fragment belongs to another version. Thus configuring N versions and two test
types yields two actual cluster runs for each matching FBC, not 2N clusters for
every fragment. Review coverage across the matching snapshots: skipped rows
are not evidence that another OCP version passed.

The current GitOps change enables only the two OCP 4.22 pilot rows. Before
adding all supported versions, confirm the product support matrix, provisioning
availability, and upgrade baselines with the team. The `coo-1.5` ART group's
`OCP_TARGET_VERSIONS` currently contains 4.12, 4.14, 4.16, 4.18, 4.19, 4.20,
4.21, 4.22, and 5.0. That list is **build targeting, not by itself a support or
stable-release availability declaration**. In particular:

- Older supported versions may require a full-cluster workflow rather than the
  current HyperShift management cluster. Obtain access to the appropriate
  profile before configuring it; `aws-konflux-prod` is the shared HyperShift
  default, not a blanket authorization for other workflows.
- A version without a stable payload needs an explicitly reviewed
  `cluster_releases` selector. Do not silently substitute a newer OCP version.
- Non-HyperShift cluster setup uses IDMS where available and the older ICSP API
  otherwise. Validate provisioning, mirrors, and monitoring on every newly
  enabled legacy row rather than assuming the 4.22 pilot proves compatibility.

## Upgrade coverage

The upgrade path mirrors the tracing team's released-to-FBC approach:

1. Install `cluster-observability-operator` from the cluster's `redhat-operators`
   stable channel. Resolve and record the baseline CSV, or use
   `upgrade_from_csv` to select a specific supported older release.
2. Create a MonitoringStack with persistent storage and Alertmanager using that
   released operator. Wait for both workloads and record the identity of the
   bound PVC mounted at the Prometheus data directory.
3. Switch the **same Subscription** to the immutable candidate FBC and approve
   only an InstallPlan containing the resolved candidate CSV.
4. Verify that exact candidate is installed and its shipped configuration works.
5. Change the stack's external labels and verify the new operator reconciles
   that change. Check that both workloads are ready and Prometheus still mounts
   the original bound PVC.
6. Run the unchanged COO Go e2e suite against the upgraded installation.

Deleting the old CSV, Subscription, CRDs, or workloads and reinstalling the
candidate is not an acceptable substitute for this path. A baseline equal to
the candidate fails explicitly rather than reporting a successful "upgrade".
If production has caught up with the candidate, choose an older supported
baseline. If the FBC's upgrade graph does not allow the transition, the test
fails; it does not bypass OLM.

This is deliberately bounded initial upgrade coverage: OLM transition, workload
readiness, storage preservation, reconciliation, and post-upgrade functionality.
It does **not** yet prove historical query data, uninterrupted collection/alerting,
browser compatibility, every UIPlugin migration, or every supported upgrade
baseline. Those can be added as product-specific pre/post-upgrade fixtures
without duplicating provisioning or introducing another framework.

## Rollout and source revisions

1. Merge the pipeline **and `tasks/` directory together** into
   `rhobs/observability-operator` before the tenant ITS references them.
2. Grant `art-coo-tenant` on `kflux-ocp-p01` access to `aws-konflux-prod` through
   `openshift/release`'s cluster-profile configuration. Confirm the integration
   runner's quota and namespaced TestPlatformCluster create/get/list/patch access.
3. Have ART confirm a tenant `kubernetes.io/dockerconfigjson` Secret named
   `stage-registry-creds`, key `.dockerconfigjson`, with stage pull credentials.
   Include FBC-registry credentials if private. Override `stage_registry_secret`
   if the actual name differs; never commit Secret data.
4. Merge the ITS source, Kustomize entry, and regenerated tenant manifests.
   Keep the scenarios optional until matching-version passes, intentional
   failures, upgrade preservation, and cleanup have been demonstrated.

`pipeline_repository` and `pipeline_revision` resolve this pipeline's native
Tasks. They are **not** the product's test source. If the ITS resolver points
at a fork/branch or pinned pipeline commit, set these two parameters to the
same repository/revision. The defaults match an ITS resolving upstream `main`.
`tests_ref`, independently, selects the existing public COO Go test source.
The resolved test commit is recorded and its `VERSION` must match the FBC's
COO major/minor. The FBC's Git revision is not necessarily the operator source
revision. Authenticated embargo-specific source checkout is not enabled by
this public-source pilot.

Default provisioning uses HyperShift on `hosted-mgmt2`, three `m5.2xlarge`
workers, and the requested stable OCP minor. Guest catalog placement and a
digest-preserving mirror allow stage pulls without rewriting shipped images.
The cluster must provide the service CA, platform monitoring certificates,
console, and a default StorageClass. Missing prerequisites fail explicitly.

Fresh-install ITS timeouts are 4h total / 3h30m tasks / 30m finally; the upgrade
pilot allows 6h / 5h30m / 30m. Set all three together. Review the suite's 60-minute
timeout, 90-minute test Task, two-hour provisioning Task, and four-hour
post-provisioning lifetime bound when extending a matrix row's budgets.
Cleanup signals only claims owned by the PipelineRun and also handles partial
provisioning. Claim ownership and the lifetime timeout are interruption fallbacks.

## Results and validation

The standard CI Go build root compiles the existing `./test/e2e/` package;
`gotestsum` and `test2json` already ship there. The standard Konflux tools image
runs that binary with the actual installation namespace. Failures remain
failures, all-skipped functional suites cannot pass, and JUnit plus diagnostics
are printed in TaskRun logs. There is no optional OCI uploader or artifact
registry to configure in this initial implementation.

Validate native Pipeline/Task YAML, Bash syntax, and ShellCheck before rollout;
validate tenant manifests with that repository's Kustomize/tox workflow.
Run the cluster-free upgrade fixture regression tests with
`bash konflux/integration-tests/tests/upgrade-fixture_test.sh` (requires `jq`
and `yq` v4). No custom Go test harness or Makefile change is required for the
pipeline. Offline checks do not replace a live fresh-install and upgrade pilot.

The suite and upgrade fixtures mutate cluster-wide resources. Never run them
on shared or production installations. Do not log kubeconfigs or Secret data.
These ITS runs complement upstream PR CI and Conforma. Required FBC ITS results
are not a universal gate for earlier image staging or ART/manual shipments;
those release paths require explicit checks of the matching snapshot's results.

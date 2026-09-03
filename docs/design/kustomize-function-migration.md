# Kustomize overlay migration: deleted Go functions

The kustomize overlay refactor replaced Go functions that constructed
Kubernetes resources as struct literals with declarative YAML under `config/`.
The overlay engine (`pkg/overlay`) reads these YAML files at runtime and applies
patches and replacements to produce the final resource set.

## UIPlugin resources (`pkg/controllers/uiplugin/`)

### Core plugin resources

Each plugin type had its resources built by `pluginComponentReconcilers` and
helpers in `components.go`. These are now kustomize components under
`config/uiplugins/components/<type>/resources/`.

| Deleted function | Replacement |
|---|---|
| `newDeployment` | `<type>/resources/deployment.yaml` |
| `newService` | `<type>/resources/service.yaml` |
| `newServiceAccount` | `<type>/resources/serviceaccount.yaml` |
| `newRole` | `<type>/resources/role.yaml` |
| `newRoleBinding` | `<type>/resources/rolebinding.yaml` |
| `componentLabels` | Labels baked into each YAML file |

### ConsolePlugin

| Deleted function | Replacement |
|---|---|
| `newConsolePlugin` | `consoleplugin/v1/consoleplugin.yaml` |
| `newRhobsConsolePlugin` | `consoleplugin/rhobs-v1/consoleplugin.yaml` |
| `newLegacyConsolePlugin` | `consoleplugin/v1alpha1/consoleplugin.yaml` |
| `PluginProxy.ToV1Alpha1` / `ToRhobsV1` / `ToUpstreamV1` | Proxy config built as patches in `addConsolePluginPatch` |

### Korrel8r (troubleshooting-panel)

| Deleted function | Replacement |
|---|---|
| `newKorrel8rDeployment` | `troubleshooting-panel/korrel8r/korrel8r-deployment.yaml` |
| `newKorrel8rService` | `troubleshooting-panel/korrel8r/korrel8r-service.yaml` |
| `newKorrel8rConfigMap` | `troubleshooting-panel/korrel8r/korrel8r-configmap.yaml` (patched dynamically) |

### Health analyzer (monitoring)

| Deleted function | Replacement |
|---|---|
| `newHealthAnalyzerDeployment` | `monitoring/health-analyzer/deployment.yaml` |
| `newHealthAnalyzerService` | `monitoring/health-analyzer/service.yaml` |
| `newHealthAnalyzerServiceMonitor` | `monitoring/health-analyzer/servicemonitor.yaml` |
| `newHealthAnalyzerPrometheusRole` | `monitoring/health-analyzer/role-prometheus-k8s.yaml` |
| `newHealthAnalyzerPrometheusRoleBinding` | `monitoring/health-analyzer/rolebinding-prometheus-k8s.yaml` |
| `newComponentHealthConfig` | `monitoring/health-analyzer/configmap.yaml` |
| `componentsHealthClusterRole` | `monitoring/health-analyzer/clusterrole.yaml` |

### Perses (monitoring)

| Deleted function | Replacement |
|---|---|
| `newPersesClusterRole` | `monitoring/perses/clusterrole.yaml` |
| `newClusterRoleBinding` (perses) | `monitoring/perses/clusterrolebinding-perses.yaml` |
| `newClusterRoleBinding` (auth-delegator) | `monitoring/perses/clusterrolebinding-system-auth-delegator.yaml` |

The following Perses resources remain as Go code because they use the Perses SDK
to generate their spec and cannot be expressed as static YAML:

- `newPerses` — Perses instance
- `newPrometheusGlobalDatasource` — PersesGlobalDatasource
- `newAcceleratorsDatasource` — PersesDatasource
- `newAcceleratorsDashboard` — PersesDashboard
- `newAPMDashboard` — PersesDashboard

### Other

| Deleted function | Replacement |
|---|---|
| `GenerateUIPluginObjects` | `ResolvePlugin` (calls `BuildOverlay`) |
| `pluginComponentReconcilers` | `BuildOverlay` + `addMonitoringComponents` |

## ObservabilityInstaller resources (`pkg/controllers/observability/`)

| Deleted function | Replacement |
|---|---|
| `otelCollector` | `tracing/collector/resources/opentelemetrycollector.yaml` |
| `otelCollectorComponentsRBAC` | `tracing/collector/resources/rbac.yaml` (ClusterRole + ClusterRoleBinding) |
| `otelCollectorTempoRBAC` | `tracing/collector/resources/rbac.yaml` (ClusterRole + ClusterRoleBinding) |
| `otelCollectorName` | Inlined (identity function) |
| `subscription` | `tracing/collector/operator/subscription.yaml`, `tracing/store/operator/subscription.yaml` |
| `GenerateAllInstallerObjects` / `GenerateInstallerObjects` | Overlay-based reconciler in `overlay.go` / `tracing_overlay.go` |

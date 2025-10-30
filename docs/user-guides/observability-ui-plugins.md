# Observability UI Plugins

Using the Observability UI, you can install and manage plugins that extend the observability functionality of the OpenShift web console. Plugins are installed and managed by the Observability Operator.

## Plugins

- [dashboards](#dashboards): Add enhanced dashboards to the OpenShift web console. This plugin allows you to add other Prometheus datasources present in the cluster, apart from the in-cluster one, to the default dashboards.
- [troubleshooting-panel](#troubleshooting-panel): Add the troubleshooting panel to the OpenShift web console. This plugin adds a troubleshooting panel to the console dashboard, which queries and displays results from [Korrel8r](https://github.com/korrel8r/korrel8r) to help troubleshoot issues.
- [distributed-tracing](#distributed-tracing): Add the Observability > Traces page to the Openshift web console. This plugin allows a user to select a [Tempo](https://docs.openshift.com/container-platform/4.13/observability/distr_tracing/distr_tracing_arch/distr-tracing-architecture.html#distr-tracing-architecture_distributed-tracing-architecture) instance and view trace data from it.
- [monitoring](#monitoring): Add the a number of Observing pages to the Openshift web related to Alerting. This plugin allows a user to view Alerts, Silences, and Alert rules.

| __COO Version__ |   __OCP Versions__  | __Dashboards__ | __Distributed Tracing__ | __Logging__ | __Troubleshooting Panel__ | __Monitoring__ |
| --------------- | ------------------- | -------------- | ----------------------- | ----------- | ------------------------- | ---------------|
| 0.2.0           | 4.11                |       ✔        |             ✘           |       ✘     |             ✘             |       ✘       |
| 0.3.0 - 0.4.0   | 4.11 - 4.15         |       ✔        |             ✔           |       ✔     |             ✘             |       ✘       |
| 0.3.0 - 0.4.0   | 4.16+               |       ✔        |             ✔           |       ✔     |             ✔             |       ✘       |
| 1.0.0+          | 4.11 - 4.14         |       ✔        |             ✔           |       ✔     |             ✘             |       ✘       |
| 1.0.0+          | 4.15                |       ✔        |             ✔           |       ✔     |             ✘             |       ✔       |
| 1.0.0+          | 4.16+               |       ✔        |             ✔           |       ✔     |             ✔             |       ✔       |

Some plugin offer additional features that are available dependant on the cluster version. COO will always deploy all features available for the cluster it is running on.

### Dashboards

The plugin will search for datasources as ConfigMaps in the `openshift-config-managed` namespace with the `console.openshift.io/dashboard-datasource: 'true'` label. The namespace `openshift-config-managed` is required, more details on how to create a datasource ConfigMap can be found in the [console-dashboards-plugin](https://github.com/openshift/console-dashboards-plugin/blob/main/docs/add-datasource.md)

#### Plugin Creation

To enable the console dashboards plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the console dashboards plugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: dashboards
spec:
  type: Dashboards
```

#### Feature Matrix

| __COO Version__ |   __OCP Versions__  | __Features__                                          |
| --------------- | ------------------- | ----------------------------------------------------- |
| 0.2.0+          | 4.11+               | _No features configuration, just core functionality_  |

### Troubleshooting Panel

The plugin adds a UI panel meant to assist in the troubleshooting journey, through exploring related pages. Creating this `UIPlugin` will deploy a [Korrel8r](https://github.com/korrel8r/korrel8r) service named `korrel8r` in the same namespace which is able to locate related observability signals and kubernetes resources from its correlation engine.

To use the Troubleshooting Panel, in the admin perspective navigate to `Observe > Alerts` and then select an alert. If the alert has correlated items then a "Troubleshooting Panel" will appear above the chart on the alert detail page. This button opens a panel consisting of query details and a topology graph of the query results. The alert page you are on is converted into a Korrel8r query string and sent to the `korrel8r` service. The results are displayed as a graph network connecting the returned signals and resources. The nodes on the graph will take you to the corresponding OpenShift wab console pages when clicked.

#### Plugin Creation

To enable the troubleshooting panel console plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the troubleshooting panel console plugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: troubleshooting-panel
spec:
  type: TroubleshootingPanel
```

#### Feature Matrix

| __COO Version__ |   __OCP Versions__  | __Features__                                          |
| --------------- | ------------------- | ----------------------------------------------------- |
| 0.3.0+          | 4.16+               | _No features configuration, just core functionality_  |

### Distributed Tracing

The plugin adds tracing related UI features to the OpenShift web console. A new tab can be located in the admin perspective under `Observe > Traces`. This tab allows a user to select a supported Tempo instance ([TempoStack](https://docs.openshift.com/container-platform/4.16/observability/distr_tracing/distr_tracing_tempo/distr-tracing-tempo-installing.html#installing-a-tempostack-instance) or [TempoMonolithic](https://docs.openshift.com/container-platform/4.16/observability/distr_tracing/distr_tracing_tempo/distr-tracing-tempo-installing.html#installing-a-tempomonolithic-instance) with multi-tenancy) running in their cluster as well as a set a time range and query for the traces being loaded. These traces are displayed on a scatter-plot showing the trace start time, duration, and number of spans. Underneath the scatter plot there is a list of traces showing information such as the `Trace Name`, number of `Spans`, and `Duration`. The trace name contains a link which takes the user to the trace detail page for the selected trace containing a Gantt Chart of all of the spans within the trace. Once selected, the spans show a breakdown of their configured attributes.

#### Plugin Creation

To enable to distributed tracing console plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the distributed tracing console plugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: distributed-tracing
spec:
  type: DistributedTracing
```

#### Feature Matrix

| __COO Version__ |   __OCP Versions__  | __Features__                                          |
| --------------- | ------------------- | ----------------------------------------------------- |
| 0.3.0+          | 4.11+               | _No features configuration, just core functionality_  |

### Logging

The plugin adds various Logging UI functionalities to the OpenShift web console. The core functionality of this plugin adds a new admin perspective tab `Observe > Logs`. This page includes query and log filters, the list of logs, and a histogram showing log frequency by severity. The logs can be filtered by an number of tags, such as `tenant` and `namespace`. The page also has controls for the length of time to query over, the refresh rate of the logging page, and whether to show kubernetes resource information in the results, such as `pod` and `container`. The results section shows a list of collapsed logs, which can then be expanded to show more detailed information for each log.

When a __TroubleshootingPanel__ `UIPlugin` is deployed the plugin will connect the [Korrel8r](https://github.com/korrel8r/korrel8r) service and add direct links from the admin perspective `Observe > Logs` page to the `Observe > Metrics` page with a correlated PromQL query. It will also add a "See Related Logs" link from the admin perspective `Observe > Alerting` on an alerting detail page to the `Observe > Logs` page with a correlated filter set selected.

#### Feature List

| __Feature__   | __Description__                                                                                                                                                            | __Support Level__ |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- |
| `dev-console` | Adds the logging view to the developer perspective                                                                                                                         | General Availability |
| `alerts`      | Merges the OpenShift console alerts with log-based alerts defined in the Loki ruler. Adds a log-based metrics chart in the alert detail view                               | General Availability |
| `dev-alerts`  | Merges the OpenShift console alerts with log-based alerts defined in the Loki ruler. Adds a log-based metrics chart in the alert detail view for the developer perspective | General Availability |

#### Feature Matrix

| __COO version__ | __OCP versions__ | __Features__                                          |
| --------------- | ---------------- | ----------------------------------------------------- |
| 0.3.0+          | 4.11             | _No features configuration, just core functionality_  |
| 0.3.0+          | 4.12             | `dev-console`                                         |
| 0.3.0+          | 4.13             | `dev-console`, `alerts`                               |
| 0.3.0+          | 4.14+            | `dev-console`, `alerts`, `dev-alerts`                 |

#### Plugin Creation

To enable to logging view plugin, create a `UIPlugin` CR. This CR has three parameters located under `spec.logging`, used to control the behavior of the logging-view-plugin. The `spec.logging.lokiStack` required parameter locates the LokiStack instance in the `openshift-logging` namespace to connect to. The `spec.logging.logLimit` and the `spec.logging.timeout` determine the number of logs returned from a query and the time before the query timeouts respectively.

The following example shows how to create a CR to enable the logging view plugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: logging
spec:
  type: Logging
  logging:
    lokiStack:
      name: logging-loki
    logsLimit: 50
    timeout: 30s
    schema: otel
```

### Monitoring

The plugin adds monitoring related UI features to the OpenShift web console, related to the Advance Cluster Management (ACM) perspective, incidents (cluster health analysis), and [Perses](https://github.com/perses/perses). A number of new pages and features are enabled through this plugin. Including, but not limited to:
- `ACM > Observe > Alerting`
- `ACM > Observe > Alerting > Silences`
- `ACM > Observe > Alerting > Alert rules`
- `OCP > Observe > Dashboards (Perses)`
- `OCP > Observe > Incidents`

To deploy ACM related features the `acm-alerting` configuration must be enabled. In the UIPlugin Custom Resource (CR) you must pass the Alertmanager and ThanosQuerier Service endpoint (e.g. `https://alertmanager.open-cluster-management-observability.svc:9095` and `https://rbac-query-proxy.open-cluster-management-observability.svc:8443`). See the example in the next section `Plugin Creation.`

To deploy the Incidents feature, the `incidents` configuration must be enabled. See the example in the next section, `Plugin Creation.`

To deploy the Perses dashboard feature, the `perses-dashboards` configuration must be enabled. In the UIPlugin CR, you can optionally pass the service name and namespace of your Perses instance (e.g., `serviceName: perses-api-http` and `namespace: perses`). If these fields are left blank and `spec.monitoring.perses.enabled: true`, then default values will be assigned. These default values are `serviceName: perses-api-http` and `namespace: perses`. See the example in the next section, `Plugin Creation.`
Besides, when `spec.monitoring.perses.enabled: true`, Accelerator Perses dashboard and Accelerator Perses datasource are both created.

ObO/COO operator creates the following roles:
- persesdashboard-editor-role - ability to create, read, update and delete perses dashboards CRD instance presented on ObO/COO operator under PersesDashboards tab, and view perses dashboards presentation in Dashboards (Perses)
- persesdashboard-viewer-role - ability to only read/view perses dashboards CRD instance presented on ObO/COO operator under PersesDashboards tab, and view perses dashboards presentation in Dashboards (Perses)
- persesdatasource-editor-role - ability to create, read, update and delete perses datasources CRD instance presented on ObO/COO operator under PersesDatasources tab, and view perses dashboards with data being loaded from perses datasource in Dashboards (Perses)
- persesdatasource-viewer-role - ability to only read/view perses datasources CRD instance presented on ObO/COO operator under PersesDatasources tab, and view perses dashboards with data being loaded from perses datasource in Dashboards (Perses)

When assigned via ClusterRoleBinding, user has access to all perses dashboards and perses datasources presented in all namespaces/projects. When assigned via RoleBinding, user has access to all perses dashboards and perses datasources presented in a given namespace/project.

Examples:
- user1 RoleBinding as persesdashboard-viewer-role and persesdatasource-viewer-role in openshift-cluster-observability-operator namespace:
```yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: user1-viewer-dashboard
  namespace: openshift-cluster-observability-operator
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: user1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: persesdashboard-viewer-role

kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: user1-viewer-datasource
  namespace: openshift-cluster-observability-operator
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: user1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: persesdatasource-viewer-role
```

- user1 ClusterRoleBinding as persesdashboard-editor-role and persesdatasource-editor-role:
```yaml
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: user1-editor-dashboard
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: user1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: persesdashboard-editor-role

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: user1-editor-datasource
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: user1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: persesdatasource-editor-role
```

Other pages which are typically distributed with the monitoring-plugin, such as `Admin > Observe > Dashboards`, are only available in the monitoring-plugin when deployed through [CMO](https://github.com/openshift/cluster-monitoring-operator).

#### Plugin Creation

To enable to monitoring console plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the monitoring console plugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: monitoring
spec:
  type: Monitoring
  monitoring:
    acm:
      enabled: true
      alertmanager:
        url: 'https://alertmanager.open-cluster-management-observability.svc:9095'
      thanosQuerier:
        url: 'https://rbac-query-proxy.open-cluster-management-observability.svc:8443'
    perses:
      enabled: true
    incidents:
      enabled: true
```

#### Feature List

| __Feature__         | __Description__                                                                                                          | __Support Level__    |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------ | -----------------    |
| `acm-alerting`      | Adds alerting UI to multi-cluster view. Configures proxies to connect with any alertmanager and thanos-querier.          | Dev Preview          |
| `incidents`         | Adds incidents UI to `Observe` section of OpenShift Console Platform. Deploys the [Cluster Health Analyzer](https://github.com/openshift/cluster-health-analyzer) and configures proxies in the plugin to connect with it. | General Availability |
| `perses-dashboards` | Adds perses UI to `Observe` section of OpenShift Console Platform. Configures proxies to connect with a Perses instance. Installs Accelerator Perses Dashboard and Accelerator Perses Datasource. See details [here](./perses-dashboards.md) | Dev Preview          |


#### Feature Matrix

| __COO Version__ |   __OCP Versions__  | __Features__                      |
| --------------- | ------------------- | --------------------------------- |
| 1.0.0+          | 4.14+               | `acm-alerting`                    |
| 1.1.0+          | 4.15+               | `acm-alerting, perses-dashboards` |
| 1.2.0           | 4.19+               | `acm-alerting, perses-dashboards, incidents (Tech Preview)` |
| 1.3.0+          | 4.19+               | `acm-alerting, perses-dashboards, incidents (General Availability)` |

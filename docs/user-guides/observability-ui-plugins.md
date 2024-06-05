# Observability UI Plugins

Using the Observability UI, you can install and manage plugins that extend the observability functionality of the OpenShift web console. Plugins are installed and managed by the Observability Operator.

## Plugins

- [dashboards](#dashboards): Add enhanced dashboards to the OpenShift web console. This plugin allows you to add other Prometheus datasources present in the cluster, apart from the in-cluster one, to the default dashboards.
- [troubleshooting-panel](#troubleshooting-panel): Add the troubleshooting panel to the OpenShift web console. This plugin adds a troubleshooting panel to the console dashboard, which queries and displays results from [Korrel8r](https://github.com/korrel8r/korrel8r) to help troubleshoot issues.
- [distributed-tracing](#distributed-tracing): Add the Observability > Traces page to the Openshift web console. This plugin allows a user to select a [TempoStack](https://docs.openshift.com/container-platform/4.15/observability/distr_tracing/distr_tracing_rn/distr-tracing-rn-3-1-1.html) instance and view trace data from it.

### Dashboards

The plugin will search for datasources as ConfigMaps in the `openshift-config-managed` namespace with the `console.openshift.io/dashboard-datasource: 'true'` label. The namespace `openshift-config-managed` is required, more details on how to create a datasource ConfigMap can be found in the [console-dashboards-plugin](https://github.com/openshift/console-dashboards-plugin/blob/main/docs/add-datasource.md)

To enable the console dashboards plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the console dashboards plugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: ui-dashboards
spec:
  type: Dashboards
```

### Troubleshooting Panel

The plugin will connect to a Korrel8r instance named `korrel8r` in the `korrel8r` namespace. A "Troubleshooting Panel" button is added to the alerts page, which will convert the current alert into a Korrel8r query, then retrieve related neighbors and display them in a topology view.

To enable the troubleshooting panel console plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the troubleshooting panel console plugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: troubleshooting-panel-console-plugin
spec:
  type: TroubleshootingPanel
```

### Distributed Tracing

The plugin allows a user to select a TempoStack instance and query traces from it to display them as a table and a scatter plot.

To enable to distributed tracing console plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the distributed tracing console plugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: distributed-tracing-console-plugin
spec:
  type: DistributedTracing
```

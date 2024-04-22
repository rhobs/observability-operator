# Observability UI Plugins

Using the Observability UI, you can install and manage plugins that extend the observability functionality of the OpenShift web console. Plugins are installed and managed by the Observability Operator.

## Plugins

- [dashboards](#dashboards): Add enhanced dashboards to the OpenShift web console. This plugin allows you to add other Prometheus datasources present in the cluster, apart from the in-cluster one, to the default dashboards.

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

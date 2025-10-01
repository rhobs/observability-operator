# Perses Dashboards

In conjunction with the Monitoring UIPlugin, the Cluster Observability Operator (COO) allows you to add dashboards and datasources to the OpenShift Console using [Perses](https://perses.dev/). This feature is currently in Dev Preview.

## Requirements to display Perses Dashboards in OpenShift Console

- OpenShift 4.15 or later
- The Cluster Observability Operator 1.2 or later installed and running
- The Monitoring UIPlugin installed with the perses feature enabled

## Adding a Perses Dashboard and Datasource

COO embeds the [Perses Operator](https://github.com/perses/perses-operator), which is responsible for managing Perses dashboards and datasources. The COO provides directly the `PersesDashboard` and `PersesDatasource` custom resources (CRs) which are namespaced. This allows you to define RBAC policies for them using the standard Kubernetes RBAC model.

### Add a Perses Dashboard and Datasource manually

To add a Perses dashboard to the OpenShift Console, a `PersesDashboard` CR must be created. The Perses dashboard CR is namespaced, meaning it is scoped to a specific namespace in your OpenShift cluster.

```yaml
apiVersion: perses.dev/v1alpha1
kind: PersesDashboard
metadata:
  name: my-dashboard
  namespace: my-namespace
spec:
  # Perses Dashboard specification goes here
```

Found a complete dashboard example [here](https://github.com/perses/perses-operator/blob/main/config/samples/openshift/openshift-cluster-sample-dashboard.yaml)

The dashboard specification to create a Perses dashboard CR can be obtained in two ways:

1. **Export from Perses UI**: Export the specification directly from an existing Perses dashboard through the Perses UI.
> [!NOTE]
> The Perses UI can now export the CR directly.
2. **Convert from Grafana**: Convert an existing Grafana dashboard definition to Perses format using the [`percli`](https://perses.dev/perses/docs/migration/).


Similarly, to add a Perses datasource, a `PersesDatasource` CR must be created:

```yaml
apiVersion: perses.dev/v1alpha1
kind: PersesDatasource
metadata:
  name: my-datasource
  namespace: my-namespace
spec:
  config:
    # Perses Datasource specification goes here
```

The Openshift Console monitors the cluster using the Cluster Monitoring Operator (CMO) and the Prometheus Operator. A Perses Datasource can be created to connect to the Prometheus instance in the cluster trough the thanos querier. The datasource CR can look like this:

```yaml
apiVersion: perses.dev/v1alpha1
kind: PersesDatasource
metadata:
  name: thanos-querier-datasource
  namespace: perses-dev
spec:
  config:
    display:
      name: "Thanos Querier Datasource"
    default: true
    plugin:
      kind: "PrometheusDatasource"
      spec:
        proxy:
          kind: HTTPProxy
          spec:
            url: https://thanos-querier.openshift-monitoring.svc.cluster.local:9091
            secret: thanos-querier-datasource-secret
  client:
    tls:
      enable: true
      caCert:
        type: file
        certPath: /ca/service-ca.crt
```
> [!IMPORTANT]
> The name `thanos-querier-datasource-secret` in the example isn't a Kubernetes secret. It's a reference to a Perses secret that the Perses Operator automatically generates from the datasource name and stores in the Perses backend. Therefore, the secret's name must match the datasource name, followed by the `-secret` suffix.

This will allow a dashboard in the `perses-dev` namespace to fetch cluster metrics.

### Add a Perses Dashboard and Datasource from an operator

Similar to Prometheus ServiceMonitors, you can create Perses dashboards and datasources from an operator to define default dashboards and datasources based on the applications it deploys. Dashboards and datasources can be created programatically in the reconciliation loop. Either by creating the CR and definition directly from the [Perses Operator API](https://pkg.go.dev/github.com/perses/perses-operator/api/v1alpha1#Dashboard) or using the [Perses Go SDK](https://perses.dev/perses/docs/dac/go/dashboard/#example). For example:

```go
package dashboards

import (
  "context"
  "time"

  "github.com/perses/community-dashboards/pkg/promql"
  persesv1 "github.com/perses/perses-operator/api/v1alpha1"
  common "github.com/perses/perses/go-sdk/common"
  dashboard "github.com/perses/perses/go-sdk/dashboard"
  "github.com/perses/perses/go-sdk/panel"
  panelgroup "github.com/perses/perses/go-sdk/panel-group"
  listvariable "github.com/perses/perses/go-sdk/variable/list-variable"
  "github.com/perses/plugins/prometheus/sdk/go/query"
  labelValuesVar "github.com/perses/plugins/prometheus/sdk/go/variable/label-values"
  timeSeriesPanel "github.com/perses/plugins/timeserieschart/sdk/go"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "sigs.k8s.io/controller-runtime/pkg/client"
)

func BuilderToOperatorResource(builder dashboard.Builder) client.Object {
  return &persesv1.PersesDashboard{
    TypeMeta: metav1.TypeMeta{
      Kind:       "PersesDashboard",
      APIVersion: "perses.dev/v1alpha1",
    },
    ObjectMeta: metav1.ObjectMeta{
      Name:      builder.Dashboard.Metadata.Name,
      Namespace: builder.Dashboard.Metadata.Project,
      Labels: map[string]string{
        "app.kubernetes.io/name":      "perses-dashboard",
        "app.kubernetes.io/instance":  builder.Dashboard.Metadata.Name,
        "app.kubernetes.io/part-of":   "my-operator",
        "app.kubernetes.io/component": "dashboard",
      },
    },
    Spec: persesv1.Dashboard{
      DashboardSpec: builder.Dashboard.Spec,
    },
  }
}

func GetPersesDashboard() (dashboard.Builder, error) {
  return dashboard.New("Example Dashboard",
    dashboard.ProjectName("my-namespace"),
    dashboard.RefreshInterval(1*time.Minute),
    dashboard.Duration(24*time.Hour),

    // VARIABLES
    dashboard.AddVariable("job",
      listvariable.List(
        labelValuesVar.PrometheusLabelValues("job",
          labelValuesVar.Matchers("perses_build_info{}"),
        ),
        listvariable.DisplayName("job"),
      ),
    ),
    dashboard.AddVariable("instance",
      listvariable.List(
        labelValuesVar.PrometheusLabelValues("instance",
          labelValuesVar.Matchers(
            promql.SetLabelMatchers(
              "perses_build_info",
              []promql.LabelMatcher{{Name: "job", Type: "=", Value: "$job"}},
            ),
          ),
        ),
        listvariable.DisplayName("instance"),
      ),
    ),

    // ROWS
    dashboard.AddPanelGroup("Latency Metrics",
      panelgroup.PanelsPerLine(3),

      // PANELS
      panelgroup.AddPanel("HTTP Requests Latency",
        panel.Description("Displays the latency of HTTP requests over a 5-minute window."),
        timeSeriesPanel.Chart(
          timeSeriesPanel.WithYAxis(timeSeriesPanel.YAxis{
            Format: &common.Format{
              Unit: string(common.SecondsUnit),
            },
          }),
          timeSeriesPanel.WithLegend(timeSeriesPanel.Legend{
            Position: timeSeriesPanel.RightPosition,
            Mode:     timeSeriesPanel.TableMode,
          }),
          timeSeriesPanel.WithVisual(timeSeriesPanel.Visual{
            Display:      timeSeriesPanel.LineDisplay,
            ConnectNulls: false,
            LineWidth:    0.25,
            AreaOpacity:  0.5,
            Palette:      timeSeriesPanel.Palette{Mode: timeSeriesPanel.AutoMode},
          }),
        ),
        panel.AddQuery(
          query.PromQL(
            "sum by (handler, method) (rate(perses_http_request_duration_second_sum{job=~'$job', instance=~'$instance'}[$__rate_interval])) / sum by (handler, method) (rate(perses_http_request_duration_second_count{job=~'$job', instance=~'$instance'}[$__rate_interval]))",
            query.SeriesNameFormat("{{handler}} {{method}}"),
          ),
        ),
      ),
    ),
  )
}

func Reconcile(k8sClient client.Client) {
  dashboardBuilder, err := GetPersesDashboard()
  if err != nil {
    panic(err)
  }
  dashboardResource := BuilderToOperatorResource(dashboardBuilder)

  // Here you would typically use a Kubernetes client to apply the dashboardResource idempotently.
  k8sClient.Create(context.TODO(), dashboardResource)
}
```

More examples can be found in the [community dashboards repository](https://github.com/perses/community-dashboards)

> [!IMPORTANT]
> **Automatic Datasource Detection**: Notice that the above example does not set a specific datasource for the dashboard. This is because Perses will automatically detect the available datasources in the namespace and use the default one it finds. A specific datasource can be set by adding a `datasource` field in the panel query or by adding a datasource variable to the dashboard so users can select the datasource they want to use.

## Perses dashboards and datasources RBAC

The Perses operator creates the following cluster roles for datasources and dashboards. 

- persesdashboard-editor-role
- persesdashboard-viewer-role
- persesdatasource-editor-role
- persesdatasource-viewer-role

The following role bindings illustrate how to add viewer permissions for `user1` in the `my-namespace` namespace:

```yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: user1-viewer-dashboard
  namespace: my-namespace
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: user1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: persesdashboard-viewer-role
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: user1-viewer-datasource
  namespace: my-namespace
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: user1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: persesdatasource-viewer-role
```

## Viewing Perses Dashboards in OpenShift Console

Once the `PersesDashboard` and `PersesDatasource` CRs are created and the appropriate RBAC permissions are granted, you can view the dashboards in the OpenShift Console under the "Observe -> Perses Dashboards" section. A namespace selector will be available to filter dashboards by the namespaces where the user has been granted Perses RBAC permissions.

package uiplugin

import (
	"encoding/json"
	"fmt"

	"github.com/perses/perses/go-sdk/dashboard"
	"github.com/perses/perses/go-sdk/panel"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	listvariable "github.com/perses/perses/go-sdk/variable/list-variable"
	textvariable "github.com/perses/perses/go-sdk/variable/text-variable"
	logstable "github.com/perses/plugins/logstable/sdk/go"
	lokiquery "github.com/perses/plugins/loki/sdk/go/query/log"
	markdown "github.com/perses/plugins/markdown/sdk/go"
	staticlist "github.com/perses/plugins/staticlistvariable/sdk/go"
	specCommon "github.com/perses/spec/go/common"
	dsSpec "github.com/perses/spec/go/datasource"
	persesv1alpha2 "github.com/rhobs/perses-operator/api/v1alpha2"
	persesv1 "github.com/rhobs/perses/pkg/model/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

const (
	auditDatasourceName = "audit-loki-datasource"

	// auditVarQuery builds a metric expression that returns all unique values
	// of a field from audit logs. Using count_over_time aggregation returns a
	// matrix result (one series per unique value) rather than a streams result,
	// avoiding the default 100-entry limit that log queries have.
	//
	// Only used for low-cardinality fields (verb, resource type) where the
	// series count stays well under Loki's 500 series limit. High-cardinality
	// fields (username, namespace, resource name) use TextVariable instead
	// because exceeding 500 series causes Loki to return HTTP 400, silently
	// breaking the dropdown. See: https://github.com/perses/perses/issues/4199
	auditVarQueryFmt = `count by (%s) (count_over_time({log_type="audit", openshift_log_source="kubeAPI"} | json [1h]))`

	// OTLP query: openshift_log_source is a stream label (indexed), avoiding post-filter scan.
	// Audit fields still require | json until structured metadata extraction lands upstream.
	auditLogQuery = `{log_type="audit", openshift_log_source="kubeAPI"} | json | user_username!~"${exclude_sa}" | user_username=~"(?i).*(?:${username}).*" | verb=~"${verb}" | objectRef_resource=~"${resource}" | objectRef_resource!~"${exclude_resource}" | objectRef_namespace=~".*(?:${namespace}).*" | objectRef_name=~"(?i).*(?:${resource_name}).*" | responseStatus_code=~"${response_code}" | userAgent=~"(?i).*(?:${client}).*" | line_format "User={{.user_username}} | Verb={{.verb}} | Namespace={{.objectRef_namespace}} | Resource={{.objectRef_resource}} | Resource Name={{.objectRef_name}} | Status={{.responseStatus_code}} | Client={{.userAgent}}" ${filter}`

	excludeSACustomAllValue       = `system:serviceaccount:.*|system:node:.*|system:kube.*|system:openshift.*|system:apiserver.*|system:aggregator.*|system:open-cluster-management:.*|system:ovn-node:.*|system:authenticated.*|system:unauthenticated.*|system:monitoring.*|system:master.*|system:multus.*`
	excludeResourceCustomAllValue = `events|endpoints|endpointslices|leases|tokenreviews|subjectaccessreviews|selfsubjectaccessreviews|selfsubjectrulesreviews`

	auditHelpText = "**Requires:** OpenShift Logging with OTLP data model enabled.\n\n**Filters:** All text filters support regex. Leave empty = match all.\n- **Username:** e.g. `admin`, `.*@example.com`, `user1|user2`\n- **Resource Type:** e.g. `pods`, `deploy.*`, `configmaps|secrets`\n- **Resource Name:** e.g. `my-pod.*`, `nginx`, `etcd.*`\n- **Namespace:** e.g. `openshift-.*`, `my-app`, `kube-system|default`\n- **Client:** e.g. `oc`, `kubectl`, `console`, `argocd`\n\n**LogQL Filter:** Raw stage, e.g. `|~ \"error\"` (include), `!~ \"health\"` (exclude), `| user_username!~\"bot.*\"`\n\n**Tip:** Use shorter time ranges for faster queries."

	auditQueryDisplay = "Active Query: `" + auditLogQuery + "`"
)

// Loki variable plugin specs — the Go SDK does not have builders for these yet
// (only the JS/TS UI has them, added in perses/plugins#651).
// We construct the plugin spec directly, following the same pattern as staticlist.StaticList.

type lokiDatasourceRef struct {
	Kind string `json:"kind"`
	Name string `json:"name,omitempty"`
}

type lokiLogQLVarSpec struct {
	Datasource *lokiDatasourceRef `json:"datasource,omitempty"`
	Expr       string             `json:"expr"`
	LabelName  string             `json:"labelName"`
}

func newLokiDatasourceRef(name string) *lokiDatasourceRef {
	return &lokiDatasourceRef{Kind: "LokiDatasource", Name: name}
}

// lokiLogQLVariable returns a listvariable.Option that populates the variable
// dropdown by running a LogQL expression and extracting unique values of labelName.
func lokiLogQLVariable(datasourceName, expr, labelName string) listvariable.Option {
	return func(builder *listvariable.Builder) error {
		builder.ListVariableSpec.Plugin.Kind = "LokiLogQLVariable"
		builder.ListVariableSpec.Plugin.Spec = lokiLogQLVarSpec{
			Datasource: newLokiDatasourceRef(datasourceName),
			Expr:       expr,
			LabelName:  labelName,
		}
		return nil
	}
}

// labeledValue pairs a query-time value with a human-readable display label.
// The Go SDK's staticlist.Values only accepts plain strings; this bypasses it
// to use the {value, label} object form supported by the StaticListVariable schema.
type labeledValue struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

// staticListWithLabels returns a listvariable.Option that uses StaticListVariable
// with {value, label} pairs so the dropdown shows friendly names while the query
// uses the raw value.
func staticListWithLabels(values ...labeledValue) listvariable.Option {
	return func(builder *listvariable.Builder) error {
		builder.ListVariableSpec.Plugin.Kind = "StaticListVariable"
		builder.ListVariableSpec.Plugin.Spec = struct {
			Values []labeledValue `json:"values"`
		}{Values: values}
		return nil
	}
}

func newAuditDatasource(namespace string, lokiStack *types.NamespacedName) *persesv1alpha2.PersesDatasource {
	gatewayURL := ""
	if lokiStack != nil {
		gatewayURL = fmt.Sprintf("https://%s-gateway-http.%s.svc.cluster.local:8080/api/logs/v1/audit",
			lokiStack.Name, lokiStack.Namespace)
	}

	return &persesv1alpha2.PersesDatasource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha2.GroupVersion.String(),
			Kind:       "PersesDatasource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      auditDatasourceName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha2.DatasourceSpec{
			Config: persesv1alpha2.Datasource{
				Spec: dsSpec.Spec{
					Display: &specCommon.Display{
						Name: "Loki Audit Logs",
					},
					Default: false,
					Plugin: specCommon.Plugin{
						Kind: "LokiDatasource",
						Spec: map[string]interface{}{
							"proxy": map[string]interface{}{
								"kind": "HTTPProxy",
								"spec": map[string]interface{}{
									"url":    gatewayURL,
									"secret": auditDatasourceName + "-secret",
								},
							},
						},
					},
				},
			},
			Client: &persesv1alpha2.Client{
				TLS: &persesv1alpha2.TLS{
					Enable: ptr.To(true),
					CaCert: &persesv1alpha2.Certificate{
						SecretSource: persesv1alpha2.SecretSource{
							Type: persesv1alpha2.SecretSourceTypeFile,
						},
						CertPath: "/ca/service-ca.crt",
					},
				},
			},
		},
	}
}

func buildAuditDashboardOTLP() (dashboard.Builder, error) {
	return dashboard.New("ocp-audit-log-viewer",
		dashboard.Name("Audit Log Viewer"),
		dashboard.DurationAsString("1h"),
		dashboard.RefreshIntervalAsString("0s"),

		dashboard.AddVariable("username",
			textvariable.Text("",
				textvariable.DisplayName("Username"),
				textvariable.Description("Regex filter. Examples: username, .*@redhat.com, admin|user"),
			),
		),
		dashboard.AddVariable("exclude_sa",
			listvariable.List(
				staticlist.StaticList(
					staticlist.Values(
						"^$",
						"system:serviceaccount:.*",
						"system:node:.*",
						"system:kube.*",
						"system:openshift.*",
						"system:apiserver.*",
						"system:aggregator.*",
						"system:open-cluster-management:.*",
						"system:ovn-node:.*",
						"system:authenticated.*",
						"system:unauthenticated.*",
						"system:monitoring.*",
						"system:master.*",
						"system:multus.*",
					),
				),
				listvariable.DisplayName("Exclude System Users"),
				listvariable.Description("Select None to show all users including system accounts"),
				listvariable.AllowAllValue(true),
				listvariable.AllowMultiple(true),
				listvariable.CustomAllValue(excludeSACustomAllValue),
				listvariable.DefaultValue("$__all"),
			),
		),
		dashboard.AddVariable("verb",
			listvariable.List(
				staticlist.StaticList(
					staticlist.Values("create", "update", "patch", "delete", "deletecollection", "get", "list", "watch"),
				),
				listvariable.DisplayName("Verb"),
				listvariable.Description("Filter by API verb"),
				listvariable.AllowAllValue(true),
				listvariable.AllowMultiple(true),
				listvariable.CustomAllValue(".*"),
				listvariable.DefaultValue("$__all"),
			),
		),
		dashboard.AddVariable("resource",
			listvariable.List(
				lokiLogQLVariable(auditDatasourceName, fmt.Sprintf(auditVarQueryFmt, "objectRef_resource"), "objectRef_resource"),
				listvariable.DisplayName("Resource"),
				listvariable.Description("Filter by resource type (populated from audit logs)"),
				listvariable.AllowAllValue(true),
				listvariable.AllowMultiple(true),
				listvariable.CustomAllValue(".*"),
				listvariable.DefaultValue("$__all"),
			),
		),
		dashboard.AddVariable("resource_name",
			textvariable.Text("",
				textvariable.DisplayName("Resource Name"),
				textvariable.Description("Regex filter. Examples: my-pod.*, nginx, etcd.*"),
			),
		),
		dashboard.AddVariable("namespace",
			textvariable.Text("",
				textvariable.DisplayName("Namespace"),
				textvariable.Description("Regex filter. Examples: openshift-.*, my-app, kube-system|default"),
			),
		),
		dashboard.AddVariable("response_code",
			listvariable.List(
				staticListWithLabels(
					labeledValue{Value: "200", Label: "200 OK"},
					labeledValue{Value: "201", Label: "201 Created"},
					labeledValue{Value: "204", Label: "204 No Content"},
					labeledValue{Value: "304", Label: "304 Not Modified"},
					labeledValue{Value: "400", Label: "400 Bad Request"},
					labeledValue{Value: "401", Label: "401 Unauthorized"},
					labeledValue{Value: "403", Label: "403 Forbidden"},
					labeledValue{Value: "404", Label: "404 Not Found"},
					labeledValue{Value: "409", Label: "409 Conflict"},
					labeledValue{Value: "422", Label: "422 Unprocessable"},
					labeledValue{Value: "500", Label: "500 Internal Error"},
					labeledValue{Value: "503", Label: "503 Unavailable"},
				),
				listvariable.DisplayName("Response Code"),
				listvariable.Description("Filter by HTTP response code"),
				listvariable.AllowAllValue(true),
				listvariable.CustomAllValue(".*"),
				listvariable.DefaultValue("$__all"),
			),
		),
		dashboard.AddVariable("exclude_resource",
			listvariable.List(
				staticlist.StaticList(
					staticlist.Values(
						"^$",
						"events",
						"endpoints",
						"endpointslices",
						"leases",
						"tokenreviews",
						"subjectaccessreviews",
						"selfsubjectaccessreviews",
						"selfsubjectrulesreviews",
					),
				),
				listvariable.DisplayName("Exclude Resources"),
				listvariable.Description("Select None to show all resource types"),
				listvariable.AllowAllValue(true),
				listvariable.AllowMultiple(true),
				listvariable.CustomAllValue(excludeResourceCustomAllValue),
				listvariable.DefaultValue("$__all"),
			),
		),
		dashboard.AddVariable("client",
			textvariable.Text("",
				textvariable.DisplayName("Client"),
				textvariable.Description("User agent regex. Examples: oc, kubectl, console, argocd"),
			),
		),
		dashboard.AddVariable("filter",
			textvariable.Text("",
				textvariable.DisplayName("LogQL Filter"),
				textvariable.Description(`Raw LogQL stage. Examples: |~ "error" (include), !~ "health" (exclude), | user_username!~"bot.*"`),
			),
		),

		dashboard.AddPanelGroup("Help",
			panelgroup.Collapsed(true),
			panelgroup.PanelHeight(5),
			panelgroup.AddPanel("Usage Guide",
				markdown.Markdown(auditHelpText),
			),
		),

		dashboard.AddPanelGroup("Active Query",
			panelgroup.Collapsed(true),
			panelgroup.PanelHeight(6),
			panelgroup.AddPanel("Current LogQL Query",
				markdown.Markdown(auditQueryDisplay),
			),
		),

		dashboard.AddPanelGroup("",
			panelgroup.PanelHeight(20),
			panelgroup.AddPanel("Audit Logs",
				logstable.LogsTable(
					logstable.EnableDetails(true),
					logstable.ShowTime(true),
				),
				panel.AddQuery(
					lokiquery.LokiLogQuery(auditLogQuery,
						lokiquery.Datasource(auditDatasourceName),
					),
				),
			),
		),
	)
}

func newAuditDashboardOTLP(namespace string) (*persesv1alpha2.PersesDashboard, error) {
	builder, err := buildAuditDashboardOTLP()
	if err != nil {
		return nil, err
	}

	// Workaround because of type conflict between Perses plugin types and Perses fork in rhobs org
	rhobsDashboard := persesv1.Dashboard{}
	bytes, err := json.Marshal(builder.Dashboard)
	if err != nil {
		return nil, err
	}
	err = rhobsDashboard.UnmarshalJSON(bytes)
	if err != nil {
		return nil, err
	}

	return &persesv1alpha2.PersesDashboard{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha2.GroupVersion.String(),
			Kind:       "PersesDashboard",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocp-audit-log-viewer",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha2.PersesDashboardSpec{
			Config: persesv1alpha2.Dashboard{
				Spec: rhobsDashboard.Spec,
			},
		},
	}, nil
}

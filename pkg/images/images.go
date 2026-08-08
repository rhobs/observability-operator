package images

import obopo "github.com/rhobs/obo-prometheus-operator/pkg/operator"

// DefaultImages map of default image values.
//
// Prometheus and Alertmanager are handled by prometheus-operator.
// For thanos we use the default version from prometheus-operator.
var DefaultImages = map[string]string{
	"alertmanager":                 "",
	"health-analyzer":              "quay.io/openshiftanalytics/cluster-health-analyzer:v1.1.1",
	"korrel8r":                     "quay.io/korrel8r/korrel8r:0.11.1",
	"perses":                       "quay.io/openshift-observability-ui/perses:v0.54.0",
	"prometheus":                   "",
	"thanos":                       obopo.DefaultThanosImage,
	"ui-dashboards":                "quay.io/openshift-observability-ui/console-dashboards-plugin:v0.4.3",
	"ui-distributed-tracing":       "quay.io/openshift-observability-ui/distributed-tracing-console-plugin:v1.1.0",
	"ui-distributed-tracing-pf4":   "quay.io/openshift-observability-ui/distributed-tracing-console-plugin:v0.3.3",
	"ui-distributed-tracing-pf5":   "quay.io/openshift-observability-ui/distributed-tracing-console-plugin:v0.4.3",
	"ui-distributed-tracing-pf6":   "quay.io/openshift-observability-ui/distributed-tracing-console-plugin:v1.0.3",
	"ui-logging":                   "quay.io/openshift-observability-ui/logging-view-plugin:v6.2.1",
	"ui-logging-pf4":               "quay.io/openshift-observability-ui/logging-view-plugin:v6.0.5",
	"ui-logging-pf5":               "quay.io/openshift-observability-ui/logging-view-plugin:v6.1.6",
	"ui-monitoring":                "quay.io/openshift-observability-ui/monitoring-console-plugin:v1.0.0",
	"ui-monitoring-pf5":            "quay.io/openshift-observability-ui/monitoring-console-plugin:v0.4.5",
	"ui-monitoring-pf6":            "quay.io/openshift-observability-ui/monitoring-console-plugin:v0.5.4",
	"ui-troubleshooting-panel":     "quay.io/openshift-observability-ui/troubleshooting-panel-console-plugin:v1.0.0",
	"ui-troubleshooting-panel-pf6": "quay.io/openshift-observability-ui/troubleshooting-panel-console-plugin:v0.4.5",
}

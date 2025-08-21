package uiplugin

import (
	"encoding/json"
	"fmt"

	"github.com/perses/perses/go-sdk/common"
	"github.com/perses/perses/go-sdk/dashboard"
	"github.com/perses/perses/go-sdk/panel"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	listvariable "github.com/perses/perses/go-sdk/variable/list-variable"
	"github.com/perses/plugins/prometheus/sdk/go/query"
	labelvalues "github.com/perses/plugins/prometheus/sdk/go/variable/label-values"
	table "github.com/perses/plugins/table/sdk/go"
	timeseries "github.com/perses/plugins/timeserieschart/sdk/go"
	persesv1alpha1 "github.com/rhobs/perses-operator/api/v1alpha1"
	persesv1 "github.com/rhobs/perses/pkg/model/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func withServiceMetrics(variableMatchers string) dashboard.Option {
	return dashboard.AddPanelGroup("Service Metrics",
		panelgroup.PanelsPerLine(3),
		panelgroup.AddPanel("Request rate",
			timeseries.Chart(),
			panel.AddQuery(
				query.PromQL(
					fmt.Sprintf("sum(rate(traces_span_metrics_calls{%s}[$__rate_interval]))", variableMatchers),
					query.SeriesNameFormat("req/s"),
				),
			),
		),
		panelgroup.AddPanel("Error rate",
			timeseries.Chart(),
			panel.AddQuery(
				query.PromQL(
					fmt.Sprintf("sum(rate(traces_span_metrics_calls{%s, status_code=\"STATUS_CODE_ERROR\"}[$__rate_interval])) or vector(0)", variableMatchers),
					query.SeriesNameFormat("error/s"),
				),
			),
		),
		panelgroup.AddPanel("Duration",
			timeseries.Chart(
				timeseries.WithYAxis(timeseries.YAxis{
					Format: &common.Format{
						Unit: string(common.MilliSecondsUnit),
					},
				}),
				timeseries.WithLegend(timeseries.Legend{
					Position: timeseries.BottomPosition,
				}),
			),
			panel.AddQuery(
				query.PromQL(
					fmt.Sprintf("histogram_quantile(.95, sum(rate(traces_span_metrics_duration_bucket{%s}[$__rate_interval])) by (le))", variableMatchers),
					query.SeriesNameFormat("95th"),
				),
			),
			panel.AddQuery(
				query.PromQL(
					fmt.Sprintf("histogram_quantile(.75, sum(rate(traces_span_metrics_duration_bucket{%s}[$__rate_interval])) by (le))", variableMatchers),
					query.SeriesNameFormat("75th"),
				),
			),
			panel.AddQuery(
				query.PromQL(
					fmt.Sprintf("histogram_quantile(.50, sum(rate(traces_span_metrics_duration_bucket{%s}[$__rate_interval])) by (le))", variableMatchers),
					query.SeriesNameFormat("50th"),
				),
			),
		),
	)
}

func withOperationMetrics(variableMatchers string) dashboard.Option {
	return dashboard.AddPanelGroup("Operations",
		panelgroup.PanelsPerLine(1),
		panelgroup.AddPanel("Operation metrics",
			table.Table(
				table.Transform([]common.Transform{
					{
						Kind: common.MergeSeriesKind,
						Spec: common.MergeSeriesSpec{},
					},
				}),
				table.WithColumnSettings([]table.ColumnSettings{
					{
						Name:          "span_name",
						Header:        "Name",
						EnableSorting: true,
					},
					{
						Name:   "value #1",
						Header: "Request rate",
						Format: &common.Format{
							Unit:          string(common.RequestsPerSecondsUnit),
							DecimalPlaces: 3,
						},
					},
					{
						Name:   "value #2",
						Header: "Error rate",
						Format: &common.Format{
							Unit:          string(common.DecimalUnit),
							DecimalPlaces: 3,
						},
					},
					{
						Name:   "value #3",
						Header: "Duration",
						Format: &common.Format{
							Unit:          string(common.MilliSecondsUnit),
							DecimalPlaces: 3,
						},
					},
					{
						Name: "timestamp",
						Hide: true,
					},
				}),
			),
			panel.AddQuery(
				query.PromQL(
					fmt.Sprintf("sum(rate(traces_span_metrics_calls{%s}[$__rate_interval])) by (span_name) > 0", variableMatchers),
					query.SeriesNameFormat("req/s"),
				),
			),
			panel.AddQuery(
				query.PromQL(
					fmt.Sprintf("sum(rate(traces_span_metrics_calls{%s, status_code=\"STATUS_CODE_ERROR\"}[$__rate_interval])) by (span_name) > 0", variableMatchers),
					query.SeriesNameFormat("Error rate"),
				),
			),
			panel.AddQuery(
				query.PromQL(
					fmt.Sprintf("sum(rate(traces_span_metrics_duration_sum{%s}[5m]) / rate(traces_span_metrics_duration_count{%s}[5m])) by (span_name) > 0", variableMatchers, variableMatchers),
					query.SeriesNameFormat("95th"),
				),
			),
		),
	)
}

func buildAPMDashboard() (dashboard.Builder, error) {
	variableMatchers := "namespace=\"$namespace\", service=\"$collector\", service_name=\"$service\""

	return dashboard.New("apm",
		dashboard.Name("Application Performance Monitoring (APM)"),
		dashboard.AddVariable("namespace",
			listvariable.List(
				listvariable.DisplayName("OTEL Collector Namespace"),
				labelvalues.PrometheusLabelValues("namespace",
					labelvalues.Matchers("traces_span_metrics_calls{}"),
				),
			),
		),
		dashboard.AddVariable("collector",
			listvariable.List(
				listvariable.DisplayName("OTEL Collector"),
				labelvalues.PrometheusLabelValues("service",
					labelvalues.Matchers("traces_span_metrics_calls{namespace=\"$namespace\"}"),
				),
			),
		),
		dashboard.AddVariable("service",
			listvariable.List(
				listvariable.DisplayName("Service"),
				labelvalues.PrometheusLabelValues("service_name",
					labelvalues.Matchers("traces_span_metrics_calls{namespace=\"$namespace\", service=\"$collector\"}"),
				),
			),
		),
		withServiceMetrics(variableMatchers),
		withOperationMetrics(variableMatchers),
	)
}

func newAPMDashboard(namespace string) (*persesv1alpha1.PersesDashboard, error) {
	builder, err := buildAPMDashboard()
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

	return &persesv1alpha1.PersesDashboard{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha1.GroupVersion.String(),
			Kind:       "PersesDashboard",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apm-dashboard",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha1.Dashboard{
			DashboardSpec: rhobsDashboard.Spec,
		},
	}, nil
}

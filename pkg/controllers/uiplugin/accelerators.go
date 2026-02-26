package uiplugin

import (
	"encoding/json"

	"github.com/perses/perses/go-sdk/common"
	"github.com/perses/perses/go-sdk/dashboard"
	"github.com/perses/perses/go-sdk/panel"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	listvariable "github.com/perses/perses/go-sdk/variable/list-variable"
	"github.com/perses/perses/pkg/model/api/v1/variable"
	"github.com/perses/plugins/prometheus/sdk/go/query"
	labelvalues "github.com/perses/plugins/prometheus/sdk/go/variable/label-values"
	timeseries "github.com/perses/plugins/timeserieschart/sdk/go"
	persesv1alpha2 "github.com/rhobs/perses-operator/api/v1alpha2"
	persesv1 "github.com/rhobs/perses/pkg/model/api/v1"
	persescommon "github.com/rhobs/perses/pkg/model/api/v1/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func newAcceleratorsDatasource(namespace string) *persesv1alpha2.PersesDatasource {
	return &persesv1alpha2.PersesDatasource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha2.GroupVersion.String(),
			Kind:       "PersesDatasource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "accelerators-thanos-querier-datasource",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha2.DatasourceSpec{
			Config: persesv1alpha2.Datasource{
				DatasourceSpec: persesv1.DatasourceSpec{
					Display: &persescommon.Display{
						Name: "Accelerators Datasource",
					},
					Default: true,
					Plugin: persescommon.Plugin{
						Kind: "PrometheusDatasource",
						Spec: map[string]interface{}{
							"proxy": map[string]interface{}{
								"kind": "HTTPProxy",
								"spec": map[string]interface{}{
									"url":    "https://thanos-querier.openshift-monitoring.svc.cluster.local:9091",
									"secret": "accelerators-thanos-querier-datasource-secret",
								},
							},
						},
					},
				},
			},
			Client: &persesv1alpha2.Client{
				TLS: &persesv1alpha2.TLS{
					Enable: true,
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

func acceleratorPanel(panelName, targetMetric string) panelgroup.Option {
	return panelgroup.AddPanel(panelName,
		timeseries.Chart(
			timeseries.WithLegend(timeseries.Legend{
				Mode:     timeseries.ListMode,
				Position: timeseries.BottomPosition,
				Values:   []common.Calculation{},
			}),
			timeseries.WithVisual(timeseries.Visual{
				AreaOpacity:  1,
				ConnectNulls: false,
				Display:      timeseries.LineDisplay,
				LineWidth:    0.25,
				Stack:        timeseries.AllStack,
			}),
			timeseries.WithYAxis(timeseries.YAxis{
				Format: &common.Format{
					Unit: ptr.To(string(common.DecimalUnit)),
				},
				Min: 0,
			}),
		),
		panel.AddQuery(
			query.PromQL(targetMetric,
				query.SeriesNameFormat("{{vendor_id}}"),
			),
		),
	)
}

func buildAcceleratorsDashboard() (dashboard.Builder, error) {
	return dashboard.New("accelerators-dashboard",
		dashboard.Name("Accelerators common metrics"),
		dashboard.AddVariable("cluster",
			listvariable.List(
				listvariable.DisplayName("Cluster"),
				listvariable.Hidden(false),
				listvariable.AllowAllValue(false),
				listvariable.AllowMultiple(false),
				listvariable.SortingBy(variable.SortAlphabeticalAsc),
				labelvalues.PrometheusLabelValues("cluster",
					labelvalues.Matchers("up{job=\"kubelet\", metrics_path=\"/metrics/cadvisor\"}"),
				),
			),
		),
		dashboard.AddPanelGroup("Accelerators",
			panelgroup.PanelsPerLine(2),
			acceleratorPanel("GPU Utilization", "accelerator_gpu_utilization"),
			acceleratorPanel("Memory Used Bytes", "accelerator_memory_used_bytes"),
			acceleratorPanel("Memory Total Bytes", "accelerator_memory_total_bytes"),
			acceleratorPanel("Power Usage (Watts)", "accelerator_power_usage_watts"),
			acceleratorPanel("Temperature (Celsius)", "accelerator_temperature_celsius"),
			acceleratorPanel("SM Clock (Hertz)", "accelerator_sm_clock_hertz"),
			acceleratorPanel("Memory Clock (Hertz)", "accelerator_memory_clock_hertz"),
		),
	)
}

func newAcceleratorsDashboard(namespace string) (*persesv1alpha2.PersesDashboard, error) {
	builder, err := buildAcceleratorsDashboard()
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
			Name:      "accelerators-dashboard",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha2.PersesDashboardSpec{
			Config: persesv1alpha2.Dashboard{
				DashboardSpec: rhobsDashboard.Spec,
			},
		},
	}, nil
}

package uiplugin

import (
	persesv1alpha1 "github.com/rhobs/perses-operator/api/v1alpha1"
	persesv1 "github.com/rhobs/perses/pkg/model/api/v1"
	"github.com/rhobs/perses/pkg/model/api/v1/common"
	"github.com/rhobs/perses/pkg/model/api/v1/dashboard"
	"github.com/rhobs/perses/pkg/model/api/v1/variable"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func newAcceleratorsDatasource(namespace string) *persesv1alpha1.PersesDatasource {
	return &persesv1alpha1.PersesDatasource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha1.GroupVersion.String(),
			Kind:       "PersesDatasource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "accelerators-thanos-querier-datasource",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha1.DatasourceSpec{
			Config: persesv1alpha1.Datasource{
				DatasourceSpec: persesv1.DatasourceSpec{
					Display: &common.Display{
						Name: "acceelerators datasource",
					},
					Default: true,
					Plugin: common.Plugin{
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
			Client: &persesv1alpha1.Client{
				TLS: &persesv1alpha1.TLS{
					Enable: true,
					CaCert: &persesv1alpha1.Certificate{
						SecretSource: persesv1alpha1.SecretSource{
							Type: persesv1alpha1.SecretSourceTypeFile,
						},
						CertPath: "/ca/service-ca.crt",
					},
				},
			},
		},
	}
}

func newAcceleratorsDashboard(namespace string) *persesv1alpha1.PersesDashboard {
	return &persesv1alpha1.PersesDashboard{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha1.GroupVersion.String(),
			Kind:       "PersesDashboard",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "accelerators-dashboard",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha1.Dashboard{
			DashboardSpec: persesv1.DashboardSpec{
				Display: &common.Display{
					Name: "Accelerators common metrics",
				},
				Variables: []dashboard.Variable{
					{
						Kind: variable.KindList,
						Spec: &dashboard.ListVariableSpec{
							Name: "cluster",
							ListSpec: variable.ListSpec{
								Display: &variable.Display{
									Hidden: false,
								},
								AllowAllValue: false,
								AllowMultiple: false,
								Sort:          ptr.To(variable.SortAlphabeticalAsc),
								Plugin: common.Plugin{
									Kind: "PrometheusLabelValuesVariable",
									Spec: map[string]interface{}{
										"labelName": "cluster",
										"matchers": []interface{}{
											"up{job=\"kubelet\", metrics_path=\"/metrics/cadvisor\"}",
										},
									},
								},
							},
						},
					},
				},
				Panels: map[string]*persesv1.Panel{
					"0_0": getPanel("GPU Utilization", "accelerator_gpu_utilization"),
					"0_1": getPanel("Memory Used Bytes", "accelerator_memory_used_bytes"),
					"0_2": getPanel("Memory Total Bytes", "accelerator_memory_total_bytes"),
					"0_3": getPanel("Power Usage (Watts)", "accelerator_power_usage_watts"),
					"0_4": getPanel("Temperature (Celsius)", "accelerator_temperature_celsius"),
					"0_5": getPanel("SM Clock (Hertz)", "accelerator_sm_clock_hertz"),
					"0_6": getPanel("Memory Clock (Hertz)", "accelerator_memory_clock_hertz"),
				},
				Layouts: []dashboard.Layout{
					{
						Kind: dashboard.KindGridLayout,
						Spec: dashboard.GridLayoutSpec{
							Display: &dashboard.GridLayoutDisplay{
								Title: "Accelerators",
								Collapse: &dashboard.GridLayoutCollapse{
									Open: true,
								},
							},
							Items: []dashboard.GridItem{
								getGridItem(0, 0, "#/spec/panels/0_0"),
								getGridItem(12, 0, "#/spec/panels/0_1"),
								getGridItem(0, 7, "#/spec/panels/0_2"),
								getGridItem(12, 7, "#/spec/panels/0_3"),
								getGridItem(0, 14, "#/spec/panels/0_4"),
								getGridItem(12, 14, "#/spec/panels/0_5"),
								getGridItem(0, 21, "#/spec/panels/0_6"),
							},
						},
					},
				},
			},
		},
	}
}

func getPanel(panelName, targetMetric string) *persesv1.Panel {
	return &persesv1.Panel{
		Kind: "Panel",
		Spec: persesv1.PanelSpec{
			Display: persesv1.PanelDisplay{
				Name: panelName,
			},
			Plugin: common.Plugin{
				Kind: "TimeSeriesChart",
				Spec: map[string]interface{}{
					"legend": map[string]interface{}{
						"mode":     "list",
						"position": "bottom",
						"values":   []interface{}{}, // Empty array
					},
					"visual": map[string]interface{}{
						"areaOpacity":  1,
						"connectNulls": false,
						"display":      "line",
						"lineWidth":    0.25,
						"stack":        "all",
					},
					"yAxis": map[string]interface{}{
						"format": map[string]interface{}{
							"unit": "decimal",
						},
						"min": 0,
					},
				},
			},
			Queries: []persesv1.Query{
				{
					Kind: "TimeSeriesQuery",
					Spec: persesv1.QuerySpec{
						Plugin: common.Plugin{
							Kind: "PrometheusTimeSeriesQuery",
							Spec: map[string]interface{}{
								"datasource": map[string]interface{}{
									"kind": "PrometheusDatasource",
								},
								"query":            targetMetric,
								"seriesNameFormat": "{{vendor_id}}",
							},
						},
					},
				},
			},
		},
	}
}

func getGridItem(xPos, yPos int, ref string) dashboard.GridItem {
	return dashboard.GridItem{
		X:      xPos,
		Y:      yPos,
		Width:  12,
		Height: 7,
		Content: &common.JSONRef{
			Ref: ref,
		},
	}
}

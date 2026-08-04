package uiplugin

import (
	specCommon "github.com/perses/spec/go/common"
	dsSpec "github.com/perses/spec/go/datasource"
	pluginSpec "github.com/perses/spec/go/plugin"
	persesv1alpha2 "github.com/rhobs/perses-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func newPrometheusGlobalDatasource() *persesv1alpha2.PersesDatasource {
	return &persesv1alpha2.PersesDatasource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha2.GroupVersion.String(),
			Kind:       "PersesGlobalDatasource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "global-thanos-querier-datasource",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha2.DatasourceSpec{
			Config: persesv1alpha2.Datasource{
				Spec: dsSpec.Spec{
					Display: &specCommon.Display{
						Name: "Prometheus Global Datasource",
					},
					Default: true,
					Plugin: pluginSpec.Plugin{
						Kind: "PrometheusDatasource",
						Spec: map[string]interface{}{
							"proxy": map[string]interface{}{
								"kind": "HTTPProxy",
								"spec": map[string]interface{}{
									"url":    "https://thanos-querier.openshift-monitoring.svc.cluster.local:9091",
									"secret": "global-thanos-querier-datasource-secret",
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

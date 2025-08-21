package operator

import (
	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	persesv1alpha1 "github.com/rhobs/perses-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	rhobsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func NewScheme(cfg *OperatorConfiguration) *runtime.Scheme {
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(rhobsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	utilruntime.Must(uiv1alpha1.AddToScheme(scheme))
	utilruntime.Must(obsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(otelv1beta1.AddToScheme(scheme))
	utilruntime.Must(tempov1alpha1.AddToScheme(scheme))

	if cfg.FeatureGates.OpenShift.Enabled {
		utilruntime.Must(osv1.Install(scheme))
		utilruntime.Must(osv1alpha1.Install(scheme))
		utilruntime.Must(operatorv1.Install(scheme))
		utilruntime.Must(corev1.AddToScheme(scheme))
		utilruntime.Must(monv1.AddToScheme(scheme))
		utilruntime.Must(persesv1alpha1.AddToScheme(scheme))
		utilruntime.Must(olmv1alpha1.AddToScheme(scheme))
	}

	return scheme
}

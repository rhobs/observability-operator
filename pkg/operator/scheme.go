package operator

import (
	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	persesv1alpha1 "github.com/perses/perses-operator/api/v1alpha1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
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

	if cfg.FeatureGates.OpenShift.Enabled {
		utilruntime.Must(osv1.Install(scheme))
		utilruntime.Must(osv1alpha1.Install(scheme))
		utilruntime.Must(operatorv1.Install(scheme))
		utilruntime.Must(corev1.AddToScheme(scheme))
		utilruntime.Must(monv1.AddToScheme(scheme))
		utilruntime.Must(persesv1alpha1.AddToScheme(scheme))
	}

	return scheme
}

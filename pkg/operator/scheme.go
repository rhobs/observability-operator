package operator

import (
	rhobslogv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/logging/v1alpha1"
	rhobsmonv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	rhobsuiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observabilityui/v1alpha1"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	openshiftoperatorsv1 "github.com/openshift/api/operator/v1"
	loggingv1 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(rhobsmonv1alpha1.AddToScheme(scheme))
	utilruntime.Must(rhobslogv1alpha1.AddToScheme(scheme))
	utilruntime.Must(rhobsuiv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	utilruntime.Must(lokiv1.AddToScheme(scheme))
	utilruntime.Must(loggingv1.AddToScheme(scheme))
	utilruntime.Must(olmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(olmv1.AddToScheme(scheme))
	utilruntime.Must(openshiftoperatorsv1.AddToScheme(scheme))

	return scheme
}

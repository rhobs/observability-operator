package operator

import (
	"context"
	"fmt"
	"rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"
	poctrl "rhobs/monitoring-stack-operator/pkg/controllers/prometheus-operator"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Operator struct {
	manager manager.Manager
}

func New(metricsAddr string, poOpts poctrl.Options) (*Operator, error) {
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create manager: %w", err)
	}

	if err := poctrl.RegisterWithManager(mgr, poOpts); err != nil {
		return nil, fmt.Errorf("unable to register prometheus-operator controller: %w", err)
	}

	return &Operator{
		manager: mgr,
	}, nil
}

func (o *Operator) Start(ctx context.Context) error {
	if err := o.manager.Start(ctx); err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	return nil
}

func (o *Operator) GetClient() client.Client {
	return o.manager.GetClient()
}

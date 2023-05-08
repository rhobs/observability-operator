package operator

import (
	"context"
	"fmt"

	stackctrl "github.com/rhobs/observability-operator/pkg/controllers/monitoring/monitoring-stack"
	tqctrl "github.com/rhobs/observability-operator/pkg/controllers/monitoring/thanos-querier"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NOTE: The instance selector label is hardcoded in static assets.
// Any change to that must be reflected here as well
const instanceSelector = "app.kubernetes.io/managed-by=observability-operator"

const ObservabilityOperatorName = "observability-operator"

// Operator embedds manager and exposes only the minimal set of functions
type Operator struct {
	manager manager.Manager
}

func New(metricsAddr, healthProbeAddr string) (*Operator, error) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 NewScheme(),
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: healthProbeAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create manager: %w", err)
	}

	if err := stackctrl.RegisterWithManager(mgr, stackctrl.Options{InstanceSelector: instanceSelector}); err != nil {
		return nil, fmt.Errorf("unable to register monitoring stack controller: %w", err)
	}

	if err := tqctrl.RegisterWithManager(mgr); err != nil {
		return nil, fmt.Errorf("unable to register the thanos querier controller with the manager: %w", err)
	}

	if err := mgr.AddHealthzCheck("health probe", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to add health probe: %w", err)
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

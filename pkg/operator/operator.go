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
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// NOTE: The instance selector label is hardcoded in static assets.
// Any change to that must be reflected here as well
const instanceSelector = "app.kubernetes.io/managed-by=observability-operator"

const ObservabilityOperatorName = "observability-operator"

// Operator embedds manager and exposes only the minimal set of functions
type Operator struct {
	manager manager.Manager
}

type OperandConfiguration struct {
	Image   string
	Version string
}

type OperatorConfiguration struct {
	MetricsAddr     string
	HealthProbeAddr string
	Prometheus      stackctrl.PrometheusConfiguration
	Alertmanager    stackctrl.AlertmanagerConfiguration
	ThanosSidecar   stackctrl.ThanosConfiguration
	ThanosQuerier   tqctrl.ThanosConfiguration
}

func NewOperatorConfiguration(metricsAddr string, healthProbeAddr string, images map[string]string) OperatorConfiguration {
	return OperatorConfiguration{
		MetricsAddr:     metricsAddr,
		HealthProbeAddr: healthProbeAddr,
		Prometheus:      stackctrl.PrometheusConfiguration{Image: images["prometheus"]},
		Alertmanager:    stackctrl.AlertmanagerConfiguration{Image: images["alertmanager"]},
		ThanosSidecar:   stackctrl.ThanosConfiguration{Image: images["thanos"]},
		ThanosQuerier:   tqctrl.ThanosConfiguration{Image: images["thanos"]},
	}
}

func New(cfg OperatorConfiguration) (*Operator, error) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: NewScheme(),
		Metrics: metricsserver.Options{
			BindAddress: cfg.MetricsAddr,
		},
		HealthProbeBindAddress: cfg.HealthProbeAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create manager: %w", err)
	}

	if err := stackctrl.RegisterWithManager(mgr, stackctrl.Options{
		InstanceSelector: instanceSelector,
		Prometheus:       cfg.Prometheus,
		Alertmanager:     cfg.Alertmanager}); err != nil {
		return nil, fmt.Errorf("unable to register monitoring stack controller: %w", err)
	}

	if err := tqctrl.RegisterWithManager(mgr, tqctrl.Options{Thanos: cfg.ThanosQuerier}); err != nil {
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

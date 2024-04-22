package operator

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	stackctrl "github.com/rhobs/observability-operator/pkg/controllers/monitoring/monitoring-stack"
	tqctrl "github.com/rhobs/observability-operator/pkg/controllers/monitoring/thanos-querier"
	uictrl "github.com/rhobs/observability-operator/pkg/controllers/uiplugin"
)

// NOTE: The instance selector label is hardcoded in static assets.
// Any change to that must be reflected here as well
const instanceSelector = "app.kubernetes.io/managed-by=observability-operator"

const ObservabilityOperatorName = "observability-operator"

// Operator embedds manager and exposes only the minimal set of functions
type Operator struct {
	manager manager.Manager
}

type OpenShiftFeatureGates struct {
	Enabled bool `json:"enabled,omitempty"`
}

type FeatureGates struct {
	OpenShift OpenShiftFeatureGates `json:"openshift,omitempty"`
}

type OperatorConfiguration struct {
	MetricsAddr     string
	HealthProbeAddr string
	Prometheus      stackctrl.PrometheusConfiguration
	Alertmanager    stackctrl.AlertmanagerConfiguration
	ThanosSidecar   stackctrl.ThanosConfiguration
	ThanosQuerier   tqctrl.ThanosConfiguration
	UIPlugins       uictrl.UIPluginsConfiguration
	FeatureGates    FeatureGates
}

func WithPrometheusImage(image string) func(*OperatorConfiguration) {
	return func(oc *OperatorConfiguration) {
		oc.Prometheus.Image = image
	}
}

func WithAlertmanagerImage(image string) func(*OperatorConfiguration) {
	return func(oc *OperatorConfiguration) {
		oc.Alertmanager.Image = image
	}
}

func WithThanosSidecarImage(image string) func(*OperatorConfiguration) {
	return func(oc *OperatorConfiguration) {
		oc.ThanosSidecar.Image = image
	}
}

func WithThanosQuerierImage(image string) func(*OperatorConfiguration) {
	return func(oc *OperatorConfiguration) {
		oc.ThanosQuerier.Image = image
	}
}

func WithMetricsAddr(addr string) func(*OperatorConfiguration) {
	return func(oc *OperatorConfiguration) {
		oc.MetricsAddr = addr
	}
}

func WithHealthProbeAddr(addr string) func(*OperatorConfiguration) {
	return func(oc *OperatorConfiguration) {
		oc.HealthProbeAddr = addr
	}
}

func WithUIPlugins(namespace string, images map[string]string) func(*OperatorConfiguration) {
	return func(oc *OperatorConfiguration) {
		oc.UIPlugins.Images = images
		oc.UIPlugins.ResourcesNamespace = namespace
	}
}

func WithFeatureGates(featureGates FeatureGates) func(*OperatorConfiguration) {
	return func(oc *OperatorConfiguration) {
		oc.FeatureGates = featureGates
	}
}

func NewOperatorConfiguration(opts ...func(*OperatorConfiguration)) *OperatorConfiguration {
	cfg := &OperatorConfiguration{}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

func New(cfg *OperatorConfiguration) (*Operator, error) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: NewScheme(cfg),
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
		Alertmanager:     cfg.Alertmanager,
		Thanos:           cfg.ThanosSidecar,
	}); err != nil {
		return nil, fmt.Errorf("unable to register monitoring stack controller: %w", err)
	}

	if err := tqctrl.RegisterWithManager(mgr, tqctrl.Options{Thanos: cfg.ThanosQuerier}); err != nil {
		return nil, fmt.Errorf("unable to register the thanos querier controller with the manager: %w", err)
	}

	if cfg.FeatureGates.OpenShift.Enabled {
		if err := uictrl.RegisterWithManager(mgr, uictrl.Options{PluginsConf: cfg.UIPlugins}); err != nil {
			return nil, fmt.Errorf("unable to register observability-ui-plugin controller: %w", err)
		}
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

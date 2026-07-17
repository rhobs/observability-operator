package observability

import (
	"fmt"
	"io/fs"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/overlay"
)

type OverlayConfig struct {
	ConfigFS              fs.FS
	COOName               string
	COONamespace          string
	OpenTelemetryOperator OperatorInstallConfig
	TempoOperator         OperatorInstallConfig
}

const (
	PlaceholderInstallerName      = "placeholder-observability-installer-name"
	PlaceholderInstallerNamespace = "placeholder-observability-installer-namespace"
	PlaceholderOperatorNamespace  = "placeholder-operator-namespace"
)

func BuildOverlay(instance *obsv1alpha1.ObservabilityInstaller, cfg OverlayConfig) (*overlay.Overlay, error) {
	ns := instance.Namespace
	if ns == "" {
		ns = "default"
	}

	o := overlay.New(cfg.ConfigFS)
	o.SetNamespace(cfg.COONamespace)
	o.AddSubstitution(PlaceholderInstallerNamespace, ns)
	o.AddSubstitution(PlaceholderOperatorNamespace, cfg.COONamespace)
	o.AddSubstitution(PlaceholderInstallerName, instance.Name)

	tracing := instance.Spec.GetCapabilities().GetTracing()
	if tracing != nil && tracing.Enabled {
		o.SetBase("../../observabilityinstaller/base")
		if err := addOtelCollector(o, instance, cfg.OpenTelemetryOperator); err != nil {
			return nil, fmt.Errorf("building OTel collector overlay: %w", err)
		}
		if err := addTempoStack(o, instance, cfg.TempoOperator); err != nil {
			return nil, fmt.Errorf("building TempoStack overlay: %w", err)
		}
	} else if tracing != nil && tracing.GetOperators() != nil &&
		tracing.GetOperators().Install != nil && *tracing.GetOperators().Install {
		o.SetBase("../../observabilityinstaller/base")
		o.AddComponent("../../observabilityinstaller/components/collectors/tracing/operator")
		if err := addSubscriptionPatch(o, "opentelemetry-product", cfg.OpenTelemetryOperator); err != nil {
			return nil, err
		}
		o.AddComponent("../../observabilityinstaller/components/stores/tempostack/operator")
		if err := addSubscriptionPatch(o, "tempo-product", cfg.TempoOperator); err != nil {
			return nil, err
		}
	}

	return o, nil
}

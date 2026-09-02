package observability

import (
	"context"
	"fmt"
	"io/fs"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
	"github.com/rhobs/observability-operator/pkg/overlay"
)

type OverlayConfig struct {
	Options
	Scheme    *runtime.Scheme
	ConfigFS  fs.FS
	Operators bool // include operator components
	Resources bool // include resource components
}

func (cfg *OverlayConfig) Capabilities(cc obsv1alpha1.CommonCapabilitiesSpec) {
	cfg.Resources = cfg.Resources && cc.Enabled
	cfg.Operators = cfg.Operators && cc.InstallOperators()
}

// ResolveInstaller builds the object set that reconciles an
// ObservabilityInstaller: the overlay build output plus, when resources are
// enabled, any secrets assembled from source secrets read through reader.
// It is shared by the reconciler and the generator so both produce the same
// objects.
func ResolveInstaller(ctx context.Context, reader client.Reader, instance *obsv1alpha1.ObservabilityInstaller, cfg OverlayConfig) ([]client.Object, error) {
	o, err := BuildInstallerOverlay(instance, cfg)
	if err != nil {
		return nil, fmt.Errorf("building overlay for %s: %w", instance.Name, err)
	}
	objects, err := o.Build()
	if err != nil {
		return nil, fmt.Errorf("building overlay for %s: %w", instance.Name, err)
	}
	if cfg.Resources {
		if tracing := instance.Spec.GetCapabilities().GetTracing(); tracing != nil && tracing.Enabled {
			secrets, err := BuildTempoSecrets(ctx, reader, *instance)
			if err != nil {
				return nil, fmt.Errorf("building secrets for %s: %w", instance.Name, err)
			}
			for _, s := range secrets {
				util.AddCommonLabels(s, instance.Name)
			}
			objects = append(objects, secrets...)
		}
	}
	return objects, nil
}

// BuildInstallerOverlay creates an overlay that modifies the base configuration from the config/ directory
func BuildInstallerOverlay(instance *obsv1alpha1.ObservabilityInstaller, cfg OverlayConfig) (*overlay.Overlay, error) {
	ns := instance.Namespace
	if ns == "" {
		return nil, fmt.Errorf("no namespace: ObservabilityInstaller/%v", instance.Name)
	}
	o := overlay.New(cfg.Scheme, cfg.ConfigFS)
	o.SetNamespace(ns)
	o.SetOwnerName(instance.Name)
	o.SetClusterScoped("observability.openshift.io", "UIPlugin")

	// Cross-references first: collector SA name in CRB subjects (before CRB names change)
	o.ReplaceValue(instance.Name+"-collector",
		overlay.TargetKindName("ClusterRoleBinding", "coo-otelcol-tracing-components", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("ClusterRoleBinding", "coo-otelcol-tracing-tempo", "subjects.[kind=ServiceAccount].name"),
	)

	// Installer name on TempoStack and OpenTelemetryCollector
	o.ReplaceValue(instance.Name,
		overlay.TargetName("tempostack", "metadata.name"),
		overlay.TargetName("tracing", "metadata.name"),
	)

	// RBAC compound names (changes metadata.name, must be after cross-refs).
	// ClusterRole and ClusterRoleBinding are targeted separately so roleRef.name
	// is set on the CRB in the same pass as its metadata.name.
	o.ReplaceValue("coo-otelcol-"+instance.Name+"-components",
		overlay.TargetKindName("ClusterRole", "coo-otelcol-tracing-components", "metadata.name"),
		overlay.TargetKindName("ClusterRoleBinding", "coo-otelcol-tracing-components", "metadata.name", "roleRef.name"),
	)
	o.ReplaceValue("coo-otelcol-"+instance.Name+"-tempo",
		overlay.TargetKindName("ClusterRole", "coo-otelcol-tracing-tempo", "metadata.name"),
		overlay.TargetKindName("ClusterRoleBinding", "coo-otelcol-tracing-tempo", "metadata.name", "roleRef.name"),
	)

	if instance.Spec.Capabilities == nil {
		return o, nil
	}

	if err := addTracing(o, instance, cfg); err != nil {
		return o, err
	}
	return o, nil
}

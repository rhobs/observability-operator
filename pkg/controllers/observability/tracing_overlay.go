package observability

import (
	"fmt"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/overlay"
)

func addTracing(o *overlay.Overlay, instance *obsv1alpha1.ObservabilityInstaller, cfg OverlayConfig) error {
	tracing := instance.Spec.Capabilities.Tracing
	if tracing == nil {
		return nil
	}
	cfg.Capabilities(tracing.CommonCapabilitiesSpec)
	if cfg.Operators {
		// operator/ subscriptions go in COONamespace, not the instance namespace.
		o.ReplaceValue(cfg.COONamespace,
			overlay.TargetKindName("Subscription", "opentelemetry-product", "metadata.namespace"),
			overlay.TargetKindName("Subscription", "tempo-product", "metadata.namespace"),
		)
	}
	if cfg.Resources {
		if err := addOtelCollector(o, instance, cfg); err != nil {
			return fmt.Errorf("building OTel collector overlay: %w", err)
		}
		if err := addTempoStack(o, instance, cfg); err != nil {
			return fmt.Errorf("building TempoStack overlay: %w", err)
		}
	}
	return nil
}

func addOtelCollector(o *overlay.Overlay, instance *obsv1alpha1.ObservabilityInstaller, cfg OverlayConfig) error {
	if cfg.Operators {
		o.AddComponent("observabilityinstaller/components/tracing/collector/operator")
		if err := addSubscriptionPatch(o, "opentelemetry-product", cfg.OpenTelemetryOperator); err != nil {
			return err
		}
	}

	if cfg.Resources {
		o.AddComponent("observabilityinstaller/components/tracing/collector/resources")

		endpoint := fmt.Sprintf(
			"https://tempo-%s-gateway.%s.svc.cluster.local:8080/api/traces/v1/%s",
			tempoName(instance.Name), instance.Namespace, TracingTenant,
		)

		if err := o.AddPatch("patches/opentelemetrycollector.yaml", map[string]any{
			"apiVersion": "opentelemetry.io/v1beta1",
			"kind":       "OpenTelemetryCollector",
			"metadata":   map[string]any{"name": "tracing"},
			"spec": map[string]any{
				"config": map[string]any{
					"exporters": map[string]any{
						"otlphttp/tempo": map[string]any{"endpoint": endpoint},
					},
				},
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func addTempoStack(o *overlay.Overlay, instance *obsv1alpha1.ObservabilityInstaller, cfg OverlayConfig) error {
	if cfg.Operators {
		o.AddComponent("observabilityinstaller/components/tracing/store/operator")
		if err := addSubscriptionPatch(o, "tempo-product", cfg.TempoOperator); err != nil {
			return err
		}
	}

	if cfg.Resources {
		o.AddComponent("observabilityinstaller/components/tracing/store/resources")

		storage := instance.Spec.GetCapabilities().GetTracing().GetStorage()
		oss := storage.GetObjectStorageSpec()

		secretMap := map[string]any{
			"type":           string(toTempoStorageType(oss)),
			"credentialMode": string(toTempoCredentialMode(oss)),
			"name":           tempoSecretName(instance.Name),
		}

		storageMap := map[string]any{"secret": secretMap}

		if oss != nil {
			tls := oss.GetTLS()
			enableTLS := tls != nil || s3hasHTTPSEndpoint(*oss)
			if enableTLS {
				tlsMap := map[string]any{"enabled": true}
				if tls != nil {
					if tls.CAConfigMap != nil {
						tlsMap["ca"] = tempoStorageCAConfigMapName(instance.Name)
					}
					if tls.CertSecret != nil {
						tlsMap["cert"] = tempoStorageSecretName(instance.Name)
					}
					if tls.MinVersion != "" {
						tlsMap["minVersion"] = tls.MinVersion
					}
				}
				storageMap["tls"] = tlsMap
			}
		}

		if err := o.AddPatch("patches/tempostack.yaml", map[string]any{
			"apiVersion": "tempo.grafana.com/v1alpha1",
			"kind":       "TempoStack",
			"metadata":   map[string]any{"name": "tempostack"},
			"spec":       map[string]any{"storage": storageMap},
		}); err != nil {
			return err
		}
	}

	return nil
}

func addSubscriptionPatch(o *overlay.Overlay, subscriptionName string, cfg OperatorInstallConfig) error {
	spec := map[string]any{}
	if cfg.StartingCSV != "" {
		spec["startingCSV"] = cfg.StartingCSV
	}
	if cfg.Channel != "" {
		spec["channel"] = cfg.Channel
	}
	if len(spec) == 0 {
		return nil
	}
	return o.AddPatch(fmt.Sprintf("patches/subscription-%s.yaml", subscriptionName), map[string]any{
		"apiVersion": "operators.coreos.com/v1alpha1",
		"kind":       "Subscription",
		"metadata":   map[string]any{"name": subscriptionName, "namespace": "placeholder"},
		"spec":       spec,
	})
}

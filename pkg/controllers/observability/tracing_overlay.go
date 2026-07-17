package observability

import (
	"fmt"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/overlay"
)

func addOtelCollector(o *overlay.Overlay, instance *obsv1alpha1.ObservabilityInstaller, cfg OperatorInstallConfig) error {
	o.AddComponent("../../observabilityinstaller/components/collectors/tracing/operator")
	o.AddComponent("../../observabilityinstaller/components/collectors/tracing/resources")
	if err := addSubscriptionPatch(o, "opentelemetry-product", cfg); err != nil {
		return err
	}

	endpoint := fmt.Sprintf(
		"https://tempo-%s-gateway.%s.svc.cluster.local:8080/api/traces/v1/%s",
		tempoName(instance.Name), instance.Namespace, tenantName,
	)

	return o.AddPatchMap("patches/opentelemetrycollector.yaml", map[string]any{
		"apiVersion": "opentelemetry.io/v1beta1",
		"kind":       "OpenTelemetryCollector",
		"metadata":   map[string]any{"name": PlaceholderInstallerName, "namespace": PlaceholderInstallerNamespace},
		"spec": map[string]any{
			"config": map[string]any{
				"exporters": map[string]any{
					"otlphttp/tempo": map[string]any{"endpoint": endpoint},
				},
			},
		},
	})
}

func addTempoStack(o *overlay.Overlay, instance *obsv1alpha1.ObservabilityInstaller, cfg OperatorInstallConfig) error {
	o.AddComponent("../../observabilityinstaller/components/stores/tempostack/operator")
	o.AddComponent("../../observabilityinstaller/components/stores/tempostack/resources")
	if err := addSubscriptionPatch(o, "tempo-product", cfg); err != nil {
		return err
	}

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

	return o.AddPatchMap("patches/tempostack.yaml", map[string]any{
		"apiVersion": "tempo.grafana.com/v1alpha1",
		"kind":       "TempoStack",
		"metadata":   map[string]any{"name": PlaceholderInstallerName, "namespace": PlaceholderInstallerNamespace},
		"spec":       map[string]any{"storage": storageMap},
	})
}

func addSubscriptionPatch(o *overlay.Overlay, subscriptionName string, cfg OperatorInstallConfig) error {
	if cfg.StartingCSV == "" && cfg.Channel == "" {
		return nil
	}
	spec := map[string]any{}
	if cfg.StartingCSV != "" {
		spec["startingCSV"] = cfg.StartingCSV
	}
	if cfg.Channel != "" {
		spec["channel"] = cfg.Channel
	}
	metadata := map[string]any{"name": subscriptionName}
	if cfg.Namespace != "" {
		metadata["namespace"] = cfg.Namespace
	}
	return o.AddPatchMap(fmt.Sprintf("patches/subscription-%s.yaml", subscriptionName), map[string]any{
		"apiVersion": "operators.coreos.com/v1alpha1",
		"kind":       "Subscription",
		"metadata":   metadata,
		"spec":       spec,
	})
}

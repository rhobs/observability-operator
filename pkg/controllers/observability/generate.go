package observability

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

// operatorsRequested reports whether the tracing spec requests installing the
// OTel and Tempo operators.
func operatorsRequested(tracing *obsv1alpha1.TracingSpec) bool {
	if tracing == nil {
		return false
	}
	if tracing.Enabled {
		return true
	}
	return tracing.GetOperators() != nil && tracing.GetOperators().Install != nil && *tracing.GetOperators().Install
}

// GenerateAllInstallerObjects returns the universe of operand objects across all
// possible capabilities. The reconciler uses this to know which objects to clean
// up when capabilities are disabled.
func GenerateAllInstallerObjects(ctx context.Context, reader client.Reader, instance *obsv1alpha1.ObservabilityInstaller, opts Options) ([]client.Object, error) {
	allInstance := instance.DeepCopy()
	if allInstance.Spec.Capabilities == nil {
		allInstance.Spec.Capabilities = &obsv1alpha1.CapabilitiesSpec{}
	}
	if allInstance.Spec.Capabilities.Tracing == nil {
		allInstance.Spec.Capabilities.Tracing = &obsv1alpha1.TracingSpec{}
	}
	allInstance.Spec.Capabilities.Tracing.Enabled = true
	return GenerateInstallerObjects(ctx, reader, allInstance, opts, false, true)
}

// GenerateInstallerObjects builds the object set that reconciles an ObservabilityInstaller.
//
// It is called by the reconciler and the offline generator so both produce the same objects.
// Subscriptions are included if the CR requests operator installation.
// When installOperators is false, Subscriptions are omitted; when installResources is false, operand resources are omitted.
func GenerateInstallerObjects(ctx context.Context, reader client.Reader, instance *obsv1alpha1.ObservabilityInstaller, opts Options, installOperators, installResources bool) ([]client.Object, error) {
	var objects []client.Object
	tracing := instance.Spec.GetCapabilities().GetTracing()

	if installOperators && operatorsRequested(tracing) {
		objects = append(objects, subscription(opts.OpenTelemetryOperator))
		objects = append(objects, subscription(opts.TempoOperator))
	}

	if installResources && tracing != nil && tracing.Enabled {
		otelCol, err := otelCollector(instance)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenTelemetryCollector: %w", err)
		}
		objects = append(objects, otelCol)

		otelcolRBAC, otelcolRBACBinding := otelCollectorComponentsRBAC(instance)
		objects = append(objects, otelcolRBAC, otelcolRBACBinding)

		objects = append(objects, tempoStack(instance))

		secrets, err := tempoStackSecrets(ctx, reader, *instance)
		if err != nil {
			return nil, fmt.Errorf("failed to create TempoStack secret: %w", err)
		}
		if secrets.objectStorage != nil {
			objects = append(objects, secrets.objectStorage)
		}
		if secrets.objectStorageTLSSecret != nil {
			objects = append(objects, secrets.objectStorageTLSSecret)
		}
		if secrets.objectStorageCAConfigMap != nil {
			objects = append(objects, secrets.objectStorageCAConfigMap)
		}

		otelcolTempoRBAC, otelcolTempoRBACBinding := otelCollectorTempoRBAC(instance)
		objects = append(objects, otelcolTempoRBAC, otelcolTempoRBACBinding)

		objects = append(objects, uiPlugin())
	}

	return objects, nil
}

package tracing

import (
	"context"
	"fmt"
	"slices"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	storage "github.com/rhobs/observability-operator/pkg/controllers/observability/storage"
)

// Options contains only the operators used by this capability.
type Options struct {
	OpenTelemetryOperator capability.OperatorConfig
	TempoOperator         capability.OperatorConfig
}

// New constructs the tracing capability.
func New(opts Options) capability.Definition {
	return capability.Definition{
		Name:         "tracing",
		Shared:       func(instance *obsv1alpha1.ObservabilityInstaller) capability.Plan { return shared(instance, opts) },
		Plan:         planTracing,
		UpdateStatus: updateTracingStatus,
		WatchTypes:   []client.Object{&otelv1beta1.OpenTelemetryCollector{}, &tempov1alpha1.TempoStack{}},
	}
}

func planTracing(ctx context.Context, instance *obsv1alpha1.ObservabilityInstaller, reader client.Reader) (capability.Plan, error) {
	otelCol, err := otelCollector(instance)
	if err != nil {
		return capability.Plan{}, fmt.Errorf("create OpenTelemetryCollector: %w", err)
	}
	objects := []client.Object{otelCol}
	otelRole, otelBinding := otelCollectorComponentsRBAC(instance)
	objects = append(objects, otelRole, otelBinding, tempoStack(instance))
	tempoRole, tempoBinding := otelCollectorTempoRBAC(instance)
	objects = append(objects, tempoRole, tempoBinding)
	names := storage.Names{Secret: tempoSecretName(instance.Name), TLSSecret: tempoStorageSecretName(instance.Name), CAConfigMap: tempoStorageCAConfigMapName(instance.Name)}
	plan := capability.Plan{Owned: slices.Concat(objects, legacyOTelCollectorRBACInventory(ctx, reader, instance), storage.Inventory(instance.Namespace, names))}
	if !tracingEnabled(instance) {
		return plan, nil
	}

	secrets, err := tempoStackSecrets(ctx, reader, *instance)
	if err != nil {
		return capability.Plan{}, fmt.Errorf("create TempoStack storage resources: %w", err)
	}
	plan.Desired = slices.Concat(objects, secrets.Objects())
	return plan, nil
}

func updateTracingStatus(ctx context.Context, instance *obsv1alpha1.ObservabilityInstaller, reader client.Reader) capability.StatusResult {
	instance.Status.Tempo = ""
	instance.Status.OpenTelemetry = ""
	if !tracingEnabled(instance) {
		return capability.StatusResult{}
	}

	otelcol := &otelv1beta1.OpenTelemetryCollector{}
	if err := reader.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: otelCollectorName(instance.Name)}, otelcol); err != nil {
		return capability.StatusReadError("OpenTelemetryCollector", err)
	}
	tempo := &tempov1alpha1.TempoStack{}
	if err := reader.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: tempoName(instance.Name)}, tempo); err != nil {
		return capability.StatusReadError("TempoStack", err)
	}

	instance.Status.Tempo = fmt.Sprintf("%s/%s (%s)", instance.Namespace, tempoName(instance.Name), tempo.Status.TempoVersion)
	instance.Status.OpenTelemetry = fmt.Sprintf("%s/%s (%s)", instance.Namespace, otelCollectorName(instance.Name), otelcol.Status.Version)
	return capability.StatusResult{}
}

func tracingEnabled(instance *obsv1alpha1.ObservabilityInstaller) bool {
	spec := instance.Spec.GetCapabilities().GetTracing()
	return instance.DeletionTimestamp == nil && spec != nil && spec.Enabled
}

// shared evaluates cluster-wide requirements without reading operand data.
func shared(instance *obsv1alpha1.ObservabilityInstaller, opts Options) capability.Plan {
	spec := instance.Spec.GetCapabilities().GetTracing()
	enabled, install := tracingEnabled(instance), false
	if instance.DeletionTimestamp == nil && spec != nil {
		operators := spec.GetOperators()
		install = operators != nil && operators.Install != nil && *operators.Install
	}
	plan := capability.Plan{Operators: capability.DesiredOperators(enabled || install,
		capability.NewOperatorRequirement("opentelemetry", opts.OpenTelemetryOperator),
		capability.NewOperatorRequirement("tempo", opts.TempoOperator),
	)}
	plan.Owned = []client.Object{uiPlugin()}
	if enabled {
		plan.Desired = plan.Owned
	}
	return plan
}

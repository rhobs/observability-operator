package tracing

import (
	"context"
	"fmt"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
)

// Options contains only the operators used by this capability.
type Options struct {
	OpenTelemetryOperator capability.OperatorConfig
	TempoOperator         capability.OperatorConfig
}

type tracingCapability struct {
	opts Options
}

// New constructs the tracing capability.
func New(opts Options) capability.Capability {
	return tracingCapability{opts: opts}
}

func (tracingCapability) Name() string { return "tracing" }

func (tracingCapability) WatchTypes() []client.Object {
	return []client.Object{&otelv1beta1.OpenTelemetryCollector{}, &tempov1alpha1.TempoStack{}}
}

func (t tracingCapability) Operators() []capability.OperatorRequirement {
	return []capability.OperatorRequirement{
		capability.NewOperatorRequirement("opentelemetry", t.opts.OpenTelemetryOperator),
		capability.NewOperatorRequirement("tempo", t.opts.TempoOperator),
	}
}

func (tracingCapability) OperatorsDesired(instance *obsv1alpha1.ObservabilityInstaller) bool {
	if tracingEnabled(instance) {
		return true
	}
	spec := instance.Spec.GetCapabilities().GetTracing()
	if spec == nil {
		return false
	}
	operators := spec.GetOperators()
	return operators != nil && operators.Install != nil && *operators.Install
}

func (tracingCapability) SharedInventory() []client.Object {
	return []client.Object{uiPlugin()}
}

func (t tracingCapability) SharedDesired(instance *obsv1alpha1.ObservabilityInstaller) []client.Object {
	if !tracingEnabled(instance) {
		return nil
	}
	return t.SharedInventory()
}

func (tracingCapability) Inventory(instance *obsv1alpha1.ObservabilityInstaller) []client.Object {
	otelcolRBAC, otelcolRBACBinding := otelCollectorComponentsRBAC(instance)
	otelcolTempoRBAC, otelcolTempoRBACBinding := otelCollectorTempoRBAC(instance)
	objectMeta := func(name string) metav1.ObjectMeta {
		return metav1.ObjectMeta{Name: name, Namespace: instance.Namespace}
	}
	secretType := metav1.TypeMeta{Kind: "Secret", APIVersion: corev1.SchemeGroupVersion.String()}
	return []client.Object{
		&otelv1beta1.OpenTelemetryCollector{
			TypeMeta:   metav1.TypeMeta{Kind: "OpenTelemetryCollector", APIVersion: otelv1beta1.GroupVersion.String()},
			ObjectMeta: objectMeta(otelCollectorName(instance.Name)),
		},
		otelcolRBAC, otelcolRBACBinding, tempoStack(instance),
		&corev1.Secret{TypeMeta: secretType, ObjectMeta: objectMeta(tempoSecretName(instance.Name))},
		&corev1.Secret{TypeMeta: secretType, ObjectMeta: objectMeta(tempoStorageSecretName(instance.Name))},
		&corev1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: corev1.SchemeGroupVersion.String()},
			ObjectMeta: objectMeta(tempoStorageCAConfigMapName(instance.Name)),
		},
		otelcolTempoRBAC, otelcolTempoRBACBinding,
	}
}

func (tracingCapability) Desired(ctx context.Context, instance *obsv1alpha1.ObservabilityInstaller, reader client.Reader) ([]client.Object, error) {
	if !tracingEnabled(instance) {
		return nil, nil
	}
	otelCol, err := otelCollector(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetryCollector: %w", err)
	}
	objects := []client.Object{otelCol}
	otelcolRBAC, otelcolRBACBinding := otelCollectorComponentsRBAC(instance)
	objects = append(objects, otelcolRBAC, otelcolRBACBinding, tempoStack(instance))

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
	return append(objects, otelcolTempoRBAC, otelcolTempoRBACBinding), nil
}

func (tracingCapability) UpdateStatus(ctx context.Context, instance *obsv1alpha1.ObservabilityInstaller, reader client.Reader) capability.StatusResult {
	return updateTracingStatus(ctx, instance, reader)
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

package observability

import (
	"fmt"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

func getReconcilers(instance *obsv1alpha1.ClusterObservability, opts Options, storageSecret *corev1.Secret, subsByCSVName map[string]olmv1alpha1.Subscription) ([]reconciler.Reconciler, error) {
	var reconcilers []reconciler.Reconciler
	var operatorObjects []client.Object
	var instanceObjects []client.Object
	installedObjects := map[string]client.Object{}

	// the OTEL and Tempo operators are rolling release, meaning only the latest released versions are supported.
	// At the moment there are no compatibility issues between the operands of these two operators, so we can
	// install them together in any versions.
	if _, otelOperatorInstalled := subsByCSVName["opentelemetry-operator"]; !otelOperatorInstalled {
		operatorObjects = append(operatorObjects, subscription(opts.OpenTelemetryOperator))
	}
	if _, tempoOperatorInstalled := subsByCSVName["tempo-operator"]; !tempoOperatorInstalled {
		operatorObjects = append(operatorObjects, subscription(opts.TempoOperator))
	}

	// instance objects
	instanceObjects = append(instanceObjects, newOperandsNamespace(opts.OperandsNamespace))
	otelCol, err := otelCollector(opts.OperandsNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetryCollector: %w", err)
	}
	instanceObjects = append(instanceObjects, otelCol)
	otelcolRBAC, otelcolRBACBinding := otelCollectorComponentsRBAC(opts.OperandsNamespace)
	instanceObjects = append(instanceObjects, otelcolRBAC)
	instanceObjects = append(instanceObjects, otelcolRBACBinding)
	instanceObjects = append(instanceObjects, tempoStack(instance.Spec.Storage, opts.OperandsNamespace))
	instanceObjects = append(instanceObjects, tempoStackSecret(instance.Spec.Storage, opts.OperandsNamespace, storageSecret))
	otelcolTempoRBAC, otelcolTempoRBACBinding := otelCollectorTempoRBAC(opts.OperandsNamespace)
	instanceObjects = append(instanceObjects, otelcolTempoRBAC)
	instanceObjects = append(instanceObjects, otelcolTempoRBACBinding)
	instanceObjects = append(instanceObjects, uiPlugin())

	if instance.ObjectMeta.DeletionTimestamp != nil {
		for _, obj := range instanceObjects {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}
		for _, obj := range operatorObjects {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}

		reconcilers = append(reconcilers, reconciler.NewDeleter(&olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      opts.OpenTelemetryOperator.StartingCSV,
				Namespace: opts.COONamespace,
			},
		}))
		reconcilers = append(reconcilers, reconciler.NewDeleter(&olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      opts.TempoOperator.StartingCSV,
				Namespace: opts.COONamespace,
			},
		}))
		return reconcilers, nil
	}

	// Install operators and instances
	if instance.Spec.Capabilities != nil && instance.Spec.Capabilities.Tracing.CommonCapabilitiesSpec.Enabled {
		// install operators and instances
		for _, obj := range operatorObjects {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(obj, instance))
			installedObjects[gvaNameIdentifier(obj)] = obj
		}
		if storageSecret == nil {
			return nil, fmt.Errorf("storage secret is required when the tracing capability is enabled")
		}
		for _, obj := range instanceObjects {
			reconcilers = append(reconcilers, reconciler.NewUpdater(obj, instance))
			installedObjects[gvaNameIdentifier(obj)] = obj
		}
	}
	// install operators only
	if instance.Spec.Capabilities != nil &&
		(instance.Spec.Capabilities.Tracing.CommonCapabilitiesSpec.Operators.Install != nil && *instance.Spec.Capabilities.Tracing.CommonCapabilitiesSpec.Operators.Install) {
		// install operators only
		for _, obj := range operatorObjects {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(obj, instance))
			installedObjects[gvaNameIdentifier(obj)] = obj
		}
	}

	// rest to delete
	for _, obj := range instanceObjects {
		if installedObjects[gvaNameIdentifier(obj)] == nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}
	}
	// Delete CSV explicitly because it is not deleted when the subscription is deleted.
	for _, obj := range operatorObjects {
		// operator is not installed so make sure the CSV is deleted
		if installedObjects[gvaNameIdentifier(obj)] == nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))

			var csv *olmv1alpha1.ClusterServiceVersion
			if obj.GetName() == opts.OpenTelemetryOperator.PackageName {
				csv = &olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      opts.OpenTelemetryOperator.StartingCSV,
						Namespace: opts.COONamespace,
					},
				}
			} else if obj.GetName() == opts.TempoOperator.PackageName {
				csv = &olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      opts.TempoOperator.StartingCSV,
						Namespace: opts.COONamespace,
					},
				}
			}
			reconcilers = append(reconcilers, reconciler.NewDeleter(csv))
		}
	}

	return reconcilers, nil
}

func gvaNameIdentifier(obj client.Object) string {
	return fmt.Sprintf("%s/%s", obj.GetObjectKind().GroupVersionKind().String(), obj.GetName())
}

func newOperandsNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

package observability

import (
	"context"
	"fmt"
	"strings"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type operatorsStatus struct {
	cooNamespace string
	// CSVs installed in the operator's namespace.
	// The CSV name is always in a form opentelemetry-operator.v0.129.1 - it's for both the product and the OOS version.
	csvs []olmv1alpha1.ClusterServiceVersion
	// Subscriptions installed in all namespaces.
	// The subscription name can be opentelemetry-product or opentelemetry-operator.
	subs []olmv1alpha1.Subscription
}

// ShouldInstall checks if the operator should be uninstalled.
func (s *operatorsStatus) ShouldInstall(operatorName string) bool {
	for _, sub := range s.subs {
		if strings.HasPrefix(sub.Name, operatorName) && sub.Namespace != s.cooNamespace {
			return false
		}
	}
	return true
}

func (s *operatorsStatus) ShouldUnInstall(operatorName string) bool {
	for _, sub := range s.subs {
		if strings.HasPrefix(sub.Name, operatorName) && sub.Namespace == s.cooNamespace {
			return true
		}
	}
	return false
}

func (s *operatorsStatus) getCSVByName(operatorName string) *olmv1alpha1.ClusterServiceVersion {
	for _, csv := range s.csvs {
		if strings.HasPrefix(csv.Name, operatorName) {
			return &csv
		}
	}
	return nil
}

// getReconcilers returns a list of reconcilers for the ClusterObservability instance.
// The subByName is used to check if the operators are already installed, if not, they will be installed.
// The csvByName is used to uninstall the operators, the name of the CSV contains the version therefore it must be retrieved from the cluster.
// The CSV is not deleted when the subscription is deleted, so we need to delete it explicitly.
func getReconcilers(ctx context.Context, k8sClient client.Client, instance *obsv1alpha1.ClusterObservability, opts Options, operatorsStatus operatorsStatus) ([]reconciler.Reconciler, error) {
	var reconcilers []reconciler.Reconciler
	//var otelOperator client.Object
	//var tempoOperator client.Object
	var instanceObjects []client.Object
	installedObjects := map[string]client.Object{}

	// the OTEL and Tempo operators are rolling release, meaning only the latest released versions are supported.
	// At the moment there are no compatibility issues between the operands of these two operators, so we can
	// install them together in any versions.

	otelSubs := subscription(opts.OpenTelemetryOperator)
	tempoSubs := subscription(opts.TempoOperator)

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
	instanceObjects = append(instanceObjects, tempoStack(instance.Spec.Storage, opts.OperandsNamespace, instance.Name))

	secrets, err := tempoStackSecrets(ctx, k8sClient, *instance, opts.OperandsNamespace, opts.COONamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create TempoStack secret: %w", err)
	}
	if secrets.objectStorage != nil {
		instanceObjects = append(instanceObjects, secrets.objectStorage)
	}
	if secrets.objectStorageTLSSecret != nil {
		instanceObjects = append(instanceObjects, secrets.objectStorageTLSSecret)
	}
	if secrets.objectStorageCAConfigMap != nil {
		instanceObjects = append(instanceObjects, secrets.objectStorageCAConfigMap)
	}

	otelcolTempoRBAC, otelcolTempoRBACBinding := otelCollectorTempoRBAC(opts.OperandsNamespace)
	instanceObjects = append(instanceObjects, otelcolTempoRBAC)
	instanceObjects = append(instanceObjects, otelcolTempoRBACBinding)
	instanceObjects = append(instanceObjects, uiPlugin())

	if instance.ObjectMeta.DeletionTimestamp != nil {
		for _, obj := range instanceObjects {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}
		if operatorsStatus.ShouldUnInstall("opentelemetry") {
			reconcilers = append(reconcilers, reconciler.NewDeleter(otelSubs))
			if otelCSV := operatorsStatus.getCSVByName("opentelemetry-operator"); otelCSV != nil {
				reconcilers = append(reconcilers, reconciler.NewDeleter(otelCSV))
			}
		}
		if operatorsStatus.ShouldUnInstall("tempo") {
			reconcilers = append(reconcilers, reconciler.NewDeleter(tempoSubs))
			if tempoCSV := operatorsStatus.getCSVByName("tempo-operator"); tempoCSV != nil {
				reconcilers = append(reconcilers, reconciler.NewDeleter(tempoCSV))
			}
		}
		return reconcilers, nil
	}

	// Install operators and instances
	if instance.Spec.Capabilities != nil && instance.Spec.Capabilities.Tracing.CommonCapabilitiesSpec.Enabled {
		// install operators and instances
		if operatorsStatus.ShouldInstall("opentelemetry") {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(otelSubs, instance))
			installedObjects[gvaNameIdentifier(otelSubs)] = otelSubs
		}
		if operatorsStatus.ShouldInstall("tempo") {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(tempoSubs, instance))
			installedObjects[gvaNameIdentifier(tempoSubs)] = tempoSubs
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
		if operatorsStatus.ShouldInstall("opentelemetry") {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(otelSubs, instance))
			installedObjects[gvaNameIdentifier(otelSubs)] = otelSubs
		}
		if operatorsStatus.ShouldInstall("tempo") {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(tempoSubs, instance))
			installedObjects[gvaNameIdentifier(tempoSubs)] = tempoSubs
		}
	}

	// Delete not created objects.
	for _, obj := range instanceObjects {
		if installedObjects[gvaNameIdentifier(obj)] == nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}
	}
	// Delete CSV explicitly because it is not deleted when the subscription is deleted.
	if operatorsStatus.ShouldInstall("opentelemetry") && installedObjects[gvaNameIdentifier(otelSubs)] == nil {
		reconcilers = append(reconcilers, reconciler.NewDeleter(otelSubs))
		if otelCSV := operatorsStatus.getCSVByName("opentelemetry-operator"); otelCSV != nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(otelCSV))
		}
	}
	if operatorsStatus.ShouldInstall("tempo") && installedObjects[gvaNameIdentifier(tempoSubs)] == nil {
		reconcilers = append(reconcilers, reconciler.NewDeleter(tempoSubs))
		if tempoCSV := operatorsStatus.getCSVByName("tempo-operator"); tempoCSV != nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(tempoCSV))
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

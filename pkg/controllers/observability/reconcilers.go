package observability

import (
	"context"
	"fmt"
	"strings"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type operatorsStatus struct {
	cooNamespace string
	// Subscriptions installed in all namespaces.
	// The subscription name can be opentelemetry-product or opentelemetry-operator.
	subs []olmv1alpha1.Subscription
}

// ShouldInstall checks if the operator should be uninstalled.
// The operator should be installed only if it is not installed
func (s *operatorsStatus) ShouldInstall(operatorName string) bool {
	for _, sub := range s.subs {
		if strings.HasPrefix(sub.Name, operatorName) && sub.Labels[util.ResourceLabel] != util.OpName {
			return false
		}
	}
	return true
}

func (s *operatorsStatus) cooManages(operatorName string) *olmv1alpha1.Subscription {
	for _, sub := range s.subs {
		if strings.HasPrefix(sub.Name, operatorName) && sub.Labels[util.ResourceLabel] == util.OpName {
			return &sub
		}
	}
	return nil
}

// getReconcilers returns a list of reconcilers for the ObservabilityInstaller instance.
// The subByName is used to check if the operators are already installed, if not, they will be installed.
// The csvByName is used to uninstall the operators, the name of the CSV contains the version therefore it must be retrieved from the cluster.
// The CSV is not deleted when the subscription is deleted, so we need to delete it explicitly.
func getReconcilers(ctx context.Context, k8sClient client.Client, k8sReader client.Reader, instance *obsv1alpha1.ObservabilityInstaller, opts Options, operatorsStatus operatorsStatus) ([]reconciler.Reconciler, error) {
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
	otelCol, err := otelCollector(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetryCollector: %w", err)
	}
	instanceObjects = append(instanceObjects, otelCol)
	otelcolRBAC, otelcolRBACBinding := otelCollectorComponentsRBAC(instance)
	instanceObjects = append(instanceObjects, otelcolRBAC)
	instanceObjects = append(instanceObjects, otelcolRBACBinding)
	instanceObjects = append(instanceObjects, tempoStack(instance))

	secrets, err := tempoStackSecrets(ctx, k8sClient, k8sReader, *instance)
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

	otelcolTempoRBAC, otelcolTempoRBACBinding := otelCollectorTempoRBAC(instance)
	instanceObjects = append(instanceObjects, otelcolTempoRBAC)
	instanceObjects = append(instanceObjects, otelcolTempoRBACBinding)
	instanceObjects = append(instanceObjects, uiPlugin())

	if instance.ObjectMeta.DeletionTimestamp != nil {
		for _, obj := range instanceObjects {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}
		if otelSub := operatorsStatus.cooManages("opentelemetry"); otelSub != nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(otelSub))
			reconcilers = append(reconcilers, reconciler.NewDeleter(
				&olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      otelSub.Status.CurrentCSV,
						Namespace: otelSub.Namespace,
					},
				}))
		}
		if tempoSub := operatorsStatus.cooManages("tempo"); tempoSub != nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(tempoSub))
			reconcilers = append(reconcilers, reconciler.NewDeleter(
				&olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tempoSub.Status.CurrentCSV,
						Namespace: tempoSub.Namespace,
					},
				}))
		}
		return reconcilers, nil
	}

	// Install operators and instances
	if instance.Spec.Capabilities != nil && instance.Spec.Capabilities.Tracing.CommonCapabilitiesSpec.Enabled {
		// install operators and instances
		if operatorsStatus.ShouldInstall("opentelemetry") {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(otelSubs, instance))
			installedObjects[gvkNameIdentifier(otelSubs)] = otelSubs
		}
		if operatorsStatus.ShouldInstall("tempo") {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(tempoSubs, instance))
			installedObjects[gvkNameIdentifier(tempoSubs)] = tempoSubs
		}
		for _, obj := range instanceObjects {
			reconcilers = append(reconcilers, reconciler.NewUpdater(obj, instance))
			installedObjects[gvkNameIdentifier(obj)] = obj
		}
	}
	// install operators only
	if instance.Spec.Capabilities != nil &&
		(instance.Spec.Capabilities.Tracing.CommonCapabilitiesSpec.Operators.Install != nil && *instance.Spec.Capabilities.Tracing.CommonCapabilitiesSpec.Operators.Install) {
		// install operators only
		if operatorsStatus.ShouldInstall("opentelemetry") {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(otelSubs, instance))
			installedObjects[gvkNameIdentifier(otelSubs)] = otelSubs
		}
		if operatorsStatus.ShouldInstall("tempo") {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(tempoSubs, instance))
			installedObjects[gvkNameIdentifier(tempoSubs)] = tempoSubs
		}
	}

	// Delete not created objects.
	for _, obj := range instanceObjects {
		if installedObjects[gvkNameIdentifier(obj)] == nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}
	}
	// Delete CSV explicitly because it is not deleted when the subscription is deleted.
	// This handles the uninstall case when the capability is disabled or the operators installation is disabled.
	if otelSub := operatorsStatus.cooManages("opentelemetry"); otelSub != nil && installedObjects[gvkNameIdentifier(otelSubs)] == nil {
		reconcilers = append(reconcilers, reconciler.NewDeleter(otelSub))
		reconcilers = append(reconcilers, reconciler.NewDeleter(
			&olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      otelSub.Status.CurrentCSV,
					Namespace: otelSub.Namespace,
				},
			}))
	}
	if tempoSub := operatorsStatus.cooManages("tempo"); tempoSub != nil && installedObjects[gvkNameIdentifier(tempoSubs)] == nil {
		reconcilers = append(reconcilers, reconciler.NewDeleter(tempoSub))
		reconcilers = append(reconcilers, reconciler.NewDeleter(
			&olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tempoSub.Status.CurrentCSV,
					Namespace: tempoSub.Namespace,
				},
			}))
	}

	return reconcilers, nil
}

func gvkNameIdentifier(obj client.Object) string {
	return fmt.Sprintf("%s/%s", obj.GetObjectKind().GroupVersionKind().String(), obj.GetName())
}

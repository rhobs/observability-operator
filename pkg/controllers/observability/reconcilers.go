package observability

import (
	"context"
	"fmt"
	"slices"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type operatorsStatus struct {
	// Subscriptions installed in all namespaces.
	// A subscription name may include a product-specific suffix.
	subs []olmv1alpha1.Subscription
}

// ShouldInstall reports whether COO should install the operator. An existing
// subscription not managed by COO takes precedence, including a subscription to
// an equivalent package such as the community build.
func (s *operatorsStatus) ShouldInstall(packageNames []string) bool {
	for _, sub := range s.subs {
		if sub.Spec != nil && slices.Contains(packageNames, sub.Spec.Package) && sub.Labels[util.ResourceLabel] != util.OpName {
			return false
		}
	}
	return true
}

func (s *operatorsStatus) cooManages(packageName string) *olmv1alpha1.Subscription {
	for _, sub := range s.subs {
		if sub.Spec != nil && sub.Spec.Package == packageName && sub.Labels[util.ResourceLabel] == util.OpName {
			return &sub
		}
	}
	return nil
}

// operatorReconcilers returns the reconcilers that install operators and,
// separately, those that uninstall them. The uninstall reconcilers must run
// after the operand deleters: removing an operator first would leave nothing to
// clear the finalizers its operands carry.
func operatorReconcilers(requirements []capability.OperatorRequirement, status operatorsStatus) (install, uninstall []reconciler.Reconciler) {
	for _, requirement := range requirements {
		packageNames := append([]string{requirement.Subscription.Spec.Package}, requirement.EquivalentPackages...)
		if requirement.Desired {
			if status.ShouldInstall(packageNames) {
				// Subscriptions are shared across all ObservabilityInstallers. Their
				// lifecycle is controlled by the aggregated requirements below, not
				// by garbage collection of any individual installer.
				install = append(install, reconciler.NewUnmanagedCreateUpdateReconciler(requirement.Subscription))
			}
			continue
		}
		uninstall = append(uninstall, managedOperatorDeleters(requirement.Subscription.Spec.Package, status)...)
	}
	return install, uninstall
}

func managedOperatorDeleters(name string, status operatorsStatus) []reconciler.Reconciler {
	sub := status.cooManages(name)
	if sub == nil {
		return nil
	}

	result := []reconciler.Reconciler{reconciler.NewDeleter(sub)}
	if sub.Status.CurrentCSV != "" {
		result = append(result, reconciler.NewDeleter(&olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{Name: sub.Status.CurrentCSV, Namespace: sub.Namespace},
		}))
	}
	return result
}

// getReconcilers returns the reconcilers needed to move an
// ObservabilityInstaller to its desired state.
func getReconcilers(ctx context.Context, k8sReader client.Reader, instance *obsv1alpha1.ObservabilityInstaller, definitions []capability.Definition, status operatorsStatus, instances []obsv1alpha1.ObservabilityInstaller) ([]reconciler.Reconciler, error) {
	shared := aggregateShared(definitions, instances)
	var plannedDesiredObjects, plannedOwnedObjects []client.Object
	for _, planner := range definitions {
		plan, err := planner.Plan(ctx, instance, k8sReader)
		if err != nil {
			return nil, fmt.Errorf("plan %s capability: %w", planner.Name, err)
		}
		plannedDesiredObjects = append(plannedDesiredObjects, plan.Desired...)
		plannedOwnedObjects = append(plannedOwnedObjects, plan.Owned...)
	}

	if instance.DeletionTimestamp != nil {
		plannedDesiredObjects = nil
	}
	plannedOwnedObjects = append(plannedOwnedObjects, shared.Owned...)
	plannedDesiredObjects = append(plannedDesiredObjects, shared.Desired...)
	install, uninstall := operatorReconcilers(shared.Operators, status)
	result := install
	desiredObjects := map[string]struct{}{}
	result = appendUpdaters(result, plannedDesiredObjects, instance, desiredObjects)

	for _, obj := range plannedOwnedObjects {
		if _, desired := desiredObjects[gvkNameIdentifier(obj)]; !desired {
			result = append(result, reconciler.NewDeleter(obj))
		}
	}
	// Operators are removed last, once their operands are gone.
	result = append(result, uninstall...)
	return result, nil
}

func appendUpdaters(result []reconciler.Reconciler, objects []client.Object, instance *obsv1alpha1.ObservabilityInstaller, desired map[string]struct{}) []reconciler.Reconciler {
	for _, obj := range objects {
		result = append(result, reconciler.NewUpdater(obj, instance))
		desired[gvkNameIdentifier(obj)] = struct{}{}
	}
	return result
}

func gvkNameIdentifier(obj client.Object) string {
	return fmt.Sprintf("%s/%s/%s", obj.GetObjectKind().GroupVersionKind().String(), obj.GetNamespace(), obj.GetName())
}

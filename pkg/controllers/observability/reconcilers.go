package observability

import (
	"context"
	"errors"
	"fmt"
	"slices"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type operatorsStatus struct {
	cooNamespace string
	// Subscriptions installed in all namespaces.
	subs []olmv1alpha1.Subscription
}

// ShouldInstall reports whether COO should create its own subscription,
// returning false if an equivalent package is already installed by someone else.
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

// getReconcilers returns a list of reconcilers for the ObservabilityInstaller instance.
// The operatorsStatus is used to check if the operators are already installed, if not, they will be installed.
// The CSV is not deleted when the subscription is deleted, so we need to delete it explicitly.
func getReconcilers(ctx context.Context, k8sReader client.Reader, instance *obsv1alpha1.ObservabilityInstaller, capabilities []capability.Capability, operatorsStatus operatorsStatus, instances []obsv1alpha1.ObservabilityInstaller) ([]reconciler.Reconciler, error) {
	var reconcilers []reconciler.Reconciler
	shared, err := aggregateShared(capabilities, instances)
	if err != nil {
		return nil, err
	}
	var desiredObjects, ownedObjects []client.Object
	var planErrors []error
	for _, c := range capabilities {
		inventory := c.Inventory(instance)
		if instance.DeletionTimestamp != nil {
			ownedObjects = append(ownedObjects, inventory...)
			continue
		}
		desired, err := c.Desired(ctx, instance, k8sReader)
		if err == nil {
			err = checkInventory(c.Name(), desired, inventory)
		}
		if err != nil {
			planErrors = append(planErrors, fmt.Errorf("plan %s capability: %w", c.Name(), err))
			continue
		}
		ownedObjects = append(ownedObjects, inventory...)
		desiredObjects = append(desiredObjects, desired...)
	}
	desiredObjects = append(desiredObjects, shared.desired...)
	// Shared cleanup can affect multiple capabilities; defer it until every
	// per-installer plan succeeds. Healthy capabilities still clean up their own inventory.
	if len(planErrors) == 0 {
		ownedObjects = append(ownedObjects, shared.owned...)
	}

	// Install operators and instances
	for _, operator := range shared.operators {
		packageNames := append([]string{operator.Subscription.Spec.Package}, operator.EquivalentPackages...)
		if instance.DeletionTimestamp == nil && operator.desired && operatorsStatus.ShouldInstall(packageNames) {
			reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(operator.Subscription, instance))
		}
	}
	installedObjects := map[string]client.Object{}
	for _, obj := range desiredObjects {
		// A deleting installer must not hold its finalizer while applying
		// resources requested by other installers. Their reconciles apply them.
		if instance.DeletionTimestamp == nil {
			reconcilers = append(reconcilers, reconciler.NewUpdater(obj, instance))
		}
		installedObjects[gvkNameIdentifier(obj)] = obj
	}

	// Delete not created objects.
	for _, obj := range ownedObjects {
		if installedObjects[gvkNameIdentifier(obj)] == nil {
			reconcilers = append(reconcilers, missingCRDDeleter{reconciler.NewDeleter(obj)})
		}
	}
	// Delete CSV explicitly because it is not deleted when the subscription is deleted.
	// This handles the uninstall case when the capability is disabled or the operators installation is disabled.
	for _, operator := range shared.operators {
		if sub := operatorsStatus.cooManages(operator.Subscription.Spec.Package); sub != nil && !operator.desired && len(planErrors) == 0 {
			reconcilers = append(reconcilers, reconciler.NewDeleter(sub))
			reconcilers = append(reconcilers, reconciler.NewDeleter(
				&olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      sub.Status.CurrentCSV,
						Namespace: sub.Namespace,
					},
				}))
		}
	}

	return reconcilers, errors.Join(planErrors...)
}

func gvkNameIdentifier(obj client.Object) string {
	return fmt.Sprintf("%s/%s/%s", obj.GetObjectKind().GroupVersionKind().String(), obj.GetNamespace(), obj.GetName())
}

// missingCRDDeleter treats a resource whose kind is not served as already
// deleted. Inventory includes operands whose operator, and so CRD, may never
// have been installed.
type missingCRDDeleter struct{ reconciler.Deleter }

func (d missingCRDDeleter) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	if err := d.Deleter.Reconcile(ctx, c, scheme); !meta.IsNoMatchError(err) {
		return err
	}
	return nil
}

package reconciler

import (
	"context"
	"fmt"
	"slices"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

type createUpdateReconciler struct {
	resourceOwner metav1.Object
	resource      client.Object
	desired       client.Object // desired holds a copy of the requested state.
}

func (r createUpdateReconciler) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	// If the resource owner is in the same namespace as the resource, or if the resource owner is cluster scoped set the owner reference.
	if r.resourceOwner != nil && (r.resourceOwner.GetNamespace() == r.resource.GetNamespace() || r.resourceOwner.GetNamespace() == "") {
		if err := controllerutil.SetControllerReference(r.resourceOwner, r.resource, scheme); err != nil {
			return fmt.Errorf("%s/%s (%s): updater failed to set owner reference: %w",
				r.resource.GetNamespace(), r.resource.GetName(),
				r.resource.GetObjectKind().GroupVersionKind().String(), err)
		}
	}

	_, err := ctrl.CreateOrUpdate(ctx, c, r.resource, func() error {
		// Propagate the fields COO owns onto the live Subscription so updates
		// are not silently dropped. Other fields, such as a mirrored catalog
		// source, install plan approval or spec.config set by an admin, are
		// preserved.
		live, liveOK := r.resource.(*olmv1alpha1.Subscription)
		desired, desiredOK := r.desired.(*olmv1alpha1.Subscription)
		if liveOK && desiredOK && desired.Spec != nil {
			if live.Spec == nil {
				live.Spec = &olmv1alpha1.SubscriptionSpec{}
			}
			live.Spec.Package = desired.Spec.Package
			live.Spec.Channel = desired.Spec.Channel
			live.Spec.StartingCSV = desired.Spec.StartingCSV
			if r.resourceOwner == nil {
				// Subscriptions used to be owned by individual installers. Remove
				// those legacy references so deleting one installer cannot garbage-
				// collect an operator still needed by another installer.
				live.OwnerReferences = slices.DeleteFunc(live.OwnerReferences, func(ref metav1.OwnerReference) bool {
					return ref.APIVersion == obsv1alpha1.GroupVersion.String() && ref.Kind == "ObservabilityInstaller"
				})
			}
		}
		return nil
	})

	return err
}

func NewCreateUpdateReconciler(resource client.Object, owner metav1.Object) Reconciler {
	res := util.AddCommonLabels(resource, owner.GetName())
	return createUpdateReconciler{
		resourceOwner: owner,
		resource:      res,
		desired:       res.DeepCopyObject().(client.Object),
	}
}

// NewUnmanagedCreateUpdateReconciler reconciles a resource without attaching
// it to a particular owner. Use it for resources shared by multiple owners,
// such as operator Subscriptions used by several ObservabilityInstallers:
// owning them by one installer would let garbage collection delete them when
// that installer is deleted. The caller is responsible for deleting the
// resource once no owner needs it.
func NewUnmanagedCreateUpdateReconciler(resource client.Object) Reconciler {
	res := util.AddCommonLabels(resource, util.OpName)
	return createUpdateReconciler{
		resource: res,
		desired:  res.DeepCopyObject().(client.Object),
	}
}

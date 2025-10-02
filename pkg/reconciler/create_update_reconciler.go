package reconciler

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

type createUpdateReconciler struct {
	resourceOwner metav1.Object
	resource      client.Object
}

func (r createUpdateReconciler) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	// If the resource owner is in the same namespace as the resource, or if the resource owner is cluster scoped set the owner reference.
	if r.resourceOwner.GetNamespace() == r.resource.GetNamespace() || r.resourceOwner.GetNamespace() == "" {
		if err := controllerutil.SetControllerReference(r.resourceOwner, r.resource, scheme); err != nil {
			return fmt.Errorf("%s/%s (%s): updater failed to set owner reference: %w",
				r.resource.GetNamespace(), r.resource.GetName(),
				r.resource.GetObjectKind().GroupVersionKind().String(), err)
		}
	}

	_, err := ctrl.CreateOrUpdate(ctx, c, r.resource, func() error { return nil })

	return err
}

func NewCreateUpdateReconciler(resource client.Object, owner metav1.Object) Reconciler {
	return createUpdateReconciler{
		resourceOwner: owner,
		resource:      util.AddCommonLabels(resource, owner.GetName()),
	}
}

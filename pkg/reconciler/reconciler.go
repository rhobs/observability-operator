package reconciler

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler interface {
	Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error
}

type UpdateReconciler struct {
	controller metav1.Object
	resource   client.Object
}

func (r UpdateReconciler) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	if r.controller.GetNamespace() == r.resource.GetNamespace() {
		if err := controllerutil.SetControllerReference(r.controller, r.resource, scheme); err != nil {
			return err
		}
	}

	if err := c.Patch(ctx, r.resource, client.Apply, client.ForceOwnership, client.FieldOwner(fmt.Sprintf("%s/%s", r.controller.GetNamespace(), r.controller.GetName()))); err != nil {
		return err
	}
	return nil
}

func NewUpdateReconciler(r client.Object, c metav1.Object) UpdateReconciler {
	return UpdateReconciler{
		controller: c,
		resource:   r,
	}
}

type DeleteReconciler struct {
	resource client.Object
}

func (r DeleteReconciler) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	if err := c.Delete(ctx, r.resource); client.IgnoreNotFound(err) != nil {
		return err
	}
	return nil
}

func NewDeleteReconciler(r client.Object) DeleteReconciler {
	return DeleteReconciler{resource: r}
}

type OptionalResourceReconciler struct {
	r Reconciler
}

func (r OptionalResourceReconciler) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	return r.r.Reconcile(ctx, c, scheme)
}

func NewOptionalResourceReconciler(r client.Object, c metav1.Object, cond bool) OptionalResourceReconciler {
	if cond {
		return OptionalResourceReconciler{
			r: NewUpdateReconciler(r, c),
		}
	} else {
		return OptionalResourceReconciler{
			r: NewDeleteReconciler(r),
		}
	}
}

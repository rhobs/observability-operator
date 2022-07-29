package componentUpdater

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type UpdateFunction func(ctx context.Context, c client.Client, scheme *runtime.Scheme) error

func DefaultUpdater(component client.Object, controller metav1.Object) UpdateFunction {
	return func(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
		if controller.GetNamespace() == component.GetNamespace() {
			if err := controllerutil.SetControllerReference(controller, component, scheme); err != nil {
				return err
			}
		}

		if err := c.Patch(ctx, component, client.Apply, client.ForceOwnership, client.FieldOwner(fmt.Sprintf("%s/%s", controller.GetNamespace(), controller.GetName()))); err != nil {
			return err
		}
		return nil
	}
}

func DefaultDeleteOrUpdate(component client.Object, controller metav1.Object, deleteCondition bool) UpdateFunction {
	if deleteCondition {
		return func(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
			if err := c.Delete(ctx, component); client.IgnoreNotFound(err) != nil {
				return err
			}
			return nil
		}
	} else {
		return DefaultUpdater(component, controller)
	}
}

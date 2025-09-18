package reconciler

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

const (
	// OpenshiftMonitoringNamespace is the namespace in which the OpenShift
	// monitoring components are deployed.
	OpenshiftMonitoringNamespace = "openshift-monitoring"
)

// This interface is used by the resourceManagers to reconicle the resources they
// watch. If any component needs special treatment in the reconcile loop, create
// a new type that implements this interface.
type Reconciler interface {
	Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error
}

// Updater simply updates a resource by setting a controller reference
// for resourceOwner and calling Patch on it.
type Updater struct {
	resourceOwner metav1.Object
	resource      client.Object
}

func (r Updater) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	// If the resource owner is in the same namespace as the resource, or if the resource owner is cluster scoped set the owner reference.
	if r.resourceOwner.GetNamespace() == r.resource.GetNamespace() || r.resourceOwner.GetNamespace() == "" {
		if err := controllerutil.SetControllerReference(r.resourceOwner, r.resource, scheme); err != nil {
			return fmt.Errorf("%s/%s (%s): updater failed to set owner reference: %w",
				r.resource.GetNamespace(), r.resource.GetName(),
				r.resource.GetObjectKind().GroupVersionKind().String(), err)
		}
	}

	if err := c.Patch(ctx, r.resource, client.Apply, client.ForceOwnership, client.FieldOwner("observability-operator")); err != nil {
		return fmt.Errorf("%s/%s (%s): updater failed to patch: %w",
			r.resource.GetNamespace(), r.resource.GetName(),
			r.resource.GetObjectKind().GroupVersionKind().String(), err)
	}

	return nil
}

func NewUpdater(resource client.Object, owner metav1.Object) Updater {
	return Updater{
		resourceOwner: owner,
		resource:      util.AddCommonLabels(resource, owner.GetName()),
	}
}

// Deleter deletes a resource and ignores NotFound errors.
type Deleter struct {
	resource client.Object
}

func (r Deleter) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	if err := c.Delete(ctx, r.resource); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("%s/%s (%s): deleter failed to delete: %w",
			r.resource.GetNamespace(), r.resource.GetName(),
			r.resource.GetObjectKind().GroupVersionKind().String(), err)
	}
	return nil
}

func NewDeleter(r client.Object) Deleter {
	return Deleter{resource: r}
}

type Merger struct {
	resource client.Object
}

func NewMerger(r client.Object, owner string) Merger {
	return Merger{resource: util.AddCommonLabels(r, owner)}
}

func (r Merger) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	if err := c.Patch(ctx, r.resource, client.Merge); err != nil {
		return fmt.Errorf("%s/%s (%s): merger failed to patch: %w",
			r.resource.GetNamespace(), r.resource.GetName(),
			r.resource.GetObjectKind().GroupVersionKind().String(), err)
	}
	return nil
}

// NewOptionalUpdater ensures that a resource is present or absent depending on the `cond` value (true: present).
func NewOptionalUpdater(r client.Object, c metav1.Object, cond bool) Reconciler {
	if cond {
		return NewUpdater(r, c)
	} else {
		return NewDeleter(r)
	}
}

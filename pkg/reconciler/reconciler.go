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

// This interface is used by the resourceManagers to reconcile the resources they
// watch. If any component needs special treatment in the reconcile loop, create
// a new type that implements this interface.
type Reconciler interface {
	Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error
}

// Updater simply updates a resource by setting a controller reference
// for resourceOwner and calling Patch on it.
type Updater struct {
	resourceOwner          metav1.Object
	resource               client.Object
	shouldBypassSetCtrlRef bool
}

func (r Updater) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	// Only set the controller reference if the bypass flag is false.
	// Bypassing allows other operators to own the resource
	// (e.g. Observability-operator creates the perses instance. But Perses-operator manages the perses instance)
	if !r.shouldBypassSetCtrlRef {
		// If the resource owner is in the same namespace as the resource, or if the resource owner is cluster scoped set the owner reference.
		if r.resourceOwner.GetNamespace() == r.resource.GetNamespace() || r.resourceOwner.GetNamespace() == "" {
			if err := controllerutil.SetControllerReference(r.resourceOwner, r.resource, scheme); err != nil {
				return fmt.Errorf("%s/%s (%s): updater failed to set owner reference: %w",
					r.resource.GetNamespace(), r.resource.GetName(),
					r.resource.GetObjectKind().GroupVersionKind().String(), err)
			}
		}
	}

	//nolint:staticcheck // Ignore SA1019 this function is marked as deprecated.
	if err := c.Patch(ctx, r.resource, client.Apply, client.ForceOwnership, client.FieldOwner("observability-operator")); err != nil {
		return fmt.Errorf("%s/%s (%s): updater failed to patch: %w",
			r.resource.GetNamespace(), r.resource.GetName(),
			r.resource.GetObjectKind().GroupVersionKind().String(), err)
	}

	return nil
}

func NewUpdater(resource client.Object, owner metav1.Object) (Updater, error) {
	return newUpdater(resource, owner, false)
}

// NewUnmanagedUpdater creates an Updater that does not set a controller reference.
func NewUnmanagedUpdater(resource client.Object, owner metav1.Object) (Updater, error) {
	return newUpdater(resource, owner, true)
}

func newUpdater(resource client.Object, owner metav1.Object, bypassOwnerRef bool) (Updater, error) {
	labeled, err := util.AddCommonLabels(resource, owner.GetName())
	if err != nil {
		return Updater{}, err
	}
	return Updater{
		resourceOwner:          owner,
		resource:               labeled,
		shouldBypassSetCtrlRef: bypassOwnerRef,
	}, nil
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

func NewMerger(r client.Object, owner string) (Merger, error) {
	labeled, err := util.AddCommonLabels(r, owner)
	if err != nil {
		return Merger{}, err
	}
	return Merger{resource: labeled}, nil
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
func NewOptionalUpdater(r client.Object, c metav1.Object, cond bool) (Reconciler, error) {
	if cond {
		return NewUpdater(r, c)
	}
	return NewDeleter(r), nil
}

func NewOptionalUnmanagedUpdater(r client.Object, c metav1.Object, cond bool) (Reconciler, error) {
	if cond {
		return NewUnmanagedUpdater(r, c)
	}
	return NewDeleter(r), nil
}

// ReconcilerBuilder accumulates Reconciler instances and the first error encountered.
// It allows constructing a list of reconcilers without verbose per-call error handling.
type ReconcilerBuilder struct {
	recs []Reconciler
	err  error
}

// Add appends r to the builder if no prior error was recorded. If err is non-nil it is
// stored and subsequent Add calls are no-ops.
func (b *ReconcilerBuilder) Add(r Reconciler, err error) *ReconcilerBuilder {
	if b.err != nil {
		return b
	}
	if err != nil {
		b.err = err
		return b
	}
	b.recs = append(b.recs, r)
	return b
}

// Build returns the accumulated reconcilers and the first error encountered, if any.
func (b *ReconcilerBuilder) Build() ([]Reconciler, error) {
	return b.recs, b.err
}

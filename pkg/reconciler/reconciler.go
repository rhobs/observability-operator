package reconciler

import (
	"context"
	"encoding/json"
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

	if err := c.Apply(ctx, &clientObjectApplyConfig{obj: r.resource}, client.ForceOwnership, client.FieldOwner("observability-operator")); err != nil {
		return fmt.Errorf("%s/%s (%s): updater failed to apply: %w",
			r.resource.GetNamespace(), r.resource.GetName(),
			r.resource.GetObjectKind().GroupVersionKind().String(), err)
	}

	return nil
}

func NewUpdater(resource client.Object, owner metav1.Object) Updater {
	return newUpdater(resource, owner, false)
}

// NewUnmanagedUpdater creates an Updater that does not set a controller reference.
func NewUnmanagedUpdater(resource client.Object, owner metav1.Object) Updater {
	return newUpdater(resource, owner, true)
}

func newUpdater(resource client.Object, owner metav1.Object, bypassOwnerRef bool) Updater {
	return Updater{
		resourceOwner:          owner,
		resource:               util.AddCommonLabels(resource, owner.GetName()),
		shouldBypassSetCtrlRef: bypassOwnerRef,
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
	}
	return NewDeleter(r)
}

func NewOptionalUnmanagedUpdater(r client.Object, c metav1.Object, cond bool) Reconciler {
	if cond {
		return NewUnmanagedUpdater(r, c)
	}
	return NewDeleter(r)
}

// clientObjectApplyConfig wraps a client.Object so it satisfies runtime.ApplyConfiguration,
// allowing Updater to use client.Client.Apply() instead of the deprecated
// client.Client.Patch(..., client.Apply, ...) path.
//
// The object is held as a plain field (not embedded) so the wrapper does NOT
// satisfy runtime.Object. Without this, the typed client's type-switch would
// hit the runtime.Object branch and try to look up *clientObjectApplyConfig
// in the scheme, causing "no kind is registered" errors at runtime.
//
// Serialisation is identical to the old applyPatch path: apply.NewRequest
// calls json.Marshal on this wrapper, which delegates to the underlying object.
type clientObjectApplyConfig struct {
	obj client.Object
}

func (a *clientObjectApplyConfig) IsApplyConfiguration() {}

func (a *clientObjectApplyConfig) GetName() *string {
	n := a.obj.GetName()
	return &n
}

func (a *clientObjectApplyConfig) GetNamespace() *string {
	ns := a.obj.GetNamespace()
	return &ns
}

func (a *clientObjectApplyConfig) GetKind() *string {
	k := a.obj.GetObjectKind().GroupVersionKind().Kind
	return &k
}

func (a *clientObjectApplyConfig) GetAPIVersion() *string {
	av := a.obj.GetObjectKind().GroupVersionKind().GroupVersion().String()
	return &av
}

func (a *clientObjectApplyConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.obj)
}

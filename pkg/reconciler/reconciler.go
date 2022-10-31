package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	logger        logr.Logger
}

func (r Updater) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {

	r.logger.Info("patching started")
	defer r.logger.Info("patching done")

	ns := r.resource.GetNamespace()
	name := r.resource.GetName()
	gvk := r.resource.GetObjectKind().GroupVersionKind()

	if r.resourceOwner.GetNamespace() == ns {
		if err := controllerutil.SetControllerReference(r.resourceOwner, r.resource, scheme); err != nil {
			return fmt.Errorf("%s/%s (%s): updater failed to set owner reference: %w", ns, name, gvk, err)
		}
	}
	owner := fmt.Sprintf("%s/%s", r.resourceOwner.GetNamespace(), r.resourceOwner.GetName())

	// HACK: works around the issue that the operator sometimes fails to create
	// a rolebinding  because a role that was created earlier could not be found.
	// E.g. while creating a RoleBinding "valid-loglevel-alertmanager", we often see
	// Patch fails with Error "roles.rbac.authorization.k8s.io 'valid-loglevel-alertmanager' not found"
	//
	// The workaround is to wait until the resource that was applied by Patch is created, and
	// on failure to return the error

	if err := c.Patch(ctx, r.resource, client.Apply, client.ForceOwnership, client.FieldOwner(owner)); err != nil {
		return fmt.Errorf("%s/%s (%s): updater failed to patch: %w", ns, name, gvk, err)
	}

	dup, err := copyClientObject(r.resource)
	if err != nil {
		return err
	}

	// these const makes it easier to understand the return statements in poll
	const done = true
	const retry = false

	attempt := 0
	var getErr error

	pollErr := wait.PollImmediate(10*time.Second, 60*time.Second, func() (bool, error) {
		// reset error for every attemt
		getErr = nil

		attempt++
		l := r.logger.WithValues("attempt", attempt)

		// try to get the object back again and if that fails, retry
		key := types.NamespacedName{Namespace: ns, Name: name}
		err := c.Get(ctx, key, dup)
		if err != nil {
			l.Info("resource patched but hasn't been created", "err", err)
			getErr = fmt.Errorf("%s/%s (%s): updater failed to create: %w", ns, name, gvk, err)
			return retry, nil
		}

		l.Info("resource patched and created successfully")
		return done, nil
	})

	r.logger.Info("after patch", "get", getErr, "poll", pollErr)

	if pollErr == wait.ErrWaitTimeout {
		return getErr
	}

	return nil
}

func NewUpdater(r client.Object, owner metav1.Object, l logr.Logger) Updater {
	return Updater{
		resourceOwner: owner,
		resource:      r,
		logger: l.WithName("updater").WithValues(
			"resource", r.GetName(), "ns", r.GetNamespace(),
			"kind", r.GetObjectKind().GroupVersionKind().Kind,
			"owner", owner.GetName(),
		),
	}
}

// Deleter deletes a resource and ignores NotFound errors.
type Deleter struct {
	resource client.Object
	logger   logr.Logger
}

func (r Deleter) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	// these const makes is easier to understand the return statements
	r.logger.Info("delete started")
	defer r.logger.Info("delete finished")

	const done = true
	const retry = false

	ns := r.resource.GetNamespace()
	name := r.resource.GetName()
	gvk := r.resource.GetObjectKind().GroupVersionKind()

	var delErr error
	pollErr := wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		delErr = nil

		if err := c.Delete(ctx, r.resource); client.IgnoreNotFound(err) != nil {
			r.logger.Info("delete failed", "err", err)
			delErr = fmt.Errorf("%s/%s (%s): deleter failed to delete: %w", ns, name, gvk, err)
			return retry, nil
		}
		r.logger.Info("deleted successfully")
		return done, nil
	})

	r.logger.Info("after deletion", "del", delErr, "poll", pollErr)

	if pollErr == wait.ErrWaitTimeout {
		return delErr
	}

	// Get requires a client.Object but we can't use r.resource since it may
	// need to be reapplied, so deepcopy the resource and set the GVK
	dup, err := copyClientObject(r.resource)
	if err != nil {
		return err
	}

	pollErr = wait.PollImmediate(3*time.Second, 15*time.Second, func() (bool, error) {

		// reset delErr for every attempt
		delErr = nil

		// try to get the object back again and if it exits, retry
		key := types.NamespacedName{Namespace: ns, Name: name}
		if err := c.Get(ctx, key, dup); errors.IsNotFound(err) {
			r.logger.Info("resource deleted successfully")
			return done, nil
		}

		delErr = fmt.Errorf("%s/%s (%s): resource deletion is incomplete: %w", ns, name, gvk, err)
		r.logger.Info("resource deleted but still exists", "err", err)
		return retry, nil
	})

	r.logger.Info("after delete", "del-error", delErr, "poll-error", pollErr)

	if pollErr == wait.ErrWaitTimeout {
		return delErr
	}

	return nil
}

func NewDeleter(r client.Object, l logr.Logger) Deleter {
	return Deleter{
		resource: r,
		logger: l.WithName("deleter").WithValues(
			"name", r.GetName(), "ns", r.GetNamespace(),
			"kind", r.GetObjectKind().GroupVersionKind().Kind,
		),
	}
}

// NewOptionalUpdater ensures that a resource is present or absent depending on the `cond` value (true: present).
func NewOptionalUpdater(r client.Object, c metav1.Object, l logr.Logger, cond bool) Reconciler {
	if cond {
		return NewUpdater(r, c, l)
	} else {
		return NewDeleter(r, l)
	}
}

func copyClientObject(o client.Object) (client.Object, error) {

	// Get requires a client.Object but we can't use r.resource since it may
	// need to be reapplied, so deepcopy the resource and set the GVK
	gvk := o.GetObjectKind().GroupVersionKind()
	dup, ok := o.DeepCopyObject().(client.Object)
	if !ok {
		return nil, fmt.Errorf("%s/%s (%s): failed to create copy",
			o.GetNamespace(), o.GetName(), gvk)
	}

	dup.GetObjectKind().SetGroupVersionKind(gvk)
	return dup, nil
}

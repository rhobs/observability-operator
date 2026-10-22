package reconciler

import (
	"context"
	"crypto/sha256"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

type networkPolicyReconciler struct {
	resource      *networkingv1.NetworkPolicy
	resourceOwner metav1.Object
	apiReader     client.Reader
	present       bool
}

// NewNetworkPolicyUpdater reconciles a policy without adopting an existing policy
// unless its controller reference identifies the requested owner, including UID.
func NewNetworkPolicyUpdater(resource *networkingv1.NetworkPolicy, owner metav1.Object, apiReader client.Reader) Reconciler {
	return NewOptionalNetworkPolicyUpdater(resource, owner, apiReader, true)
}

// NewNetworkPolicyDeleter deletes only a policy controlled by the requested owner.
// A missing policy is already reconciled; an ownership collision is an error.
func NewNetworkPolicyDeleter(resource *networkingv1.NetworkPolicy, owner metav1.Object, apiReader client.Reader) Reconciler {
	return NewOptionalNetworkPolicyUpdater(resource, owner, apiReader, false)
}

// NewOptionalNetworkPolicyUpdater ensures an owned policy is present or absent.
// Unlike NewOptionalUpdater, both paths check existing controller ownership.
// apiReader must bypass the cache to find policies missing the managed-by label.
func NewOptionalNetworkPolicyUpdater(resource *networkingv1.NetworkPolicy, owner metav1.Object, apiReader client.Reader, present bool) Reconciler {
	return networkPolicyReconciler{resource: resource, resourceOwner: owner, apiReader: apiReader, present: present}
}

func (r networkPolicyReconciler) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	desired := r.resource.DeepCopy()
	key := client.ObjectKeyFromObject(desired)
	if desired.Name == "" || desired.Namespace == "" {
		return fmt.Errorf("NetworkPolicy %s: name and namespace must be set", key)
	}
	if r.resourceOwner.GetUID() == "" {
		return fmt.Errorf("NetworkPolicy %s: controller owner UID must be set", key)
	}
	// SetControllerReference rejects cross-namespace owners rather than silently
	// omitting the reference as the generic Updater does.
	if err := controllerutil.SetControllerReference(r.resourceOwner, desired, scheme); err != nil {
		return fmt.Errorf("NetworkPolicy %s: failed to set controller reference: %w", key, err)
	}
	desired.SetGroupVersionKind(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"))
	util.AddCommonLabels(desired, r.resourceOwner.GetName())
	desired.Labels["app.kubernetes.io/name"] = networkPolicyNameLabelValue(desired.Name)

	current := &networkingv1.NetworkPolicy{}
	err := c.Get(ctx, key, current)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("NetworkPolicy %s: failed to get: %w", key, err)
	}
	if apierrors.IsNotFound(err) && r.present {
		// SSA is an upsert: applying after a NotFound could adopt a foreign
		// policy created concurrently. Reserve the name with an atomic Create
		// first, then use SSA to manage the policy we just created.
		current = desired.DeepCopy()
		err = c.Create(ctx, current, client.FieldOwner("observability-operator"))
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("NetworkPolicy %s: failed to create: %w", key, err)
		}
	}
	if err != nil {
		// A policy leaving the cache's label selector produces a delete event,
		// but still exists on the API server. Read it directly after an
		// AlreadyExists, or before treating a cache miss as a completed deletion.
		current = &networkingv1.NetworkPolicy{}
		if err := r.apiReader.Get(ctx, key, current); err != nil {
			if !r.present && apierrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("NetworkPolicy %s: failed to get uncached: %w", key, err)
		}
	}
	// Also check the Create response in case admission changed the owner.
	if err := checkNetworkPolicyController(current, *metav1.GetControllerOf(desired)); err != nil {
		return err
	}

	if !r.present {
		// Protect against both replacement and ownership changes after the Get.
		if err := c.Delete(ctx, current, client.Preconditions{
			UID: &current.UID, ResourceVersion: &current.ResourceVersion,
		}); client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("NetworkPolicy %s: failed to delete: %w", key, err)
		}
		return nil
	}

	// Protect the ownership check against concurrent edits or replacement, and
	// prevent SSA from recreating a policy deleted after the Get.
	desired.UID = current.UID
	desired.ResourceVersion = current.ResourceVersion
	if err := c.Apply(ctx, &clientObjectApplyConfig{obj: desired}, client.ForceOwnership, client.FieldOwner("observability-operator")); err != nil {
		return fmt.Errorf("NetworkPolicy %s: failed to apply: %w", key, err)
	}
	return nil
}

// networkPolicyNameLabelValue returns a valid Kubernetes label value for the
// policy name. Policy names can be longer than the 63-character label-value
// limit when they are derived from a user-provided custom resource name.
func networkPolicyNameLabelValue(name string) string {
	const maxLength = 63
	if len(name) <= maxLength {
		return name
	}

	hash := sha256.Sum256([]byte(name))
	return fmt.Sprintf("%s-%x", name[:maxLength-9], hash[:4])
}

func checkNetworkPolicyController(policy *networkingv1.NetworkPolicy, expected metav1.OwnerReference) error {
	actual := metav1.GetControllerOf(policy)
	// API versions may change across an upgrade, but the GroupKind, name, and
	// UID must still identify the exact same controller object.
	if actual == nil || actual.UID != expected.UID || actual.Name != expected.Name ||
		schema.FromAPIVersionAndKind(actual.APIVersion, actual.Kind).GroupKind() != schema.FromAPIVersionAndKind(expected.APIVersion, expected.Kind).GroupKind() {
		return fmt.Errorf("NetworkPolicy %s: ownership conflict: expected controller %s %s %s (UID %s), got %v",
			client.ObjectKeyFromObject(policy), expected.APIVersion, expected.Kind, expected.Name, expected.UID, actual)
	}
	return nil
}

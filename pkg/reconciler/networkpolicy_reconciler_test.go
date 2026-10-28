package reconciler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	networkingac "k8s.io/client-go/applyconfigurations/networking/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestNetworkPolicyUpdater(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "create"
		if existing {
			name = "same-owner update"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			scheme, owner, desired := networkPolicyTestObjects(t)
			original := desired.DeepCopy()
			var objects []client.Object
			if existing {
				current := ownedTestNetworkPolicy(t, scheme, owner, desired)
				current.Spec.Ingress = nil
				current.Spec.Egress = nil
				current.Labels = map[string]string{"user-label": "preserved"}
				objects = append(objects, current)
			}
			base := networkPolicyTestClient(t, scheme, objects...)
			var creates, applies int
			c := interceptor.NewClient(base, interceptor.Funcs{
				Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					creates++
					return c.Create(ctx, obj, opts...)
				},
				Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
					applies++
					options := (&client.ApplyOptions{}).ApplyOptions(opts)
					require.Equal(t, "observability-operator", options.FieldManager)
					require.Equal(t, ptr.To(true), options.Force)
					policy := decodeAppliedNetworkPolicy(t, obj)
					require.NotEmpty(t, policy.UID)
					require.NotEmpty(t, policy.ResourceVersion)
					return c.Apply(ctx, obj, opts...)
				},
			})

			r := NewNetworkPolicyUpdater(desired, owner, base)
			require.NoError(t, r.Reconcile(ctx, c, scheme))
			require.Equal(t, 1, applies)
			if existing {
				require.Zero(t, creates)
			} else {
				require.Equal(t, 1, creates)
			}
			current := &networkingv1.NetworkPolicy{}
			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(desired), current))
			require.Equal(t, desired.Spec, current.Spec)
			require.Equal(t, "observability-operator", current.Labels["app.kubernetes.io/managed-by"])
			require.Equal(t, owner.Name, current.Labels["app.kubernetes.io/part-of"])
			require.Equal(t, desired.Name, current.Labels["app.kubernetes.io/name"])
			ref := metav1.GetControllerOf(current)
			require.NotNil(t, ref)
			require.Equal(t, owner.UID, ref.UID)
			require.Equal(t, owner.Name, ref.Name)
			require.Equal(t, "ConfigMap", ref.Kind)
			require.Equal(t, ptr.To(true), ref.BlockOwnerDeletion)
			if existing {
				require.Equal(t, "original-policy", string(current.UID))
				require.Equal(t, "preserved", current.Labels["user-label"])
			}
			// A second pass updates the same object instead of reserving again.
			require.NoError(t, r.Reconcile(ctx, c, scheme))
			require.Equal(t, 2, applies)
			require.Equal(t, original, desired, "reconciliation must not mutate the builder's policy")
		})
	}
}

func TestNetworkPolicyCacheMiss(t *testing.T) {
	tt := []struct {
		name      string
		existing  bool
		present   bool
		managedBy string
		wantCalls []string
	}{
		{"create missing policy", false, true, "", []string{"create", "apply"}},
		{"restore removed label", true, true, "", []string{"create", "get uncached", "apply"}},
		{"restore changed label", true, true, "someone-else", []string{"create", "get uncached", "apply"}},
		{"stale cache", true, true, "observability-operator", []string{"create", "get uncached", "apply"}},
		{"delete unlabeled policy", true, false, "", []string{"get uncached", "delete"}},
		{"delete missing policy", false, false, "", []string{"get uncached"}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			scheme, owner, desired := networkPolicyTestObjects(t)
			var objects []client.Object
			if tc.existing {
				current := ownedTestNetworkPolicy(t, scheme, owner, desired)
				current.Spec.Ingress = nil
				current.Spec.Egress = nil
				current.Labels = map[string]string{"user-label": "preserved"}
				if tc.managedBy != "" {
					current.Labels["app.kubernetes.io/managed-by"] = tc.managedBy
				}
				objects = append(objects, current)
			}
			base := networkPolicyTestClient(t, scheme, objects...)
			var calls []string
			apiReader := interceptor.NewClient(base, interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					calls = append(calls, "get uncached")
					return c.Get(ctx, key, obj, opts...)
				},
			})
			c := interceptor.NewClient(networkPolicyCacheMissClient(base), interceptor.Funcs{
				Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					calls = append(calls, "create")
					return c.Create(ctx, obj, opts...)
				},
				Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
					calls = append(calls, "apply")
					return c.Apply(ctx, obj, opts...)
				},
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					calls = append(calls, "delete")
					return c.Delete(ctx, obj, opts...)
				},
			})

			err := NewOptionalNetworkPolicyUpdater(desired, owner, apiReader, tc.present).Reconcile(ctx, c, scheme)
			require.NoError(t, err)
			require.Equal(t, tc.wantCalls, calls)
			actual := &networkingv1.NetworkPolicy{}
			err = base.Get(ctx, client.ObjectKeyFromObject(desired), actual)
			if !tc.present {
				require.True(t, apierrors.IsNotFound(err), "%v", err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, desired.Spec, actual.Spec)
			require.Equal(t, "observability-operator", actual.Labels["app.kubernetes.io/managed-by"])
			require.Equal(t, owner.UID, metav1.GetControllerOf(actual).UID)
			if tc.existing {
				require.Equal(t, types.UID("original-policy"), actual.UID, "repair must not recreate the policy")
				require.Equal(t, "preserved", actual.Labels["user-label"])
			}
		})
	}
}

func TestNetworkPolicyOwnershipCollisions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*networkingv1.NetworkPolicy)
	}{
		{"foreign controller", func(p *networkingv1.NetworkPolicy) {
			p.OwnerReferences[0].Name = "foreign-owner"
			p.OwnerReferences[0].UID = "foreign-uid"
		}},
		{"recreated owner with old UID", func(p *networkingv1.NetworkPolicy) { p.OwnerReferences[0].UID = "old-uid" }},
		{"unmanaged with spoofed labels", func(p *networkingv1.NetworkPolicy) {
			p.OwnerReferences = nil
			p.Labels = map[string]string{"app.kubernetes.io/managed-by": "observability-operator"}
		}},
		{"non-controller owner reference", func(p *networkingv1.NetworkPolicy) { p.OwnerReferences[0].Controller = ptr.To(false) }},
		{"foreign owner name with same UID", func(p *networkingv1.NetworkPolicy) { p.OwnerReferences[0].Name = "foreign-name" }},
		{"foreign owner kind with same UID", func(p *networkingv1.NetworkPolicy) { p.OwnerReferences[0].Kind = "Secret" }},
		{"foreign owner group with same UID", func(p *networkingv1.NetworkPolicy) { p.OwnerReferences[0].APIVersion = "foreign.example/v1" }},
	} {
		for _, cacheMiss := range []bool{false, true} {
			name := tc.name
			if cacheMiss {
				name += " with cache miss"
			}
			t.Run(name, func(t *testing.T) {
				for _, present := range []bool{true, false} {
					scheme, owner, desired := networkPolicyTestObjects(t)
					current := ownedTestNetworkPolicy(t, scheme, owner, desired)
					tc.mutate(current)
					base := networkPolicyTestClient(t, scheme, current)
					cached := base
					if cacheMiss {
						cached = networkPolicyCacheMissClient(base)
					}
					var creates, writes int
					c := interceptor.NewClient(cached, interceptor.Funcs{
						Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
							creates++
							return c.Create(ctx, obj, opts...)
						},
						Apply: func(context.Context, client.WithWatch, runtime.ApplyConfiguration, ...client.ApplyOption) error {
							writes++
							return nil
						},
						Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
							writes++
							return nil
						},
					})
					before := &networkingv1.NetworkPolicy{}
					require.NoError(t, base.Get(context.Background(), client.ObjectKeyFromObject(current), before))
					err := NewOptionalNetworkPolicyUpdater(desired, owner, base, present).Reconcile(context.Background(), c, scheme)
					require.ErrorContains(t, err, "ownership conflict")
					require.Zero(t, writes)
					if cacheMiss && present {
						require.Equal(t, 1, creates)
					} else {
						require.Zero(t, creates)
					}
					after := &networkingv1.NetworkPolicy{}
					require.NoError(t, base.Get(context.Background(), client.ObjectKeyFromObject(current), after))
					require.Equal(t, before, after)
				}
			})
		}
	}
}

func TestNetworkPolicySafeDeletion(t *testing.T) {
	for _, optional := range []bool{false, true} {
		ctx := context.Background()
		scheme, owner, desired := networkPolicyTestObjects(t)
		current := ownedTestNetworkPolicy(t, scheme, owner, desired)
		base := networkPolicyTestClient(t, scheme, current)
		var deletes int
		c := interceptor.NewClient(base, interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				deletes++
				options := (&client.DeleteOptions{}).ApplyOptions(opts)
				require.NotNil(t, options.Preconditions)
				require.Equal(t, ptr.To(current.UID), options.Preconditions.UID)
				require.Equal(t, ptr.To(current.ResourceVersion), options.Preconditions.ResourceVersion)
				return c.Delete(ctx, obj, opts...)
			},
		})
		r := NewNetworkPolicyDeleter(desired, owner, base)
		if optional {
			r = NewOptionalNetworkPolicyUpdater(desired, owner, base, false)
		}
		require.NoError(t, r.Reconcile(ctx, c, scheme))
		require.Equal(t, 1, deletes)
		require.True(t, apierrors.IsNotFound(base.Get(ctx, client.ObjectKeyFromObject(desired), &networkingv1.NetworkPolicy{})))
		require.NoError(t, r.Reconcile(ctx, c, scheme), "missing policy must be a no-op")
		require.Equal(t, 1, deletes)
		// Re-enable the component after its policy was removed.
		require.NoError(t, NewOptionalNetworkPolicyUpdater(desired, owner, base, true).Reconcile(ctx, c, scheme))
		require.NoError(t, base.Get(ctx, client.ObjectKeyFromObject(desired), &networkingv1.NetworkPolicy{}))
	}
}

func TestNetworkPolicyConcurrentCreationCollision(t *testing.T) {
	ctx := context.Background()
	scheme, owner, desired := networkPolicyTestObjects(t)
	base := networkPolicyTestClient(t, scheme)
	foreign := desired.DeepCopy()
	foreign.Spec.Ingress = nil
	c := interceptor.NewClient(base, interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			require.NoError(t, c.Create(ctx, foreign))
			return c.Create(ctx, obj, opts...)
		},
		Apply: func(context.Context, client.WithWatch, runtime.ApplyConfiguration, ...client.ApplyOption) error {
			t.Fatal("must not apply when another writer reserved the name")
			return nil
		},
	})
	err := NewNetworkPolicyUpdater(desired, owner, base).Reconcile(ctx, c, scheme)
	require.ErrorContains(t, err, "ownership conflict")
	current := &networkingv1.NetworkPolicy{}
	require.NoError(t, base.Get(ctx, client.ObjectKeyFromObject(desired), current))
	require.Empty(t, current.OwnerReferences)
	require.Nil(t, current.Spec.Ingress)
}

func TestNetworkPolicyConcurrentOwnershipChange(t *testing.T) {
	for _, present := range []bool{true, false} {
		ctx := context.Background()
		scheme, owner, desired := networkPolicyTestObjects(t)
		current := ownedTestNetworkPolicy(t, scheme, owner, desired)
		base := networkPolicyTestClient(t, scheme, current)
		var changed *networkingv1.NetworkPolicy
		changeOwner := func(c client.WithWatch) {
			changed = &networkingv1.NetworkPolicy{}
			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(desired), changed))
			changed.OwnerReferences[0].UID = "foreign-uid"
			require.NoError(t, c.Update(ctx, changed))
		}
		c := interceptor.NewClient(base, interceptor.Funcs{
			Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
				changeOwner(c)
				return c.Apply(ctx, obj, opts...)
			},
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				changeOwner(c)
				return c.Delete(ctx, obj, opts...)
			},
		})
		err := NewOptionalNetworkPolicyUpdater(desired, owner, base, present).Reconcile(ctx, c, scheme)
		require.True(t, apierrors.IsConflict(err), "%v", err)
		after := &networkingv1.NetworkPolicy{}
		require.NoError(t, base.Get(ctx, client.ObjectKeyFromObject(desired), after))
		require.Equal(t, changed, after, "raced mutation must not overwrite or delete the foreign policy")
	}
}

func TestNetworkPolicyConcurrentReplacement(t *testing.T) {
	for _, present := range []bool{true, false} {
		ctx := context.Background()
		scheme, owner, desired := networkPolicyTestObjects(t)
		current := ownedTestNetworkPolicy(t, scheme, owner, desired)
		base := networkPolicyTestClient(t, scheme, current)
		var replacement *networkingv1.NetworkPolicy
		replace := func(c client.WithWatch) {
			require.NoError(t, c.Delete(ctx, current))
			replacement = desired.DeepCopy()
			replacement.UID = "foreign-policy"
			replacement.Spec.Ingress = nil
			require.NoError(t, c.Create(ctx, replacement))
		}
		c := interceptor.NewClient(base, interceptor.Funcs{
			Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
				replace(c)
				return c.Apply(ctx, obj, opts...)
			},
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				replace(c)
				return c.Delete(ctx, obj, opts...)
			},
		})
		err := NewOptionalNetworkPolicyUpdater(desired, owner, base, present).Reconcile(ctx, c, scheme)
		require.True(t, apierrors.IsConflict(err), "%v", err)
		after := &networkingv1.NetworkPolicy{}
		require.NoError(t, base.Get(ctx, client.ObjectKeyFromObject(desired), after))
		require.Equal(t, replacement, after)
	}
}

func TestNetworkPolicyAdmissionOwnerChange(t *testing.T) {
	scheme, owner, desired := networkPolicyTestObjects(t)
	base := networkPolicyTestClient(t, scheme)
	c := interceptor.NewClient(base, interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			obj.GetOwnerReferences()[0].UID = "admission-owner"
			return c.Create(ctx, obj, opts...)
		},
		Apply: func(context.Context, client.WithWatch, runtime.ApplyConfiguration, ...client.ApplyOption) error {
			t.Fatal("must not apply after admission changed the controller")
			return nil
		},
	})
	err := NewNetworkPolicyUpdater(desired, owner, base).Reconcile(context.Background(), c, scheme)
	require.ErrorContains(t, err, "ownership conflict")
	current := &networkingv1.NetworkPolicy{}
	require.NoError(t, base.Get(context.Background(), client.ObjectKeyFromObject(desired), current))
	require.Equal(t, types.UID("admission-owner"), metav1.GetControllerOf(current).UID)
}

func TestNetworkPolicyAPIErrors(t *testing.T) {
	gr := schema.GroupResource{Group: networkingv1.GroupName, Resource: "networkpolicies"}
	for _, operation := range []string{"get", "create", "apply", "delete", "delete-not-found"} {
		t.Run(operation, func(t *testing.T) {
			scheme, owner, desired := networkPolicyTestObjects(t)
			failure := apierrors.NewForbidden(gr, desired.Name, errors.New("denied"))
			var objects []client.Object
			if operation != "create" {
				objects = append(objects, ownedTestNetworkPolicy(t, scheme, owner, desired))
			}
			base := networkPolicyTestClient(t, scheme, objects...)
			funcs := interceptor.Funcs{}
			switch operation {
			case "get":
				funcs.Get = func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return failure
				}
			case "create":
				funcs.Create = func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error { return failure }
			case "apply":
				funcs.Apply = func(context.Context, client.WithWatch, runtime.ApplyConfiguration, ...client.ApplyOption) error {
					return failure
				}
			case "delete":
				funcs.Delete = func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error { return failure }
			case "delete-not-found":
				funcs.Delete = func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
					return apierrors.NewNotFound(gr, desired.Name)
				}
			}
			present := operation != "delete" && operation != "delete-not-found"
			err := NewOptionalNetworkPolicyUpdater(desired, owner, base, present).Reconcile(context.Background(), interceptor.NewClient(base, funcs), scheme)
			if operation == "delete-not-found" {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, failure)
				require.True(t, apierrors.IsForbidden(err))
			}
		})
	}
}

func TestNetworkPolicyUncachedReadErrors(t *testing.T) {
	tt := []struct {
		name     string
		present  bool
		notFound bool
	}{
		{"update forbidden", true, false},
		{"deleted after create conflict", true, true},
		{"delete forbidden", false, false},
		{"delete not found", false, true},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			scheme, owner, desired := networkPolicyTestObjects(t)
			current := ownedTestNetworkPolicy(t, scheme, owner, desired)
			base := networkPolicyTestClient(t, scheme, current)
			before := &networkingv1.NetworkPolicy{}
			require.NoError(t, base.Get(context.Background(), client.ObjectKeyFromObject(desired), before))
			gr := schema.GroupResource{Group: networkingv1.GroupName, Resource: "networkpolicies"}
			failure := apierrors.NewForbidden(gr, desired.Name, errors.New("denied"))
			if tc.notFound {
				failure = apierrors.NewNotFound(gr, desired.Name)
			}
			apiReader := interceptor.NewClient(base, interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return failure
				},
			})
			err := NewOptionalNetworkPolicyUpdater(desired, owner, apiReader, tc.present).Reconcile(context.Background(), networkPolicyCacheMissClient(base), scheme)
			if !tc.present && tc.notFound {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, failure)
			}
			actual := &networkingv1.NetworkPolicy{}
			require.NoError(t, base.Get(context.Background(), client.ObjectKeyFromObject(desired), actual))
			require.Equal(t, before, actual, "failed live reads must not modify or delete the policy")
		})
	}
}

func TestNetworkPolicyOwnerScope(t *testing.T) {
	scheme, owner, desired := networkPolicyTestObjects(t)
	for _, tc := range []struct {
		name      string
		owner     metav1.Object
		wantError string
	}{
		{"same namespace", owner, ""},
		{"cluster scoped", &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cluster-owner", UID: "cluster-owner-uid"}}, ""},
		{"cross namespace", &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: owner.Name, Namespace: "other-ns", UID: owner.UID}}, "cross-namespace owner references are disallowed"},
		{"missing UID", &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: owner.Name, Namespace: owner.Namespace}}, "controller owner UID must be set"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := networkPolicyTestClient(t, scheme)
			err := NewNetworkPolicyUpdater(desired, tc.owner, c).Reconcile(context.Background(), c, scheme)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.True(t, apierrors.IsNotFound(c.Get(context.Background(), client.ObjectKeyFromObject(desired), &networkingv1.NetworkPolicy{})))
			} else {
				require.NoError(t, err)
				current := &networkingv1.NetworkPolicy{}
				require.NoError(t, c.Get(context.Background(), client.ObjectKeyFromObject(desired), current))
				require.Equal(t, tc.owner.GetUID(), metav1.GetControllerOf(current).UID)
			}
		})
	}
}

func TestNetworkPolicyNameLabelValue(t *testing.T) {
	const labelName = "app.kubernetes.io/name"
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "within limit", value: strings.Repeat("a", 63)},
		{name: "over limit", value: strings.Repeat("a", 64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme, owner, policy := networkPolicyTestObjects(t)
			policy.Name = tc.value
			client := networkPolicyTestClient(t, scheme)

			require.NoError(t, NewNetworkPolicyUpdater(policy, owner, client).Reconcile(context.Background(), client, scheme))

			actual := &networkingv1.NetworkPolicy{}
			require.NoError(t, client.Get(context.Background(), clientObjectKey(policy), actual))
			require.LessOrEqual(t, len(actual.Labels[labelName]), 63)
			if len(tc.value) <= 63 {
				require.Equal(t, tc.value, actual.Labels[labelName])
			} else {
				require.NotEqual(t, tc.value, actual.Labels[labelName])
				require.True(t, strings.HasPrefix(actual.Labels[labelName], tc.value[:54]+"-"))
			}
		})
	}
}

func clientObjectKey(obj client.Object) client.ObjectKey {
	return client.ObjectKeyFromObject(obj)
}

func networkPolicyTestObjects(t *testing.T) (*runtime.Scheme, *corev1.ConfigMap, *networkingv1.NetworkPolicy) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, networkingv1.AddToScheme(scheme))
	owner := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns", UID: "owner-uid"}}
	policy := NewAllowAllNetworkPolicy("example-prometheus", owner.Namespace, map[string]string{"app.kubernetes.io/instance": "example"})
	return scheme, owner, policy
}

func ownedTestNetworkPolicy(t *testing.T, scheme *runtime.Scheme, owner metav1.Object, desired *networkingv1.NetworkPolicy) *networkingv1.NetworkPolicy {
	t.Helper()
	policy := desired.DeepCopy()
	policy.UID = "original-policy"
	policy.ResourceVersion = "10"
	require.NoError(t, controllerutil.SetControllerReference(owner, policy, scheme))
	return policy
}

func decodeAppliedNetworkPolicy(t *testing.T, obj runtime.ApplyConfiguration) *networkingv1.NetworkPolicy {
	t.Helper()
	raw, err := json.Marshal(obj)
	require.NoError(t, err)
	policy := &networkingv1.NetworkPolicy{}
	require.NoError(t, json.Unmarshal(raw, policy))
	return policy
}

func networkPolicyCacheMissClient(base client.WithWatch) client.WithWatch {
	return interceptor.NewClient(base, interceptor.Funcs{
		Get: func(_ context.Context, _ client.WithWatch, key client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
			return apierrors.NewNotFound(schema.GroupResource{Group: networkingv1.GroupName, Resource: "networkpolicies"}, key.Name)
		},
	})
}

func networkPolicyTestClient(t *testing.T, scheme *runtime.Scheme, objects ...client.Object) client.WithWatch {
	t.Helper()
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).WithGlobalResourceVersionCounter().
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				// The fake client does not allocate UIDs like the API server.
				if obj.GetUID() == "" {
					obj.SetUID(types.UID("created-policy"))
				}
				return c.Create(ctx, obj, opts...)
			},
			Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
				// fake.Apply zeroes its input before decoding the response, which
				// cannot preserve the generic wrapper's unexported object field.
				// Use an equivalent generated configuration for the fake's SSA.
				raw, err := json.Marshal(obj)
				if err != nil {
					return err
				}
				config := &networkingac.NetworkPolicyApplyConfiguration{}
				if err := json.Unmarshal(raw, config); err != nil {
					return err
				}
				return c.Apply(ctx, config, opts...)
			},
		}).Build()
}

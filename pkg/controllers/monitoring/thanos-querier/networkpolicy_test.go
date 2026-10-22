package thanos_querier

import (
	"context"
	"encoding/json"
	"testing"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	msoapi "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

func TestNewNetworkPolicy(t *testing.T) {
	np := newNetworkPolicy("thanos-querier-example", "test-ns")

	require.Equal(t, "thanos-querier-example-network-policy", np.Name)
	require.Equal(t, "test-ns", np.Namespace)
	require.Equal(t, componentLabels("thanos-querier-example"), np.Spec.PodSelector.MatchLabels)
	require.Equal(t, []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}, np.Spec.PolicyTypes)
	require.Equal(t, []networkingv1.NetworkPolicyIngressRule{{}}, np.Spec.Ingress)
	require.Equal(t, []networkingv1.NetworkPolicyEgressRule{{}}, np.Spec.Egress)
}

func TestThanosComponentReconcilers_ReconcileNetworkPolicy(t *testing.T) {
	querier := &msoapi.ThanosQuerier{
		TypeMeta: metav1.TypeMeta{
			APIVersion: msoapi.GroupVersion.String(),
			Kind:       "ThanosQuerier",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "test-ns",
			UID:       "thanos-querier-uid",
		},
	}

	scheme := k8sruntime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, monv1.AddToScheme(scheme))
	require.NoError(t, msoapi.AddToScheme(scheme))

	rec := newRecordingClient()
	for _, r := range thanosComponentReconcilers(querier, nil, ThanosConfiguration{}, nil, rec) {
		require.NoError(t, r.Reconcile(context.Background(), rec, scheme))
	}

	np := appliedNetworkPolicy(t, rec.applied, "thanos-querier-example-network-policy")
	require.Equal(t, componentLabels("thanos-querier-example"), np.Spec.PodSelector.MatchLabels)
	require.Equal(t, []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}, np.Spec.PolicyTypes)
	require.Equal(t, []networkingv1.NetworkPolicyIngressRule{{}}, np.Spec.Ingress)
	require.Equal(t, []networkingv1.NetworkPolicyEgressRule{{}}, np.Spec.Egress)

	require.Len(t, np.OwnerReferences, 1)
	require.Equal(t, querier.Name, np.OwnerReferences[0].Name)
	require.Equal(t, "ThanosQuerier", np.OwnerReferences[0].Kind)
	require.NotNil(t, np.OwnerReferences[0].Controller)
	require.True(t, *np.OwnerReferences[0].Controller)
}

type recordingClient struct {
	client.Client
	applied  []k8sruntime.ApplyConfiguration
	deleted  []client.Object
	policies map[client.ObjectKey]*networkingv1.NetworkPolicy
}

func newRecordingClient() *recordingClient {
	return &recordingClient{policies: map[client.ObjectKey]*networkingv1.NetworkPolicy{}}
}

func (r *recordingClient) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	policy, ok := r.policies[key]
	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{Group: networkingv1.GroupName, Resource: "networkpolicies"}, key.Name)
	}
	np, ok := obj.(*networkingv1.NetworkPolicy)
	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{Group: networkingv1.GroupName, Resource: "networkpolicies"}, key.Name)
	}
	*np = *policy.DeepCopy()
	return nil
}

func (r *recordingClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	np, ok := obj.(*networkingv1.NetworkPolicy)
	if !ok {
		return nil
	}
	key := client.ObjectKeyFromObject(np)
	if _, exists := r.policies[key]; exists {
		return apierrors.NewAlreadyExists(schema.GroupResource{Group: networkingv1.GroupName, Resource: "networkpolicies"}, np.Name)
	}
	np.UID = types.UID("created-" + np.Name)
	np.ResourceVersion = "1"
	r.policies[key] = np.DeepCopy()
	return nil
}

func (r *recordingClient) Apply(_ context.Context, obj k8sruntime.ApplyConfiguration, _ ...client.ApplyOption) error {
	r.applied = append(r.applied, obj)
	return nil
}

func (r *recordingClient) Delete(_ context.Context, obj client.Object, _ ...client.DeleteOption) error {
	r.deleted = append(r.deleted, obj)
	if np, ok := obj.(*networkingv1.NetworkPolicy); ok {
		delete(r.policies, client.ObjectKeyFromObject(np))
	}
	return nil
}

func appliedNetworkPolicy(t *testing.T, objs []k8sruntime.ApplyConfiguration, name string) *networkingv1.NetworkPolicy {
	t.Helper()
	for _, obj := range objs {
		raw, err := json.Marshal(obj)
		require.NoError(t, err)

		var meta metav1.TypeMeta
		require.NoError(t, json.Unmarshal(raw, &meta))
		if meta.Kind != "NetworkPolicy" {
			continue
		}

		var np networkingv1.NetworkPolicy
		require.NoError(t, json.Unmarshal(raw, &np))
		if np.Name == name {
			return &np
		}
	}

	t.Fatalf("network policy %s was not reconciled", name)
	return nil
}

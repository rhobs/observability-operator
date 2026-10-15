package monitoringstack

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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

func TestNewPrometheusNetworkPolicy(t *testing.T) {
	ms := testMonitoringStack(false)

	np := newPrometheusNetworkPolicy(ms)

	require.Equal(t, "example-prometheus-network-policy", np.Name)
	require.Equal(t, "test-ns", np.Namespace)
	assertPermissiveNetworkPolicySpec(t, np, podLabels("prometheus", ms.Name))
}

func TestNewAlertmanagerNetworkPolicy(t *testing.T) {
	ms := testMonitoringStack(false)

	np := newAlertmanagerNetworkPolicy(ms)

	require.Equal(t, "example-alertmanager-network-policy", np.Name)
	require.Equal(t, "test-ns", np.Namespace)
	assertPermissiveNetworkPolicySpec(t, np, podLabels("alertmanager", ms.Name))
}

func TestStackComponentReconcilers_ReconcileNetworkPolicies(t *testing.T) {
	ms := testMonitoringStack(false)
	scheme := testMonitoringStackScheme(t)
	rec := newRecordingClient()

	for _, r := range stackComponentReconcilers(ms, ThanosConfiguration{}, PrometheusConfiguration{}, AlertmanagerConfiguration{}) {
		require.NoError(t, r.Reconcile(context.Background(), rec, scheme))
	}

	applied := appliedNetworkPolicies(t, rec.applied)
	require.Contains(t, applied, "example-prometheus-network-policy")
	require.Contains(t, applied, "example-alertmanager-network-policy")

	promNP := applied["example-prometheus-network-policy"]
	assertPermissiveNetworkPolicySpec(t, &promNP, podLabels("prometheus", ms.Name))
	require.Len(t, promNP.OwnerReferences, 1)
	require.Equal(t, ms.Name, promNP.OwnerReferences[0].Name)
	require.Equal(t, "MonitoringStack", promNP.OwnerReferences[0].Kind)
	require.NotNil(t, promNP.OwnerReferences[0].Controller)
	require.True(t, *promNP.OwnerReferences[0].Controller)

	alertNP := applied["example-alertmanager-network-policy"]
	assertPermissiveNetworkPolicySpec(t, &alertNP, podLabels("alertmanager", ms.Name))
}

func TestStackComponentReconcilers_DeletesAlertmanagerNetworkPolicyWhenDisabled(t *testing.T) {
	ms := testMonitoringStack(false)
	scheme := testMonitoringStackScheme(t)
	rec := newRecordingClient()

	for _, r := range stackComponentReconcilers(ms, ThanosConfiguration{}, PrometheusConfiguration{}, AlertmanagerConfiguration{}) {
		require.NoError(t, r.Reconcile(context.Background(), rec, scheme))
	}

	disabled := ms.DeepCopy()
	disabled.Spec.AlertmanagerConfig.Disabled = true
	for _, r := range stackComponentReconcilers(disabled, ThanosConfiguration{}, PrometheusConfiguration{}, AlertmanagerConfiguration{}) {
		require.NoError(t, r.Reconcile(context.Background(), rec, scheme))
	}

	require.Contains(t, deletedNetworkPolicyNames(rec.deleted), "example-alertmanager-network-policy")
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

func appliedNetworkPolicies(t *testing.T, objs []k8sruntime.ApplyConfiguration) map[string]networkingv1.NetworkPolicy {
	t.Helper()
	result := map[string]networkingv1.NetworkPolicy{}

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
		result[np.Name] = np
	}

	return result
}

func deletedNetworkPolicyNames(objs []client.Object) []string {
	names := make([]string, 0)
	for _, obj := range objs {
		np, ok := obj.(*networkingv1.NetworkPolicy)
		if !ok {
			continue
		}
		names = append(names, np.Name)
	}
	return names
}

func assertPermissiveNetworkPolicySpec(t *testing.T, np *networkingv1.NetworkPolicy, selector map[string]string) {
	t.Helper()
	require.Equal(t, selector, np.Spec.PodSelector.MatchLabels)
	require.Equal(t, []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}, np.Spec.PolicyTypes)
	require.Equal(t, []networkingv1.NetworkPolicyIngressRule{{}}, np.Spec.Ingress)
	require.Equal(t, []networkingv1.NetworkPolicyEgressRule{{}}, np.Spec.Egress)
}

func testMonitoringStackScheme(t *testing.T) *k8sruntime.Scheme {
	t.Helper()
	scheme := k8sruntime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, monv1.AddToScheme(scheme))
	require.NoError(t, stack.AddToScheme(scheme))
	return scheme
}

func testMonitoringStack(disabledAlertmanager bool) *stack.MonitoringStack {
	return &stack.MonitoringStack{
		TypeMeta: metav1.TypeMeta{
			APIVersion: stack.GroupVersion.String(),
			Kind:       "MonitoringStack",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "test-ns",
			UID:       "monitoring-stack-uid",
		},
		Spec: stack.MonitoringStackSpec{
			PrometheusConfig: &stack.PrometheusConfig{
				Replicas: ptr.To(int32(1)),
			},
			AlertmanagerConfig: stack.AlertmanagerConfig{
				Disabled: disabledAlertmanager,
				Replicas: ptr.To(int32(1)),
			},
		},
	}
}

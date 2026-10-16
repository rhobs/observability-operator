package reconciler

import (
	"testing"

	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestNewAllowAllNetworkPolicy(t *testing.T) {
	selector := map[string]string{
		"app.kubernetes.io/component": "prometheus",
		"app.kubernetes.io/part-of":   "example-stack",
	}

	np := NewAllowAllNetworkPolicy("example-prometheus", "test-ns", selector)

	require.Equal(t, "NetworkPolicy", np.Kind)
	require.Equal(t, networkingv1.SchemeGroupVersion.String(), np.APIVersion)
	require.Equal(t, "example-prometheus", np.Name)
	require.Equal(t, "test-ns", np.Namespace)
	require.Equal(t, selector, np.Spec.PodSelector.MatchLabels)
	require.Equal(t, []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}, np.Spec.PolicyTypes)
	require.Equal(t, []networkingv1.NetworkPolicyIngressRule{{}}, np.Spec.Ingress)
	require.Equal(t, []networkingv1.NetworkPolicyEgressRule{{}}, np.Spec.Egress)
}

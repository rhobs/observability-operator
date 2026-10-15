package uiplugin

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"gotest.tools/v3/assert"
)

type expectedNetworkPolicyIngressRule struct {
	from  []expectedNetworkPolicyPeer
	ports []int32
}

type expectedNetworkPolicyPeer struct {
	namespacePolicyGroup string
	podSelector          map[string]string
}

func TestNetworkPolicyBuilders(t *testing.T) {
	const namespace = "openshift-operators"

	testCases := []struct {
		name                string
		policy              *networkingv1.NetworkPolicy
		expectedName        string
		expectedPodSelector map[string]string
		expectedIngress     []expectedNetworkPolicyIngressRule
	}{
		{
			name:                "monitoring plugin backend allows console and acm proxy ports",
			policy:              newPluginBackendNetworkPolicy("monitoring", uiv1alpha1.TypeMonitoring, namespace),
			expectedName:        "monitoring-network-policy",
			expectedPodSelector: componentLabels("monitoring"),
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespacePolicyGroup: consolePolicyGroup},
				},
				ports: []int32{pluginBackendPort, monitoringAlertmanagerProxyPort, monitoringThanosQueryProxyPort},
			}},
		},
		{
			name:                "other plugin backend remains console to 9443",
			policy:              newPluginBackendNetworkPolicy("logging", uiv1alpha1.TypeLogging, namespace),
			expectedName:        "logging-network-policy",
			expectedPodSelector: componentLabels("logging"),
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespacePolicyGroup: consolePolicyGroup},
				},
				ports: []int32{pluginBackendPort},
			}},
		},
		{
			name:                "korrel8r",
			policy:              newKorrel8rNetworkPolicy(namespace),
			expectedName:        "korrel8r-network-policy",
			expectedPodSelector: componentLabels(korrel8rName),
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespacePolicyGroup: consolePolicyGroup},
				},
				ports: []int32{korrel8rPort},
			}},
		},
		{
			name:                "perses allows console and perses operator controller pods",
			policy:              newPersesNetworkPolicy(namespace),
			expectedName:        "perses-network-policy",
			expectedPodSelector: persesWorkloadSelector(),
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespacePolicyGroup: consolePolicyGroup},
					{podSelector: map[string]string{
						"app.kubernetes.io/name":      "perses-operator",
						"app.kubernetes.io/component": "controller",
						"app.kubernetes.io/part-of":   "observability-operator",
					}},
				},
				ports: []int32{persesPort},
			}},
		},
		{
			name:                "health analyzer",
			policy:              newHealthAnalyzerNetworkPolicy(namespace),
			expectedName:        "health-analyzer-network-policy",
			expectedPodSelector: componentLabels(healthAnalyzerWorkloadName),
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespacePolicyGroup: monitoringPolicyGroup},
				},
				ports: []int32{healthAnalyzerPort},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedName, tc.policy.Name)
			assert.DeepEqual(t, tc.expectedPodSelector, tc.policy.Spec.PodSelector.MatchLabels)
			assert.DeepEqual(t, []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}, tc.policy.Spec.PolicyTypes)
			assert.DeepEqual(t, []networkingv1.NetworkPolicyEgressRule{{}}, tc.policy.Spec.Egress)
			assertNetworkPolicyIngressRules(t, tc.expectedIngress, tc.policy.Spec.Ingress)
		})
	}
}

func assertNetworkPolicyIngressRules(t *testing.T, expectedRules []expectedNetworkPolicyIngressRule, ingressRules []networkingv1.NetworkPolicyIngressRule) {
	t.Helper()

	assert.Equal(t, len(expectedRules), len(ingressRules))

	for i, expectedRule := range expectedRules {
		ingressRule := ingressRules[i]
		assert.Equal(t, len(expectedRule.from), len(ingressRule.From))

		for j, expectedPeer := range expectedRule.from {
			actualPeer := ingressRule.From[j]

			if expectedPeer.namespacePolicyGroup != "" {
				expectedNamespaceSelector := &metav1.LabelSelector{
					MatchLabels: map[string]string{namespacePolicyGroupLabel: expectedPeer.namespacePolicyGroup},
				}
				assert.DeepEqual(t, expectedNamespaceSelector, actualPeer.NamespaceSelector)
			} else {
				assert.Assert(t, actualPeer.NamespaceSelector == nil)
			}

			if expectedPeer.podSelector != nil {
				expectedPodSelector := &metav1.LabelSelector{MatchLabels: expectedPeer.podSelector}
				assert.DeepEqual(t, expectedPodSelector, actualPeer.PodSelector)
			} else {
				assert.Assert(t, actualPeer.PodSelector == nil)
			}
		}

		assert.Equal(t, len(expectedRule.ports), len(ingressRule.Ports))
		for j, expectedPort := range expectedRule.ports {
			tcpProtocol := corev1.ProtocolTCP
			tcpPort := intstr.FromInt32(expectedPort)

			assert.DeepEqual(t, &tcpProtocol, ingressRule.Ports[j].Protocol)
			assert.DeepEqual(t, &tcpPort, ingressRule.Ports[j].Port)
		}
	}
}

func TestPersesWorkloadSelector(t *testing.T) {
	assert.DeepEqual(t, map[string]string{
		"app.kubernetes.io/instance":   "perses",
		"app.kubernetes.io/managed-by": "perses-operator",
	}, persesWorkloadSelector())
}

func TestPersesOperatorPodSelector(t *testing.T) {
	assert.DeepEqual(t, map[string]string{
		"app.kubernetes.io/name":      "perses-operator",
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/part-of":   "observability-operator",
	}, persesOperatorPodSelector())
}

func TestPluginNetworkPolicyReconcilers(t *testing.T) {
	testCases := []struct {
		name                  string
		pluginType            uiv1alpha1.UIPluginType
		monitoringConfig      *uiv1alpha1.MonitoringConfig
		korrel8rEnabled       bool
		healthAnalyzerEnabled bool
		persesEnabled         bool
		expectedPolicies      int
	}{
		{
			name:             "troubleshooting with korrel8r enabled",
			pluginType:       uiv1alpha1.TypeTroubleshootingPanel,
			korrel8rEnabled:  true,
			expectedPolicies: 2,
		},
		{
			name:             "troubleshooting with korrel8r disabled",
			pluginType:       uiv1alpha1.TypeTroubleshootingPanel,
			korrel8rEnabled:  false,
			expectedPolicies: 2,
		},
		{
			name:                  "monitoring with optional workloads enabled",
			pluginType:            uiv1alpha1.TypeMonitoring,
			healthAnalyzerEnabled: true,
			persesEnabled:         true,
			expectedPolicies:      3,
		},
		{
			name:                  "monitoring with optional workloads disabled",
			pluginType:            uiv1alpha1.TypeMonitoring,
			healthAnalyzerEnabled: false,
			persesEnabled:         false,
			expectedPolicies:      3,
		},
		{
			name:             "logging",
			pluginType:       uiv1alpha1.TypeLogging,
			expectedPolicies: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := &uiv1alpha1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: uiv1alpha1.UIPluginSpec{
					Type:       tc.pluginType,
					Monitoring: tc.monitoringConfig,
				},
			}

			reconcilers := pluginNetworkPolicyReconcilers(plugin, UIPluginInfo{Name: "plugin"}, "openshift-operators", tc.korrel8rEnabled, tc.healthAnalyzerEnabled, tc.persesEnabled)

			assert.Equal(t, tc.expectedPolicies, len(reconcilers))
		})
	}
}

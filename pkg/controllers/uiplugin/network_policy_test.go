package uiplugin

import (
	"context"
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"gotest.tools/v3/assert"
)

type expectedNetworkPolicyIngressRule struct {
	from  []expectedNetworkPolicyPeer
	ports []int32
}

type expectedNetworkPolicyPeer struct {
	namespaceSelector *metav1.LabelSelector
	podSelector       map[string]string
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
					{namespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-console"}}},
				},
				ports: []int32{9443, 9444, 9445},
			}},
		},
		{
			name:                "other plugin backend remains console to 9443",
			policy:              newPluginBackendNetworkPolicy("logging", uiv1alpha1.TypeLogging, namespace),
			expectedName:        "logging-network-policy",
			expectedPodSelector: componentLabels("logging"),
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-console"}}},
				},
				ports: []int32{9443},
			}},
		},
		{
			name:                "korrel8r",
			policy:              newKorrel8rNetworkPolicy(namespace),
			expectedName:        "korrel8r-network-policy",
			expectedPodSelector: componentLabels(korrel8rName),
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-console"}}},
				},
				ports: []int32{9443},
			}},
		},
		{
			name:         "perses allows console and perses operator controller pods",
			policy:       newPersesNetworkPolicy(namespace),
			expectedName: "perses-network-policy",
			expectedPodSelector: map[string]string{
				"app.kubernetes.io/instance":   "perses",
				"app.kubernetes.io/managed-by": "perses-operator",
			},
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-console"}}},
					{podSelector: map[string]string{
						"app.kubernetes.io/name":      "perses-operator",
						"app.kubernetes.io/component": "controller",
						"app.kubernetes.io/part-of":   "observability-operator",
					}},
				},
				ports: []int32{8080},
			}},
		},
		{
			name:                "health analyzer",
			policy:              newHealthAnalyzerNetworkPolicy(namespace),
			expectedName:        "health-analyzer-network-policy",
			expectedPodSelector: componentLabels(healthAnalyzerWorkloadName),
			expectedIngress: []expectedNetworkPolicyIngressRule{{
				from: []expectedNetworkPolicyPeer{
					{namespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "kubernetes.io/metadata.name",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"openshift-monitoring", "openshift-user-workload-monitoring"},
						}},
					}},
				},
				ports: []int32{8443},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, networkingv1.SchemeGroupVersion.String(), tc.policy.APIVersion)
			assert.Equal(t, "NetworkPolicy", tc.policy.Kind)
			assert.Equal(t, tc.expectedName, tc.policy.Name)
			assert.Equal(t, namespace, tc.policy.Namespace)
			assert.DeepEqual(t, metav1.LabelSelector{MatchLabels: tc.expectedPodSelector}, tc.policy.Spec.PodSelector)
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
			peer := networkingv1.NetworkPolicyPeer{NamespaceSelector: expectedPeer.namespaceSelector}

			if expectedPeer.podSelector != nil {
				peer.PodSelector = &metav1.LabelSelector{MatchLabels: expectedPeer.podSelector}
			}
			assert.DeepEqual(t, peer, actualPeer)
		}

		assert.Equal(t, len(expectedRule.ports), len(ingressRule.Ports))
		for j, expectedPort := range expectedRule.ports {
			tcpProtocol := corev1.ProtocolTCP
			tcpPort := intstr.FromInt32(expectedPort)

			assert.DeepEqual(t, networkingv1.NetworkPolicyPort{Protocol: &tcpProtocol, Port: &tcpPort}, ingressRule.Ports[j])
		}
	}
}

func TestNetworkPolicyNamespaceSelectors(t *testing.T) {
	testCases := []struct {
		name              string
		policy            *networkingv1.NetworkPolicy
		allowedNamespaces map[string]bool
	}{
		{
			name:              "plugin backend",
			policy:            newPluginBackendNetworkPolicy("logging", uiv1alpha1.TypeLogging, "operator"),
			allowedNamespaces: map[string]bool{"openshift-console": true},
		},
		{
			name:              "monitoring plugin backend",
			policy:            newPluginBackendNetworkPolicy("monitoring", uiv1alpha1.TypeMonitoring, "operator"),
			allowedNamespaces: map[string]bool{"openshift-console": true},
		},
		{
			name:              "korrel8r",
			policy:            newKorrel8rNetworkPolicy("operator"),
			allowedNamespaces: map[string]bool{"openshift-console": true},
		},
		{
			name:              "perses",
			policy:            newPersesNetworkPolicy("operator"),
			allowedNamespaces: map[string]bool{"openshift-console": true},
		},
		{
			name:              "health analyzer",
			policy:            newHealthAnalyzerNetworkPolicy("operator"),
			allowedNamespaces: map[string]bool{"openshift-monitoring": true, "openshift-user-workload-monitoring": true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selector, err := metav1.LabelSelectorAsSelector(tc.policy.Spec.Ingress[0].From[0].NamespaceSelector)
			assert.NilError(t, err)
			for _, namespace := range []string{"openshift-console", "openshift-monitoring", "openshift-user-workload-monitoring", "unrelated"} {
				// The policy-group label must not affect whether a namespace matches.
				for _, group := range []string{"", "console", "monitoring"} {
					namespaceLabels := labels.Set{"kubernetes.io/metadata.name": namespace}
					if group != "" {
						namespaceLabels["network.openshift.io/policy-group"] = group
					}
					assert.Equal(t, tc.allowedNamespaces[namespace], selector.Matches(namespaceLabels), "namespace %s, policy group %q", namespace, group)
				}
			}
			assert.Equal(t, false, selector.Matches(labels.Set{}))
		})
	}
}

func TestPluginNetworkPolicyReconcilers_ReconcileExpectedPolicies(t *testing.T) {
	testCases := []struct {
		name                  string
		pluginName            string
		pluginType            uiv1alpha1.UIPluginType
		korrel8rEnabled       bool
		healthAnalyzerEnabled bool
		persesEnabled         bool
		expectedPolicyNames   []string
	}{
		{
			name:                "troubleshooting with korrel8r enabled",
			pluginName:          "troubleshooting-panel",
			pluginType:          uiv1alpha1.TypeTroubleshootingPanel,
			korrel8rEnabled:     true,
			expectedPolicyNames: []string{"korrel8r-network-policy", "troubleshooting-panel-network-policy"},
		},
		{
			name:                "troubleshooting with korrel8r disabled",
			pluginName:          "troubleshooting-panel",
			pluginType:          uiv1alpha1.TypeTroubleshootingPanel,
			korrel8rEnabled:     false,
			expectedPolicyNames: []string{"troubleshooting-panel-network-policy"},
		},
		{
			name:                  "monitoring with optional workloads enabled",
			pluginName:            "monitoring",
			pluginType:            uiv1alpha1.TypeMonitoring,
			healthAnalyzerEnabled: true,
			persesEnabled:         true,
			expectedPolicyNames:   []string{"health-analyzer-network-policy", "monitoring-network-policy", "perses-network-policy"},
		},
		{
			name:                "monitoring with optional workloads disabled",
			pluginName:          "monitoring",
			pluginType:          uiv1alpha1.TypeMonitoring,
			expectedPolicyNames: []string{"monitoring-network-policy"},
		},
		{
			name:                "logging backend policy",
			pluginName:          "logging",
			pluginType:          uiv1alpha1.TypeLogging,
			expectedPolicyNames: []string{"logging-network-policy"},
		},
		{
			name:                "distributed tracing backend policy",
			pluginName:          "distributed-tracing",
			pluginType:          uiv1alpha1.TypeDistributedTracing,
			expectedPolicyNames: []string{"distributed-tracing-network-policy"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := &uiv1alpha1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: tc.pluginName, UID: types.UID("plugin-uid")},
				Spec: uiv1alpha1.UIPluginSpec{
					Type: tc.pluginType,
				},
			}
			client := newNetworkPolicyRecordingClient()

			reconcilers := pluginNetworkPolicyReconcilers(plugin, UIPluginInfo{Name: tc.pluginName}, "openshift-operators", client, tc.korrel8rEnabled, tc.healthAnalyzerEnabled, tc.persesEnabled)
			for _, reconciler := range reconcilers {
				assert.NilError(t, reconciler.Reconcile(context.Background(), client, networkPolicyTestScheme(t)))
			}

			assert.DeepEqual(t, tc.expectedPolicyNames, client.policyNames())
		})
	}
}

type networkPolicyRecordingClient struct {
	client.Client
	policies map[client.ObjectKey]*networkingv1.NetworkPolicy
}

func newNetworkPolicyRecordingClient() *networkPolicyRecordingClient {
	return &networkPolicyRecordingClient{policies: map[client.ObjectKey]*networkingv1.NetworkPolicy{}}
}

func (c *networkPolicyRecordingClient) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	policy, ok := c.policies[key]
	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{Group: networkingv1.GroupName, Resource: "networkpolicies"}, key.Name)
	}

	target := obj.(*networkingv1.NetworkPolicy)
	*target = *policy.DeepCopy()
	return nil
}

func (c *networkPolicyRecordingClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	policy := obj.(*networkingv1.NetworkPolicy)
	key := client.ObjectKeyFromObject(policy)
	if _, ok := c.policies[key]; ok {
		return apierrors.NewAlreadyExists(schema.GroupResource{Group: networkingv1.GroupName, Resource: "networkpolicies"}, policy.Name)
	}

	policy.UID = types.UID("policy-" + policy.Name)
	policy.ResourceVersion = "1"
	c.policies[key] = policy.DeepCopy()
	return nil
}

func (c *networkPolicyRecordingClient) Apply(_ context.Context, _ runtime.ApplyConfiguration, _ ...client.ApplyOption) error {
	return nil
}

func (c *networkPolicyRecordingClient) policyNames() []string {
	names := make([]string, 0, len(c.policies))
	for _, policy := range c.policies {
		names = append(names, policy.Name)
	}

	sort.Strings(names)
	return names
}

func networkPolicyTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	assert.NilError(t, networkingv1.AddToScheme(scheme))
	assert.NilError(t, uiv1alpha1.AddToScheme(scheme))
	return scheme
}

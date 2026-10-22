package uiplugin

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

const (
	networkPolicyNameSuffix      = "-network-policy"
	healthAnalyzerWorkloadName   = "health-analyzer"
	persesOperatorManagedByLabel = "perses-operator"
	persesOperatorNameLabel      = "perses-operator"
	persesOperatorComponentLabel = "controller"
	persesOperatorPartOfLabel    = "observability-operator"

	namespacePolicyGroupLabel = "network.openshift.io/policy-group"
	consolePolicyGroup        = "console"
	monitoringPolicyGroup     = "monitoring"

	pluginBackendPort               int32 = 9443
	monitoringAlertmanagerProxyPort int32 = 9444
	monitoringThanosQueryProxyPort  int32 = 9445
	korrel8rPort                    int32 = 9443
	persesPort                      int32 = 8080
	healthAnalyzerPort              int32 = 8443
)

func pluginNetworkPolicyReconcilers(plugin *uiv1alpha1.UIPlugin, pluginInfo UIPluginInfo, namespace string, apiReader client.Reader, korrel8rEnabled, healthAnalyzerEnabled, persesEnabled bool) []reconciler.Reconciler {
	components := []reconciler.Reconciler{
		reconciler.NewNetworkPolicyUpdater(newPluginBackendNetworkPolicy(pluginInfo.Name, plugin.Spec.Type, namespace), plugin, apiReader),
	}

	if plugin.Spec.Type == uiv1alpha1.TypeTroubleshootingPanel {
		components = append(components, reconciler.NewOptionalNetworkPolicyUpdater(newKorrel8rNetworkPolicy(namespace), plugin, apiReader, korrel8rEnabled))
	}

	if plugin.Spec.Type == uiv1alpha1.TypeMonitoring {
		components = append(components,
			reconciler.NewOptionalNetworkPolicyUpdater(newHealthAnalyzerNetworkPolicy(namespace), plugin, apiReader, healthAnalyzerEnabled),
			reconciler.NewOptionalNetworkPolicyUpdater(newPersesNetworkPolicy(namespace), plugin, apiReader, persesEnabled),
		)
	}

	return components
}

func newPluginBackendNetworkPolicy(pluginName string, pluginType uiv1alpha1.UIPluginType, namespace string) *networkingv1.NetworkPolicy {
	ports := []int32{pluginBackendPort}
	if pluginType == uiv1alpha1.TypeMonitoring {
		ports = append(ports, monitoringAlertmanagerProxyPort, monitoringThanosQueryProxyPort)
	}

	return newPolicyGroupRestrictedIngressNetworkPolicy(
		pluginName+networkPolicyNameSuffix,
		namespace,
		componentLabels(pluginName),
		consolePolicyGroup,
		ports...,
	)
}

func newKorrel8rNetworkPolicy(namespace string) *networkingv1.NetworkPolicy {
	return newPolicyGroupRestrictedIngressNetworkPolicy(
		korrel8rName+networkPolicyNameSuffix,
		namespace,
		componentLabels(korrel8rName),
		consolePolicyGroup,
		korrel8rPort,
	)
}

func newHealthAnalyzerNetworkPolicy(namespace string) *networkingv1.NetworkPolicy {
	return newPolicyGroupRestrictedIngressNetworkPolicy(
		healthAnalyzerWorkloadName+networkPolicyNameSuffix,
		namespace,
		componentLabels(healthAnalyzerWorkloadName),
		monitoringPolicyGroup,
		healthAnalyzerPort,
	)
}

func newPersesNetworkPolicy(namespace string) *networkingv1.NetworkPolicy {
	return newRestrictedIngressNetworkPolicy(
		persesServiceName+networkPolicyNameSuffix,
		namespace,
		persesWorkloadSelector(),
		[]networkingv1.NetworkPolicyPeer{
			newNamespacePolicyGroupPeer(consolePolicyGroup),
			newPodSelectorPeer(persesOperatorPodSelector()),
		},
		[]int32{persesPort},
	)
}

func newPolicyGroupRestrictedIngressNetworkPolicy(name, namespace string, podSelector map[string]string, sourcePolicyGroup string, ports ...int32) *networkingv1.NetworkPolicy {
	return newRestrictedIngressNetworkPolicy(
		name,
		namespace,
		podSelector,
		[]networkingv1.NetworkPolicyPeer{newNamespacePolicyGroupPeer(sourcePolicyGroup)},
		ports,
	)
}

func newRestrictedIngressNetworkPolicy(name, namespace string, podSelector map[string]string, peers []networkingv1.NetworkPolicyPeer, ports []int32) *networkingv1.NetworkPolicy {
	networkPolicy := reconciler.NewAllowAllNetworkPolicy(name, namespace, podSelector)

	networkPolicy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{{
		From:  peers,
		Ports: newTCPNetworkPolicyPorts(ports...),
	}}

	return networkPolicy
}

func newNamespacePolicyGroupPeer(policyGroup string) networkingv1.NetworkPolicyPeer {
	return networkingv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{namespacePolicyGroupLabel: policyGroup},
		},
	}
}

func newPodSelectorPeer(labels map[string]string) networkingv1.NetworkPolicyPeer {
	return networkingv1.NetworkPolicyPeer{
		PodSelector: &metav1.LabelSelector{MatchLabels: labels},
	}
}

func newTCPNetworkPolicyPorts(ports ...int32) []networkingv1.NetworkPolicyPort {
	networkPolicyPorts := make([]networkingv1.NetworkPolicyPort, 0, len(ports))
	for _, port := range ports {
		tcpProtocol := corev1.ProtocolTCP
		tcpPort := intstr.FromInt32(port)
		networkPolicyPorts = append(networkPolicyPorts, networkingv1.NetworkPolicyPort{
			Protocol: &tcpProtocol,
			Port:     &tcpPort,
		})
	}

	return networkPolicyPorts
}

// persesWorkloadSelector matches generated Perses pods from rhobs/perses-operator.
// The operator always sets app.kubernetes.io/instance and app.kubernetes.io/managed-by
// in common.LabelsForPerses() and reuses that map for Deployment/StatefulSet pod selectors.
func persesWorkloadSelector() map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance":   persesServiceName,
		"app.kubernetes.io/managed-by": persesOperatorManagedByLabel,
	}
}

func persesOperatorPodSelector() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      persesOperatorNameLabel,
		"app.kubernetes.io/component": persesOperatorComponentLabel,
		"app.kubernetes.io/part-of":   persesOperatorPartOfLabel,
	}
}

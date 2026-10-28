package uiplugin

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
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

	consoleNamespace                = "openshift-console"
	userWorkloadMonitoringNamespace = "openshift-user-workload-monitoring"

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

	switch plugin.Spec.Type {
	case uiv1alpha1.TypeTroubleshootingPanel:
		components = append(components, reconciler.NewOptionalNetworkPolicyUpdater(newKorrel8rNetworkPolicy(namespace), plugin, apiReader, korrel8rEnabled))
	case uiv1alpha1.TypeMonitoring:
		components = append(components,
			reconciler.NewOptionalNetworkPolicyUpdater(newHealthAnalyzerNetworkPolicy(namespace), plugin, apiReader, healthAnalyzerEnabled),
			reconciler.NewOptionalNetworkPolicyUpdater(newPersesNetworkPolicy(namespace), plugin, apiReader, persesEnabled),
		)
	}

	return components
}

func newPluginBackendNetworkPolicy(pluginName string, pluginType uiv1alpha1.UIPluginType, namespace string) *networkingv1.NetworkPolicy {
	ports := []networkingv1.NetworkPolicyPort{{
		Protocol: ptr.To(corev1.ProtocolTCP),
		Port:     ptr.To(intstr.FromInt32(pluginBackendPort)),
	}}
	if pluginType == uiv1alpha1.TypeMonitoring {
		ports = append(ports,
			networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt32(monitoringAlertmanagerProxyPort)),
			},
			networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt32(monitoringThanosQueryProxyPort)),
			},
		)
	}

	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{APIVersion: networkingv1.SchemeGroupVersion.String(), Kind: "NetworkPolicy"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginName + networkPolicyNameSuffix,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: componentLabels(pluginName)},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{corev1.LabelMetadataName: consoleNamespace},
					},
				}},
				Ports: ports,
			}},
			// An empty egress rule allows all outbound traffic.
			Egress: []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
}

func newKorrel8rNetworkPolicy(namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{APIVersion: networkingv1.SchemeGroupVersion.String(), Kind: "NetworkPolicy"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      korrel8rName + networkPolicyNameSuffix,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: componentLabels(korrel8rName)},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{corev1.LabelMetadataName: consoleNamespace},
					},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt32(korrel8rPort)),
				}},
			}},
			Egress: []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
}

func newHealthAnalyzerNetworkPolicy(namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{APIVersion: networkingv1.SchemeGroupVersion.String(), Kind: "NetworkPolicy"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      healthAnalyzerWorkloadName + networkPolicyNameSuffix,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: componentLabels(healthAnalyzerWorkloadName)},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      corev1.LabelMetadataName,
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{reconciler.OpenshiftMonitoringNamespace, userWorkloadMonitoringNamespace},
						}},
					},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt32(healthAnalyzerPort)),
				}},
			}},
			Egress: []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
}

func newPersesNetworkPolicy(namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{APIVersion: networkingv1.SchemeGroupVersion.String(), Kind: "NetworkPolicy"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      persesServiceName + networkPolicyNameSuffix,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			// LabelsForPerses in rhobs/perses-operator sets these labels on Perses pods.
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/instance":   persesServiceName,
				"app.kubernetes.io/managed-by": persesOperatorManagedByLabel,
			}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{
					{NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{corev1.LabelMetadataName: consoleNamespace},
					}},
					// A pod selector without a namespace selector matches only this namespace.
					{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						"app.kubernetes.io/name":      persesOperatorNameLabel,
						"app.kubernetes.io/component": persesOperatorComponentLabel,
						"app.kubernetes.io/part-of":   persesOperatorPartOfLabel,
					}}},
				},
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt32(persesPort)),
				}},
			}},
			Egress: []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
}

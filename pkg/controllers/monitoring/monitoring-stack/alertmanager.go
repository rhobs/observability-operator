package monitoringstack

import (
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

func newAlertmanager(
	ms *stack.MonitoringStack,
	rbacResourceName string,
	alertmanagerCfg AlertmanagerConfiguration,
) *monv1.Alertmanager {
	resourceSelector := ms.Spec.ResourceSelector
	if resourceSelector == nil {
		resourceSelector = &metav1.LabelSelector{}
	}

	am := &monv1.Alertmanager{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "Alertmanager",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name,
			Namespace: ms.Namespace,
		},
		Spec: monv1.AlertmanagerSpec{
			PodMetadata: &monv1.EmbeddedObjectMetadata{
				Labels: podLabels("alertmanager", ms.Name),
			},
			Replicas:                   ms.Spec.AlertmanagerConfig.Replicas,
			ServiceAccountName:         rbacResourceName,
			AlertmanagerConfigSelector: resourceSelector,
			NodeSelector:               ms.Spec.NodeSelector,
			Tolerations:                ms.Spec.Tolerations,
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							TopologyKey: "kubernetes.io/hostname",
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: podLabels("alertmanager", ms.Name),
							},
						},
					},
					// We cannot expect all clusters to be multi-AZ, especially in the CI.
					// This is why we set zone-spread as preferred instead of required.
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 100,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: podLabels("alertmanager", ms.Name),
								},
								TopologyKey: "topology.kubernetes.io/zone",
							},
						},
					},
				},
			},
			SecurityContext: &corev1.PodSecurityContext{
				FSGroup:      ptr.To(AlertmanagerUserFSGroupID),
				RunAsNonRoot: ptr.To(true),
				RunAsUser:    ptr.To(AlertmanagerUserFSGroupID),
			},
			AlertmanagerConfigNamespaceSelector: ms.Spec.NamespaceSelector,
		},
	}
	if alertmanagerCfg.Image != "" {
		am.Spec.Image = ptr.To(alertmanagerCfg.Image)
	}
	if ms.Spec.AlertmanagerConfig.WebTLSConfig != nil {
		tlsConfig := ms.Spec.AlertmanagerConfig.WebTLSConfig
		am.Spec.Web = &monv1.AlertmanagerWebSpec{
			WebConfigFileFields: monv1.WebConfigFileFields{
				TLSConfig: &monv1.WebTLSConfig{
					KeySecret: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: tlsConfig.PrivateKey.Name,
						},
						Key: tlsConfig.PrivateKey.Key,
					},
					Cert: monv1.SecretOrConfigMap{
						Secret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: tlsConfig.Certificate.Name,
							},
							Key: tlsConfig.Certificate.Key,
						},
					},
				},
			},
		}
	}
	return am
}

func newAlertmanagerService(ms *stack.MonitoringStack) *corev1.Service {
	name := ms.Name + "-alertmanager"
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: podLabels("alertmanager", ms.Name),
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       9093,
					TargetPort: intstr.FromInt(9093),
				},
			},
		},
	}
}

func newAlertmanagerPDB(ms *stack.MonitoringStack) *policyv1.PodDisruptionBudget {
	name := ms.Name + "-alertmanager"
	selector := podLabels("alertmanager", ms.Name)

	return &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyv1.SchemeGroupVersion.String(),
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
		},
	}
}

func newAlertManagerClusterRole(rbacResourceName string, rbacVerbs []string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: rbacResourceName,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups:     []string{"security.openshift.io"},
			Resources:     []string{"securitycontextconstraints"},
			ResourceNames: []string{"nonroot", "nonroot-v2"},
			Verbs:         []string{"use"},
		}},
	}
}

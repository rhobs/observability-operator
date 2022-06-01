package monitoringstack

import (
	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	policyv1 "k8s.io/api/policy/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newAlertmanager(
	ms *stack.MonitoringStack,
	rbacResourceName string,
	instanceSelectorKey string,
	instanceSelectorValue string,
) *monv1.Alertmanager {
	resourceSelector := ms.Spec.ResourceSelector
	if resourceSelector == nil {
		resourceSelector = &metav1.LabelSelector{}
	}
	replicas := int32(2)

	return &monv1.Alertmanager{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "Alertmanager",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name,
			Namespace: ms.Namespace,
			Labels:    objectLabels(ms.Name, ms.Name, instanceSelectorKey, instanceSelectorValue),
		},
		Spec: monv1.AlertmanagerSpec{
			PodMetadata: &monv1.EmbeddedObjectMetadata{
				Labels: podLabels("alertmanager", ms.Name),
			},
			Replicas:                   &replicas,
			ServiceAccountName:         rbacResourceName,
			AlertmanagerConfigSelector: resourceSelector,
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
		},
	}
}

func newAlertmanagerService(ms *stack.MonitoringStack, instanceSelectorKey string, instanceSelectorValue string) *corev1.Service {
	name := ms.Name + "-alertmanager"
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
			Labels:    objectLabels(name, ms.Name, instanceSelectorKey, instanceSelectorValue),
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

func newAlertmanagerPDB(ms *stack.MonitoringStack, instanceSelectorKey string, instanceSelectorValue string) *policyv1.PodDisruptionBudget {
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
			Labels:    objectLabels(name, ms.Name, instanceSelectorKey, instanceSelectorValue),
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

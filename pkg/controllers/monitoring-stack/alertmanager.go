package monitoringstack

import (
	stack "github.com/rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"

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
	replicas := int32(3)

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
			Replicas:                            &replicas,
			ServiceAccountName:                  rbacResourceName,
			AlertmanagerConfigSelector:          resourceSelector,
			AlertmanagerConfigNamespaceSelector: nil,
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

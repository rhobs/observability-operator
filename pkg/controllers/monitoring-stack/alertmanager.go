package monitoringstack

import (
	stack "rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
			Labels:    commonLabels(ms.Name, instanceSelectorKey, instanceSelectorValue),
		},
		Spec: monv1.AlertmanagerSpec{
			PodMetadata: &monv1.EmbeddedObjectMetadata{
				Labels: commonLabels(ms.Name, instanceSelectorKey, instanceSelectorValue),
			},
			Replicas:                            &replicas,
			ServiceAccountName:                  rbacResourceName,
			AlertmanagerConfigSelector:          resourceSelector,
			AlertmanagerConfigNamespaceSelector: nil,
		},
	}
}

func newAlertmanagerService(ms *stack.MonitoringStack) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name + "-alertmanager",
			Namespace: ms.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/part-of": ms.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/part-of": ms.Name,
			},
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

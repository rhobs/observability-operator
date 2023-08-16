package thanos_querier

import (
	"fmt"

	"github.com/rhobs/observability-operator/pkg/reconciler"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	msoapi "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	po "github.com/rhobs/obo-prometheus-operator/pkg/operator"
)

func thanosComponentReconcilers(thanos *msoapi.ThanosQuerier, sidecarUrls []string) []reconciler.Reconciler {
	name := "thanos-querier-" + thanos.Name
	return []reconciler.Reconciler{
		reconciler.NewUpdater(newServiceAccount(name, thanos.Namespace), thanos),
		reconciler.NewUpdater(newThanosQuerierDeployment(name, thanos, sidecarUrls), thanos),
		reconciler.NewUpdater(newService(name, thanos.Namespace), thanos),
		reconciler.NewUpdater(newServiceMonitor(name, thanos.Namespace), thanos),
	}
}

func newThanosQuerierDeployment(name string, spec *msoapi.ThanosQuerier, sidecarUrls []string) *appsv1.Deployment {
	args := []string{
		"query",
		"--grpc-address=127.0.0.1:10901",
		"--http-address=127.0.0.1:9090",
		"--log.format=logfmt",
		"--query.replica-label=prometheus_replica",
		"--query.auto-downsampling",
	}
	for _, endpoint := range sidecarUrls {
		args = append(args, fmt.Sprintf("--endpoint=%s", endpoint))
	}

	for _, rl := range spec.Spec.ReplicaLabels {
		args = append(args, fmt.Sprintf("--query.replica-label=%s", rl))
	}

	thanos := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: spec.Namespace,
			Labels:    componentLabels(name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: spec.Namespace,
					Labels:    componentLabels(name),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "thanos-querier",
							Args:  args,
							Image: po.DefaultThanosImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9090,
									Name:          "metrics",
								},
							},
							TerminationMessagePolicy: "FallbackToLogsOnError",
						},
					},
					NodeSelector: map[string]string{
						"kubernetes.io/os": "linux",
					},
				},
			},
			ProgressDeadlineSeconds: func(i int32) *int32 { return &i }(300),
		},
	}

	return thanos
}

func newServiceAccount(name string, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newService(name string, namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 9090,
					Name: "http",
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/instance": name,
			},
			Type: "ClusterIP",
		},
	}
}

func newServiceMonitor(name string, namespace string) *monv1.ServiceMonitor {
	return &monv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    componentLabels(name),
		},
		Spec: monv1.ServiceMonitorSpec{
			Endpoints: []monv1.Endpoint{
				{
					Port:   "http",
					Scheme: "http",
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": name,
				},
			},
		},
	}
}

func componentLabels(querierName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance":   querierName,
		"app.kubernetes.io/part-of":    "ThanosQuerier",
		"app.kubernetes.io/managed-by": "observability-operator",
	}
}

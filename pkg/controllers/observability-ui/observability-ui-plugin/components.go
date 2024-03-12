package observability_ui_plugin

import (
	"fmt"

	"github.com/rhobs/observability-operator/pkg/reconciler"

	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	obsui "github.com/rhobs/observability-operator/pkg/apis/observability-ui/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func pluginComponentReconcilers(plugin *obsui.ObservabilityUIPlugin, pluginInfo ObservabilityUIPluginInfo) []reconciler.Reconciler {
	hasClusterRole := pluginInfo.ClusterRole != nil
	hasClusterRoleBinding := pluginInfo.ClusterRoleBinding != nil
	namespace := plugin.Namespace

	return []reconciler.Reconciler{
		reconciler.NewUpdater(newServiceAccount(pluginInfo, namespace), plugin),
		reconciler.NewOptionalUpdater(newClusterRole(pluginInfo), plugin, hasClusterRole),
		reconciler.NewOptionalUpdater(newClusterRoleBinding(pluginInfo), plugin, hasClusterRoleBinding),
		reconciler.NewUpdater(newDeployment(pluginInfo, namespace), plugin),
		reconciler.NewUpdater(newService(pluginInfo, namespace), plugin),
		reconciler.NewUpdater(newConsolePlugin(pluginInfo, namespace), plugin),
	}
}

func newServiceAccount(info ObservabilityUIPluginInfo, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      info.Name + "-sa",
			Namespace: namespace,
		},
	}
}

func newClusterRole(info ObservabilityUIPluginInfo) *rbacv1.ClusterRole {
	return info.ClusterRole
}

func newClusterRoleBinding(info ObservabilityUIPluginInfo) *rbacv1.ClusterRoleBinding {
	return info.ClusterRoleBinding
}

func newConsolePlugin(info ObservabilityUIPluginInfo, namespace string) *osv1alpha1.ConsolePlugin {
	return &osv1alpha1.ConsolePlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: osv1alpha1.SchemeGroupVersion.String(),
			Kind:       "ConsolePlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: info.ConsoleName,
		},
		Spec: osv1alpha1.ConsolePluginSpec{
			DisplayName: info.DisplayName,
			Service: osv1alpha1.ConsolePluginService{
				Name:      info.Name,
				Namespace: namespace,
				Port:      9443,
				BasePath:  "/",
			},
			Proxy: info.Proxies,
		},
	}
}

func newDeployment(info ObservabilityUIPluginInfo, namespace string) *appsv1.Deployment {
	plugin := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      info.Name,
			Namespace: namespace,
			Labels:    componentLabels(info.Name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": info.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      info.Name,
					Namespace: namespace,
					Labels:    componentLabels(info.Name),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: info.Name + "-sa",
					Containers: []corev1.Container{
						{
							Name:  info.Name,
							Image: info.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9443,
									Name:          "web",
								},
							},
							TerminationMessagePolicy: "FallbackToLogsOnError",
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot:             &[]bool{true}[0],
								AllowPrivilegeEscalation: &[]bool{false}[0],
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "serving-cert",
									ReadOnly:  true,
									MountPath: "/var/serving-cert",
								},
							},
							Args: []string{
								fmt.Sprintf("-port=%d", 9443),
								"-cert=/var/serving-cert/tls.crt",
								"-key=/var/serving-cert/tls.key",
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "serving-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  info.Name,
									DefaultMode: &[]int32{420}[0],
								},
							},
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

	return plugin
}

func newService(info ObservabilityUIPluginInfo, namespace string) *corev1.Service {
	annotations := map[string]string{
		"service.alpha.openshift.io/serving-cert-secret-name": info.Name,
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        info.Name,
			Namespace:   namespace,
			Labels:      componentLabels(info.Name),
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       9443,
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(9443),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/instance": info.Name,
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func componentLabels(pluginName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance":   pluginName,
		"app.kubernetes.io/part-of":    "ObservabilityUIPlugin",
		"app.kubernetes.io/managed-by": "observability-operator",
	}
}

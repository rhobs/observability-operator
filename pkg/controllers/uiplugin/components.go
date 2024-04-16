package uiplugin

import (
	"fmt"

	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

const (
	port                  = 9443
	serviceAccountSuffix  = "-sa"
	servingCertVolumeName = "serving-cert"
)

func pluginComponentReconcilers(plugin *uiv1alpha1.UIPlugin, pluginInfo UIPluginInfo) []reconciler.Reconciler {
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

func newServiceAccount(info UIPluginInfo, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      info.Name + serviceAccountSuffix,
			Namespace: namespace,
		},
	}
}

func newClusterRole(info UIPluginInfo) *rbacv1.ClusterRole {
	return info.ClusterRole
}

func newClusterRoleBinding(info UIPluginInfo) *rbacv1.ClusterRoleBinding {
	return info.ClusterRoleBinding
}

func newConsolePlugin(info UIPluginInfo, namespace string) *osv1alpha1.ConsolePlugin {
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
				Port:      port,
				BasePath:  "/",
			},
			Proxy: info.Proxies,
		},
	}
}

func newDeployment(info UIPluginInfo, namespace string) *appsv1.Deployment {
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
			Replicas: ptr.To(int32(1)),
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
					ServiceAccountName: info.Name + serviceAccountSuffix,
					Containers: []corev1.Container{
						{
							Name:  info.Name,
							Image: info.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: port,
									Name:          "web",
								},
							},
							TerminationMessagePolicy: "FallbackToLogsOnError",
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot:             ptr.To(bool(true)),
								AllowPrivilegeEscalation: ptr.To(bool(false)),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      servingCertVolumeName,
									ReadOnly:  true,
									MountPath: "/var/serving-cert",
								},
							},
							Args: []string{
								fmt.Sprintf("-port=%d", port),
								"-cert=/var/serving-cert/tls.crt",
								"-key=/var/serving-cert/tls.key",
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: servingCertVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  info.Name,
									DefaultMode: ptr.To(int32(420)),
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"kubernetes.io/os": "linux",
					},
				},
			},
			ProgressDeadlineSeconds: ptr.To(int32(300)),
		},
	}

	return plugin
}

func newService(info UIPluginInfo, namespace string) *corev1.Service {
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
					Port:       port,
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(port),
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
		"app.kubernetes.io/part-of":    "UIPlugin",
		"app.kubernetes.io/managed-by": "observability-operator",
	}
}

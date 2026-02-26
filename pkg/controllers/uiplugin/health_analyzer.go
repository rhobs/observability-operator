package uiplugin

import (
	_ "embed"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	name                      = "health-analyzer"
	volumeMountName           = name + "-tls"
	componentConfigVolumeName = "components-health-config"
	componentConfigMapName    = "components-config"
)

//go:embed config/health-analyzer.yaml
var componentHealthConfig string

func newHealthAnalyzerPrometheusRole(namespace string) *rbacv1.Role {
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-k8s",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	return role
}

func newHealthAnalyzerPrometheusRoleBinding(namespace string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-k8s",
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     "prometheus-k8s",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "prometheus-k8s",
				Namespace: "openshift-monitoring",
			},
		},
	}
	return roleBinding
}

func newHealthAnalyzerService(namespace string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": volumeMountName,
			},
			Labels: componentLabels(name),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "metrics",
					Port:       8443,
					TargetPort: intstr.FromString("metrics"),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/instance": name,
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	return service
}

func newHealthAnalyzerDeployment(namespace string,
	serviceAccountName string,
	image string) *appsv1.Deployment {

	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    componentLabels(name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: componentLabels(name),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:           serviceAccountName,
					AutomountServiceAccountToken: ptr.To(true),
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           image,
							ImagePullPolicy: corev1.PullAlways,
							Args: []string{
								"serve",
								"--tls-cert-file=/etc/tls/private/tls.crt",
								"--tls-private-key-file=/etc/tls/private/tls.key",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PROM_URL",
									Value: "https://thanos-querier.openshift-monitoring.svc.cluster.local:9091/",
								},
								{
									Name:  "ALERTMANAGER_URL",
									Value: "https://alertmanager-main.openshift-monitoring.svc.cluster.local:9094",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot:             ptr.To(true),
								AllowPrivilegeEscalation: ptr.To(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
									Name:          "metrics",
								},
							},
							TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/etc/tls/private",
									Name:      volumeMountName,
									ReadOnly:  true,
								},
								{
									Name:      componentConfigVolumeName,
									MountPath: "/etc/config",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: volumeMountName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: volumeMountName,
								},
							},
						},
						{
							Name: componentConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: componentConfigMapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return deploy
}

func newHealthAnalyzerServiceMonitor(namespace string) *monv1.ServiceMonitor {
	serviceMonitor := &monv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: monv1.ServiceMonitorSpec{
			Endpoints: []monv1.Endpoint{
				{
					Interval: "30s",
					Port:     "metrics",
					Scheme:   ptr.To(monv1.Scheme("https")),
					HTTPConfigWithProxyAndTLSFiles: monv1.HTTPConfigWithProxyAndTLSFiles{
						HTTPConfigWithTLSFiles: monv1.HTTPConfigWithTLSFiles{
							TLSConfig: &monv1.TLSConfig{
								TLSFilesConfig: monv1.TLSFilesConfig{
									CAFile:   "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
									CertFile: "/etc/prometheus/secrets/metrics-client-certs/tls.crt",
									KeyFile:  "/etc/prometheus/secrets/metrics-client-certs/tls.key",
								},
								SafeTLSConfig: monv1.SafeTLSConfig{
									ServerName: ptr.To(name + "." + namespace + ".svc"),
								},
							},
						},
					},
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": name,
				},
			},
		},
	}

	return serviceMonitor
}

// newComponentHealthConfig creates a new ConfigMap
// that defines the components whose health is evaluated.
func newComponentHealthConfig(namespace string) *v1.ConfigMap {
	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      componentConfigMapName,
			Labels:    componentLabels("monitoring"),
		},
		Data: map[string]string{
			"components.yaml": componentHealthConfig,
		},
	}

	return &cm
}

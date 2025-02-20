package thanos_querier

import (
	"fmt"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	msoapi "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

func thanosComponentReconcilers(
	thanos *msoapi.ThanosQuerier,
	sidecarUrls []string,
	thanosCfg ThanosConfiguration,
	tlsHashes map[string]string,
) []reconciler.Reconciler {
	name := "thanos-querier-" + thanos.Name
	return []reconciler.Reconciler{
		reconciler.NewUpdater(newServiceAccount(name, thanos.Namespace), thanos),
		reconciler.NewUpdater(newThanosQuerierDeployment(name, thanos, sidecarUrls, thanosCfg, tlsHashes), thanos),
		reconciler.NewUpdater(newService(name, thanos.Namespace), thanos),
		reconciler.NewUpdater(newServiceMonitor(name, thanos.Namespace, thanos), thanos),
		reconciler.NewOptionalUpdater(newHttpConfConfigMap(name, thanos), thanos, thanos.Spec.WebTLSConfig != nil),
	}
}

func newHttpConfConfigMap(name string, thanos *msoapi.ThanosQuerier) *corev1.ConfigMap {
	httpConf := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-http-conf", name),
			Namespace: thanos.Namespace,
		},
	}
	if thanos.Spec.WebTLSConfig != nil {
		httpConf.Data = map[string]string{
			"http.conf": `
tls_server_config:
  cert_file: /etc/thanos/tls-assets/web-cert-secret/` + thanos.Spec.WebTLSConfig.Certificate.Key + `
  key_file: /etc/thanos/tls-assets/web-key-secret/` + thanos.Spec.WebTLSConfig.PrivateKey.Key,
		}
	}

	return httpConf
}

func newThanosQuerierDeployment(
	name string,
	spec *msoapi.ThanosQuerier,
	sidecarUrls []string,
	thanosCfg ThanosConfiguration,
	tlsHashes map[string]string,
) *appsv1.Deployment {
	httpConfCMName := fmt.Sprintf("%s-http-conf", name)

	args := []string{
		"query",
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

	if spec.Spec.WebTLSConfig != nil {
		args = append(args, "--http.config=/etc/thanos/tls-assets/web-http-conf-cm/http.conf")
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
			Replicas: ptr.To(int32(1)),
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
							Image: thanosCfg.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 10902,
									Name:          "metrics",
								},
							},
							TerminationMessagePolicy: "FallbackToLogsOnError",
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
								RunAsNonRoot: ptr.To(true),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"kubernetes.io/os": "linux",
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			},
			ProgressDeadlineSeconds: ptr.To(int32(300)),
		},
	}
	if spec.Spec.WebTLSConfig != nil {
		thanos.Spec.Template.Spec.Volumes = append(thanos.Spec.Template.Spec.Volumes, []corev1.Volume{
			{
				Name: "thanos-web-tls-key",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: spec.Spec.WebTLSConfig.PrivateKey.Name,
					},
				},
			},
			{
				Name: "thanos-web-tls-cert",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: spec.Spec.WebTLSConfig.Certificate.Name,
					},
				},
			},
			{
				Name: "thanos-web-http-conf",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: httpConfCMName,
						},
					},
				},
			},
		}...)
		thanos.Spec.Template.Spec.Containers[0].VolumeMounts = append(thanos.Spec.Template.Spec.Containers[0].VolumeMounts, []corev1.VolumeMount{
			{
				Name:      "thanos-web-tls-key",
				MountPath: "/etc/thanos/tls-assets/web-cert-secret",
				ReadOnly:  true,
			},
			{
				Name:      "thanos-web-tls-cert",
				MountPath: "/etc/thanos/tls-assets/web-key-secret",
				ReadOnly:  true,
			},
			{
				Name:      "thanos-web-http-conf",
				MountPath: "/etc/thanos/tls-assets/web-http-conf-cm",
				ReadOnly:  true,
			},
		}...)
		tlsAnnotations := map[string]string{}
		for name, hash := range tlsHashes {
			tlsAnnotations[fmt.Sprintf("monitoring.openshift.io/%s-hash", name)] = hash
		}
		thanos.ObjectMeta.Annotations = tlsAnnotations
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
					Port: 10902,
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

func newServiceMonitor(name string, namespace string, thanos *msoapi.ThanosQuerier) *monv1.ServiceMonitor {
	serviceMonitor := &monv1.ServiceMonitor{
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
	if thanos.Spec.WebTLSConfig != nil {
		serviceMonitor.Spec.Endpoints[0].Scheme = "https"
		serviceMonitor.Spec.Endpoints[0].TLSConfig = &monv1.TLSConfig{
			SafeTLSConfig: monv1.SafeTLSConfig{
				CA: monv1.SecretOrConfigMap{
					Secret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: thanos.Spec.WebTLSConfig.CertificateAuthority.Name,
						},
						Key: thanos.Spec.WebTLSConfig.CertificateAuthority.Key,
					},
				},
				ServerName: ptr.To(name),
			},
		}
	}
	return serviceMonitor
}

func componentLabels(querierName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance":   querierName,
		"app.kubernetes.io/part-of":    "ThanosQuerier",
		"app.kubernetes.io/managed-by": "observability-operator",
	}
}

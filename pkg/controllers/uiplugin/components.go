package uiplugin

import (
	"embed"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"text/template"

	osv1 "github.com/openshift/api/console/v1"
	osRhobsv1 "github.com/rhobs/openshift-api/console/v1"
	osv1alpha1 "github.com/rhobs/openshift-api/console/v1alpha1"
	"golang.org/x/mod/semver"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

const (
	port                   = 9443
	serviceAccountSuffix   = "-sa"
	servingCertVolumeName  = "serving-cert"
	DefaultLokiStackName   = "logging-loki"
	OpenshiftLoggingNs     = "openshift-logging"
	OpenshiftNetobservNs   = "netobserv"
	OpenshiftTracingNs     = "openshift-tracing"
	monitorClusterroleName = "cluster-monitoring-view"

	annotationPrefix = "observability.openshift.io/ui-plugin-"
)

var (
	defaultNodeSelector = map[string]string{
		"kubernetes.io/os": "linux",
	}

	hashSeparator = []byte("\n")

	//go:embed config/korrel8r.yaml
	korrel8rConfigYAMLTmplFile embed.FS

	korrel8rConfigTmpl = template.Must(template.ParseFS(korrel8rConfigYAMLTmplFile, "config/korrel8r.yaml"))
)

func IsVersionAheadOrEqual(currentVersion, version string) bool {
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}
	if version == "" {
		return false
	}

	canonicalMinVersion := fmt.Sprintf("%s-0", semver.Canonical(version))

	return semver.Compare(currentVersion, canonicalMinVersion) >= 0
}

func newClusterRoleBinding(namespace string, serviceAccountName string, roleName string, name string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  corev1.SchemeGroupVersion.Group,
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     roleName,
		},
	}
}

func newServiceAccount(name string, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + serviceAccountSuffix,
			Namespace: namespace,
		},
	}
}

func newRole(info UIPluginInfo) *rbacv1.Role {
	return info.Role
}

func newRoleBinding(info UIPluginInfo) *rbacv1.RoleBinding {
	return info.RoleBinding
}

func newLegacyConsolePlugin(info UIPluginInfo, namespace string) *osv1alpha1.ConsolePlugin {
	proxies := make([]osv1alpha1.ConsolePluginProxy, len(info.Proxies))
	for i, p := range info.Proxies {
		proxies[i] = p.ToV1Alpha1()
	}

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
			Proxy: proxies,
		},
	}
}

func newRhobsConsolePlugin(info UIPluginInfo, namespace string) *osRhobsv1.ConsolePlugin {
	proxies := make([]osRhobsv1.ConsolePluginProxy, len(info.Proxies))
	for i, p := range info.Proxies {
		proxies[i] = p.ToRhobsV1()
	}

	return &osRhobsv1.ConsolePlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: osRhobsv1.SchemeGroupVersion.String(),
			Kind:       "ConsolePlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: info.ConsoleName,
		},
		Spec: osRhobsv1.ConsolePluginSpec{
			DisplayName: info.DisplayName,
			Backend: osRhobsv1.ConsolePluginBackend{
				Type: osRhobsv1.Service,
				Service: &osRhobsv1.ConsolePluginService{
					Name:      info.Name,
					Namespace: namespace,
					Port:      port,
					BasePath:  "/",
				},
			},
			Proxy: proxies,
			I18n:  osRhobsv1.ConsolePluginI18n{LoadType: osRhobsv1.Preload},
		},
	}
}

func newConsolePlugin(info UIPluginInfo, namespace string) *osv1.ConsolePlugin {
	proxies := make([]osv1.ConsolePluginProxy, len(info.Proxies))
	for i, p := range info.Proxies {
		proxies[i] = p.ToUpstreamV1()
	}

	return &osv1.ConsolePlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: osv1.SchemeGroupVersion.String(),
			Kind:       "ConsolePlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: info.ConsoleName,
		},
		Spec: osv1.ConsolePluginSpec{
			DisplayName: info.DisplayName,
			Backend: osv1.ConsolePluginBackend{
				Type: osv1.Service,
				Service: &osv1.ConsolePluginService{
					Name:      info.Name,
					Namespace: namespace,
					Port:      port,
					BasePath:  "/",
				},
			},
			Proxy:                 proxies,
			I18n:                  osv1.ConsolePluginI18n{LoadType: osv1.Preload},
			ContentSecurityPolicy: []osv1.ConsolePluginCSP{},
		},
	}
}

func newDeployment(info UIPluginInfo, namespace string, config *uiv1alpha1.DeploymentConfig) *appsv1.Deployment {
	pluginArgs := []string{
		fmt.Sprintf("-port=%d", port),
		"-cert=/var/serving-cert/tls.crt",
		"-key=/var/serving-cert/tls.key",
	}

	if len(info.ExtraArgs) > 0 {
		pluginArgs = append(pluginArgs, info.ExtraArgs...)
	}

	if info.TLSMinVersion != "" {
		pluginArgs = append(pluginArgs, fmt.Sprintf("-tls-min-version=%s", info.TLSMinVersion))
	}
	if len(info.TLSCiphers) > 0 {
		pluginArgs = append(pluginArgs, fmt.Sprintf("-tls-cipher-suites=%s", strings.Join(info.TLSCiphers, ",")))
	}

	volumes := []corev1.Volume{
		{
			Name: servingCertVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  info.Name,
					DefaultMode: ptr.To(int32(420)),
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      servingCertVolumeName,
			ReadOnly:  true,
			MountPath: "/var/serving-cert",
		},
	}

	podAnnotations := map[string]string{}
	if info.ConfigMap != nil {
		podAnnotations[annotationPrefix+"config-hash"] = computeConfigMapHash(info.ConfigMap)
		volumes = append(volumes, corev1.Volume{
			Name: "plugin-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: info.Name,
					},
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "plugin-config",
			ReadOnly:  true,
			MountPath: "/etc/plugin/config",
		})
	}

	nodeSelector, tolerations := createNodeSelectorAndTolerations(config)

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
				MatchLabels: componentLabels(info.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        info.Name,
					Namespace:   namespace,
					Labels:      componentLabels(info.Name),
					Annotations: podAnnotations,
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
							VolumeMounts: volumeMounts,
							Args:         pluginArgs,
						},
					},
					Volumes:       volumes,
					NodeSelector:  nodeSelector,
					Tolerations:   tolerations,
					RestartPolicy: "Always",
					DNSPolicy:     "ClusterFirst",
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

	return plugin
}

func computeConfigMapHash(cm *corev1.ConfigMap) string {
	keys := make([]string, 0, len(cm.Data))
	for k := range cm.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := fnv.New32a()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write(hashSeparator)
		h.Write([]byte(cm.Data[k]))
		h.Write(hashSeparator)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func createNodeSelectorAndTolerations(config *uiv1alpha1.DeploymentConfig) (map[string]string, []corev1.Toleration) {
	if config == nil {
		return defaultNodeSelector, nil
	}

	nodeSelector := config.NodeSelector
	if nodeSelector == nil {
		nodeSelector = defaultNodeSelector
	}

	return nodeSelector, config.Tolerations
}

func newService(info UIPluginInfo, namespace string) *corev1.Service {
	if info.ConsoleName == "monitoring-console-plugin" {
		return newMonitoringService(info.Name, namespace)
	}

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
			Selector: componentLabels(info.Name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

// componentsHealthClusterRole creates a new clusterrole with the provided name.
// The clusterrole has read permissions to the cluster resources and it is required
// for the component health evaluation.
func componentsHealthClusterRole(name string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{"config.openshift.io"},
				Resources: []string{"clusteroperators"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{"machineconfiguration.openshift.io"},
				Resources: []string{"machineconfigpools"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{"kubevirt.io"},
				Resources: []string{"kubevirts"},
				Verbs:     []string{"get", "list"},
			},
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

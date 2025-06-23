package uiplugin

import (
	"bytes"
	"embed"
	"fmt"
	"hash/fnv"
	"io"
	"sort"
	"strings"
	"text/template"

	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	"golang.org/x/mod/semver"
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
	port                   = 9443
	serviceAccountSuffix   = "-sa"
	servingCertVolumeName  = "serving-cert"
	korrel8rName           = "korrel8r"
	Korrel8rConfigFileName = "korrel8r.yaml"
	Korrel8rConfigMountDir = "/config/"
	OpenshiftLoggingNs     = "openshift-logging"
	OpenshiftNetobservNs   = "netobserv"
	OpenshiftTracingNs     = "openshift-tracing"

	annotationPrefix = "observability.openshift.io/ui-plugin-"
)

var (
	defaultNodeSelector = map[string]string{
		"kubernetes.io/os": "linux",
	}

	hashSeparator = []byte("\n")

	//go:embed config/korrel8r.yaml
	korrel8rConfigYAMLTmplFile embed.FS
)

func isVersionAheadOrEqual(currentVersion, version string) bool {
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}
	if version == "" {
		return false
	}

	canonicalMinVersion := fmt.Sprintf("%s-0", semver.Canonical(version))

	return semver.Compare(currentVersion, canonicalMinVersion) >= 0
}

func pluginComponentReconcilers(plugin *uiv1alpha1.UIPlugin, pluginInfo UIPluginInfo, clusterVersion string) []reconciler.Reconciler {
	namespace := pluginInfo.ResourceNamespace

	components := []reconciler.Reconciler{
		reconciler.NewUpdater(newServiceAccount(pluginInfo, namespace), plugin),
		reconciler.NewUpdater(newDeployment(pluginInfo, namespace, plugin.Spec.Deployment), plugin),
		reconciler.NewUpdater(newService(pluginInfo, namespace), plugin),
	}

	if isVersionAheadOrEqual(clusterVersion, "v4.17") {
		components = append(components, reconciler.NewUpdater(newConsolePlugin(pluginInfo, namespace), plugin))
	} else {
		components = append(components, reconciler.NewUpdater(newLegacyConsolePlugin(pluginInfo, namespace), plugin))
	}

	if pluginInfo.Role != nil {
		components = append(components, reconciler.NewUpdater(newRole(pluginInfo), plugin))
	}

	if pluginInfo.RoleBinding != nil {
		components = append(components, reconciler.NewUpdater(newRoleBinding(pluginInfo), plugin))
	}

	if pluginInfo.ConfigMap != nil {
		components = append(components, reconciler.NewUpdater(pluginInfo.ConfigMap, plugin))
	}

	for _, role := range pluginInfo.ClusterRoles {
		if role != nil {
			components = append(components, reconciler.NewUpdater(role, plugin))
		}
	}

	for _, roleBinding := range pluginInfo.ClusterRoleBindings {
		if roleBinding != nil {
			components = append(components, reconciler.NewUpdater(roleBinding, plugin))
		}
	}

	if pluginInfo.Korrel8rImage != "" {
		components = append(components, reconciler.NewUpdater(newKorrel8rService(korrel8rName, namespace), plugin))
		korrel8rCm, err := newKorrel8rConfigMap(korrel8rName, namespace, pluginInfo)
		if err == nil && korrel8rCm != nil {
			components = append(components, reconciler.NewUpdater(korrel8rCm, plugin))
			components = append(components, reconciler.NewUpdater(newKorrel8rDeployment(korrel8rName, namespace, pluginInfo), plugin))
		}
	}

	if pluginInfo.HealthAnalyzerImage != "" {
		serviceAccountName := plugin.Name + serviceAccountSuffix
		components = append(components, reconciler.NewUpdater(newClusterRoleBinding(namespace, serviceAccountName, "cluster-monitoring-view", "cluster-monitoring-view"), plugin))
		components = append(components, reconciler.NewUpdater(newClusterRoleBinding(namespace, serviceAccountName, "system:auth-delegator", serviceAccountName+":system:auth-delegator"), plugin))
		components = append(components, reconciler.NewUpdater(newHealthAnalyzerPrometheusRole(namespace), plugin))
		components = append(components, reconciler.NewUpdater(newHealthAnalyzerPrometheusRoleBinding(namespace), plugin))
		components = append(components, reconciler.NewUpdater(newHealthAnalyzerService(namespace), plugin))
		components = append(components, reconciler.NewUpdater(newHealthAnalyzerDeployment(namespace, serviceAccountName, pluginInfo), plugin))
		components = append(components, reconciler.NewUpdater(newHealthAnalyzerServiceMonitor(namespace), plugin))
	}

	if pluginInfo.PersesImage != "" {
		components = append(components, reconciler.NewUpdater(newPerses(namespace, pluginInfo.PersesImage), plugin))
		components = append(components, reconciler.NewUpdater(newAcceleratorsDatasource(namespace), plugin))
		components = append(components, reconciler.NewUpdater(newAcceleratorsDashboard(namespace), plugin))
	}

	return components
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

func newRole(info UIPluginInfo) *rbacv1.Role {
	return info.Role
}

func newRoleBinding(info UIPluginInfo) *rbacv1.RoleBinding {
	return info.RoleBinding
}

func newLegacyConsolePlugin(info UIPluginInfo, namespace string) *osv1alpha1.ConsolePlugin {
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
			Proxy: info.LegacyProxies,
		},
	}
}

func newConsolePlugin(info UIPluginInfo, namespace string) *osv1.ConsolePlugin {
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
			Proxy: info.Proxies,
			I18n:  osv1.ConsolePluginI18n{LoadType: osv1.Preload},
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

func newKorrel8rDeployment(name string, namespace string, info UIPluginInfo) *appsv1.Deployment {
	volumes := []corev1.Volume{
		{
			Name: servingCertVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  name,
					DefaultMode: ptr.To(int32(420)),
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      servingCertVolumeName,
			ReadOnly:  true,
			MountPath: "/secrets/",
		},
	}

	volumes = append(volumes, corev1.Volume{
		Name: "korrel8r-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "korrel8r-config",
		ReadOnly:  true,
		MountPath: Korrel8rConfigMountDir,
	})

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
			Selector: &metav1.LabelSelector{
				MatchLabels: componentLabels(name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels:    componentLabels(name),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: info.Name + serviceAccountSuffix,
					Containers: []corev1.Container{
						{
							Name:    name,
							Image:   info.Korrel8rImage,
							Command: []string{"korrel8r", "web", fmt.Sprintf("--https=:%d", port), "--cert=/secrets/tls.crt", "--key=/secrets/tls.key", "--config=/config/korrel8r.yaml"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot:             ptr.To(true),
								AllowPrivilegeEscalation: ptr.To(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
	return deploy
}

func newKorrel8rService(name string, namespace string) *corev1.Service {
	annotations := map[string]string{
		"service.beta.openshift.io/serving-cert-secret-name": name,
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      componentLabels(name),
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					Name:       "web",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(port),
				},
			},
			Selector: componentLabels(name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func newKorrel8rConfigMap(name string, namespace string, info UIPluginInfo) (*corev1.ConfigMap, error) {

	korrel8rData := map[string]string{
		"Metric":      "thanos-querier",
		"MetricAlert": "alertmanager-main",
		"Log":         "logging-loki-gateway-http",
		"Netflow":     "loki-gateway-http", "Trace": "tempo-platform-gateway",
		"MonitoringNs": reconciler.OpenshiftMonitoringNamespace,
		"LoggingNs":    OpenshiftLoggingNs,
		"NetobservNs":  OpenshiftNetobservNs,
		"TracingNs":    OpenshiftTracingNs,
	}

	if info.LokiServiceNames[OpenshiftLoggingNs] != "" {
		korrel8rData["Log"] = info.LokiServiceNames[OpenshiftLoggingNs]
	}
	if info.LokiServiceNames[OpenshiftNetobservNs] != "" {
		korrel8rData["Netflow"] = info.LokiServiceNames[OpenshiftNetobservNs]
	}
	if info.TempoServiceNames[OpenshiftTracingNs] != "" {
		korrel8rData["Trace"] = info.TempoServiceNames[OpenshiftTracingNs]
	}

	var korrel8rConfigYAMLTmpl = template.Must(template.ParseFS(korrel8rConfigYAMLTmplFile, "config/korrel8r.yaml"))
	w := bytes.NewBuffer(nil)
	err := korrel8rConfigYAMLTmpl.Execute(w, korrel8rData)
	if err != nil {
		return nil, err
	}

	cfg, _ := io.ReadAll(w)

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    componentLabels(name),
		},
		Data: map[string]string{
			Korrel8rConfigFileName: string(cfg),
		},
	}, nil
}

func componentLabels(pluginName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance":   pluginName,
		"app.kubernetes.io/part-of":    "UIPlugin",
		"app.kubernetes.io/managed-by": "observability-operator",
	}
}

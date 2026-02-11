package uiplugin

import (
	"fmt"
	"strings"

	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	persesv1alpha2 "github.com/rhobs/perses-operator/api/v1alpha2"
	persesconfig "github.com/rhobs/perses/pkg/model/api/config"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

const persesServiceName = "perses"

/*
Requirements for ACM enablement
1. UIPlugin configuration requires acm.enabled, acm.thanosQuerier.Url, and acm.alertmanager.Url
2. OpenShift Container Platform requirement: v4.14+
*/
func validateACMConfig(config *uiv1alpha1.MonitoringConfig) bool {
	enabled := config.ACM != nil && config.ACM.Enabled

	if !enabled {
		return false
	}

	// alertManager and thanosQuerier url configurations are required to enable 'acm-alerting'
	validAlertManagerUrl := config.ACM.Alertmanager.Url != ""
	validThanosQuerierUrl := config.ACM.ThanosQuerier.Url != ""
	isValidAcmAlertingConfig := validAlertManagerUrl && validThanosQuerierUrl

	return isValidAcmAlertingConfig && enabled
}

func validatePersesConfig(config *uiv1alpha1.MonitoringConfig) bool {
	return config.Perses != nil && config.Perses.Enabled
}

func validateHealthanalyzerConfig(config *uiv1alpha1.MonitoringConfig, clusterVersion string) bool {
	enabled := config.ClusterHealthAnalyzer != nil &&
		config.ClusterHealthAnalyzer.Enabled

	if !strings.HasPrefix(clusterVersion, "v") {
		clusterVersion = "v" + clusterVersion
	}
	canonicalClusterVersion := fmt.Sprintf("%s-0", semver.Canonical(clusterVersion))
	minClusterVersionMet := semver.Compare(canonicalClusterVersion, "v4.19.0-0") >= 0

	return enabled && minClusterVersionMet
}

func validateIncidentsConfig(config *uiv1alpha1.MonitoringConfig, clusterVersion string) bool {
	enabled := config.Incidents != nil && config.Incidents.Enabled

	if !strings.HasPrefix(clusterVersion, "v") {
		clusterVersion = "v" + clusterVersion
	}
	canonicalClusterVersion := fmt.Sprintf("%s-0", semver.Canonical(clusterVersion))
	minClusterVersionMet := semver.Compare(canonicalClusterVersion, "v4.19.0-0") >= 0

	return enabled && minClusterVersionMet
}

func addFeatureFlags(plugin *UIPluginInfo, features []string) {
	featureField := fmt.Sprintf("-features=%s", strings.Join(features, ","))
	plugin.ExtraArgs = append(plugin.ExtraArgs, featureField)
}

func getBasePluginInfo(namespace, name, image string) *UIPluginInfo {
	return &UIPluginInfo{
		Image:       image,
		Name:        name,
		ConsoleName: "monitoring-console-plugin",
		DisplayName: "Monitoring Console Plugin",
		ExtraArgs: []string{
			"-config-path=/opt/app-root/config",
			"-static-path=/opt/app-root/web/dist",
		},
		ResourceNamespace: namespace,
		Proxies: []osv1.ConsolePluginProxy{
			{
				Alias:         "backend",
				Authorization: "UserToken",
				Endpoint: osv1.ConsolePluginProxyEndpoint{
					Type: osv1.ProxyTypeService,
					Service: &osv1.ConsolePluginProxyServiceConfig{
						Name:      name,
						Namespace: namespace,
						Port:      port,
					},
				},
			},
		},
		LegacyProxies: []osv1alpha1.ConsolePluginProxy{
			{
				Type:      "Service",
				Alias:     "backend",
				Authorize: true,
				Service: osv1alpha1.ConsolePluginProxyServiceConfig{
					Name:      name,
					Namespace: namespace,
					Port:      9443,
				},
			},
		},
	}
}

func addPersesProxy(pluginInfo *UIPluginInfo, namespace string) {
	pluginInfo.Proxies = append(pluginInfo.Proxies, osv1.ConsolePluginProxy{
		Alias:         "perses",
		Authorization: "UserToken",
		Endpoint: osv1.ConsolePluginProxyEndpoint{
			Type: osv1.ProxyTypeService,
			Service: &osv1.ConsolePluginProxyServiceConfig{
				Name:      persesServiceName,
				Namespace: namespace,
				Port:      8080,
			},
		},
	})
	pluginInfo.LegacyProxies = append(pluginInfo.LegacyProxies, osv1alpha1.ConsolePluginProxy{
		Type:      "Service",
		Alias:     "perses",
		Authorize: true,
		Service: osv1alpha1.ConsolePluginProxyServiceConfig{
			Name:      persesServiceName,
			Namespace: namespace,
			Port:      8080,
		},
	})
}

func addAcmAlertingProxy(pluginInfo *UIPluginInfo, name string, namespace string, config *uiv1alpha1.MonitoringConfig) {
	pluginInfo.ExtraArgs = append(pluginInfo.ExtraArgs,
		fmt.Sprintf("-alertmanager=%s", config.ACM.Alertmanager.Url),
		fmt.Sprintf("-thanos-querier=%s", config.ACM.ThanosQuerier.Url),
	)
	pluginInfo.Proxies = append(pluginInfo.Proxies,
		osv1.ConsolePluginProxy{
			Alias:         "alertmanager-proxy",
			Authorization: "UserToken",
			Endpoint: osv1.ConsolePluginProxyEndpoint{
				Type: osv1.ProxyTypeService,
				Service: &osv1.ConsolePluginProxyServiceConfig{
					Name:      name,
					Namespace: namespace,
					Port:      9444,
				},
			},
		},
		osv1.ConsolePluginProxy{
			Alias:         "thanos-proxy",
			Authorization: "UserToken",
			Endpoint: osv1.ConsolePluginProxyEndpoint{
				Type: osv1.ProxyTypeService,
				Service: &osv1.ConsolePluginProxyServiceConfig{
					Name:      name,
					Namespace: namespace,
					Port:      9445,
				},
			},
		},
	)
	pluginInfo.LegacyProxies = append(pluginInfo.LegacyProxies,
		osv1alpha1.ConsolePluginProxy{
			Type:      "Service",
			Alias:     "alertmanager-proxy",
			Authorize: true,
			Service: osv1alpha1.ConsolePluginProxyServiceConfig{
				Name:      name,
				Namespace: namespace,
				Port:      9444,
			},
		},
		osv1alpha1.ConsolePluginProxy{
			Type:      "Service",
			Alias:     "thanos-proxy",
			Authorize: true,
			Service: osv1alpha1.ConsolePluginProxyServiceConfig{
				Name:      name,
				Namespace: namespace,
				Port:      9445,
			},
		},
	)
}

func createMonitoringPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string, clusterVersion string, healthAnalyzerImage string, persesImage string) (*UIPluginInfo, error) {
	config := plugin.Spec.Monitoring
	if config == nil {
		return nil, fmt.Errorf("monitoring configuration can not be empty for plugin type %s", plugin.Spec.Type)
	}

	// Validate feature configuration and cluster conditions support enablement
	isValidAcmConfig := validateACMConfig(config)
	isValidPersesConfig := validatePersesConfig(config)
	isValidIncidentsConfig := validateIncidentsConfig(config, clusterVersion)
	isValidHealthAnalyzerConfig := validateHealthanalyzerConfig(config, clusterVersion)

	atLeastOneValidConfig := isValidAcmConfig || isValidPersesConfig || isValidIncidentsConfig || isValidHealthAnalyzerConfig

	pluginInfo := getBasePluginInfo(namespace, name, image)
	if !atLeastOneValidConfig {
		pluginInfo.AreMonitoringFeatsDisabled = true
		// pluginInfo must be return to controller to delete related components
		return pluginInfo, fmt.Errorf("all uiplugin monitoring configurations are invalid or not supported in this cluster version")
	}

	//  Add proxies and feature flags
	if isValidAcmConfig {
		addAcmAlertingProxy(pluginInfo, name, namespace, config)
		features = append(features, "acm-alerting")
	}
	if isValidPersesConfig {
		addPersesProxy(pluginInfo, namespace)
		features = append(features, "perses-dashboards")
		pluginInfo.PersesImage = persesImage
	}
	if isValidIncidentsConfig {
		pluginInfo.HealthAnalyzerImage = healthAnalyzerImage
		features = append(features, "incidents")
	}
	if isValidHealthAnalyzerConfig {
		pluginInfo.HealthAnalyzerImage = healthAnalyzerImage
		features = append(features, "cluster-health-analyzer")
	}
	addFeatureFlags(pluginInfo, features)

	return pluginInfo, nil
}

func newMonitoringService(name string, namespace string) *corev1.Service {
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
					Port:       9443,
					Name:       "backend",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(9443),
				},
				{
					Port:       9444,
					Name:       "alertmanager-proxy",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(9444),
				},
				{
					Port:       9445,
					Name:       "thanos-proxy",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(9445),
				},
			},
			Selector: componentLabels(name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func newPerses(namespace string, persesImage string) *persesv1alpha2.Perses {
	name := "perses"
	return &persesv1alpha2.Perses{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha2.GroupVersion.String(),
			Kind:       "Perses",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "perses",
				"app.kubernetes.io/instance":   "perses-observability-operator",
				"app.kubernetes.io/component":  "perses",
				"app.kubernetes.io/part-of":    "perses",
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Spec: persesv1alpha2.PersesSpec{
			Config: persesv1alpha2.PersesConfig{
				Config: persesconfig.Config{
					Security: persesconfig.Security{
						EnableAuth: true,
						Authorization: persesconfig.AuthorizationConfig{
							Provider: persesconfig.AuthorizationProvider{
								Kubernetes: persesconfig.KubernetesAuthorizationProvider{
									Enable: true,
								},
							},
						},
						Authentication: persesconfig.AuthenticationConfig{
							DisableSignUp: true,
							Providers: persesconfig.AuthenticationProviders{
								KubernetesProvider: persesconfig.K8sAuthnProvider{
									Enable: true,
								},
							},
						},
					},
					Database: persesconfig.Database{
						File: &persesconfig.File{
							Folder:    "/perses",
							Extension: persesconfig.YAMLExtension,
						},
					},
				},
			},
			Image:         persesImage,
			ContainerPort: 8080,
			// Set an empty PodSecurityContext to prevent the Perses operator from defaulting to invalid values
			PodSecurityContext: &corev1.PodSecurityContext{},
			TLS: &persesv1alpha2.TLS{
				Enable: true,
				UserCert: &persesv1alpha2.Certificate{
					SecretSource: persesv1alpha2.SecretSource{
						Type:      persesv1alpha2.SecretSourceTypeSecret,
						Name:      name,
						Namespace: namespace,
					},
					CertPath:       "tls.crt",
					PrivateKeyPath: "tls.key",
				},
				CaCert: &persesv1alpha2.Certificate{
					SecretSource: persesv1alpha2.SecretSource{
						Type:      persesv1alpha2.SecretSourceTypeConfigMap,
						Name:      "openshift-service-ca.crt",
						Namespace: namespace,
					},
					CertPath: "service-ca.crt",
				},
			},
			Client: &persesv1alpha2.Client{
				TLS: &persesv1alpha2.TLS{
					Enable: true,
					CaCert: &persesv1alpha2.Certificate{
						SecretSource: persesv1alpha2.SecretSource{
							Type:      persesv1alpha2.SecretSourceTypeConfigMap,
							Name:      "openshift-service-ca.crt",
							Namespace: namespace,
						},
						CertPath: "service-ca.crt",
					},
				},
				KubernetesAuth: &persesv1alpha2.KubernetesAuth{
					Enable: true,
				},
			},
			Service: &persesv1alpha2.PersesService{
				Annotations: map[string]string{
					"service.beta.openshift.io/serving-cert-secret-name": name,
				},
			},
			ServiceAccountName: "perses" + serviceAccountSuffix,
		},
	}
}

func newPersesClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "perses-cr",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "observability-operator",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"list", "get"},
			},
			{
				APIGroups: []string{"perses.dev"},
				Resources: []string{"persesdashboards", "persesdatasources", "persesglobaldatasources"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete", "patch"},
			},
			{
				APIGroups:     []string{"security.openshift.io"},
				Resources:     []string{"securitycontextconstraints"},
				ResourceNames: []string{"nonroot", "nonroot-v2"},
				Verbs:         []string{"use"},
			},
		},
	}
}

func newAlertManagerViewRoleBinding(serviceAccountName, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alertmanager-view-rolebinding",
			Namespace: "openshift-monitoring",
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
			Kind:     "Role",
			Name:     "monitoring-alertmanager-view",
		},
	}
}

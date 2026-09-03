package uiplugin

import (
	"fmt"
	"strings"

	persesv1alpha2 "github.com/rhobs/perses-operator/api/v1alpha2"
	persesconfig "github.com/rhobs/perses/pkg/model/api/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

const (
	persesServiceName   = "perses"
	PersesUserFSGroupID = int64(65534)
)

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
	return config.ClusterHealthAnalyzer != nil &&
		config.ClusterHealthAnalyzer.Enabled &&
		IsVersionAheadOrEqual(clusterVersion, "v4.19")
}

func validateIncidentsConfig(config *uiv1alpha1.MonitoringConfig, clusterVersion string) bool {
	return config.Incidents != nil &&
		config.Incidents.Enabled &&
		IsVersionAheadOrEqual(clusterVersion, "v4.19")
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
		Proxies: []PluginProxy{
			{
				Alias:            "backend",
				ServiceName:      name,
				ServiceNamespace: namespace,
				ServicePort:      port,
				Authorize:        true,
			},
		},
	}
}

func addPersesProxy(pluginInfo *UIPluginInfo, namespace string) {
	pluginInfo.Proxies = append(pluginInfo.Proxies, PluginProxy{
		Alias:            "perses",
		ServiceName:      persesServiceName,
		ServiceNamespace: namespace,
		ServicePort:      8080,
		Authorize:        true,
	})
}

func addAcmAlertingProxy(pluginInfo *UIPluginInfo, name string, namespace string, config *uiv1alpha1.MonitoringConfig) {
	pluginInfo.ExtraArgs = append(pluginInfo.ExtraArgs,
		fmt.Sprintf("-alertmanager=%s", config.ACM.Alertmanager.Url),
		fmt.Sprintf("-thanos-querier=%s", config.ACM.ThanosQuerier.Url),
	)
	pluginInfo.Proxies = append(pluginInfo.Proxies,
		PluginProxy{
			Alias:            "alertmanager-proxy",
			ServiceName:      name,
			ServiceNamespace: namespace,
			ServicePort:      9444,
			Authorize:        true,
		},
		PluginProxy{
			Alias:            "thanos-proxy",
			ServiceName:      name,
			ServiceNamespace: namespace,
			ServicePort:      9445,
			Authorize:        true,
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

	features = append(features, "mcp-overview")

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
	if isValidIncidentsConfig || isValidHealthAnalyzerConfig {
		pluginInfo.HealthAnalyzerImage = healthAnalyzerImage
		features = append(features, "cluster-health-analyzer")
	}
	addFeatureFlags(pluginInfo, features)

	return pluginInfo, nil
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
							Folder:        "/perses",
							Extension:     persesconfig.YAMLExtension,
							CaseSensitive: true,
						},
					},
				},
			},
			Image:         ptr.To(persesImage),
			ContainerPort: ptr.To(int32(8080)),
			// Set PodSecurityContext to run as non-root user (nobody/65534) for OpenShift SCC compatibility
			PodSecurityContext: &corev1.PodSecurityContext{
				FSGroup:      ptr.To(PersesUserFSGroupID),
				RunAsNonRoot: ptr.To(true),
				RunAsUser:    ptr.To(PersesUserFSGroupID),
			},
			ContainerSecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsNonRoot:           ptr.To(true),
				ReadOnlyRootFilesystem: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			TLS: &persesv1alpha2.TLS{
				Enable: ptr.To(true),
				UserCert: &persesv1alpha2.Certificate{
					SecretSource: persesv1alpha2.SecretSource{
						Type:      persesv1alpha2.SecretSourceTypeSecret,
						Name:      ptr.To(name),
						Namespace: ptr.To(namespace),
					},
					CertPath:       "tls.crt",
					PrivateKeyPath: ptr.To("tls.key"),
				},
				CaCert: &persesv1alpha2.Certificate{
					SecretSource: persesv1alpha2.SecretSource{
						Type:      persesv1alpha2.SecretSourceTypeConfigMap,
						Name:      ptr.To("openshift-service-ca.crt"),
						Namespace: ptr.To(namespace),
					},
					CertPath: "service-ca.crt",
				},
			},
			Client: &persesv1alpha2.Client{
				TLS: &persesv1alpha2.TLS{
					Enable: ptr.To(true),
					CaCert: &persesv1alpha2.Certificate{
						SecretSource: persesv1alpha2.SecretSource{
							Type:      persesv1alpha2.SecretSourceTypeConfigMap,
							Name:      ptr.To("openshift-service-ca.crt"),
							Namespace: ptr.To(namespace),
						},
						CertPath: "service-ca.crt",
					},
				},
				KubernetesAuth: &persesv1alpha2.KubernetesAuth{
					Enable: ptr.To(true),
				},
			},
			Service: &persesv1alpha2.PersesService{
				Annotations: map[string]string{
					"service.beta.openshift.io/serving-cert-secret-name": name,
				},
			},
			ServiceAccountName: ptr.To("perses" + serviceAccountSuffix),
		},
	}
}

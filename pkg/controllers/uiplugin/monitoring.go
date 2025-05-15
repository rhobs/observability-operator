package uiplugin

import (
	"fmt"
	"strings"
	"time"

	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	persesv1alpha1 "github.com/perses/perses-operator/api/v1alpha1"
	persesconfig "github.com/perses/perses/pkg/model/api/config"
	"github.com/perses/perses/pkg/model/api/v1/common"
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

	atLeastOneValidConfig := isValidAcmConfig || isValidPersesConfig || isValidIncidentsConfig
	if !atLeastOneValidConfig {
		return nil, fmt.Errorf("all uiplugin monitoring configurations are invalid")
	}

	//  Add proxies and feature flags
	pluginInfo := getBasePluginInfo(namespace, name, image)
	if isValidAcmConfig {
		addAcmAlertingProxy(pluginInfo, name, namespace, config)
		features = append(features, "acm-alerting")
	}
	if isValidPersesConfig {
		addPersesProxy(pluginInfo, namespace)
		pluginInfo.PersesImage = persesImage
		features = append(features, "perses-dashboards")
	}
	if isValidIncidentsConfig {
		pluginInfo.HealthAnalyzerImage = healthAnalyzerImage
		features = append(features, "incidents")
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

func newPerses(namespace string, persesImage string) *persesv1alpha1.Perses {
	name := "perses"
	return &persesv1alpha1.Perses{
		TypeMeta: metav1.TypeMeta{
			APIVersion: persesv1alpha1.GroupVersion.String(),
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
		Spec: persesv1alpha1.PersesSpec{
			Config: persesv1alpha1.PersesConfig{
				Config: persesconfig.Config{
					Security: persesconfig.Security{
						EnableAuth: false,
					},
					Database: persesconfig.Database{
						File: &persesconfig.File{
							Folder:    "/perses",
							Extension: persesconfig.YAMLExtension,
						},
					},
					Schemas: persesconfig.Schemas{
						PanelsPath:      "/etc/perses/cue/schemas/panels",
						QueriesPath:     "/etc/perses/cue/schemas/queries",
						DatasourcesPath: "/etc/perses/cue/schemas/datasources",
						VariablesPath:   "/etc/perses/cue/schemas/variables",
						Interval:        common.Duration(time.Hour * 6),
					},
				},
			},
			Image:         persesImage,
			ContainerPort: 8080,
			TLS: &persesv1alpha1.TLS{
				Enable: true,
				UserCert: &persesv1alpha1.Certificate{
					Type:           persesv1alpha1.CertificateTypeSecret,
					Name:           name,
					CertPath:       "tls.crt",
					PrivateKeyPath: "tls.key",
				},
				CaCert: &persesv1alpha1.Certificate{
					Type:     persesv1alpha1.CertificateTypeConfigMap,
					Name:     "openshift-service-ca.crt",
					CertPath: "service-ca.crt",
				},
			},
			Client: &persesv1alpha1.Client{
				TLS: &persesv1alpha1.TLS{
					Enable: true,
					CaCert: &persesv1alpha1.Certificate{
						Type:     persesv1alpha1.CertificateTypeSecret,
						CertPath: "ca.crt",
					},
				},
			},
			Service: &persesv1alpha1.PersesService{
				Annotations: map[string]string{
					"service.beta.openshift.io/serving-cert-secret-name": name,
				},
			},
		},
	}
}

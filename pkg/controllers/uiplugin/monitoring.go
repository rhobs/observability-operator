package uiplugin

import (
	"fmt"
	"slices"
	"strings"

	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

var AcmErrorMsg = `if you intend to enable "acm-alerting" thanos querier url and alertmanager url needs to be configured in the custom resource UIPlugin. `
var PersesErrorMsg = `if you intend to enable "perses-dashboards" a perses service name and namespace needs to be configured in the custom resource UIPlugin.`
var AlertmanagerEmptyMsg = "alertmanager location is empty for plugin type monitoring."
var ThanosEmptyMsg = "thanosQuerier location is empty for plugin type monitoring."
var PersesNameEmptyMsg = "persesName location is empty for plugin type monitoring."
var PersesNamespaceEmptyMsg = "persesNamespace location is empty for plugin type monitoring."
var IncompatibleFeaturesAndConfigsErrorMsg = "UIPlugin configurations are incompatible with feature flags"

func getConfigError(config *uiv1alpha1.MonitoringConfig) (bool, bool, bool, string) {
	errorSlice := []string{}
	errorMessage := ""

	// alertManager and thanosQuerier url configurations are required to enable 'acm-alerting'
	validAlertManagerUrl := config.Alertmanager.Url != ""
	if !validAlertManagerUrl {
		errorSlice = append(errorSlice, AlertmanagerEmptyMsg)
	}

	validThanosQuerierUrl := config.ThanosQuerier.Url != ""
	if !validThanosQuerierUrl {
		errorSlice = append(errorSlice, ThanosEmptyMsg)
	}
	isValidAcmAlertingConfig := validAlertManagerUrl && validThanosQuerierUrl

	// perses name and namespace are required to enable 'perses-dashboards'
	validPersesName := config.Perses.Name != ""
	if !validPersesName {
		errorSlice = append(errorSlice, PersesNameEmptyMsg)
	}
	validPersesNamespace := config.Perses.Namespace != ""
	if !validPersesNamespace {
		errorSlice = append(errorSlice, PersesNamespaceEmptyMsg)
	}
	isValidPersesConfig := validPersesName && validPersesNamespace

	// build error message by converting slice into one string
	if len(errorSlice) > 0 {
		// Add extra information in the error message based on type of incorrect configurations
		if !isValidPersesConfig {
			errorSlice = append([]string{PersesErrorMsg}, errorSlice...)
		}
		if !isValidAcmAlertingConfig {
			errorSlice = append([]string{AcmErrorMsg}, errorSlice...)
		}
		errorMessage = strings.Join(errorSlice, "")
	}

	allConfigsInvalid := !isValidAcmAlertingConfig && !isValidPersesConfig

	return allConfigsInvalid, isValidAcmAlertingConfig, isValidPersesConfig, errorMessage
}

func getBasePluginInfo(namespace, name, image string, features []string) *UIPluginInfo {
	return &UIPluginInfo{
		Image:       image,
		Name:        name,
		ConsoleName: "monitoring-console-plugin",
		DisplayName: "Monitoring Console Plugin",
		ExtraArgs: []string{
			fmt.Sprintf("-features=%s", strings.Join(features, ",")),
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

func addPersesProxy(pluginInfo *UIPluginInfo, persesName string, persesNamespace string) {
	pluginInfo.Proxies = append(pluginInfo.Proxies, osv1.ConsolePluginProxy{
		Alias:         "perses",
		Authorization: "UserToken",
		Endpoint: osv1.ConsolePluginProxyEndpoint{
			Type: osv1.ProxyTypeService,
			Service: &osv1.ConsolePluginProxyServiceConfig{
				Name:      persesName,
				Namespace: persesNamespace,
				Port:      8080,
			},
		},
	})
	pluginInfo.LegacyProxies = append(pluginInfo.LegacyProxies, osv1alpha1.ConsolePluginProxy{
		Type:      "Service",
		Alias:     "perses",
		Authorize: true,
		Service: osv1alpha1.ConsolePluginProxyServiceConfig{
			Name:      persesName,
			Namespace: persesNamespace,
			Port:      8080,
		},
	})
}

func addAcmAlertingProxy(pluginInfo *UIPluginInfo, name string, namespace string, config *uiv1alpha1.MonitoringConfig) {
	pluginInfo.ExtraArgs = append(pluginInfo.ExtraArgs,
		fmt.Sprintf("-alertmanager=%s", config.Alertmanager.Url),
		fmt.Sprintf("-thanos-querier=%s", config.ThanosQuerier.Url),
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

func createMonitoringPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string, acmVersion string) (*UIPluginInfo, error) {
	config := plugin.Spec.Monitoring
	if config == nil {
		return nil, fmt.Errorf("monitoring configuration can not be empty for plugin type %s", plugin.Spec.Type)
	}

	// Validate UIPlugin configuration
	allConfigsInvalid, validAcmAlertingConfig, validPersesConfig, errorMessage := getConfigError(config)
	if allConfigsInvalid {
		return nil, fmt.Errorf("%s", errorMessage)
	}

	//  Inject feature flags based on valid UIPlugin Configuration and version of dependencies
	if (semver.Compare(acmVersion, "v2.11") >= 0) && validAcmAlertingConfig {
		// "acm-alerting" feature is supported in ACM v2.11+
		features = append(features, "acm-alerting")
	}
	if validPersesConfig {
		features = append(features, "perses-dashboards")
	}

	// Validate at least one feature flag is present
	persesDashboardsFeatureEnabled := slices.Contains(features, "perses-dashboards")
	acmAlertingFeatureEnabled := slices.Contains(features, "acm-alerting")
	if !acmAlertingFeatureEnabled && !persesDashboardsFeatureEnabled {
		return nil, fmt.Errorf("monitoring feature flags were not set, check cluster compatibility")
	}

	// Validate at least one proxy can be added to monitoring plugin info
	validPersesProxyConditions := persesDashboardsFeatureEnabled && validPersesConfig
	validAcmAlertingProxyConditions := acmAlertingFeatureEnabled && validAcmAlertingConfig
	invalidProxyConditions := !validPersesProxyConditions && !validAcmAlertingProxyConditions
	if invalidProxyConditions {
		return nil, fmt.Errorf("%s", IncompatibleFeaturesAndConfigsErrorMsg)
	}

	pluginInfo := getBasePluginInfo(namespace, name, image, features)
	if validPersesProxyConditions {
		addPersesProxy(pluginInfo, config.Perses.Name, config.Perses.Namespace)
	}
	if validAcmAlertingProxyConditions {
		addAcmAlertingProxy(pluginInfo, name, namespace, config)
	}

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

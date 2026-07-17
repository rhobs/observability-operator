package uiplugin

import (
	"bytes"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/overlay"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

const (
	PlaceholderUIPluginName   = "placeholder-uiplugin-name"
	PlaceholderUIPluginCRName = "placeholder-uiplugin-crname"
)

type UIPluginBuildConfig struct {
	ConfigFS       fs.FS
	Images         map[string]string
	OperatorName   string
	Namespace      string
	ClusterVersion string
	TLSCiphers     []string
	TLSMinVersion  string
	// Pre-resolved values for plugins that need cluster state.
	// Empty strings use defaults.
	LokiStackName      string
	LokiStackNamespace string
	LokiServiceNames   map[string]string
	TempoServiceNames  map[string]string
}

func BuildUIPluginOverlay(plugin *uiv1alpha1.UIPlugin, conf UIPluginBuildConfig, logger logr.Logger) (*overlay.Overlay, *UIPluginInfo, error) {
	compatibilityInfo, err := lookupImageAndFeatures(plugin.Spec.Type, conf.ClusterVersion)
	if err != nil {
		return nil, nil, err
	}

	image := conf.Images[compatibilityInfo.ImageKey]
	if image == "" {
		return nil, nil, fmt.Errorf("no image provided for plugin type %s with key %s", plugin.Spec.Type, compatibilityInfo.ImageKey)
	}

	namespace := conf.Namespace
	features := slices.Clone(compatibilityInfo.Features)

	var pluginInfo *UIPluginInfo
	var pluginInfoErr error

	switch plugin.Spec.Type {
	case uiv1alpha1.TypeDashboards:
		pluginInfo, pluginInfoErr = createDashboardsPluginInfo(plugin, namespace, plugin.Name, image)

	case uiv1alpha1.TypeDistributedTracing:
		pluginInfo, pluginInfoErr = createDistributedTracingPluginInfo(plugin, namespace, plugin.Name, image, features)

	case uiv1alpha1.TypeLogging:
		lokiName := conf.LokiStackName
		lokiNs := conf.LokiStackNamespace
		if lokiName == "" {
			lokiName = resolveLokiStackNameFromCR(plugin)
		}
		if lokiNs == "" {
			lokiNs = OpenshiftLoggingNs
		}
		pluginInfo, pluginInfoErr = createLoggingPluginInfo(plugin, namespace, plugin.Name, image, features, lokiName, lokiNs)

	case uiv1alpha1.TypeTroubleshootingPanel:
		pluginInfo, pluginInfoErr = createTroubleshootingPanelPluginInfo(plugin, namespace, plugin.Name, image, features, conf.ClusterVersion, logger)
		if pluginInfoErr == nil {
			pluginInfo.Korrel8rImage = conf.Images["korrel8r"]
			if conf.LokiServiceNames != nil {
				pluginInfo.LokiServiceNames = conf.LokiServiceNames
			}
			if conf.TempoServiceNames != nil {
				pluginInfo.TempoServiceNames = conf.TempoServiceNames
			}
		}

	case uiv1alpha1.TypeMonitoring:
		pluginInfo, pluginInfoErr = createMonitoringPluginInfo(plugin, namespace, plugin.Name, image, features, conf.ClusterVersion, conf.Images["health-analyzer"], conf.Images["perses"])

	default:
		return nil, nil, fmt.Errorf("plugin type not supported: %s", plugin.Spec.Type)
	}

	if pluginInfo == nil {
		if pluginInfoErr != nil {
			return nil, nil, pluginInfoErr
		}
		return nil, nil, fmt.Errorf("failed to build plugin info for %s", plugin.Spec.Type)
	}

	pluginInfo.TLSMinVersion = conf.TLSMinVersion
	pluginInfo.TLSCiphers = conf.TLSCiphers

	o := overlay.New(conf.ConfigFS)
	o.SetNamespace(namespace)
	o.AddSubstitution(PlaceholderUIPluginName, pluginInfo.Name)
	o.AddSubstitution(PlaceholderUIPluginCRName, plugin.Name)

	var marshalErr error
	addResource := func(name string, obj client.Object) {
		if marshalErr != nil {
			return
		}
		data, err := kyaml.Marshal(obj)
		if err != nil {
			marshalErr = fmt.Errorf("marshaling %s: %w", name, err)
			return
		}
		o.AddResource("resources/"+name, data)
	}

	componentDir := pluginTypeToComponentDir(plugin.Spec.Type)
	if componentDir != "" {
		o.AddComponent(fmt.Sprintf("../../uiplugins/components/%s/resources", componentDir))
		if err := addPluginDeploymentPatch(o, *pluginInfo, plugin.Spec.Deployment); err != nil {
			return nil, nil, err
		}

		if pluginInfo.ConfigMap != nil {
			if err := addConfigMapPatch(o, *pluginInfo); err != nil {
				return nil, nil, err
			}
		}

		if plugin.Spec.Type == uiv1alpha1.TypeTroubleshootingPanel && pluginInfo.Korrel8rImage != "" {
			o.AddComponent("../../uiplugins/components/troubleshooting-panel/korrel8r")
			if err := addKorrel8rPatches(o, *pluginInfo); err != nil {
				return nil, nil, err
			}
		}
	} else {
		addResource("serviceaccount.yaml", newServiceAccount(pluginInfo.Name, namespace))
		addResource("deployment.yaml", newDeployment(*pluginInfo, namespace, plugin.Spec.Deployment))
		addResource("service.yaml", newService(*pluginInfo, namespace))

		if pluginInfo.Role != nil {
			addResource("role.yaml", newRole(*pluginInfo))
		}
		if pluginInfo.RoleBinding != nil {
			addResource("rolebinding.yaml", newRoleBinding(*pluginInfo))
		}
		if pluginInfo.ConfigMap != nil {
			addResource("configmap.yaml", pluginInfo.ConfigMap)
		}

		for i, role := range pluginInfo.ClusterRoles {
			if role != nil {
				addResource(fmt.Sprintf("clusterrole-%d.yaml", i), role)
			}
		}
		for i, binding := range pluginInfo.ClusterRoleBindings {
			if binding != nil {
				addResource(fmt.Sprintf("clusterrolebinding-%d.yaml", i), binding)
			}
		}
	}

	// ConsolePlugin is always generated from Go — its apiVersion varies by cluster version.
	if IsVersionAheadOrEqual(conf.ClusterVersion, "v4.19") {
		addResource("consoleplugin.yaml", newConsolePlugin(*pluginInfo, namespace))
	} else if IsVersionAheadOrEqual(conf.ClusterVersion, "v4.17") {
		addResource("consoleplugin.yaml", newRhobsConsolePlugin(*pluginInfo, namespace))
	} else {
		addResource("consoleplugin.yaml", newLegacyConsolePlugin(*pluginInfo, namespace))
	}

	if plugin.Spec.Type == uiv1alpha1.TypeMonitoring {
		monitoringConfig := plugin.Spec.Monitoring
		serviceAccountName := plugin.Name + serviceAccountSuffix
		incidentsEnabled := monitoringConfig != nil &&
			monitoringConfig.Incidents != nil &&
			monitoringConfig.Incidents.Enabled &&
			pluginInfo.HealthAnalyzerImage != ""

		healthAnalyzerEnabled := monitoringConfig != nil &&
			monitoringConfig.ClusterHealthAnalyzer != nil &&
			monitoringConfig.ClusterHealthAnalyzer.Enabled &&
			pluginInfo.HealthAnalyzerImage != ""

		deployHealthAnalyzer := incidentsEnabled || healthAnalyzerEnabled
		if deployHealthAnalyzer {
			addResource("ha-clusterrole.yaml", componentsHealthClusterRole("components-health-view"))
			addResource("ha-clusterrolebinding-components.yaml", newClusterRoleBinding(namespace, serviceAccountName, "components-health-view", plugin.Name+"-components-health-view"))
			addResource("ha-configmap.yaml", newComponentHealthConfig(namespace))
			addResource("ha-clusterrolebinding-monitoring.yaml", newClusterRoleBinding(namespace, serviceAccountName, "cluster-monitoring-view", plugin.Name+"cluster-monitoring-view"))
			addResource("ha-clusterrolebinding-auth.yaml", newClusterRoleBinding(namespace, serviceAccountName, "system:auth-delegator", serviceAccountName+"-system-auth-delegator"))
			addResource("ha-alertmanager-rolebinding.yaml", newAlertManagerViewRoleBinding(serviceAccountName, namespace))
			addResource("ha-prometheus-role.yaml", newHealthAnalyzerPrometheusRole(namespace))
			addResource("ha-prometheus-rolebinding.yaml", newHealthAnalyzerPrometheusRoleBinding(namespace))
			addResource("ha-service.yaml", newHealthAnalyzerService(namespace))
			addResource("ha-deployment.yaml", newHealthAnalyzerDeployment(namespace, serviceAccountName, *pluginInfo))
			addResource("ha-servicemonitor.yaml", newHealthAnalyzerServiceMonitor(namespace))
		}

		persesEnabled := monitoringConfig != nil && monitoringConfig.Perses != nil && monitoringConfig.Perses.Enabled
		if persesEnabled {
			persesServiceAccountName := "perses" + serviceAccountSuffix
			addResource("perses-serviceaccount.yaml", newServiceAccount("perses", namespace))
			addResource("perses-clusterrolebinding-auth.yaml", newClusterRoleBinding(namespace, persesServiceAccountName, "system:auth-delegator", persesServiceAccountName+"-system-auth-delegator"))
			addResource("perses-clusterrole.yaml", newPersesClusterRole())
			addResource("perses-clusterrolebinding.yaml", newClusterRoleBinding(namespace, persesServiceAccountName, "perses-cr", persesServiceAccountName+"-perses-cr"))
			addResource("perses.yaml", newPerses(namespace, pluginInfo.PersesImage))
			addResource("perses-datasource.yaml", newAcceleratorsDatasource(namespace))

			acceleratorsDashboard, err := newAcceleratorsDashboard(namespace)
			if err != nil {
				logger.Error(err, "Cannot build Accelerators dashboard")
			} else {
				addResource("perses-accelerators-dashboard.yaml", acceleratorsDashboard)
			}
			apmDashboard, err := newAPMDashboard(namespace)
			if err != nil {
				logger.Error(err, "Cannot build APM dashboard")
			} else {
				addResource("perses-apm-dashboard.yaml", apmDashboard)
			}
		}
	}

	if marshalErr != nil {
		return nil, nil, marshalErr
	}
	return o, pluginInfo, pluginInfoErr
}

func resolveLokiStackNameFromCR(plugin *uiv1alpha1.UIPlugin) string {
	if plugin.Spec.Logging != nil && plugin.Spec.Logging.LokiStack != nil && plugin.Spec.Logging.LokiStack.Name != "" {
		return plugin.Spec.Logging.LokiStack.Name
	}
	return DefaultLokiStackName
}

func pluginTypeToComponentDir(pluginType uiv1alpha1.UIPluginType) string {
	switch pluginType {
	case uiv1alpha1.TypeDashboards:
		return "dashboards"
	case uiv1alpha1.TypeDistributedTracing:
		return "distributed-tracing"
	case uiv1alpha1.TypeLogging:
		return "logging"
	case uiv1alpha1.TypeTroubleshootingPanel:
		return "troubleshooting-panel"
	default:
		return ""
	}
}

func addPluginDeploymentPatch(o *overlay.Overlay, info UIPluginInfo, deployConfig *uiv1alpha1.DeploymentConfig) error {
	pluginArgs := []string{
		fmt.Sprintf("-port=%d", port),
		"-cert=/var/serving-cert/tls.crt",
		"-key=/var/serving-cert/tls.key",
	}
	pluginArgs = append(pluginArgs, info.ExtraArgs...)
	if info.TLSMinVersion != "" {
		pluginArgs = append(pluginArgs, fmt.Sprintf("-tls-min-version=%s", info.TLSMinVersion))
	}
	if len(info.TLSCiphers) > 0 {
		pluginArgs = append(pluginArgs, fmt.Sprintf("-tls-cipher-suites=%s", strings.Join(info.TLSCiphers, ",")))
	}

	metadata := map[string]any{}
	if info.ConfigMap != nil {
		metadata["annotations"] = map[string]any{
			annotationPrefix + "config-hash": computeConfigMapHash(info.ConfigMap),
		}
	}

	podSpec := map[string]any{
		"containers": []any{
			map[string]any{
				"name":  PlaceholderUIPluginName,
				"image": info.Image,
				"args":  pluginArgs,
			},
		},
	}

	if deployConfig != nil {
		nodeSelector, tolerations := createNodeSelectorAndTolerations(deployConfig)
		if nodeSelector != nil {
			podSpec["nodeSelector"] = nodeSelector
		}
		if len(tolerations) > 0 {
			podSpec["tolerations"] = tolerations
		}
	}

	template := map[string]any{"spec": podSpec}
	if len(metadata) > 0 {
		template["metadata"] = metadata
	}

	return o.AddPatchMap("patches/deployment.yaml", map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": PlaceholderUIPluginName},
		"spec":       map[string]any{"template": template},
	})
}

func addConfigMapPatch(o *overlay.Overlay, info UIPluginInfo) error {
	return o.AddPatchMap("patches/configmap.yaml", map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": PlaceholderUIPluginName},
		"data":       info.ConfigMap.Data,
	})
}

func addKorrel8rPatches(o *overlay.Overlay, info UIPluginInfo) error {
	if err := o.AddPatchMap("patches/korrel8r-deployment.yaml", map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": "korrel8r"},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "korrel8r",
							"image": info.Korrel8rImage,
						},
					},
				},
			},
		},
	}); err != nil {
		return err
	}

	configYAML, err := generateKorrel8rConfig(info)
	if err != nil {
		return err
	}
	return o.AddPatchMap("patches/korrel8r-configmap.yaml", map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": "korrel8r"},
		"data":       map[string]any{"korrel8r.yaml": configYAML},
	})
}

func generateKorrel8rConfig(info UIPluginInfo) (string, error) {
	korrel8rData := map[string]string{
		"Metric":       "thanos-querier",
		"MetricAlert":  "alertmanager-main",
		"Log":          "logging-loki-gateway-http",
		"Netflow":      "loki-gateway-http",
		"Trace":        "tempo-platform-gateway",
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

	var buf bytes.Buffer
	if err := korrel8rConfigTmpl.Execute(&buf, korrel8rData); err != nil {
		return "", err
	}
	return buf.String(), nil
}

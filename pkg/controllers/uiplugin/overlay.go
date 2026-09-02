package uiplugin

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/overlay"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

func BuildOverlay(plugin *uiv1alpha1.UIPlugin, conf UIPluginInfo, logger logr.Logger) (*overlay.Overlay, *UIPluginInfo, error) {
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
	//nolint:staticcheck // allow deprecated version
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

	pluginInfo.Scheme = conf.Scheme
	pluginInfo.ConfigFS = conf.ConfigFS
	pluginInfo.Images = conf.Images
	pluginInfo.ClusterVersion = conf.ClusterVersion
	pluginInfo.Namespace = conf.Namespace
	if conf.LokiServiceNames != nil {
		pluginInfo.LokiServiceNames = conf.LokiServiceNames
	}
	if conf.TempoServiceNames != nil {
		pluginInfo.TempoServiceNames = conf.TempoServiceNames
	}
	pluginInfo.TLSMinVersion = conf.TLSMinVersion
	pluginInfo.TLSCiphers = conf.TLSCiphers

	o := overlay.New(conf.Scheme, conf.ConfigFS)
	o.SetNamespace(namespace)
	o.SetClusterScoped("console.openshift.io", "ConsolePlugin")

	var marshalErr error
	addResource := func(obj client.Object) {
		if marshalErr == nil {
			marshalErr = o.AddResource(obj, "resources")
		}
	}

	componentDir := pluginTypeToComponentDir(plugin.Spec.Type)
	o.AddComponent(fmt.Sprintf("uiplugins/components/%s/resources", componentDir))
	addUIPluginReplacements(o, componentDir, pluginInfo.Name, plugin.Name, pluginInfo.ConfigMap != nil)
	if err := addPluginDeploymentPatch(o, componentDir, *pluginInfo, plugin.Spec.Deployment); err != nil {
		return nil, nil, err
	}

	if pluginInfo.ConfigMap != nil {
		if err := addConfigMapPatch(o, componentDir, *pluginInfo); err != nil {
			return nil, nil, err
		}
	}

	if plugin.Spec.Type == uiv1alpha1.TypeTroubleshootingPanel && pluginInfo.Korrel8rImage != "" {
		o.AddComponent("uiplugins/components/troubleshooting-panel/korrel8r")
		if err := addKorrel8rPatches(o, *pluginInfo); err != nil {
			return nil, nil, err
		}
	}

	if err := addConsolePluginComponent(o, *pluginInfo, namespace, conf.ClusterVersion); err != nil {
		return nil, nil, err
	}

	if plugin.Spec.Type == uiv1alpha1.TypeMonitoring {
		if err := addMonitoringComponents(o, addResource, plugin, *pluginInfo, namespace, logger); err != nil {
			return nil, nil, err
		}
	}

	if marshalErr != nil {
		return nil, nil, marshalErr
	}
	return o, pluginInfo, pluginInfoErr
}

// ResolvePlugin builds the operand object set for a UIPlugin: the
// overlay build output plus the plugin info. It is shared by the reconciler
// and the generator so both produce the same objects.
func ResolvePlugin(plugin *uiv1alpha1.UIPlugin, conf UIPluginInfo, logger logr.Logger) ([]client.Object, *UIPluginInfo, error) {
	o, pluginInfo, pluginInfoErr := BuildOverlay(plugin, conf, logger)
	if o == nil || pluginInfo == nil {
		return nil, pluginInfo, pluginInfoErr
	}
	objects, err := o.Build()
	if err != nil {
		return nil, pluginInfo, fmt.Errorf("building UIPlugin %s overlay: %w", plugin.Name, err)
	}
	return objects, pluginInfo, pluginInfoErr
}

func resolveLokiStackNameFromCR(plugin *uiv1alpha1.UIPlugin) string {
	if plugin.Spec.Logging != nil && plugin.Spec.Logging.LokiStack != nil && plugin.Spec.Logging.LokiStack.Name != "" {
		return plugin.Spec.Logging.LokiStack.Name
	}
	return DefaultLokiStackName
}

func pluginTypeToComponentDir(pluginType uiv1alpha1.UIPluginType) string {
	switch pluginType {
	//nolint:staticcheck // allow deprecated version
	case uiv1alpha1.TypeDashboards:
		return "dashboards"
	case uiv1alpha1.TypeDistributedTracing:
		return "distributed-tracing"
	case uiv1alpha1.TypeLogging:
		return "logging"
	case uiv1alpha1.TypeTroubleshootingPanel:
		return "troubleshooting-panel"
	case uiv1alpha1.TypeMonitoring:
		return "monitoring"
	default:
		return ""
	}
}

func addPluginDeploymentPatch(o *overlay.Overlay, componentDir string, info UIPluginInfo, deployConfig *uiv1alpha1.DeploymentConfig) error {
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
				"name":  componentDir,
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

	return o.AddPatch("patches/deployment.yaml", map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": componentDir},
		"spec":       map[string]any{"template": template},
	})
}

func addConfigMapPatch(o *overlay.Overlay, componentDir string, info UIPluginInfo) error {
	return o.AddPatch("patches/configmap.yaml", map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": componentDir},
		"data":       info.ConfigMap.Data,
	})
}

func addKorrel8rPatches(o *overlay.Overlay, info UIPluginInfo) error {
	container := map[string]any{
		"name":  "korrel8r",
		"image": info.Korrel8rImage,
	}

	if info.TLSMinVersion != "" || len(info.TLSCiphers) > 0 {
		command := []any{
			"korrel8r", "web",
			fmt.Sprintf("--https=:%d", port),
			"--cert=/secrets/tls.crt",
			"--key=/secrets/tls.key",
			"--config=/config/korrel8r.yaml",
		}
		if info.TLSMinVersion != "" {
			command = append(command, fmt.Sprintf("--tls-min-version=%s", info.TLSMinVersion))
		}
		if len(info.TLSCiphers) > 0 {
			command = append(command, fmt.Sprintf("--tls-cipher-suites=%s", strings.Join(info.TLSCiphers, ",")))
		}
		container["command"] = command
	}

	if err := o.AddPatch("patches/korrel8r-deployment.yaml", map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": "korrel8r"},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{container},
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
	return o.AddPatch("patches/korrel8r-configmap.yaml", map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": "korrel8r"},
		"data":       map[string]any{"korrel8r.yaml": configYAML},
	})
}

func addConsolePluginComponent(o *overlay.Overlay, info UIPluginInfo, namespace, clusterVersion string) error {
	var serviceNamePath, serviceNsPath string
	if IsVersionAheadOrEqual(clusterVersion, "v4.17") {
		if IsVersionAheadOrEqual(clusterVersion, "v4.19") {
			o.AddComponent("uiplugins/components/consoleplugin/v1")
		} else {
			o.AddComponent("uiplugins/components/consoleplugin/rhobs-v1")
		}
		serviceNamePath = "spec.backend.service.name"
		serviceNsPath = "spec.backend.service.namespace"
	} else {
		o.AddComponent("uiplugins/components/consoleplugin/v1alpha1")
		serviceNamePath = "spec.service.name"
		serviceNsPath = "spec.service.namespace"
	}

	// Set service reference before renaming
	o.ReplaceValue(info.Name,
		overlay.TargetKindName("ConsolePlugin", "consoleplugin", serviceNamePath),
	)
	o.ReplaceValue(namespace,
		overlay.TargetKindName("ConsolePlugin", "consoleplugin", serviceNsPath),
	)

	// Rename last since it changes the resource ID
	o.ReplaceValue(info.ConsoleName,
		overlay.TargetKindName("ConsolePlugin", "consoleplugin", "metadata.name"),
	)

	return addConsolePluginPatch(o, info, clusterVersion)
}

func addConsolePluginPatch(o *overlay.Overlay, info UIPluginInfo, clusterVersion string) error {
	var proxies []any
	var apiVersion string

	if IsVersionAheadOrEqual(clusterVersion, "v4.17") {
		apiVersion = "console.openshift.io/v1"
		for _, p := range info.Proxies {
			auth := "None"
			if p.Authorize {
				auth = "UserToken"
			}
			proxies = append(proxies, map[string]any{
				"alias":         p.Alias,
				"authorization": auth,
				"endpoint": map[string]any{
					"type": "Service",
					"service": map[string]any{
						"name":      p.ServiceName,
						"namespace": p.ServiceNamespace,
						"port":      p.ServicePort,
					},
				},
			})
		}
	} else {
		apiVersion = "console.openshift.io/v1alpha1"
		for _, p := range info.Proxies {
			proxies = append(proxies, map[string]any{
				"type":      "Service",
				"alias":     p.Alias,
				"authorize": p.Authorize,
				"service": map[string]any{
					"name":      p.ServiceName,
					"namespace": p.ServiceNamespace,
					"port":      p.ServicePort,
				},
			})
		}
	}

	spec := map[string]any{
		"displayName": info.DisplayName,
	}
	if len(proxies) > 0 {
		spec["proxy"] = proxies
	}

	return o.AddPatch("patches/consoleplugin.yaml", map[string]any{
		"apiVersion": apiVersion,
		"kind":       "ConsolePlugin",
		"metadata":   map[string]any{"name": "consoleplugin"},
		"spec":       spec,
	})
}

func addMonitoringComponents(o *overlay.Overlay, addResource func(client.Object), plugin *uiv1alpha1.UIPlugin, info UIPluginInfo, namespace string, logger logr.Logger) error {
	monitoringConfig := plugin.Spec.Monitoring

	incidentsEnabled := monitoringConfig != nil &&
		monitoringConfig.Incidents != nil &&
		monitoringConfig.Incidents.Enabled &&
		info.HealthAnalyzerImage != ""

	healthAnalyzerEnabled := monitoringConfig != nil &&
		monitoringConfig.ClusterHealthAnalyzer != nil &&
		monitoringConfig.ClusterHealthAnalyzer.Enabled &&
		info.HealthAnalyzerImage != ""

	if incidentsEnabled || healthAnalyzerEnabled {
		o.AddComponent("uiplugins/components/monitoring/health-analyzer")
		if err := addHealthAnalyzerPatches(o, info, namespace); err != nil {
			return err
		}
		addHealthAnalyzerReplacements(o, info.Name, namespace)
	}

	persesEnabled := monitoringConfig != nil && monitoringConfig.Perses != nil && monitoringConfig.Perses.Enabled
	if persesEnabled {
		o.AddComponent("uiplugins/components/monitoring/perses")
		persesSA := "perses" + serviceAccountSuffix
		persesCR := "perses" + clusterRoleSuffix
		o.ReplaceValue(persesSA,
			overlay.TargetKindName("ServiceAccount", "perses", "metadata.name"),
			overlay.TargetKindName("ClusterRoleBinding", "perses-perses", "subjects.[kind=ServiceAccount].name"),
			overlay.TargetKindName("ClusterRoleBinding", "perses-system-auth-delegator", "subjects.[kind=ServiceAccount].name"),
		)
		o.ReplaceValue(persesCR,
			overlay.TargetKindName("ClusterRole", "perses", "metadata.name"),
			overlay.TargetKindName("ClusterRoleBinding", "perses-perses", "roleRef.name"),
		)
		o.ReplaceValue(persesSA+"-"+persesCR,
			overlay.TargetName("perses-perses", "metadata.name"),
		)
		o.ReplaceValue(persesSA+"-system-auth-delegator",
			overlay.TargetName("perses-system-auth-delegator", "metadata.name"),
		)
		// The Perses instance, datasource and dashboards are built in Go from
		// the Perses SDK (newPerses, newAcceleratorsDatasource,
		// newAcceleratorsDashboard, newAPMDashboard) rather than static config/
		// manifests because their bodies are SDK-generated. This is a documented
		// scope exception of the kustomize overlay refactor; do not move them to
		// config/ without a generator that produces equivalent SDK output.
		addResource(newPerses(namespace, info.PersesImage))
		addResource(newPrometheusGlobalDatasource())
		addResource(newAcceleratorsDatasource(namespace))

		if dashboard, err := newAcceleratorsDashboard(namespace); err != nil {
			logger.Error(err, "Cannot build Accelerators dashboard")
		} else {
			addResource(dashboard)
		}
		if dashboard, err := newAPMDashboard(namespace); err != nil {
			logger.Error(err, "Cannot build APM dashboard")
		} else {
			addResource(dashboard)
		}
	}
	return nil
}

func addHealthAnalyzerPatches(o *overlay.Overlay, info UIPluginInfo, namespace string) error {
	args := []any{
		"serve",
		"--tls-cert-file=/etc/tls/private/tls.crt",
		"--tls-private-key-file=/etc/tls/private/tls.key",
	}
	if len(info.TLSCiphers) > 0 {
		args = append(args, fmt.Sprintf("--tls-cipher-suites=%s", strings.Join(info.TLSCiphers, ",")))
	}
	if info.TLSMinVersion != "" {
		args = append(args, fmt.Sprintf("--tls-min-version=%s", info.TLSMinVersion))
	}

	if err := o.AddPatch("patches/health-analyzer-deployment.yaml", map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": "health-analyzer"},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "health-analyzer",
							"image": info.HealthAnalyzerImage,
							"args":  args,
						},
					},
				},
			},
		},
	}); err != nil {
		return err
	}

	return o.AddPatch("patches/health-analyzer-servicemonitor.yaml", map[string]any{
		"apiVersion": "monitoring.coreos.com/v1",
		"kind":       "ServiceMonitor",
		"metadata":   map[string]any{"name": "health-analyzer"},
		"spec": map[string]any{
			"endpoints": []any{
				map[string]any{
					"port": "metrics",
					"tlsConfig": map[string]any{
						"serverName": "health-analyzer." + namespace + ".svc",
					},
				},
			},
		},
	})
}

func addHealthAnalyzerReplacements(o *overlay.Overlay, pluginName, namespace string) {
	saName := pluginName + serviceAccountSuffix
	o.ReplaceValue(saName,
		overlay.TargetKindName("ClusterRoleBinding", "monitoring-components-health-view", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("ClusterRoleBinding", "monitoring-cluster-monitoring-view", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("ClusterRoleBinding", "monitoring-system-auth-delegator", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("RoleBinding", "monitoring-alertmanager-view-rolebinding", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("Deployment", "health-analyzer", "spec.template.spec.serviceAccountName"),
	)

	// Compound name changes
	o.ReplaceValue(pluginName+"-components-health-view",
		overlay.TargetName("monitoring-components-health-view", "metadata.name"),
	)
	o.ReplaceValue(pluginName+"-cluster-monitoring-view",
		overlay.TargetName("monitoring-cluster-monitoring-view", "metadata.name"),
	)
	o.ReplaceValue(saName+"-system-auth-delegator",
		overlay.TargetName("monitoring-system-auth-delegator", "metadata.name"),
	)
	o.ReplaceValue(pluginName+"-alertmanager-view-rolebinding",
		overlay.TargetKindName("RoleBinding", "monitoring-alertmanager-view-rolebinding", "metadata.name"),
	)
}

// addUIPluginReplacements registers kustomize replacements for UIPlugin names.
func addUIPluginReplacements(o *overlay.Overlay, configName, pluginName, crName string, hasConfigMap bool) {
	deployFieldPaths := []string{
		"metadata.name",
		`metadata.labels.app\.kubernetes\.io/instance`,
		`spec.selector.matchLabels.app\.kubernetes\.io/instance`,
		"spec.template.metadata.name",
		`spec.template.metadata.labels.app\.kubernetes\.io/instance`,
		"spec.template.spec.containers.0.name",
		"spec.template.spec.volumes.[name=serving-cert].secret.secretName",
	}
	if hasConfigMap {
		deployFieldPaths = append(deployFieldPaths, "spec.template.spec.volumes.[name=plugin-config].configMap.name")
	}

	// Cross-reference replacements run before metadata.name changes, since
	// changing a resource's name prevents subsequent replacements from matching
	// by the original name.

	saName := pluginName + serviceAccountSuffix
	o.ReplaceValue(saName,
		overlay.TargetKindName("ServiceAccount", configName, "metadata.name"),
		overlay.TargetKindName("Deployment", configName, "spec.template.spec.serviceAccountName"),
		overlay.TargetKindName("ClusterRoleBinding", configName, "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("ClusterRoleBinding", configName+"-cluster-monitoring-view", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("ClusterRoleBinding", configName+"-korrel8r", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("RoleBinding", configName+"-rolebinding", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("RoleBinding", "monitoring-alertmanager-view-rolebinding", "subjects.[kind=ServiceAccount].name"),
		overlay.TargetKindName("Deployment", "korrel8r", "spec.template.spec.serviceAccountName"),
	)

	o.ReplaceValue(pluginName+clusterRoleSuffix,
		overlay.TargetKindName("ClusterRole", configName, "metadata.name"),
		overlay.TargetKindName("ClusterRoleBinding", configName, "roleRef.name"),
	)

	// Plugin name replacement for Deployment, Service, ConfigMap.
	o.ReplaceValue(pluginName,
		overlay.TargetKindName("Deployment", configName, deployFieldPaths...),
		overlay.TargetKindName("Service", configName,
			"metadata.name",
			`metadata.labels.app\.kubernetes\.io/instance`,
			`metadata.annotations.service\.alpha\.openshift\.io/serving-cert-secret-name`,
			`spec.selector.app\.kubernetes\.io/instance`,
		),
		overlay.TargetKindName("ConfigMap", configName, "metadata.name"),
	)

	// ClusterRoleBinding: "-crb" suffix (name change, must be after cross-refs).
	o.ReplaceValue(pluginName+clusterRoleBindSuffix,
		overlay.TargetKindName("ClusterRoleBinding", configName, "metadata.name"),
	)

	// Datasource reader role/rolebinding (must run before rolebinding name changes)
	o.ReplaceValue(crName+"-datasource-reader",
		overlay.TargetName(configName+"-datasource-reader", "metadata.name"),
		overlay.TargetKindName("RoleBinding", configName+"-rolebinding", "roleRef.name"),
	)

	// Compound name changes (change metadata.name, must be last)
	o.ReplaceValue(pluginName+"-cluster-monitoring-view",
		overlay.TargetName(configName+"-cluster-monitoring-view", "metadata.name"),
	)
	o.ReplaceValue(pluginName+"-korrel8r",
		overlay.TargetName(configName+"-korrel8r", "metadata.name"),
	)
	o.ReplaceValue(pluginName+"-rolebinding",
		overlay.TargetName(configName+"-rolebinding", "metadata.name"),
	)
}

// generateKorrel8rConfig renders the korrel8r configuration from the Go
// template in pkg/controllers/uiplugin/config/korrel8r.yaml, substituting the
// per-cluster Loki/Tempo service names. The template is dynamic (service names vary per cluster),
// so it cannot live as a static manifest under config/.
func generateKorrel8rConfig(info UIPluginInfo) (string, error) {
	korrel8rData := map[string]string{
		"Metric":        "thanos-querier",
		"MetricAlert":   "alertmanager-main",
		"Log":           "logging-loki-gateway-http",
		"Netflow":       "loki-gateway-http",
		"Trace":         defaultTempoService,
		"MonitoringNs":  reconciler.OpenshiftMonitoringNamespace,
		"LoggingNs":     OpenshiftLoggingNs,
		"NetobservNs":   OpenshiftNetobservNs,
		"TracingNs":     OpenshiftTracingNs,
		"TracingTenant": info.TracingTenant,
	}

	if info.LokiServiceNames[OpenshiftLoggingNs] != "" {
		korrel8rData["Log"] = info.LokiServiceNames[OpenshiftLoggingNs]
	}
	if info.LokiServiceNames[OpenshiftNetobservNs] != "" {
		korrel8rData["Netflow"] = info.LokiServiceNames[OpenshiftNetobservNs]
	}
	for ns, svc := range info.TempoServiceNames {
		if svc != "" {
			korrel8rData["Trace"] = svc
			korrel8rData["TracingNs"] = ns
			break
		}
	}

	var buf bytes.Buffer
	if err := korrel8rConfigTmpl.Execute(&buf, korrel8rData); err != nil {
		return "", err
	}
	return buf.String(), nil
}

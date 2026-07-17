package generator

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/rhobs/observability-operator/config"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability"
	"github.com/rhobs/observability-operator/pkg/controllers/uiplugin"
	"github.com/rhobs/observability-operator/pkg/images"
)

type resourceKey struct {
	Kind      string
	Namespace string
	Name      string
}

func (k resourceKey) String() string {
	if k.Namespace != "" {
		return fmt.Sprintf("%s/%s/%s", k.Kind, k.Namespace, k.Name)
	}
	return fmt.Sprintf("%s/%s", k.Kind, k.Name)
}

func buildUIPluginResources(t *testing.T, plugin *uiv1alpha1.UIPlugin, conf uiplugin.UIPluginBuildConfig) map[resourceKey]*unstructured.Unstructured {
	t.Helper()
	o, _, err := uiplugin.BuildUIPluginOverlay(plugin, conf, logr.Discard())
	if err != nil {
		t.Fatalf("BuildUIPluginOverlay: %v", err)
	}
	objects, err := o.Build()
	if err != nil {
		t.Fatalf("overlay.Build: %v", err)
	}
	m := make(map[resourceKey]*unstructured.Unstructured, len(objects))
	for _, obj := range objects {
		u := obj.(*unstructured.Unstructured)
		key := resourceKey{Kind: u.GetKind(), Namespace: u.GetNamespace(), Name: u.GetName()}
		if _, exists := m[key]; exists {
			t.Errorf("duplicate resource: %s", key)
		}
		m[key] = u
	}
	return m
}

func compareResourceSets(t *testing.T, got map[resourceKey]*unstructured.Unstructured, want []resourceKey) {
	t.Helper()
	wantSet := make(map[resourceKey]bool, len(want))
	for _, k := range want {
		wantSet[k] = true
	}
	for k := range got {
		if !wantSet[k] {
			t.Errorf("extra resource: %s", k)
		}
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing resource: %s", k)
		}
	}
}

func getNestedString(t *testing.T, obj *unstructured.Unstructured, fields ...string) string {
	t.Helper()
	val, found, err := unstructured.NestedString(obj.Object, fields...)
	if err != nil || !found {
		t.Errorf("%s %s: field %v not found", obj.GetKind(), obj.GetName(), fields)
		return ""
	}
	return val
}

func getContainerField(t *testing.T, obj *unstructured.Unstructured, index int, field string) string {
	t.Helper()
	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found || index >= len(containers) {
		t.Errorf("%s %s: containers[%d] not found", obj.GetKind(), obj.GetName(), index)
		return ""
	}
	c, ok := containers[index].(map[string]any)
	if !ok {
		t.Errorf("%s %s: containers[%d] not a map", obj.GetKind(), obj.GetName(), index)
		return ""
	}
	val, _ := c[field].(string)
	return val
}

// TestUIPluginOverlay verifies that BuildUIPluginOverlay produces the correct set of
// resources for each plugin type, with correct names after placeholder substitution.
func TestUIPluginOverlay(t *testing.T) {
	ns := "test-operator-ns"
	conf := uiplugin.UIPluginBuildConfig{
		ConfigFS:       config.FS,
		Images:         images.DefaultImages,
		Namespace:      ns,
		ClusterVersion: "4.22",
	}

	tests := []struct {
		name     string
		plugin   *uiv1alpha1.UIPlugin
		conf     uiplugin.UIPluginBuildConfig
		wantKeys []resourceKey
		checks   func(t *testing.T, resources map[resourceKey]*unstructured.Unstructured)
	}{
		{
			name: "dashboards",
			plugin: &uiv1alpha1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: "dashboards"},
				Spec:       uiv1alpha1.UIPluginSpec{Type: uiv1alpha1.TypeDashboards},
			},
			wantKeys: []resourceKey{
				{Kind: "ServiceAccount", Namespace: ns, Name: "observability-ui-dashboards-sa"},
				{Kind: "Deployment", Namespace: ns, Name: "observability-ui-dashboards"},
				{Kind: "Service", Namespace: ns, Name: "observability-ui-dashboards"},
				{Kind: "ConsolePlugin", Namespace: ns, Name: "console-dashboards-plugin"},
				{Kind: "Role", Namespace: "openshift-config-managed", Name: "dashboards-datasource-reader"},
				{Kind: "RoleBinding", Namespace: "openshift-config-managed", Name: "observability-ui-dashboards-rolebinding"},
			},
			checks: func(t *testing.T, m map[resourceKey]*unstructured.Unstructured) {
				deploy := m[resourceKey{Kind: "Deployment", Namespace: ns, Name: "observability-ui-dashboards"}]
				if deploy == nil {
					return
				}
				if image := getContainerField(t, deploy, 0, "image"); image != images.DefaultImages["ui-dashboards"] {
					t.Errorf("deployment image = %q, want %q", image, images.DefaultImages["ui-dashboards"])
				}
				if sa := getNestedString(t, deploy, "spec", "template", "spec", "serviceAccountName"); sa != "observability-ui-dashboards-sa" {
					t.Errorf("serviceAccountName = %q, want %q", sa, "observability-ui-dashboards-sa")
				}
			},
		},
		{
			name: "distributed-tracing",
			plugin: &uiv1alpha1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: "distributed-tracing"},
				Spec:       uiv1alpha1.UIPluginSpec{Type: uiv1alpha1.TypeDistributedTracing},
			},
			wantKeys: []resourceKey{
				{Kind: "ServiceAccount", Namespace: ns, Name: "distributed-tracing-sa"},
				{Kind: "Deployment", Namespace: ns, Name: "distributed-tracing"},
				{Kind: "Service", Namespace: ns, Name: "distributed-tracing"},
				{Kind: "ConsolePlugin", Namespace: ns, Name: "distributed-tracing-console-plugin"},
				{Kind: "ConfigMap", Namespace: ns, Name: "distributed-tracing"},
				{Kind: "ClusterRole", Name: "distributed-tracing-cr"},
				{Kind: "ClusterRoleBinding", Name: "distributed-tracing-crb"},
			},
			checks: func(t *testing.T, m map[resourceKey]*unstructured.Unstructured) {
				deploy := m[resourceKey{Kind: "Deployment", Namespace: ns, Name: "distributed-tracing"}]
				if deploy == nil {
					return
				}
				if image := getContainerField(t, deploy, 0, "image"); image != images.DefaultImages["ui-distributed-tracing"] {
					t.Errorf("deployment image = %q, want %q", image, images.DefaultImages["ui-distributed-tracing"])
				}
			},
		},
		{
			name: "logging",
			plugin: &uiv1alpha1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: "logging"},
				Spec: uiv1alpha1.UIPluginSpec{
					Type: uiv1alpha1.TypeLogging,
					Logging: &uiv1alpha1.LoggingConfig{
						LokiStack: &uiv1alpha1.LokiStackReference{Name: "logging-loki"},
					},
				},
			},
			wantKeys: []resourceKey{
				{Kind: "ServiceAccount", Namespace: ns, Name: "logging-sa"},
				{Kind: "Deployment", Namespace: ns, Name: "logging"},
				{Kind: "Service", Namespace: ns, Name: "logging"},
				{Kind: "ConsolePlugin", Namespace: ns, Name: "logging-view-plugin"},
				{Kind: "ConfigMap", Namespace: ns, Name: "logging"},
				{Kind: "ClusterRole", Name: "cluster-logging-application-view"},
				{Kind: "ClusterRole", Name: "cluster-logging-infrastructure-view"},
				{Kind: "ClusterRole", Name: "cluster-logging-audit-view"},
			},
			checks: func(t *testing.T, m map[resourceKey]*unstructured.Unstructured) {
				deploy := m[resourceKey{Kind: "Deployment", Namespace: ns, Name: "logging"}]
				if deploy == nil {
					return
				}
				if image := getContainerField(t, deploy, 0, "image"); image != images.DefaultImages["ui-logging"] {
					t.Errorf("deployment image = %q, want %q", image, images.DefaultImages["ui-logging"])
				}
			},
		},
		{
			name: "troubleshooting-panel with korrel8r",
			plugin: &uiv1alpha1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: "troubleshooting-panel"},
				Spec:       uiv1alpha1.UIPluginSpec{Type: uiv1alpha1.TypeTroubleshootingPanel},
			},
			wantKeys: []resourceKey{
				{Kind: "ServiceAccount", Namespace: ns, Name: "troubleshooting-panel-sa"},
				{Kind: "Deployment", Namespace: ns, Name: "troubleshooting-panel"},
				{Kind: "Service", Namespace: ns, Name: "troubleshooting-panel"},
				{Kind: "ConsolePlugin", Namespace: ns, Name: "troubleshooting-panel-console-plugin"},
				{Kind: "ConfigMap", Namespace: ns, Name: "troubleshooting-panel"},
				{Kind: "RoleBinding", Namespace: "openshift-monitoring", Name: "monitoring-alertmanager-view-rolebinding"},
				{Kind: "ClusterRoleBinding", Name: "troubleshooting-panel-cluster-monitoring-view"},
				// korrel8r resources (default images include korrel8r)
				{Kind: "Deployment", Namespace: ns, Name: "korrel8r"},
				{Kind: "Service", Namespace: ns, Name: "korrel8r"},
				{Kind: "ConfigMap", Namespace: ns, Name: "korrel8r"},
				{Kind: "ClusterRole", Name: "korrel8r-view"},
				{Kind: "ClusterRoleBinding", Name: "troubleshooting-panel-korrel8r"},
			},
			checks: func(t *testing.T, m map[resourceKey]*unstructured.Unstructured) {
				deploy := m[resourceKey{Kind: "Deployment", Namespace: ns, Name: "korrel8r"}]
				if deploy == nil {
					return
				}
				if image := getContainerField(t, deploy, 0, "image"); image != images.DefaultImages["korrel8r"] {
					t.Errorf("korrel8r image = %q, want %q", image, images.DefaultImages["korrel8r"])
				}
				if sa := getNestedString(t, deploy, "spec", "template", "spec", "serviceAccountName"); sa != "troubleshooting-panel-sa" {
					t.Errorf("korrel8r serviceAccountName = %q, want %q", sa, "troubleshooting-panel-sa")
				}
			},
		},
		{
			name: "troubleshooting-panel without korrel8r",
			plugin: &uiv1alpha1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: "troubleshooting-panel"},
				Spec:       uiv1alpha1.UIPluginSpec{Type: uiv1alpha1.TypeTroubleshootingPanel},
			},
			conf: func() uiplugin.UIPluginBuildConfig {
				c := conf
				noKorrel8r := make(map[string]string)
				for k, v := range images.DefaultImages {
					if k != "korrel8r" {
						noKorrel8r[k] = v
					}
				}
				c.Images = noKorrel8r
				return c
			}(),
			wantKeys: []resourceKey{
				{Kind: "ServiceAccount", Namespace: ns, Name: "troubleshooting-panel-sa"},
				{Kind: "Deployment", Namespace: ns, Name: "troubleshooting-panel"},
				{Kind: "Service", Namespace: ns, Name: "troubleshooting-panel"},
				{Kind: "ConsolePlugin", Namespace: ns, Name: "troubleshooting-panel-console-plugin"},
				{Kind: "ConfigMap", Namespace: ns, Name: "troubleshooting-panel"},
				{Kind: "RoleBinding", Namespace: "openshift-monitoring", Name: "monitoring-alertmanager-view-rolebinding"},
				{Kind: "ClusterRoleBinding", Name: "troubleshooting-panel-cluster-monitoring-view"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := conf
			if tc.conf.Images != nil {
				c = tc.conf
			}
			resources := buildUIPluginResources(t, tc.plugin, c)
			compareResourceSets(t, resources, tc.wantKeys)
			if tc.checks != nil {
				tc.checks(t, resources)
			}
		})
	}
}

// TestEndToEndTracingGeneratesUIPlugin verifies the full pipeline:
// ObservabilityInstaller with tracing → observability overlay → resolveUIPlugins → final resources.
func TestEndToEndTracingGeneratesUIPlugin(t *testing.T) {
	input := `
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: test-installer
  namespace: test-ns
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
`
	scheme := runtime.NewScheme()
	_ = obsv1alpha1.AddToScheme(scheme)
	_ = uiv1alpha1.AddToScheme(scheme)

	installer, plugins, _, err := decodeResources(scheme, []byte(input))
	if err != nil {
		t.Fatalf("decodeResources: %v", err)
	}

	overlayCfg := observability.OverlayConfig{ConfigFS: config.FS, COONamespace: installer.Namespace}
	o, err := observability.BuildOverlay(installer, overlayCfg)
	if err != nil {
		t.Fatalf("BuildOverlay: %v", err)
	}

	pluginConf := uiplugin.UIPluginBuildConfig{
		ConfigFS:       config.FS,
		Images:         images.DefaultImages,
		Namespace:      installer.Namespace,
		ClusterVersion: "4.22",
	}
	resolveUIPlugins(o, plugins, pluginConf)

	yamlOut, err := o.BuildYAML()
	if err != nil {
		t.Fatalf("BuildYAML: %v", err)
	}

	resources, err := parseYAMLResources(yamlOut)
	if err != nil {
		t.Fatalf("parseYAMLResources: %v", err)
	}

	resourceMap := make(map[resourceKey]bool, len(resources))
	kindCounts := make(map[string]int)
	for _, r := range resources {
		key := resourceKey{Kind: r.GetKind(), Namespace: r.GetNamespace(), Name: r.GetName()}
		resourceMap[key] = true
		kindCounts[r.GetKind()]++
	}

	// Observability resources
	wantObservability := []resourceKey{
		{Kind: "TempoStack", Namespace: installer.Namespace, Name: "test-installer"},
		{Kind: "OpenTelemetryCollector", Namespace: installer.Namespace, Name: "test-installer"},
	}
	for _, k := range wantObservability {
		if !resourceMap[k] {
			t.Errorf("missing observability resource: %s", k)
		}
	}

	// UIPlugin is cluster-scoped but kustomize namespace filter may set namespace
	foundUIPlugin := false
	for k := range resourceMap {
		if k.Kind == "UIPlugin" && k.Name == "distributed-tracing" {
			foundUIPlugin = true
			break
		}
	}
	if !foundUIPlugin {
		t.Error("missing observability resource: UIPlugin distributed-tracing")
	}

	// The UIPlugin should have been resolved into concrete resources
	wantUIPlugin := []resourceKey{
		{Kind: "ConsolePlugin", Namespace: installer.Namespace, Name: "distributed-tracing-console-plugin"},
	}
	for _, k := range wantUIPlugin {
		if !resourceMap[k] {
			t.Errorf("missing resolved UIPlugin resource: %s", k)
		}
	}

	// No extra kinds should appear
	expectedKinds := map[string]bool{
		"ClusterRole": true, "ClusterRoleBinding": true, "ConfigMap": true,
		"ConsolePlugin": true, "Deployment": true,
		"OpenTelemetryCollector": true, "Service": true, "ServiceAccount": true,
		"Subscription": true, "TempoStack": true, "UIPlugin": true,
	}
	for kind, count := range kindCounts {
		if !expectedKinds[kind] {
			t.Errorf("unexpected kind %s: %d resources", kind, count)
		}
	}

	checkNamespaceConsistency(t, resources, installer.Namespace)
}

// TestNewGeneratorMatchesOldBehavior verifies that the new kustomize-based generator
// produces the same output as the old controller-based approach would have.
func TestNewGeneratorMatchesOldBehavior(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		wantKinds       map[string]int
		checkNamespaces bool
	}{
		{
			name: "tracing enabled",
			input: `
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: test-installer
  namespace: test-namespace
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
`,
			wantKinds: map[string]int{
				"Subscription":           2,
				"OpenTelemetryCollector": 1,
				"TempoStack":             1,
				"ClusterRole":            2,
				"ClusterRoleBinding":     2,
				"UIPlugin":               1,
			},
			checkNamespaces: true,
		},
		{
			name: "operators only",
			input: `
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: test-installer
  namespace: test-namespace
spec:
  capabilities:
    tracing:
      operators:
        install: true
`,
			wantKinds: map[string]int{
				"Subscription": 2,
			},
			checkNamespaces: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = obsv1alpha1.AddToScheme(scheme)
			_ = uiv1alpha1.AddToScheme(scheme)
			installer, _, _, err := decodeResources(scheme, []byte(tc.input))
			if err != nil {
				t.Fatalf("failed to decode input: %v", err)
			}

			cfg := observability.OverlayConfig{ConfigFS: config.FS, COONamespace: installer.Namespace}
			overlay, err := observability.BuildOverlay(installer, cfg)
			if err != nil {
				t.Fatalf("failed to build overlay: %v", err)
			}

			yamlOut, err := overlay.BuildYAML()
			if err != nil {
				t.Fatalf("failed to build YAML: %v", err)
			}

			resources, err := parseYAMLResources(yamlOut)
			if err != nil {
				t.Fatalf("failed to parse generated YAML: %v", err)
			}

			kindCounts := make(map[string]int)
			for _, r := range resources {
				kindCounts[r.GetKind()]++
			}

			for kind, wantCount := range tc.wantKinds {
				if gotCount := kindCounts[kind]; gotCount != wantCount {
					t.Errorf("kind %s: got %d resources, want %d", kind, gotCount, wantCount)
				}
			}

			for kind, count := range kindCounts {
				if _, expected := tc.wantKinds[kind]; !expected {
					t.Errorf("unexpected kind %s: %d resources", kind, count)
				}
			}

			if tc.checkNamespaces {
				checkNamespaceConsistency(t, resources, installer.Namespace)
			}
		})
	}
}

// TestSubscriptionNamespaces verifies that Subscription resources are created in the correct namespace
func TestSubscriptionNamespaces(t *testing.T) {
	for _, ns := range []string{"test-ns-1", "test-ns-2", "openshift-cluster-observability-operator"} {
		t.Run(ns, func(t *testing.T) {
			input := fmt.Sprintf(`
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: test
  namespace: %s
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
`, ns)

			scheme := runtime.NewScheme()
			_ = obsv1alpha1.AddToScheme(scheme)
			_ = uiv1alpha1.AddToScheme(scheme)
			installer, _, _, err := decodeResources(scheme, []byte(input))
			if err != nil {
				t.Fatalf("failed to decode input: %v", err)
			}

			cfg := observability.OverlayConfig{ConfigFS: config.FS, COONamespace: installer.Namespace}
			overlay, err := observability.BuildOverlay(installer, cfg)
			if err != nil {
				t.Fatalf("failed to build overlay: %v", err)
			}

			yamlOut, err := overlay.BuildYAML()
			if err != nil {
				t.Fatalf("failed to build YAML: %v", err)
			}

			resources, err := parseYAMLResources(yamlOut)
			if err != nil {
				t.Fatalf("failed to parse YAML: %v", err)
			}

			wantNs := map[string]string{
				"opentelemetry-product": "openshift-opentelemetry-operator",
				"tempo-product":        "openshift-tempo-operator",
			}
			for _, r := range resources {
				if r.GetKind() == "Subscription" {
					expected, ok := wantNs[r.GetName()]
					if !ok {
						t.Errorf("unexpected Subscription %s", r.GetName())
						continue
					}
					if r.GetNamespace() != expected {
						t.Errorf("Subscription %s has namespace %q, want %q",
							r.GetName(), r.GetNamespace(), expected)
					}
				}
			}
		})
	}
}

func parseYAMLResources(data []byte) ([]unstructured.Unstructured, error) {
	var resources []unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	for {
		var obj unstructured.Unstructured
		err := decoder.Decode(&obj)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}
		if obj.GetKind() != "" {
			resources = append(resources, obj)
		}
	}
	return resources, nil
}

func checkNamespaceConsistency(t *testing.T, resources []unstructured.Unstructured, expectedNs string) {
	t.Helper()
	for _, r := range resources {
		kind := r.GetKind()
		if isClusterScoped(kind) || kind == "Subscription" {
			if kind == "ClusterRoleBinding" {
				subjects, found, err := unstructured.NestedSlice(r.Object, "subjects")
				if err != nil || !found {
					continue
				}
				for _, subj := range subjects {
					subjMap, ok := subj.(map[string]any)
					if !ok {
						continue
					}
					subjKind, _, _ := unstructured.NestedString(subjMap, "kind")
					if subjKind == "ServiceAccount" {
						subjNs, found, _ := unstructured.NestedString(subjMap, "namespace")
						if found && subjNs != expectedNs {
							t.Errorf("ClusterRoleBinding %s: SA subject namespace = %q, want %q", r.GetName(), subjNs, expectedNs)
						} else if !found {
							t.Errorf("ClusterRoleBinding %s: SA subject missing namespace", r.GetName())
						}
					}
				}
			}
			continue
		}
		if ns := r.GetNamespace(); ns != expectedNs {
			t.Errorf("%s %s: namespace = %q, want %q", kind, r.GetName(), ns, expectedNs)
		}
	}
}

func isClusterScoped(kind string) bool {
	switch kind {
	case "ClusterRole", "ClusterRoleBinding", "ConsolePlugin", "CustomResourceDefinition", "Namespace", "UIPlugin":
		return true
	}
	return false
}

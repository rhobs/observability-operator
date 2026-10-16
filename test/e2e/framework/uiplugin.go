package framework

import (
	"context"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func (f *Framework) DebugUIPlugin(pluginName string) DebugFunc {
	return func(t *testing.T) {
		t.Helper()
		ctx := context.WithoutCancel(t.Context())

		t.Logf("## UIPlugin %s", pluginName)
		var plugin uiv1.UIPlugin
		if err := f.K8sClient.Get(ctx, client.ObjectKey{Name: pluginName}, &plugin); err != nil {
			t.Logf("Failed to get UIPlugin %q: %v", pluginName, err)
			return
		}
		t.Logf("* UIPlugin %q generation=%d, resourceVersion=%s", pluginName, plugin.Generation, plugin.ResourceVersion)
		t.Logf("  * UIPlugin spec.type=%s", plugin.Spec.Type)
		if plugin.Spec.Monitoring != nil {
			if plugin.Spec.Monitoring.ClusterHealthAnalyzer != nil {
				t.Logf("  * UIPlugin spec.monitoring.clusterHealthAnalyzer.enabled=%v", plugin.Spec.Monitoring.ClusterHealthAnalyzer.Enabled)
			}
			if plugin.Spec.Monitoring.Incidents != nil {
				t.Logf("  * UIPlugin spec.monitoring.incidents.enabled=%v", plugin.Spec.Monitoring.Incidents.Enabled)
			}
		}
		t.Log("* UIPlugin conditions")
		if len(plugin.Status.Conditions) == 0 {
			t.Log("  * No status conditions")
		}
		for _, c := range plugin.Status.Conditions {
			t.Logf("  * Condition: type=%s status=%s reason=%s message=%s", c.Type, c.Status, c.Reason, c.Message)
		}

		t.Logf("## UIPlugins summary")

		var plugins uiv1.UIPluginList
		if err := f.K8sClient.List(ctx, &plugins); err != nil {
			t.Logf("Failed to list UIPlugins: %v", err)
		} else {
			t.Logf("* Total number of UIPlugins: %d", len(plugins.Items))
			for _, p := range plugins.Items {
				t.Logf("  * UIPlugin: name=%s type=%s conditions=%d", p.Name, p.Spec.Type, len(p.Status.Conditions))
			}
		}
	}
}

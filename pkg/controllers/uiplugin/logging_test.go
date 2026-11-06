package uiplugin

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func TestLoggingPluginFeatures(t *testing.T) {
	compatibility, err := lookupImageAndFeatures(uiv1alpha1.TypeLogging, "4.22.0")
	assert.NilError(t, err)

	plugin := &uiv1alpha1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "logging"},
		Spec:       uiv1alpha1.UIPluginSpec{Type: uiv1alpha1.TypeLogging},
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		lokiStackResource: "LokiStackList",
	})
	info, err := PluginInfoBuilder(context.Background(), nil, dynamicClient, plugin, UIPluginsConfiguration{
		Images: map[string]string{"ui-logging": "logging-image"},
	}, compatibility, logr.Discard())
	assert.NilError(t, err)
	assert.DeepEqual(t, info.ExtraArgs, []string{
		"-plugin-config-path=/etc/plugin/config/config.yaml",
		"-features=dev-console,alerts,dev-alerts",
	})
}

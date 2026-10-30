package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func TestUIPluginReferencesLokiStack(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "sample"},
	}

	plugin := uiPlugin(instance)
	require.Equal(t, uiv1alpha1.LoggingPluginName, plugin.Name)
	require.Equal(t, uiv1alpha1.TypeLogging, plugin.Spec.Type)
	require.NotNil(t, plugin.Spec.Logging)
	require.NotNil(t, plugin.Spec.Logging.LokiStack)
	require.Equal(t, lokiStackName(instance.Name), plugin.Spec.Logging.LokiStack.Name)
	require.Equal(t, "sample", plugin.Spec.Logging.LokiStack.Namespace)
}

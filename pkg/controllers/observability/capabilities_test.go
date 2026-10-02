package observability

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func TestSharedCapabilityObjects(t *testing.T) {
	logging := obsv1alpha1.ObservabilityInstaller{Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
		Logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
	}}}
	disabled := obsv1alpha1.ObservabilityInstaller{}

	// Every plugin is always owned, so it is cleaned up once unused.
	plan := aggregateShared(capabilities(testOptions()), []obsv1alpha1.ObservabilityInstaller{logging, disabled})
	require.Len(t, plan.Owned, 2)
	require.Len(t, plan.Desired, 1)
	plugin, ok := plan.Desired[0].(*uiv1alpha1.UIPlugin)
	require.True(t, ok)
	require.Equal(t, uiv1alpha1.LoggingPluginName, plugin.Name)

	// Duplicate requests desire each shared plugin only once.
	logging.Spec.Capabilities.Tracing = &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}}
	plan = aggregateShared(capabilities(testOptions()), []obsv1alpha1.ObservabilityInstaller{logging, logging})
	require.Len(t, plan.Desired, 2)
	require.Len(t, plan.Owned, 2)

	now := metav1.Now()
	logging.DeletionTimestamp = &now
	plan = aggregateShared(capabilities(testOptions()), []obsv1alpha1.ObservabilityInstaller{logging, disabled})
	require.Empty(t, plan.Desired, "a deleting installer must not desire plugins")
	require.Len(t, plan.Owned, 2, "unused plugins are cleaned up")
}

// Two installers enabling the same capability share one cluster-scoped plugin.
// The first installer in list order supplies its configuration, so all
// installers reconcile the same plugin and cannot overwrite each other.
func TestSharedPluginConfigIsStableAcrossInstallers(t *testing.T) {
	installer := func(name, namespace string) obsv1alpha1.ObservabilityInstaller {
		return obsv1alpha1.ObservabilityInstaller{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
				Logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
			}},
		}
	}
	first, second := installer("first", "ns-a"), installer("second", "ns-b")

	for _, instances := range [][]obsv1alpha1.ObservabilityInstaller{{first, second}, {second, first}} {
		plan := aggregateShared(capabilities(testOptions()), instances)
		require.Len(t, plan.Desired, 1)
		plugin, ok := plan.Desired[0].(*uiv1alpha1.UIPlugin)
		require.True(t, ok)
		require.Equal(t, instances[0].Name+"-loki", plugin.Spec.Logging.LokiStack.Name)
		require.Equal(t, instances[0].Namespace, plugin.Spec.Logging.LokiStack.Namespace)
	}
}

func TestOperatorRequiredAcrossInstallers(t *testing.T) {
	enabled := obsv1alpha1.ObservabilityInstaller{Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
		Logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
	}}}
	disabled := obsv1alpha1.ObservabilityInstaller{}
	instances := []obsv1alpha1.ObservabilityInstaller{enabled, disabled}

	plan := aggregateShared(capabilities(testOptions()), instances)
	require.Len(t, plan.Operators, 4)
	for _, operator := range plan.Operators {
		require.Equal(t, operator.Name == "loki" || operator.Name == "cluster-logging", operator.Desired, operator.Name)
	}
}

func testOptions() Options {
	return Options{
		OpenTelemetryOperator:  OperatorInstallConfig{PackageName: "opentelemetry-product"},
		TempoOperator:          OperatorInstallConfig{PackageName: "tempo-product"},
		LokiOperator:           OperatorInstallConfig{PackageName: "loki-operator"},
		ClusterLoggingOperator: OperatorInstallConfig{PackageName: "cluster-logging"},
	}
}

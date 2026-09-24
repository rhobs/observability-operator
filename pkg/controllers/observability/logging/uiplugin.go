package logging

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

// uiPlugin returns the cluster-scoped logging console plugin. The plugin is
// shared: aggregateShared picks the LokiStack of the first installer with
// logging enabled, and the plugin is deleted once no installer enables it.
func uiPlugin(instance *obsv1alpha1.ObservabilityInstaller) *uiv1alpha1.UIPlugin {
	return &uiv1alpha1.UIPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UIPlugin",
			APIVersion: uiv1alpha1.GroupVersion.String(),
		},
		// Labelled as part of the operator, not of the applying installer, see
		// tracing.newUIPlugin.
		ObjectMeta: metav1.ObjectMeta{Name: uiv1alpha1.LoggingPluginName, Labels: map[string]string{util.PartOfLabel: util.OpName}},
		Spec: uiv1alpha1.UIPluginSpec{
			Type: uiv1alpha1.TypeLogging,
			Logging: &uiv1alpha1.LoggingConfig{
				LokiStack: &uiv1alpha1.LokiStackReference{
					Name:      lokiStackName(instance.Name),
					Namespace: instance.Namespace,
				},
			},
		},
	}
}

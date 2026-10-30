package tracing

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

func uiPlugin() *uiv1alpha1.UIPlugin {
	return newUIPlugin(uiv1alpha1.DistributedTracingPluginName, uiv1alpha1.TypeDistributedTracing)
}
func newUIPlugin(name string, pluginType uiv1alpha1.UIPluginType) *uiv1alpha1.UIPlugin {
	return &uiv1alpha1.UIPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UIPlugin",
			APIVersion: uiv1alpha1.GroupVersion.String(),
		},
		// The plugin is shared, so it is labelled as part of the operator rather
		// than of the installer that happens to apply it. Otherwise installers
		// would overwrite each other's labels on every reconcile.
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{util.PartOfLabel: util.OpName}},
		Spec:       uiv1alpha1.UIPluginSpec{Type: pluginType},
	}
}

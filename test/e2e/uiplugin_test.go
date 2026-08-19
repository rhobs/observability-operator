package e2e

import (
	"context"
	"flag"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

var uiPluginInstallNS string

func TestUIPlugin(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("The tests are skipped on non-ocp cluster")
	}

	flag.Parse()
	uiPluginInstallNS = *operatorInstallNS

	assertCRDExists(t, "uiplugins.observability.openshift.io")

	ts := []testCase{
		{
			name:     "Cluster health analyzer",
			scenario: clusterHealthAnalyzer,
		},
		{
			name:     "Create troubleshooting-panel UIPlugin",
			scenario: troubleshootingPanelUIPlugin,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func troubleshootingPanelUIPlugin(t *testing.T) {
	f.DumpOnFailure(t, f.DebugNamespaces(uiPluginInstallNS))

	tp := newTroubleshootingPanelUIPlugin(t)
	err := f.K8sClient.Create(t.Context(), tp)
	assert.NilError(t, err, "failed to create a troubleshooting-panel UIPlugin")

	tpDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, uiv1.TroubleshootingPanelPluginName, uiPluginInstallNS, &tpDeployment)
	f.AssertDeploymentReady(uiv1.TroubleshootingPanelPluginName, uiPluginInstallNS, framework.WithTimeout(5*time.Minute))(t)

	korrel8rDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, "korrel8r", uiPluginInstallNS, &korrel8rDeployment)
	f.AssertDeploymentReady("korrel8r", uiPluginInstallNS, framework.WithTimeout(5*time.Minute))(t)
}

func newTroubleshootingPanelUIPlugin(t *testing.T) *uiv1.UIPlugin {
	plugin := &uiv1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: uiv1.TroubleshootingPanelPluginName,
		},
		Spec: uiv1.UIPluginSpec{
			Type: uiv1.TypeTroubleshootingPanel,
		},
	}

	deleteUIPluginIfExists(t, plugin.Name)

	f.CleanUp(t, func() {
		ctx := context.WithoutCancel(t.Context())
		if err := f.K8sClient.Delete(ctx, plugin); err != nil && !errors.IsNotFound(err) {
			t.Logf("warning: failed to delete troubleshooting-panel UIPlugin during cleanup: %v", err)
		}
		waitForUIPluginDeletion(plugin)
	})
	return plugin
}

func waitForUIPluginDeletion(db *uiv1.UIPlugin) error {
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, wait.ForeverTestTimeout, true, func(ctx context.Context) (done bool, err error) {
		err = f.K8sClient.Get(context.Background(),
			client.ObjectKey{Name: db.Name},
			db)
		return errors.IsNotFound(err), nil
	})
}

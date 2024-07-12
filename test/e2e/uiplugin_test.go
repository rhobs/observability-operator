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

var operatorInstallNS = flag.String("operatorInstallNS", "openshift-operator", "The namespace where the operator is installed")
var uiPluginInstallNS string

func TestUIPlugin(t *testing.T) {
	flag.Parse()
	uiPluginInstallNS = *operatorInstallNS

	assertCRDExists(t, "uiplugins.observability.openshift.io")

	ts := []testCase{
		{
			name:     "Create dashboards UIPlugin",
			scenario: dashboardsUIPlugin,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func dashboardsUIPlugin(t *testing.T) {
	db := newDashboardsUIPlugin(t)
	err := f.K8sClient.Create(context.Background(), db)
	assert.NilError(t, err, "failed to create a dashboards UIPlugin")
	// Check deploy observability-ui-dashboards ius ready
	name := "observability-ui-dashboards"
	dbDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, name, uiPluginInstallNS, &dbDeployment)
	f.AssertDeploymentReady(name, uiPluginInstallNS, framework.WithTimeout(5*time.Minute))(t)
}

func newDashboardsUIPlugin(t *testing.T) *uiv1.UIPlugin {
	db := &uiv1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dashboards",
		},
		Spec: uiv1.UIPluginSpec{
			Type: uiv1.UIPluginType("Dashboards"),
		},
	}
	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), db)
		waitForDBUIPluginDeletion(db)
	})

	return db
}

func waitForDBUIPluginDeletion(db *uiv1.UIPlugin) error {
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, wait.ForeverTestTimeout, true, func(ctx context.Context) (done bool, err error) {
		err = f.K8sClient.Get(context.Background(),
			client.ObjectKey{Name: db.Name},
			db)
		return errors.IsNotFound(err), nil
	})
}

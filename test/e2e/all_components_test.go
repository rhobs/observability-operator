package e2e

import (
	"context"
	"flag"
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

func TestReadOnlyRootFilesystem(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("Skipping: requires OpenShift cluster")
	}

	f.SkipIfClusterVersionBelow(t, "4.22")
	flag.Parse()
	ns := *operatorInstallNS

	f.DumpOnFailure(t, f.DebugNamespaces(ns, e2eTestNamespace))

	plugins := []struct {
		name string
		cr   *uiv1.UIPlugin
	}{
		{
			name: "monitoring",
			cr: &uiv1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: uiv1.MonitoringPluginName},
				Spec: uiv1.UIPluginSpec{
					Type: uiv1.TypeMonitoring,
					Monitoring: &uiv1.MonitoringConfig{
						ClusterHealthAnalyzer: &uiv1.ClusterHealthAnalyzerReference{
							Enabled: true,
						},
						Perses: &uiv1.PersesReference{
							Enabled: true,
						},
					},
				},
			},
		},
		{
			name: "troubleshooting-panel",
			cr: &uiv1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: uiv1.TroubleshootingPanelPluginName},
				Spec:       uiv1.UIPluginSpec{Type: uiv1.TypeTroubleshootingPanel},
			},
		},
		{
			name: "distributed-tracing",
			cr: &uiv1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: uiv1.DistributedTracingPluginName},
				Spec:       uiv1.UIPluginSpec{Type: uiv1.TypeDistributedTracing},
			},
		},
		{
			name: "logging",
			cr: &uiv1.UIPlugin{
				ObjectMeta: metav1.ObjectMeta{Name: uiv1.LoggingPluginName},
				Spec:       uiv1.UIPluginSpec{Type: uiv1.TypeLogging},
			},
		},
	}

	for _, p := range plugins {
		deleteUIPluginIfExists(t, p.cr.Name)
		err := f.K8sClient.Create(t.Context(), p.cr)
		assert.NilError(t, err, "failed to create %s UIPlugin", p.name)
		f.CleanUp(t, func() {
			ctx := context.WithoutCancel(t.Context())
			if err := f.K8sClient.Delete(ctx, p.cr); err != nil && !errors.IsNotFound(err) {
				t.Logf("warning: failed to delete %s UIPlugin during cleanup: %v", p.name, err)
			}
			waitForUIPluginDeletion(p.cr)
		})
	}

	tq, ms := newThanosStackCombo(t, "readonly-rootfs")
	err := f.K8sClient.Create(t.Context(), ms)
	assert.NilError(t, err, "failed to create MonitoringStack")
	_ = f.GetStackWhenAvailable(t, ms.Name, ms.Namespace)

	err = f.K8sClient.Create(t.Context(), tq)
	assert.NilError(t, err, "failed to create ThanosQuerier")

	thanosDeploymentName := "thanos-querier-" + tq.Name

	expectedDeployments := []struct {
		name      string
		namespace string
	}{
		{uiv1.MonitoringPluginName, ns},
		{"health-analyzer", ns},
		{uiv1.TroubleshootingPanelPluginName, ns},
		{"korrel8r", ns},
		{uiv1.DistributedTracingPluginName, ns},
		{uiv1.LoggingPluginName, ns},
		{thanosDeploymentName, e2eTestNamespace},
	}

	for _, ed := range expectedDeployments {
		dep := appsv1.Deployment{}
		f.GetResourceWithRetry(t, ed.name, ed.namespace, &dep)
		f.AssertDeploymentReady(ed.name, ed.namespace, framework.WithTimeout(5*time.Minute))(t)
	}

	t.Log("All expected deployments are ready, checking ReadOnlyRootFilesystem on every deployment and statefulset")

	for _, checkNS := range []string{ns, e2eTestNamespace} {
		deployments := appsv1.DeploymentList{}
		err := f.K8sClient.List(t.Context(), &deployments, client.InNamespace(checkNS))
		assert.NilError(t, err, "failed to list deployments in namespace %s", checkNS)

		t.Logf("Checking ReadOnlyRootFilesystem on %d deployment(s) in namespace %s:", len(deployments.Items), checkNS)
		for i := range deployments.Items {
			dep := &deployments.Items[i]
			t.Logf("  - %s (%d container(s))", dep.Name, len(dep.Spec.Template.Spec.Containers))
			assertDeploymentContainersReadOnlyRootFilesystem(t, *dep)
		}

		statefulsets := appsv1.StatefulSetList{}
		err = f.K8sClient.List(t.Context(), &statefulsets, client.InNamespace(checkNS))
		assert.NilError(t, err, "failed to list statefulsets in namespace %s", checkNS)

		t.Logf("Checking ReadOnlyRootFilesystem on %d statefulset(s) in namespace %s:", len(statefulsets.Items), checkNS)
		for i := range statefulsets.Items {
			sts := &statefulsets.Items[i]
			t.Logf("  - %s (%d container(s))", sts.Name, len(sts.Spec.Template.Spec.Containers))
			assertStatefulSetContainersReadOnlyRootFilesystem(t, *sts)
		}
	}
}

func assertStatefulSetContainersReadOnlyRootFilesystem(t *testing.T, sts appsv1.StatefulSet) {
	t.Helper()
	for _, container := range sts.Spec.Template.Spec.Containers {
		sc := container.SecurityContext
		assert.Assert(t, sc != nil,
			fmt.Sprintf("statefulset %s/%s container %s: SecurityContext is nil", sts.Namespace, sts.Name, container.Name))
		assert.Assert(t, sc.ReadOnlyRootFilesystem != nil && *sc.ReadOnlyRootFilesystem,
			fmt.Sprintf("statefulset %s/%s container %s: ReadOnlyRootFilesystem is not true", sts.Namespace, sts.Name, container.Name))
	}
}

func assertDeploymentContainersReadOnlyRootFilesystem(t *testing.T, dep appsv1.Deployment) {
	t.Helper()
	for _, container := range dep.Spec.Template.Spec.Containers {
		sc := container.SecurityContext
		assert.Assert(t, sc != nil,
			fmt.Sprintf("deployment %s/%s container %s: SecurityContext is nil", dep.Namespace, dep.Name, container.Name))
		assert.Assert(t, sc.ReadOnlyRootFilesystem != nil && *sc.ReadOnlyRootFilesystem,
			fmt.Sprintf("deployment %s/%s container %s: ReadOnlyRootFilesystem is not true", dep.Namespace, dep.Name, container.Name))
	}
}

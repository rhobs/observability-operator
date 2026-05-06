package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

// TestUIPluginUninstallCleanup verifies that UIPlugin operands are properly
// cleaned up when the operator is uninstalled via OLM (CSV + Subscription
// deletion). This reproduces the scenario where a user uninstalls COO from the
// OpenShift console or CLI without manually deleting UIPlugin CRs first.
//
// The test:
//  1. Creates a monitoring UIPlugin with health-analyzer enabled
//  2. Waits for operand deployments to be ready
//  3. Simulates OLM uninstall by deleting the CSV and Subscription
//  4. Verifies that UIPlugin CRs and all child resources are cleaned up
func TestUIPluginUninstallCleanup(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("Skipping: requires OpenShift cluster")
	}

	f.SkipIfClusterVersionBelow(t, "4.19")

	assertCRDExists(t, "uiplugins.observability.openshift.io")

	ctx := context.Background()
	ns := f.OperatorNamespace

	// --- Phase 0: Clean up any leftover UIPlugins from previous runs ---
	// A prior test run may have left UIPlugins stuck in Terminating with
	// finalizers that can't be processed (operator already gone). Force-remove
	// them so we start clean.

	t.Log("Phase 0: Ensuring no stale UIPlugins exist")
	forceDeleteAllUIPlugins(t, ctx)

	// --- Phase 1: Create UIPlugin and verify operands are running ---

	t.Log("Phase 1: Creating monitoring UIPlugin with health-analyzer enabled")
	plugin := &uiv1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: "monitoring",
		},
		Spec: uiv1.UIPluginSpec{
			Type: uiv1.TypeMonitoring,
			Monitoring: &uiv1.MonitoringConfig{
				ClusterHealthAnalyzer: &uiv1.ClusterHealthAnalyzerReference{
					Enabled: true,
				},
			},
		},
	}

	err := f.K8sClient.Create(ctx, plugin)
	assert.NilError(t, err, "failed to create monitoring UIPlugin")

	t.Log("Waiting for monitoring plugin deployment to be ready...")
	f.AssertDeploymentReady("monitoring", ns, framework.WithTimeout(5*time.Minute))(t)

	t.Log("Waiting for health-analyzer deployment to be ready...")
	f.AssertDeploymentReady("health-analyzer", ns, framework.WithTimeout(5*time.Minute))(t)

	// --- Phase 2: Simulate OLM uninstall (delete CSV + Subscription) ---

	t.Log("Phase 2: Simulating OLM uninstall by deleting CSV and Subscription")

	csv, sub := findOLMResources(t, ctx, ns)

	// Register cleanup to reinstall the operator after the test finishes,
	// unless -retain is set (useful for inspecting the post-uninstall state).
	// Use -postpone-restoration=10m to delay restoration for manual inspection.
	if sub != nil && !f.Retain {
		savedSub := &olmv1alpha1.Subscription{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sub.Name,
				Namespace: sub.Namespace,
			},
			Spec: sub.Spec.DeepCopy(),
		}
		t.Cleanup(func() {
			if delay := *postponeRestoration; delay > 0 {
				t.Logf("Cleanup: Waiting %v before restoring operator (inspect the cluster now)", delay)
				time.Sleep(delay)
			}
			t.Log("Cleanup: Reinstalling operator Subscription so the cluster is usable for next run")
			forceDeleteAllUIPlugins(t, context.Background())
			if err := f.K8sClient.Create(context.Background(), savedSub); err != nil {
				if apierrors.IsAlreadyExists(err) {
					t.Log("Cleanup: Subscription already exists, skipping")
					return
				}
				t.Logf("Cleanup: WARNING — failed to recreate Subscription: %v", err)
				t.Log("Cleanup: Reinstall manually with: oc apply -f <subscription.yaml>")
				return
			}
			t.Log("Cleanup: Subscription recreated, OLM will reinstall the operator")
		})
	}

	if sub != nil {
		t.Logf("Deleting Subscription %s/%s", sub.Namespace, sub.Name)
		err = f.K8sClient.Delete(ctx, sub)
		if err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("failed to delete Subscription: %v", err)
		}
	}

	if csv != nil {
		t.Logf("Deleting CSV %s/%s", csv.Namespace, csv.Name)
		err = f.K8sClient.Delete(ctx, csv)
		if err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("failed to delete CSV: %v", err)
		}
	}

	t.Log("Waiting for operator deployment to be removed...")
	waitForResourceAbsent(t, "observability-operator", ns, &appsv1.Deployment{}, 5*time.Minute)

	// --- Phase 3: Verify cleanup ---

	t.Log("Phase 3: Verifying UIPlugin and operand cleanup (parallel assertions follow)")
	t.Log("--- parallel resource checks start ---")

	cleanupTimeout := 3 * time.Minute

	t.Run("UIPlugin CR is deleted", func(t *testing.T) {
		t.Parallel()
		waitForResourceAbsent(t, "monitoring", "", &uiv1.UIPlugin{}, cleanupTimeout)
	})

	t.Run("monitoring plugin deployment is deleted", func(t *testing.T) {
		t.Parallel()
		waitForResourceAbsent(t, "monitoring", ns, &appsv1.Deployment{}, cleanupTimeout)
	})

	t.Run("health-analyzer deployment is deleted", func(t *testing.T) {
		t.Parallel()
		waitForResourceAbsent(t, "health-analyzer", ns, &appsv1.Deployment{}, cleanupTimeout)
	})

	t.Run("health-analyzer service is deleted", func(t *testing.T) {
		t.Parallel()
		waitForResourceAbsent(t, "health-analyzer", ns, &corev1.Service{}, cleanupTimeout)
	})

	t.Run("monitoring plugin service is deleted", func(t *testing.T) {
		t.Parallel()
		waitForResourceAbsent(t, "monitoring", ns, &corev1.Service{}, cleanupTimeout)
	})

	t.Run("monitoring plugin service account is deleted", func(t *testing.T) {
		t.Parallel()
		waitForResourceAbsent(t, "monitoring-sa", ns, &corev1.ServiceAccount{}, cleanupTimeout)
	})

	t.Run("components-health-view ClusterRole is deleted", func(t *testing.T) {
		t.Parallel()
		waitForResourceAbsent(t, "components-health-view", "", &rbacv1.ClusterRole{}, cleanupTimeout)
	})

	t.Run("components-health-view ClusterRoleBinding is deleted", func(t *testing.T) {
		t.Parallel()
		waitForResourceAbsent(t, "monitoring-components-health-view", "", &rbacv1.ClusterRoleBinding{}, cleanupTimeout)
	})

	t.Run("no UIPlugin-managed pods remain in operator namespace", func(t *testing.T) {
		t.Parallel()
		assertNoManagedPodsRemain(t, ctx, ns)
	})

	// Note: parallel subtests complete before this function returns.
	t.Log("--- parallel resource checks done ---")
}

// findOLMResources locates the COO Subscription and CSV in the given namespace.
func findOLMResources(t *testing.T, ctx context.Context, ns string) (*olmv1alpha1.ClusterServiceVersion, *olmv1alpha1.Subscription) {
	t.Helper()

	var foundCSV *olmv1alpha1.ClusterServiceVersion
	var foundSub *olmv1alpha1.Subscription

	subs := &olmv1alpha1.SubscriptionList{}
	err := f.K8sClient.List(ctx, subs, &client.ListOptions{Namespace: ns})
	if err != nil {
		t.Logf("warning: failed to list subscriptions: %v", err)
	} else {
		for i := range subs.Items {
			if subs.Items[i].Spec.Package == "observability-operator" ||
				subs.Items[i].Spec.Package == "cluster-observability-operator" {
				foundSub = &subs.Items[i]
				t.Logf("Found Subscription: %s (package: %s)", foundSub.Name, foundSub.Spec.Package)
				break
			}
		}
	}

	csvs := &olmv1alpha1.ClusterServiceVersionList{}
	err = f.K8sClient.List(ctx, csvs, &client.ListOptions{Namespace: ns})
	if err != nil {
		t.Logf("warning: failed to list CSVs: %v", err)
	} else {
		for i := range csvs.Items {
			if strings.Contains(csvs.Items[i].Name, "observability-operator") {
				foundCSV = &csvs.Items[i]
				t.Logf("Found CSV: %s", foundCSV.Name)
				break
			}
		}
	}

	if foundCSV == nil && foundSub == nil {
		t.Fatal("Could not find COO Subscription or CSV — operator may not be installed via OLM")
	}

	return foundCSV, foundSub
}

// waitForResourceAbsent polls until the named resource no longer exists.
func waitForResourceAbsent(t *testing.T, name, namespace string, obj client.Object, timeout time.Duration) {
	t.Helper()
	key := client.ObjectKey{Name: name, Namespace: namespace}
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		if err := f.K8sClient.Get(ctx, key, obj); apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
	if wait.Interrupted(err) {
		kind := fmt.Sprintf("%T", obj)
		t.Fatalf("%s %s/%s was not cleaned up after operator uninstall (waited %v)", kind, namespace, name, timeout)
	}
}

// forceDeleteAllUIPlugins removes all UIPlugin CRs, stripping finalizers if
// necessary. This handles the case where a previous test left UIPlugins stuck
// in Terminating because the operator was already gone.
func forceDeleteAllUIPlugins(t *testing.T, ctx context.Context) {
	t.Helper()

	var plugins uiv1.UIPluginList
	if err := f.K8sClient.List(ctx, &plugins); err != nil {
		t.Logf("Could not list UIPlugins (CRD may not exist yet): %v", err)
		return
	}

	for i := range plugins.Items {
		p := &plugins.Items[i]

		if len(p.Finalizers) > 0 {
			t.Logf("Stripping finalizers from UIPlugin %s", p.Name)
			patch := client.MergeFrom(p.DeepCopy())
			p.Finalizers = nil
			if err := f.K8sClient.Patch(ctx, p, patch); err != nil && !apierrors.IsNotFound(err) {
				t.Logf("warning: failed to strip finalizers from %s: %v", p.Name, err)
			}
		}

		if p.DeletionTimestamp.IsZero() {
			t.Logf("Deleting UIPlugin %s", p.Name)
			if err := f.K8sClient.Delete(ctx, p); err != nil && !apierrors.IsNotFound(err) {
				t.Logf("warning: failed to delete UIPlugin %s: %v", p.Name, err)
			}
		}
	}

	// Wait for all UIPlugins to be gone
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		var remaining uiv1.UIPluginList
		if err := f.K8sClient.List(ctx, &remaining); err != nil {
			return false, nil
		}
		return len(remaining.Items) == 0, nil
	})
	if wait.Interrupted(err) {
		t.Fatal("Stale UIPlugins still exist after force cleanup")
	}
}

// assertNoManagedPodsRemain verifies that no UIPlugin-managed pods are left
// running in the operator namespace after uninstall.
func assertNoManagedPodsRemain(t *testing.T, ctx context.Context, namespace string) {
	t.Helper()

	managedLabels := map[string]string{
		"app.kubernetes.io/managed-by": "observability-operator",
	}

	var lastSeen []string
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		if err := f.K8sClient.List(ctx, pods,
			client.InNamespace(namespace),
			client.MatchingLabels(managedLabels),
		); err != nil {
			return false, nil
		}

		if len(pods.Items) == 0 {
			return true, nil
		}

		lastSeen = make([]string, 0, len(pods.Items))
		for _, p := range pods.Items {
			lastSeen = append(lastSeen, fmt.Sprintf("%s (phase=%s)", p.Name, p.Status.Phase))
		}
		return false, nil
	})

	if wait.Interrupted(err) {
		t.Fatalf("managed pods not cleaned up after operator uninstall: %v", lastSeen)
	}
}

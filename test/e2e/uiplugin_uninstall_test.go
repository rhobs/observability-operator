package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

// TestUIPluginUninstallCleanup verifies that UIPlugin operands are properly
// cleaned up when an admin deletes the UIPlugin CR after the operator has been
// uninstalled via OLM.
//
// Per OLM design, uninstalling an operator (deleting CSV + Subscription) does
// NOT remove CRDs or CRs — this is intentional to prevent data loss. The admin
// is expected to delete CRs manually (OLM uninstall Step 1). This test verifies
// that when the admin does delete the UIPlugin CR post-uninstall, the child
// resources are properly cascade-deleted via Kubernetes garbage collection
// (OwnerReferences), without requiring the operator to be running.
//
// Before the fix (finalizers): UIPlugin CR gets stuck in Terminating forever
// because the operator is gone and can't remove the finalizer.
// After the fix (no finalizers + OwnerReferences): UIPlugin CR deletes
// immediately and Kubernetes GC cascade-deletes all children.
//
// The test:
//  1. Creates a monitoring UIPlugin with health-analyzer enabled
//  2. Waits for operand deployments to be ready
//  3. Simulates OLM uninstall by deleting the CSV and Subscription
//  4. Deletes the UIPlugin CR (simulating admin Step 1 post-uninstall)
//  5. Verifies that all child resources are cascade-deleted
func TestUIPluginUninstallCleanup(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("Skipping: requires OpenShift cluster")
	}

	f.SkipIfClusterVersionBelow(t, "4.19")

	assertCRDExists(t, "uiplugins.observability.openshift.io")

	ctx := context.Background()
	ns := f.OperatorNamespace

	// --- Phase 0: Clean up any leftover UIPlugins from previous runs ---

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

	if sub != nil && !f.Retain {
		savedSub := &olmv1alpha1.Subscription{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sub.Name,
				Namespace: sub.Namespace,
			},
			Spec: sub.Spec.DeepCopy(),
		}
		savedSub.Spec.InstallPlanApproval = olmv1alpha1.ApprovalAutomatic
		t.Cleanup(func() {
			if delay := *postponeRestoration; delay > 0 {
				t.Logf("Cleanup: Waiting %v before restoring operator (inspect the cluster now)", delay)
				time.Sleep(delay)
			}
			t.Log("Cleanup: Reinstalling operator Subscription so the cluster is usable for next run")
			forceDeleteAllUIPlugins(t, context.Background())
			if err := f.K8sClient.Create(context.Background(), savedSub); err != nil {
				if apierrors.IsAlreadyExists(err) {
					t.Log("Cleanup: Subscription already exists, skipping create")
				} else {
					t.Logf("Cleanup: WARNING — failed to recreate Subscription: %v", err)
					t.Log("Cleanup: Reinstall manually with: oc apply -f <subscription.yaml>")
					return
				}
			} else {
				t.Log("Cleanup: Subscription recreated, OLM will reinstall the operator")
			}

			t.Log("Cleanup: Waiting for CSV to reach Succeeded phase...")
			if err := waitForCSVSucceeded(t, ns, 5*time.Minute); err != nil {
				t.Logf("Cleanup: WARNING — CSV did not reach Succeeded: %v", err)
			} else if f.IsOpenshiftCluster {
				t.Log("Cleanup: Re-enabling OpenShift mode on reinstalled CSV...")
				if err := patchCSVOpenShiftEnabled(t, ns); err != nil {
					t.Logf("Cleanup: WARNING — failed to patch CSV: %v", err)
				}
			}

			t.Log("Cleanup: Waiting for operator deployment to become ready...")
			f.AssertDeploymentReady("observability-operator", ns, framework.WithTimeout(5*time.Minute))(t)
			t.Log("Cleanup: Operator is ready")
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
	assertResourceGone(t, "observability-operator", ns, &appsv1.Deployment{}, 5*time.Minute)

	// --- Phase 3: Delete UIPlugin CR (admin cleanup step) ---
	// Per OLM docs, the admin is responsible for deleting CRs after uninstall.
	// This step simulates that. With the finalizer fix, the CR should delete
	// immediately (no operator needed). Without the fix, this would hang forever.

	t.Log("Phase 3: Deleting UIPlugin CR (simulating admin post-uninstall cleanup)")

	// Re-fetch the UIPlugin to get the latest version.
	currentPlugin := &uiv1.UIPlugin{}
	err = f.K8sClient.Get(ctx, client.ObjectKey{Name: "monitoring"}, currentPlugin)
	assert.NilError(t, err, "UIPlugin should still exist after operator uninstall")

	if len(currentPlugin.Finalizers) > 0 {
		t.Logf("UIPlugin has finalizers %v — this will block deletion (pre-fix behavior)", currentPlugin.Finalizers)
	} else {
		t.Log("UIPlugin has no finalizers — deletion should proceed immediately (post-fix behavior)")
	}

	err = f.K8sClient.Delete(ctx, currentPlugin)
	assert.NilError(t, err, "failed to delete UIPlugin CR")

	// The UIPlugin CR itself should be gone quickly (no finalizer to block it).
	// Allow a short timeout — if it exceeds this, the finalizer is likely stuck.
	// Use Errorf (not Fatalf) so Phase 4 still runs even if the CR is stuck —
	// this shows the full scope of failure on pre-fix builds.
	t.Log("Waiting for UIPlugin CR to be fully deleted...")
	if err := pollUntilResourceGone("monitoring", "", &uiv1.UIPlugin{}, 1*time.Minute); err != nil {
		t.Errorf("UIPlugin CR stuck in Terminating (finalizer not removed) — pre-fix behavior confirmed: %v", err)
	}

	// --- Phase 4: Verify cascade deletion of child resources ---

	t.Log("Phase 4: Verifying child resource cascade deletion")

	cleanupTimeout := 3 * time.Minute

	t.Run("cascade deletion", func(t *testing.T) {
		t.Run("monitoring plugin deployment is deleted", func(t *testing.T) {
			t.Parallel()
			assertResourceGone(t, "monitoring", ns, &appsv1.Deployment{}, cleanupTimeout)
		})

		t.Run("health-analyzer deployment is deleted", func(t *testing.T) {
			t.Parallel()
			assertResourceGone(t, "health-analyzer", ns, &appsv1.Deployment{}, cleanupTimeout)
		})

		t.Run("health-analyzer service is deleted", func(t *testing.T) {
			t.Parallel()
			assertResourceGone(t, "health-analyzer", ns, &corev1.Service{}, cleanupTimeout)
		})

		t.Run("monitoring plugin service is deleted", func(t *testing.T) {
			t.Parallel()
			assertResourceGone(t, "monitoring", ns, &corev1.Service{}, cleanupTimeout)
		})

		t.Run("monitoring plugin service account is deleted", func(t *testing.T) {
			t.Parallel()
			assertResourceGone(t, "monitoring-sa", ns, &corev1.ServiceAccount{}, cleanupTimeout)
		})

		t.Run("components-health-view ClusterRole is deleted", func(t *testing.T) {
			t.Parallel()
			assertResourceGone(t, "components-health-view", "", &rbacv1.ClusterRole{}, cleanupTimeout)
		})

		t.Run("components-health-view ClusterRoleBinding is deleted", func(t *testing.T) {
			t.Parallel()
			assertResourceGone(t, "monitoring-components-health-view", "", &rbacv1.ClusterRoleBinding{}, cleanupTimeout)
		})

		t.Run("no UIPlugin-managed pods remain in operator namespace", func(t *testing.T) {
			t.Parallel()
			assertNoManagedPodsRemain(t, ctx, ns)
		})
	})

	t.Log("Phase 4: All cascade deletion checks completed")
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

// assertResourceGone is a fatal wrapper around pollUntilResourceGone for use in
// subtests: it fails the test immediately if the resource is not deleted in time.
func assertResourceGone(t *testing.T, name, namespace string, obj client.Object, timeout time.Duration) {
	t.Helper()
	if err := pollUntilResourceGone(name, namespace, obj, timeout); err != nil {
		t.Fatalf("%T %s/%s was not deleted (waited %v): %v", obj, namespace, name, timeout, err)
	}
}

// pollUntilResourceGone polls until the named resource no longer exists.
// Returns nil if the resource disappeared within the timeout, or an error
// that includes the last API failure if one occurred.
func pollUntilResourceGone(name, namespace string, obj client.Object, timeout time.Duration) error {
	key := client.ObjectKey{Name: name, Namespace: namespace}
	var lastErr error
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := f.K8sClient.Get(ctx, key, obj)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			lastErr = err
		}
		return false, nil
	})
	if err != nil && lastErr != nil {
		return fmt.Errorf("%s/%s was not deleted: %w (last API error: %v)", namespace, name, err, lastErr)
	}
	return err
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

	var lastErr error
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		var remaining uiv1.UIPluginList
		if err := f.K8sClient.List(ctx, &remaining); err != nil {
			lastErr = err
			return false, nil
		}
		return len(remaining.Items) == 0, nil
	})
	if err != nil {
		t.Fatalf("Stale UIPlugins still exist after force cleanup: %v (last API error: %v)", err, lastErr)
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
	var lastErr error
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		if err := f.K8sClient.List(ctx, pods,
			client.InNamespace(namespace),
			client.MatchingLabels(managedLabels),
		); err != nil {
			lastErr = err
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

	if err != nil {
		t.Fatalf("managed pods not cleaned up after UIPlugin deletion: %v (last seen: %v) (last API error: %v)", err, lastSeen, lastErr)
	}
}

// waitForCSVSucceeded polls until the observability-operator CSV reaches the Succeeded phase.
func waitForCSVSucceeded(t *testing.T, namespace string, timeout time.Duration) error {
	t.Helper()
	var lastErr error
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		csvs := &olmv1alpha1.ClusterServiceVersionList{}
		if err := f.K8sClient.List(ctx, csvs, &client.ListOptions{Namespace: namespace}); err != nil {
			lastErr = err
			return false, nil
		}
		for i := range csvs.Items {
			if strings.Contains(csvs.Items[i].Name, "observability-operator") &&
				csvs.Items[i].Status.Phase == olmv1alpha1.CSVPhaseSucceeded {
				t.Logf("Cleanup: CSV %s is Succeeded", csvs.Items[i].Name)
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil && lastErr != nil {
		return fmt.Errorf("%w (last API error: %v)", err, lastErr)
	}
	return err
}

// patchCSVOpenShiftEnabled patches the reinstalled CSV to add --openshift.enabled=true
// to the operator container args. This is needed because operator-sdk run bundle +
// enable_openshift() only patches the initial CSV; a reinstalled CSV loses the flag.
func patchCSVOpenShiftEnabled(t *testing.T, namespace string) error {
	t.Helper()
	ctx := context.Background()

	csvs := &olmv1alpha1.ClusterServiceVersionList{}
	if err := f.K8sClient.List(ctx, csvs, &client.ListOptions{Namespace: namespace}); err != nil {
		return fmt.Errorf("listing CSVs: %w", err)
	}

	for i := range csvs.Items {
		csv := &csvs.Items[i]
		if !strings.Contains(csv.Name, "observability-operator") {
			continue
		}

		modified := false
		for di := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
			ds := &csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[di]
			if ds.Name != "observability-operator" {
				continue
			}
			for ci := range ds.Spec.Template.Spec.Containers {
				c := &ds.Spec.Template.Spec.Containers[ci]
				if c.Name != "operator" {
					continue
				}
				for _, arg := range c.Args {
					if arg == "--openshift.enabled=true" {
						t.Log("Cleanup: CSV already has --openshift.enabled=true")
						return nil
					}
				}
				c.Args = append(c.Args, "--openshift.enabled=true")
				modified = true
			}
		}

		if modified {
			if err := f.K8sClient.Update(ctx, csv); err != nil {
				return fmt.Errorf("updating CSV: %w", err)
			}
			t.Logf("Cleanup: Patched CSV %s with --openshift.enabled=true", csv.Name)
		}
		return nil
	}

	return fmt.Errorf("no observability-operator CSV found")
}

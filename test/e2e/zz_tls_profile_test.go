package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rhobs/observability-operator/test/e2e/framework"
)

const (
	operatorDeploymentName = "observability-operator"
	operatorContainerName  = "operator"
)

func TestTLSProfileWatcher(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("TLS profile watcher requires an OpenShift cluster")
	}

	// Ensure a clean baseline by resetting the TLS profile to nil (Intermediate).
	// Previous test runs may have left a non-default profile on the APIServer.
	setTLSProfile(t, nil)
	f.AssertDeploymentReady(operatorDeploymentName, f.OperatorNamespace,
		framework.WithTimeout(5*time.Minute))(t)

	ts := []testCase{
		{
			name:     "operator reads the default TLS profile and is running",
			scenario: assertOperatorRunningWithDefaultTLSProfile,
		},
		{
			name:     "operator restarts when TLS profile changes from default to Old",
			scenario: assertOperatorRestartsOnTLSProfileChange,
		},
		{
			name:     "operator restarts when TLS profile changes to Custom",
			scenario: assertOperatorRestartsOnCustomTLSProfile,
		},
		{
			name:     "operator does not restart when APIServer non-TLS field changes",
			scenario: assertOperatorStableOnNonTLSChange,
		},
	}
	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

// assertOperatorRunningWithDefaultTLSProfile verifies that the operator is
// running and healthy when the cluster uses the default TLS profile (no
// tlsSecurityProfile set on the APIServer CR, which defaults to Intermediate).
func assertOperatorRunningWithDefaultTLSProfile(t *testing.T) {
	// Verify the APIServer CR exists and is readable.
	apiServer := &configv1.APIServer{}
	err := f.K8sClient.Get(context.Background(), types.NamespacedName{Name: "cluster"}, apiServer)
	assert.NilError(t, err, "failed to get APIServer CR")

	// Verify the operator deployment is ready.
	f.AssertDeploymentReady(operatorDeploymentName, f.OperatorNamespace,
		framework.WithTimeout(2*time.Minute))(t)

	t.Logf("APIServer TLS profile type: %v", apiServer.Spec.TLSSecurityProfile)
}

// assertOperatorRestartsOnTLSProfileChange sets the TLS profile to Old,
// verifies the operator container restarts, then restores the original profile.
func assertOperatorRestartsOnTLSProfileChange(t *testing.T) {
	ctx := context.Background()

	// Save original APIServer spec so we can restore it.
	apiServer := &configv1.APIServer{}
	err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "cluster"}, apiServer)
	assert.NilError(t, err, "failed to get APIServer CR")
	originalTLSProfile := apiServer.Spec.TLSSecurityProfile

	// Ensure we restore the original profile even if the test fails.
	f.CleanUp(t, func() {
		restoreTLSProfile(t, originalTLSProfile)
	})

	// Wait for the operator to be stable before we start.
	f.AssertDeploymentReady(operatorDeploymentName, f.OperatorNamespace,
		framework.WithTimeout(2*time.Minute))(t)

	// Record the current container restart count.
	initialRestarts := getOperatorContainerRestartCount(t)
	t.Logf("container restart count before TLS change: %d", initialRestarts)

	// Change TLS profile to Old.
	t.Log("setting TLS profile to Old")
	setTLSProfile(t, &configv1.TLSSecurityProfile{
		Type: configv1.TLSProfileOldType,
		Old:  &configv1.OldTLSProfile{},
	})

	// The operator should restart: wait for the restart count to increase.
	waitForOperatorContainerRestart(t, initialRestarts)

	// After restart, the operator should become ready again.
	f.AssertDeploymentReady(operatorDeploymentName, f.OperatorNamespace,
		framework.WithTimeout(5*time.Minute))(t)
	t.Log("operator restarted and is ready after TLS profile change to Old")
}

// assertOperatorRestartsOnCustomTLSProfile sets a Custom TLS profile with
// specific ciphers and minTLSVersion, verifies the operator restarts, then
// restores the original profile.
func assertOperatorRestartsOnCustomTLSProfile(t *testing.T) {
	ctx := context.Background()

	apiServer := &configv1.APIServer{}
	err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "cluster"}, apiServer)
	assert.NilError(t, err, "failed to get APIServer CR")
	originalTLSProfile := apiServer.Spec.TLSSecurityProfile

	f.CleanUp(t, func() {
		restoreTLSProfile(t, originalTLSProfile)
	})

	f.AssertDeploymentReady(operatorDeploymentName, f.OperatorNamespace,
		framework.WithTimeout(5*time.Minute))(t)

	// Wait for the restart count to stabilize — the previous test's cleanup
	// may have triggered a restart that hasn't completed yet.
	initialRestarts := waitForStableRestartCount(t)

	// Set a Custom TLS profile that differs from the default Intermediate
	// profile by using a smaller cipher list (a subset of Intermediate's ciphers).
	t.Log("setting TLS profile to Custom")
	setTLSProfile(t, &configv1.TLSSecurityProfile{
		Type: configv1.TLSProfileCustomType,
		Custom: &configv1.CustomTLSProfile{
			TLSProfileSpec: configv1.TLSProfileSpec{
				Ciphers: []string{
					"ECDHE-ECDSA-AES128-GCM-SHA256",
					"ECDHE-RSA-AES128-GCM-SHA256",
				},
				MinTLSVersion: configv1.VersionTLS12,
			},
		},
	})

	waitForOperatorContainerRestart(t, initialRestarts)
	f.AssertDeploymentReady(operatorDeploymentName, f.OperatorNamespace,
		framework.WithTimeout(5*time.Minute))(t)
	t.Log("operator restarted and is ready after TLS profile change to Custom")
}

// assertOperatorStableOnNonTLSChange modifies a non-TLS field (an annotation)
// on the APIServer CR and verifies the operator does NOT restart. An annotation
// is used instead of a spec field like audit profile because spec changes can
// trigger MachineConfigPool rollouts which are very disruptive.
func assertOperatorStableOnNonTLSChange(t *testing.T) {
	ctx := context.Background()

	// Reset the TLS profile to nil to ensure a clean state. Previous tests'
	// cleanups may have triggered delayed restarts due to CrashLoopBackOff.
	// By explicitly setting nil here and waiting for stability, we ensure any
	// pending restarts complete before we take a baseline.
	setTLSProfile(t, nil)
	f.AssertDeploymentReady(operatorDeploymentName, f.OperatorNamespace,
		framework.WithTimeout(5*time.Minute))(t)
	initialRestarts := waitForStableRestartCount(t)

	// Update a non-TLS field: add a test annotation.
	const testAnnotation = "observability-operator.rhobs/e2e-tls-test"
	apiServer := &configv1.APIServer{}
	err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "cluster"}, apiServer)
	assert.NilError(t, err)

	f.CleanUp(t, func() {
		as := &configv1.APIServer{}
		if err := f.K8sClient.Get(context.Background(), types.NamespacedName{Name: "cluster"}, as); err != nil {
			t.Logf("cleanup: failed to get APIServer: %v", err)
			return
		}
		patch := client.MergeFrom(as.DeepCopy())
		delete(as.Annotations, testAnnotation)
		if err := f.K8sClient.Patch(context.Background(), as, patch); err != nil {
			t.Logf("cleanup: failed to remove test annotation: %v", err)
		}
	})

	patch := client.MergeFrom(apiServer.DeepCopy())
	if apiServer.Annotations == nil {
		apiServer.Annotations = map[string]string{}
	}
	apiServer.Annotations[testAnnotation] = "true"
	err = f.K8sClient.Patch(ctx, apiServer, patch)
	assert.NilError(t, err, "failed to patch APIServer with test annotation")
	t.Log("added test annotation to APIServer CR")

	// Wait a reasonable period and verify the restart count has not increased.
	t.Log("waiting 60s to confirm operator does not restart")
	err = wait.PollUntilContextTimeout(ctx, 10*time.Second, 60*time.Second, true, func(ctx context.Context) (bool, error) {
		currentRestarts := getOperatorContainerRestartCount(t)
		if currentRestarts > initialRestarts {
			return true, fmt.Errorf("operator restarted unexpectedly: restart count changed from %d to %d", initialRestarts, currentRestarts)
		}
		return false, nil // keep polling — we want this to NOT match (timeout = success)
	})

	if wait.Interrupted(err) {
		// Timeout means the restart count never changed — that's what we want.
		t.Log("operator remained stable after non-TLS APIServer change")
		return
	}
	// If we get here, it means the restart count increased — fail.
	assert.NilError(t, err, "operator should not restart on non-TLS APIServer changes")
}

// setTLSProfile patches the APIServer CR's tlsSecurityProfile.
func setTLSProfile(t *testing.T, profile *configv1.TLSSecurityProfile) {
	t.Helper()
	ctx := context.Background()

	apiServer := &configv1.APIServer{}
	err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "cluster"}, apiServer)
	assert.NilError(t, err, "failed to get APIServer CR")

	patch := client.MergeFrom(apiServer.DeepCopy())
	apiServer.Spec.TLSSecurityProfile = profile
	err = f.K8sClient.Patch(ctx, apiServer, patch)
	assert.NilError(t, err, "failed to patch APIServer TLS profile")
}

// restoreTLSProfile restores the APIServer CR's tlsSecurityProfile to the
// given value and waits for the operator to become ready again.
func restoreTLSProfile(t *testing.T, profile *configv1.TLSSecurityProfile) {
	t.Helper()
	ctx := context.Background()

	apiServer := &configv1.APIServer{}
	if err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "cluster"}, apiServer); err != nil {
		t.Logf("cleanup: failed to get APIServer: %v", err)
		return
	}

	patch := client.MergeFrom(apiServer.DeepCopy())
	apiServer.Spec.TLSSecurityProfile = profile
	if err := f.K8sClient.Patch(ctx, apiServer, patch); err != nil {
		t.Logf("cleanup: failed to restore TLS profile: %v", err)
		return
	}

	// Wait for operator to recover after the restore.
	_ = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		dep := &appsv1.Deployment{}
		if err := f.K8sClient.Get(ctx, types.NamespacedName{
			Name:      operatorDeploymentName,
			Namespace: f.OperatorNamespace,
		}, dep); err != nil {
			return false, nil
		}
		return dep.Status.ReadyReplicas == *dep.Spec.Replicas, nil
	})
}

// getOperatorContainerRestartCount returns the total restart count of the
// operator container in the running operator pod.
func getOperatorContainerRestartCount(t *testing.T) int32 {
	t.Helper()

	pod := getRunningOperatorPod(t)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == operatorContainerName {
			return cs.RestartCount
		}
	}

	t.Fatalf("container %q not found in operator pod", operatorContainerName)
	return 0
}

// getRunningOperatorPod returns the running operator pod.
func getRunningOperatorPod(t *testing.T) *corev1.Pod {
	t.Helper()

	dep := &appsv1.Deployment{}
	err := f.K8sClient.Get(context.Background(), types.NamespacedName{
		Name:      operatorDeploymentName,
		Namespace: f.OperatorNamespace,
	}, dep)
	assert.NilError(t, err, "failed to get operator deployment")

	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	assert.NilError(t, err, "failed to parse deployment selector")

	var pods corev1.PodList
	err = f.K8sClient.List(context.Background(), &pods,
		client.InNamespace(f.OperatorNamespace),
		client.MatchingLabelsSelector{Selector: selector},
	)
	assert.NilError(t, err, "failed to list operator pods")

	for i := range pods.Items {
		p := &pods.Items[i]
		if p.Status.Phase == corev1.PodRunning && p.DeletionTimestamp == nil {
			return p
		}
	}

	t.Fatal("no running operator pod found")
	return nil
}

// waitForOperatorContainerRestart polls until the operator process has
// restarted. It detects restarts by either: (1) the container's restart count
// exceeding the baseline, or (2) the pod being replaced entirely (new pod UID),
// which happens when Kubernetes recreates the pod after CrashLoopBackOff.
func waitForOperatorContainerRestart(t *testing.T, baselineRestarts int32) {
	t.Helper()

	// Record the current pod UID so we can detect pod replacement.
	originalPod := getRunningOperatorPod(t)
	originalPodUID := originalPod.UID

	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		dep := &appsv1.Deployment{}
		if err := f.K8sClient.Get(ctx, types.NamespacedName{
			Name:      operatorDeploymentName,
			Namespace: f.OperatorNamespace,
		}, dep); err != nil {
			return false, nil
		}

		selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
		if err != nil {
			return false, nil
		}

		var pods corev1.PodList
		if err := f.K8sClient.List(ctx, &pods,
			client.InNamespace(f.OperatorNamespace),
			client.MatchingLabelsSelector{Selector: selector},
		); err != nil {
			return false, nil
		}

		for _, p := range pods.Items {
			if p.DeletionTimestamp != nil {
				continue
			}
			// Detect pod replacement (new pod UID).
			if p.UID != originalPodUID && p.Status.Phase == corev1.PodRunning {
				t.Logf("operator pod replaced: old UID=%s, new UID=%s", originalPodUID, p.UID)
				return true, nil
			}
			// Detect container restart within the same pod.
			for _, cs := range p.Status.ContainerStatuses {
				if cs.RestartCount > baselineRestarts {
					t.Logf("operator container restarted: restart count %d -> %d", baselineRestarts, cs.RestartCount)
					return true, nil
				}
			}
		}
		return false, nil
	})
	assert.NilError(t, err, "operator container did not restart within timeout")
}

// waitForStableRestartCount waits until the operator container's restart count
// remains unchanged for 30 seconds AND the container has been running for at
// least 15 seconds. This ensures any pending restarts from previous test
// cleanups have completed and the watcher has had time to initialize.
func waitForStableRestartCount(t *testing.T) int32 {
	t.Helper()

	var lastCount int32 = -1
	var stableSince time.Time

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		current := getOperatorContainerRestartCount(t)
		if current != lastCount {
			lastCount = current
			stableSince = time.Now()
			return false, nil
		}
		if time.Since(stableSince) < 30*time.Second {
			return false, nil
		}

		// Also verify the container has been running long enough for
		// the watcher to process its initial reconcile event.
		pod := getRunningOperatorPod(t)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Running != nil {
				uptime := time.Since(cs.State.Running.StartedAt.Time)
				if uptime < 15*time.Second {
					t.Logf("operator container uptime %s < 15s, waiting longer", uptime.Round(time.Second))
					return false, nil
				}
			}
		}

		return true, nil
	})
	assert.NilError(t, err, "operator restart count did not stabilize within timeout")
	t.Logf("operator restart count stabilized at %d", lastCount)
	return lastCount
}

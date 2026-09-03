package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

// TestGeneratorReconcileParity verifies that the resources the generator
// produces for a UIPlugin CR are exactly what the running operator applies on a
// real cluster. It renders the live (defaulted) UIPlugin CR through the
// generator CLI and compares the output against the live operands, ignoring
// fields that Kubernetes apiservers default or manage.
func TestGeneratorReconcileParity(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("The tests are skipped on non-ocp cluster")
	}

	assertCRDExists(t, "uiplugins.observability.openshift.io")
	f.DumpOnFailure(t, f.DebugNamespaces(f.OperatorNamespace))

	ctx := context.Background()
	plugin := &uiv1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: "parity-dashboards",
		},
		Spec: uiv1.UIPluginSpec{
			Type: uiv1.UIPluginType("Dashboards"),
		},
	}
	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), plugin)
	})

	assert.NilError(t, f.K8sClient.Create(ctx, plugin), "failed to create dashboards UIPlugin")

	// Wait until the operator has reconciled the plugin.
	f.AssertDeploymentReady("observability-ui-parity-dashboards", f.OperatorNamespace,
		framework.WithTimeout(5*time.Minute))(t)

	// Render the live CR through the generator so its input matches the spec the
	// operator reconciled (including any defaulting applied by the apiserver).
	var livePlugin uiv1.UIPlugin
	assert.NilError(t, f.K8sClient.Get(ctx, client.ObjectKey{Name: plugin.Name}, &livePlugin))
	input, err := yaml.Marshal(&livePlugin)
	assert.NilError(t, err, "marshaling UIPlugin for generator input")

	expected := runGenerator(t, input, f.OperatorNamespace)

	// Compare each generator output against the live object, ignoring fields
	// that the apiserver defaults or manages.
	for _, want := range expected {
		got := &unstructured.Unstructured{}
		got.SetGroupVersionKind(want.GroupVersionKind())
		if want.GetNamespace() != "" {
			err := f.K8sClient.Get(ctx, client.ObjectKey{Name: want.GetName(), Namespace: want.GetNamespace()}, got)
			assert.NilError(t, err, "expected operand %s %s/%s does not exist",
				want.GetKind(), want.GetNamespace(), want.GetName())
		} else {
			err := f.K8sClient.Get(ctx, client.ObjectKey{Name: want.GetName()}, got)
			assert.NilError(t, err, "expected cluster-scoped operand %s %s does not exist",
				want.GetKind(), want.GetName())
		}

		normalizeForParity(want)
		normalizeForParity(got)
		assert.Equal(t, want.Object, got.Object, "drift between generator output and reconciled object")
	}
}

// runGenerator builds the generator CLI and renders the given input CR into the
// operator namespace, returning the decoded output objects (minus the resources
// OLM owns and the UIPlugin CR itself, which the operator does not create).
func runGenerator(t *testing.T, input []byte, namespace string) []*unstructured.Unstructured {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "generator")
	build := exec.Command("go", "build", "-o", bin, "./cmd/generator")
	build.Dir = filepath.Join(filepath.Dir(thisFile), "..", "..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building generator: %v\n%s", err, out)
	}

	inputFile := filepath.Join(binDir, "uiplugin.yaml")
	assert.NilError(t, os.WriteFile(inputFile, input, 0o600))
	outputDir := filepath.Join(binDir, "output")

	cmd := exec.Command(bin, "-f", inputFile, "-namespace", namespace,
		"-cluster-version", f.ClusterVersion.Status.Desired.Version, "-output", outputDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generator failed: %v\n%s", err, out)
	}

	objects := []*unstructured.Unstructured{}
	for _, file := range readYamlFiles(outputDir) {
		obj := &unstructured.Unstructured{}
		assert.NilError(t, yaml.UnmarshalStrict(file, obj), "decoding generator output %s", file)
		if isOperatorManaged(obj) {
			objects = append(objects, obj)
		}
	}
	assert.Assert(t, len(objects) > 0, "generator produced no operator-managed resources")
	return objects
}

// isOperatorManaged reports whether the object is created by the operator
// itself, excluding the resources OLM creates before the operator installs
// (the COONamespace and its OperatorGroup) and the UIPlugin CRs that the
// operator reconciles as separate objects.
func isOperatorManaged(obj *unstructured.Unstructured) bool {
	gvk := obj.GroupVersionKind()
	if gvk.Group == "observability.openshift.io" && gvk.Kind == "UIPlugin" {
		return false
	}
	if gvk.Kind == "Namespace" && gvk.Group == "" && gvk.Version == "v1" {
		return false
	}
	if gvk.Group == "operators.coreos.com" && gvk.Kind == "OperatorGroup" {
		return false
	}
	return true
}

// readYamlFiles returns the raw bytes of every YAML file in dir, sorted.
func readYamlFiles(dir string) [][]byte {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files [][]byte
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
			files = append(files, b)
		}
	}
	return files
}

// normalizeForParity removes, from both the expected and live objects, every
// field that the apiserver defaults or manages so that only the fields the
// operator declares are compared.
func normalizeForParity(obj *unstructured.Unstructured) {
	for _, f := range []string{
		"annotations",
		"creationTimestamp",
		"deletionGracePeriodSeconds",
		"deletionTimestamp",
		"finalizers",
		"generation",
		"managedFields",
		"ownerReferences",
		"resourceVersion",
		"selfLink",
		"uid",
	} {
		unstructured.RemoveNestedField(obj.Object, "metadata", f)
	}

	unstructured.RemoveNestedField(obj.Object, "status")

	switch obj.GetKind() {
	case "Deployment":
		normalizeDeployment(obj)
	case "Service":
		for _, f := range []string{"clusterIP", "clusterIPs", "healthCheckNodePort", "internalTrafficPolicy",
			"ipFamilies", "ipFamilyPolicy", "sessionAffinity"} {
			unstructured.RemoveNestedField(obj.Object, "spec", f)
		}
		if ports, ok, _ := unstructured.NestedSlice(obj.Object, "spec", "ports"); ok {
			for _, p := range ports {
				if pm, ok := p.(map[string]interface{}); ok {
					delete(pm, "nodePort")
				}
			}
			unstructured.SetNestedSlice(obj.Object, ports, "spec", "ports")
		}
	case "ServiceAccount":
		for _, f := range []string{"automountServiceAccountToken", "imagePullSecrets", "secrets"} {
			unstructured.RemoveNestedField(obj.Object, f)
		}
	}
}

func normalizeDeployment(obj *unstructured.Unstructured) {
	unstructured.RemoveNestedField(obj.Object, "spec", "revisionHistoryLimit")
	unstructured.RemoveNestedField(obj.Object, "spec", "strategy")

	for _, f := range []string{"annotations", "creationTimestamp"} {
		unstructured.RemoveNestedField(obj.Object, "spec", "template", "metadata", f)
	}

	for _, f := range []string{
		"automountServiceAccountToken",
		"deprecatedServiceAccount",
		"dnsConfig",
		"enableServiceLinks",
		"hostNetwork",
		"imagePullSecrets",
		"priority",
		"schedulerName",
		"shareProcessNamespace",
		"terminationGracePeriodSeconds",
	} {
		unstructured.RemoveNestedField(obj.Object, "spec", "template", "spec", f)
	}

	for _, path := range [][]string{
		{"spec", "template", "spec", "containers"},
		{"spec", "template", "spec", "initContainers"},
	} {
		containers, ok, _ := unstructured.NestedSlice(obj.Object, path...)
		if !ok {
			continue
		}
		for i, c := range containers {
			cm := c.(map[string]interface{})
			delete(cm, "imagePullPolicy")
			delete(cm, "resources")
			delete(cm, "terminationMessagePath")
			// Image tags are deployment-environment-specific (e2e setups often
			// override them), so normalize them away for the parity check.
			cm["image"] = "<image>"
			if ports, ok := cm["ports"].([]interface{}); ok {
				for _, p := range ports {
					pm := p.(map[string]interface{})
					delete(pm, "protocol")
					delete(pm, "hostPort")
				}
			}
			containers[i] = cm
		}
		unstructured.SetNestedSlice(obj.Object, containers, path...)
	}
}

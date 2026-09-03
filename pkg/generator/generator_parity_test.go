package generator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	kyaml "sigs.k8s.io/yaml"

	"github.com/rhobs/observability-operator/config"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability"
	"github.com/rhobs/observability-operator/pkg/controllers/uiplugin"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
	"github.com/rhobs/observability-operator/pkg/images"
	"github.com/rhobs/observability-operator/pkg/operator"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

const (
	testNamespace      = "openshift-cluster-observability-operator"
	testClusterVersion = "v4.22"
)

// installerOptions returns the operator install options the generator and the
// reconciler use, pointing at the same package/channel names so the parity
// tests compare like with like.
func installerOptions() observability.Options {
	return observability.Options{
		COONamespace: testNamespace,
		OpenTelemetryOperator: observability.OperatorInstallConfig{
			Namespace:   testNamespace,
			PackageName: "opentelemetry-product",
			StartingCSV: "",
			Channel:     "stable",
		},
		TempoOperator: observability.OperatorInstallConfig{
			Namespace:   testNamespace,
			PackageName: "tempo-product",
			StartingCSV: "",
			Channel:     "stable",
		},
	}
}

// decodeObjects parses multi-document YAML into unstructured objects.
func decodeObjects(data []byte) ([]client.Object, error) {
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	var objects []client.Object
	for {
		doc, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}
		jsonData, err := kyaml.YAMLToJSON(doc)
		if err != nil {
			return nil, err
		}
		obj := &unstructured.Unstructured{}
		if err := obj.UnmarshalJSON(jsonData); err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

// recordingClient wraps a client.Client and records every object handed to
// Apply, Create or Update. The reconcilers mutate the objects (add common
// labels, set owner references) before handing them over, so the recording
// captures exactly what the operator would apply to a cluster.
type recordingClient struct {
	client.Client
	applied []client.Object
}

func (r *recordingClient) record(obj client.Object, scheme *k8sruntime.Scheme) error {
	u, err := objectToUnstructured(obj, scheme)
	if err != nil {
		return err
	}
	r.applied = append(r.applied, u)
	return nil
}

func (r *recordingClient) Apply(ctx context.Context, obj k8sruntime.ApplyConfiguration, opts ...client.ApplyOption) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(data); err != nil {
		return err
	}
	r.applied = append(r.applied, u)
	// The fake client's Apply zeroes the apply configuration before
	// unmarshalling the patched result back into it, which loses the reconciler's
	// wrapped object (see pkg/reconciler/clientObjectApplyConfig) and fails with
	// "json: Unmarshal(nil)". The reconciler's Reconcile has already built the
	// labelled/owner-referenced object we recorded, and nothing downstream reads
	// the fake client's stored state, so recording without applying is equivalent
	// for this parity check.
	return nil
}

func (r *recordingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if err := r.record(obj, r.Client.Scheme()); err != nil {
		return err
	}
	return r.Client.Create(ctx, obj, opts...)
}

func (r *recordingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if err := r.record(obj, r.Client.Scheme()); err != nil {
		return err
	}
	return r.Client.Update(ctx, obj, opts...)
}

// objectToUnstructured converts a typed object to unstructured, filling the
// GroupVersionKind from the scheme when the object has no TypeMeta.
func objectToUnstructured(obj client.Object, scheme *k8sruntime.Scheme) (*unstructured.Unstructured, error) {
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u, nil
	}
	data, err := kyaml.Marshal(obj)
	if err != nil {
		return nil, err
	}
	jsonData, err := kyaml.YAMLToJSON(data)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(jsonData); err != nil {
		return nil, err
	}
	if u.GetKind() == "" {
		gvks, _, err := scheme.ObjectKinds(obj)
		if err != nil {
			return nil, err
		}
		u.SetGroupVersionKind(gvks[0])
	}
	return u, nil
}

// stripClusterState removes fields that only exist on objects applied to a
// cluster (and would therefore never appear in the generator's output).
func stripClusterState(u *unstructured.Unstructured) {
	for _, field := range []string{
		"resourceVersion", "uid", "generation", "creationTimestamp",
		"managedFields", "ownerReferences", "finalizers",
	} {
		unstructured.RemoveNestedField(u.Object, "metadata", field)
	}
}

// objectIdentity returns a stable identity for an object: group/kind/namespace/name.
func objectIdentity(obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return strings.Join([]string{gvk.Group, gvk.Kind, obj.GetNamespace(), obj.GetName()}, "/")
}

// mirrorOperatorLabels applies util.AddCommonLabels to installer-origin objects
// in the generator's output so the two sides can be compared on equal footing.
// UIPlugin operands get their labels from config manifests and are skipped.
func mirrorOperatorLabels(t *testing.T, objects []client.Object, installers []*obsv1alpha1.ObservabilityInstaller, others []client.Object) {
	t.Helper()
	reader := NewFallbackReader(nil, others...)
	ownerByIdentity := map[string]string{}
	for _, installer := range installers {
		cfg := observability.NewOverlayConfig(nil, installerOptions())
		installerObjects, err := observability.ResolveInstaller(context.Background(), reader, installer, cfg)
		assert.NilError(t, err)
		for _, obj := range installerObjects {
			ownerByIdentity[objectIdentity(obj)] = installer.Name
		}
	}
	for _, obj := range objects {
		if owner, ok := ownerByIdentity[objectIdentity(obj)]; ok {
			util.AddCommonLabels(obj, owner)
		}
	}
}

// TestReconcilerMatchGenerator verifies that the object set
// the operator's reconcile machinery would apply to a cluster matches the
// manifest the offline generator emits for the same input CRs.
//
// This test drives the
// real reconcile path: observability.GenerateAllInstallerObjects feeds
// reconciler.Updater / CreateUpdateReconciler, whose Reconcile methods add
// common labels and apply (server-side apply) the resources; UIPlugin operands
// are applied the same way the UIPlugin controller applies them (a
// reconciler.Updater per operand). This catches drift between the operator and
// generator that the shared functions cannot, e.g. in label handling.
//
// The two sides are not byte-identical by design:
//   - The generator emits the COO Namespace, which OLM creates when the
//     operator is installed
//   - The operator persists UIPlugin CRs as separate reconciled objects while
//     the generator resolves them into operands and drops the CRs; they are
//     dropped from the operator side.
//   - Applied objects carry owner references and server-side-apply metadata that
//     never appears in the generator output
//
// The full uiplugin.resourceManager.Reconcile (Console CR registration, status
// updates, dynamic client usage) requires a live API server and is out of scope.
func TestReconcilerMatchGenerator(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "testdata", "sample")
	inputDir := filepath.Join(testdataDir, "input")

	scheme := operator.NewScheme(&operator.OperatorConfiguration{
		FeatureGates: operator.FeatureGates{
			OpenShift: operator.OpenShiftFeatureGates{
				Enabled: true,
				Version: testClusterVersion,
			},
		}})

	yamlData, err := readInputs([]string{inputDir})
	assert.NilError(t, err)
	installers, plugins, others, err := decodeResources(scheme, yamlData)
	require.NoError(t, err)
	assert.Assert(t, len(installers) > 0, "sample input must contain at least one ObservabilityInstaller")

	// The generator's output is the ground truth for the desired object set.
	genOutput, warnings, err := Run(RunConfig{
		Input:          []string{inputDir},
		Namespace:      testNamespace,
		ClusterVersion: testClusterVersion,
	})
	require.NoError(t, err)
	assert.Equal(t, warnings, "")
	genObjects, err := decodeObjects([]byte(genOutput))
	require.NoError(t, err)

	// The generator emits Namespaces, OperatorGroups, and passthrough input
	// resources that the operator never applies; filter them out.
	passthroughIDs := make(map[string]bool)
	for _, obj := range others {
		passthroughIDs[util.Identifier(obj)] = true
	}
	genObjects = slices.DeleteFunc(genObjects, func(obj client.Object) bool {
		u := obj.(*unstructured.Unstructured)
		return u.GetKind() == "Namespace" || u.GetKind() == "OperatorGroup" || passthroughIDs[util.Identifier(obj)]
	})
	// Mirror the label mutation the operator's reconcilers perform.
	mirrorOperatorLabels(t, genObjects, installers, others)

	// Operator side: drive the reconcile machinery against a fake client,
	// recording everything applied.
	fakeClient := clientfake.NewClientBuilder().WithScheme(scheme).
		WithObjects(others...).
		Build()
	rec := &recordingClient{Client: fakeClient}
	reader := NewFallbackReader(fakeClient, others...)

	var recordedPlugins []*uiv1alpha1.UIPlugin

	// Replicate the fresh-install branch of observability.getReconcilers: with no
	// pre-installed subscriptions every object is created/updated, subscriptions
	// go through CreateUpdateReconciler and everything else through Updater.
	for _, installer := range installers {
		installer.UID = types.UID("parity-installer")
		cfg := observability.NewOverlayConfig(nil, installerOptions())
		objects, err := observability.ResolveInstaller(context.Background(), reader, installer, cfg)
		assert.NilError(t, err)
		for _, obj := range objects {
			if plugin, ok := obj.(*uiv1alpha1.UIPlugin); ok {
				recordedPlugins = append(recordedPlugins, plugin)
			}
		}

		subs := slices.DeleteFunc(slices.Clone(genObjects), func(obj client.Object) bool {
			return obj.GetObjectKind().GroupVersionKind().Kind != "Subscription"
		})
		assert.Assert(t, len(subs) == 2, "expected the two operator subscriptions, got %d", len(subs))

		var installerReconcilers []reconciler.Reconciler
		for _, sub := range subs {
			installerReconcilers = append(installerReconcilers, reconciler.NewCreateUpdateReconciler(sub.DeepCopyObject().(client.Object), installer))
		}
		for _, obj := range objects {
			installerReconcilers = append(installerReconcilers, reconciler.NewUpdater(obj, installer))
		}
		for _, installerRec := range installerReconcilers {
			require.NoError(t, installerRec.Reconcile(context.Background(), rec, scheme))
		}
	}

	// UIPlugin operands: the input plugins plus the UIPlugin CRs the installer
	// creates (typed, collected above); dedup in case the input already contains
	// a plugin with the same name.
	allPlugins := append(slices.Clone(plugins), recordedPlugins...)
	slices.SortFunc(allPlugins, func(a, b *uiv1alpha1.UIPlugin) int { return util.CompareObjects(a, b) })
	allPlugins = slices.CompactFunc(allPlugins, func(a, b *uiv1alpha1.UIPlugin) bool { return util.CompareObjects(a, b) == 0 })

	resolvedImages, err := images.Validate(nil)
	require.NoError(t, err)
	tracingInstaller := findTracingInstaller(installers)
	pluginInfo := uiplugin.UIPluginInfo{
		Scheme:            scheme,
		ConfigFS:          config.FS,
		Images:            resolvedImages,
		Namespace:         testNamespace,
		ClusterVersion:    testClusterVersion,
		TracingInstaller:  tracingInstaller,
		TempoServiceNames: tempoServiceNamesFromInstaller(tracingInstaller),
	}
	for _, plugin := range allPlugins {
		if plugin.UID == "" {
			plugin.UID = types.UID(fmt.Sprintf("parity-plugin-%s", plugin.Name))
		}
		pluginObjects, _, err := uiplugin.ResolvePlugin(plugin, pluginInfo, logr.Discard())
		assert.NilError(t, err)
		for _, obj := range pluginObjects {
			require.NoError(t, reconciler.NewUpdater(obj, plugin).Reconcile(context.Background(), rec, scheme))
		}
	}

	// The operator persists UIPlugin CRs as separate reconciled objects; the
	// generator resolves them into operands and drops them from its output.
	applied := slices.DeleteFunc(rec.applied, func(obj client.Object) bool {
		gvk := obj.GetObjectKind().GroupVersionKind()
		return gvk.Group == "observability.openshift.io" && gvk.Kind == "UIPlugin"
	})
	for _, obj := range applied {
		stripClusterState(obj.(*unstructured.Unstructured))
	}
	slices.SortFunc(applied, util.CompareObjects)
	applied = slices.CompactFunc(applied, func(a, b client.Object) bool { return util.CompareObjects(a, b) == 0 })
	slices.SortFunc(genObjects, util.CompareObjects)
	assert.DeepEqual(t, genObjects, applied)
}

// TestGoldenOutput builds the generator binary, runs it on sample input, and
// byte-compares the output against the golden expected-output.yaml.
func TestGoldenOutput(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "testdata", "sample")
	inputDir := filepath.Join(testdataDir, "input")

	want, err := os.ReadFile(filepath.Join(testdataDir, "expected-output.yaml"))
	assert.NilError(t, err)

	binDir := t.TempDir()
	bin := filepath.Join(binDir, "generator")
	build := exec.Command("go", "build", "-o", bin, "./cmd/generator")
	build.Dir = filepath.Join(filepath.Dir(thisFile), "..", "..")
	out, err := build.CombinedOutput()
	assert.NilError(t, err, "building generator:\n%s", out)

	cmd := exec.Command(bin, "-f", inputDir)
	got, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); ok {
		t.Fatalf("generator failed: %v\nstderr: %s", err, ee.Stderr)
	}
	assert.NilError(t, err)
	if string(want) != string(got) {
		gotFile := filepath.Join(t.TempDir(), "got-output.yaml")
		os.WriteFile(gotFile, got, 0644)
		diff, _ := exec.Command("diff", "-u", filepath.Join(testdataDir, "expected-output.yaml"), gotFile).CombinedOutput()
		t.Fatalf("golden output mismatch:\n%s", diff)
	}
}

// TestGoldenBuild verifies that the in-process build pipeline produces output
// matching the golden expected-output.yaml, comparing objects semantically.
func TestGoldenBuild(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "testdata", "sample")
	inputDir := filepath.Join(testdataDir, "input")

	wantData, err := os.ReadFile(filepath.Join(testdataDir, "expected-output.yaml"))
	assert.NilError(t, err)
	want, err := decodeObjects(wantData)
	assert.NilError(t, err)
	slices.SortFunc(want, util.CompareObjects)

	genOutput, warnings, err := Run(RunConfig{
		Input:          []string{inputDir},
		Namespace:      testNamespace,
		ClusterVersion: testClusterVersion,
	})
	assert.NilError(t, err)
	assert.Equal(t, warnings, "")

	got, err := decodeObjects([]byte(genOutput))
	assert.NilError(t, err)
	slices.SortFunc(got, util.CompareObjects)

	assert.DeepEqual(t, want, got)
}

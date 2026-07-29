package overlay

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	return s
}

func TestReplacementsAppliedByBuild(t *testing.T) {
	o := newTestOverlay(t)
	objects, err := o.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(objects) == 0 {
		t.Fatal("Build returned no objects")
	}
	for _, obj := range objects {
		if obj.GetNamespace() != "" && obj.GetNamespace() != "actual-namespace" {
			t.Errorf("Build: unexpected namespace: %s", obj.GetNamespace())
		}
	}
}

func TestNoReplacementsIsIdentity(t *testing.T) {
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: keep-as-is\n"),
		},
	}
	o := New(testScheme(), configFS)
	o.SetBase("base")

	dir := t.TempDir()
	if err := o.WriteToDir(dir); err != nil {
		t.Fatalf("WriteToDir: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "base/configmap.yaml"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if !strings.Contains(string(data), "keep-as-is") {
		t.Errorf("file content changed unexpectedly: %s", data)
	}
}

// newTestOverlay creates an Overlay with kustomize replacements for testing.
func newTestOverlay(t *testing.T) *Overlay {
	t.Helper()
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: dummy\n"),
		},
	}
	o := New(testScheme(), configFS)
	o.SetBase("base")
	o.SetNamespace("actual-namespace")
	o.ReplaceValue("actual-name", TargetName("dummy", "metadata.name"))
	if err := o.AddResource(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "extra"},
		Data:       map[string]string{"key": "value"},
	}); err != nil {
		panic(err)
	}
	return o
}

func TestBuildNamespaceFirst(t *testing.T) {
	t.Helper()
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- configmap1.yaml
- configmap2.yaml
- namespace1.yaml
- namespace2.yaml
`),
		},
		"base/configmap1.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: dummy1\n"),
		},
		"base/namespace1.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: ns1\n"),
		},
		"base/configmap2.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: dummy2\n"),
		},
		"base/namespace2.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: ns2\n"),
		},
	}
	o := New(testScheme(), configFS)
	o.SetBase("base")
	o.SetNamespace("actual-namespace")
	objs, err := o.Build()
	require.NoError(t, err)
	got := toStructured(objs)
	want := []client.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns1"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns2"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "dummy1", Namespace: "actual-namespace"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "dummy2", Namespace: "actual-namespace"}},
	}
	assert.Equal(t, want, got)
}

func Test_isNamespace(t *testing.T) {
	assert.True(t, isNamespace(&corev1.Namespace{}))
	assert.False(t, isNamespace(&corev1.ConfigMap{}))
}

func Test_cmpNamespaceFirst(t *testing.T) {
	assert.Equal(t, 0, cmpNamespaceFirst(&corev1.Namespace{}, &corev1.Namespace{}))
	assert.Equal(t, 0, cmpNamespaceFirst(&corev1.ConfigMap{}, &corev1.ConfigMap{}))
	assert.Equal(t, 1, cmpNamespaceFirst(&corev1.ConfigMap{}, &corev1.Namespace{}))
	assert.Equal(t, -1, cmpNamespaceFirst(&corev1.Namespace{}, &corev1.ConfigMap{}))
}

func Test_sortedNamespaceFirst(t *testing.T) {
	want := []client.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "a"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "b"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "c"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "d"}},
	}
	unsorted := []client.Object{
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "c"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "a"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "d"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "b"}},
	}
	assert.Equal(t, want, sortedNamespaceFirst(slices.Values(unsorted)))
}

func TestBuildYAML(t *testing.T) {
	o := newTestOverlay(t)
	data, err := o.BuildYAML()
	require.NoError(t, err)
	require.NotEmpty(t, data)
	assert.Contains(t, string(data), "kind: ConfigMap")
	assert.Contains(t, string(data), "name: actual-name")
}

func TestEmptyOverlayBuild(t *testing.T) {
	o := New(testScheme(), nil)
	objs, err := o.Build()
	require.NoError(t, err)
	assert.Nil(t, objs)
	data, err := o.BuildYAML()
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestSetOwnerNameAddsLabels(t *testing.T) {
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: labled\n  labels:\n    existing: preset\n"),
		},
	}
	o := New(testScheme(), configFS)
	o.SetBase("base")
	o.SetOwnerName("test-app")
	objs, err := o.Build()
	require.NoError(t, err)
	require.Len(t, objs, 1)
	labels := objs[0].GetLabels()
	assert.Equal(t, "preset", labels["existing"])
	assert.Equal(t, "observability-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "test-app", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "labled", labels["app.kubernetes.io/name"])
}

func TestSetOwnerNameDoesNotOverrideExistingLabels(t *testing.T) {
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: labled\n  labels:\n    app.kubernetes.io/managed-by: custom\n"),
		},
	}
	o := New(testScheme(), configFS)
	o.SetBase("base")
	o.SetOwnerName("test-app")
	objs, err := o.Build()
	require.NoError(t, err)
	require.Len(t, objs, 1)
	assert.Equal(t, "custom", objs[0].GetLabels()["app.kubernetes.io/managed-by"])
}

func Test_addCommonLabels(t *testing.T) {
	obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "foo", Labels: map[string]string{"app.kubernetes.io/name": "keep"}}}
	addCommonLabels(obj, "owner")
	assert.Equal(t, "keep", obj.GetLabels()["app.kubernetes.io/name"])
	assert.Equal(t, "observability-operator", obj.GetLabels()["app.kubernetes.io/managed-by"])
	assert.Equal(t, "owner", obj.GetLabels()["app.kubernetes.io/part-of"])

	obj2 := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "bar"}}
	addCommonLabels(obj2, "owner")
	assert.Equal(t, "observability-operator", obj2.GetLabels()["app.kubernetes.io/managed-by"])
	assert.Equal(t, "bar", obj2.GetLabels()["app.kubernetes.io/name"])
}

func TestSetOverlayDir(t *testing.T) {
	o := New(testScheme(), nil)
	o.SetOverlayDir("/overlays/dev")
	assert.Equal(t, "/overlays/dev", o.overlayDir)

	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: dummy\n"),
		},
	}
	o = New(testScheme(), configFS)
	o.SetBase("base")
	o.SetOverlayDir("/overlays/dev")
	dir := t.TempDir()
	require.NoError(t, o.WriteToDir(dir))
	assert.FileExists(t, filepath.Join(dir, "overlays", "dev", "kustomization.yaml"))
	objs, err := o.Build()
	require.NoError(t, err)
	assert.Len(t, objs, 1)
}

func TestTargetKindName(t *testing.T) {
	tr := TargetKindName("ConfigMap", "dummy", "metadata.name")
	require.NotNil(t, tr.Select)
	assert.Equal(t, "ConfigMap", tr.Select.ResId.Gvk.Kind)
	assert.Equal(t, "dummy", tr.Select.ResId.Name)
	assert.Equal(t, []string{"metadata.name"}, tr.FieldPaths)
}

func TestAddRemoveComponent(t *testing.T) {
	o := New(testScheme(), nil)
	o.AddComponent("a")
	o.AddComponent("b")
	require.Len(t, o.components, 2)
	o.RemoveComponent("a")
	assert.Equal(t, []string{o.relativeToOverlayDir("b")}, o.components)
	o.RemoveComponent("missing")
	assert.Len(t, o.components, 1)
	o.AddComponent("a")
	o.RemoveComponent("b")
	assert.Equal(t, []string{o.relativeToOverlayDir("a")}, o.components)
}

func TestComponentWrittenToKustomization(t *testing.T) {
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: dummy\n"),
		},
	}
	o := New(testScheme(), configFS)
	o.SetBase("base")
	o.AddComponent("observabilityinstaller/crds")
	dir := t.TempDir()
	require.NoError(t, o.WriteToDir(dir))
	data, err := os.ReadFile(filepath.Join(dir, "overlays", "generated", "kustomization.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), o.relativeToOverlayDir("observabilityinstaller/crds"))
}

func TestAddPatchApplies(t *testing.T) {
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: patched\n  labels:\n    original: \"true\"\n"),
		},
	}
	o := New(testScheme(), configFS)
	o.SetBase("base")
	require.NoError(t, o.AddPatch("patch.yaml", map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": "patched"},
		"data":       map[string]any{"key": "patched"},
	}))
	objs, err := o.Build()
	require.NoError(t, err)
	require.Len(t, objs, 1)
	cm := &corev1.ConfigMap{}
	unstr, ok := objs[0].(*unstructured.Unstructured)
	require.True(t, ok)
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.Object, cm))
	assert.Equal(t, "patched", cm.Data["key"])
	assert.Equal(t, "true", cm.Labels["original"])
}

func TestAddResourceUnregistered(t *testing.T) {
	o := New(runtime.NewScheme(), nil)
	err := o.AddResource(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "x"}})
	require.Error(t, err)
}

func Test_resourceFileGroupAndNamespace(t *testing.T) {
	o := New(testScheme(), nil)
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"metadata": map[string]any{
			"name":      "x",
			"namespace": "ns",
		},
	}}
	name, err := o.resourceFile(obj)
	require.NoError(t, err)
	assert.Equal(t, "widget-example.com-x-ns.yaml", name)
	gvk := obj.GetObjectKind().GroupVersionKind()
	assert.Equal(t, "example.com", gvk.Group)
	assert.Equal(t, "Widget", gvk.Kind)
}

// toStructured transforms any *unstructured.Unstructured objects into their
// corresponding typed Go structs if they are registered within the provided scheme.
func toStructured(objs []client.Object) []client.Object {
	result := make([]client.Object, len(objs))
	for i, obj := range objs {
		result[i] = obj // Keep the original unless we succeed in structuring
		if unstr, ok := obj.(*unstructured.Unstructured); ok {
			gvk := unstr.GroupVersionKind()
			typedObj, err := testScheme().New(gvk)
			if err != nil {
				continue
			}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.Object, typedObj)
			if err != nil {
				continue
			}
			// Assert the newly populated object matches the controller-runtime client.Object interface
			if clientObj, ok := typedObj.(client.Object); ok {
				result[i] = clientObj
				// Clear the GVK so these will be equal to typed literal structs.
				clientObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{})
			}
		}
	}
	return result
}

package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSubstitutionsAppliedByBuild(t *testing.T) {
	o := newTestOverlay(t)
	objects, err := o.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(objects) == 0 {
		t.Fatal("Build returned no objects")
	}
	for _, obj := range objects {
		checkNoPlaceholders(t, "Build", obj.GetName(), obj.GetNamespace())
	}
}

func TestSubstitutionsAppliedByBuildYAML(t *testing.T) {
	o := newTestOverlay(t)
	data, err := o.BuildYAML()
	if err != nil {
		t.Fatalf("BuildYAML: %v", err)
	}
	if strings.Contains(string(data), "PLACEHOLDER_NS") {
		t.Error("BuildYAML output contains PLACEHOLDER_NS")
	}
	if strings.Contains(string(data), "PLACEHOLDER_NAME") {
		t.Error("BuildYAML output contains PLACEHOLDER_NAME")
	}
}

func TestSubstitutionsAppliedByWriteToDir(t *testing.T) {
	o := newTestOverlay(t)
	dir := t.TempDir()
	if err := o.WriteToDir(dir); err != nil {
		t.Fatalf("WriteToDir: %v", err)
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		content := string(data)
		if strings.Contains(content, "PLACEHOLDER_NS") {
			t.Errorf("file %s contains PLACEHOLDER_NS", rel)
		}
		if strings.Contains(content, "PLACEHOLDER_NAME") {
			t.Errorf("file %s contains PLACEHOLDER_NAME", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking output dir: %v", err)
	}
}

func TestWriteToDirSubstitutesEmbeddedFiles(t *testing.T) {
	o := newTestOverlay(t)
	dir := t.TempDir()
	if err := o.WriteToDir(dir); err != nil {
		t.Fatalf("WriteToDir: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "base/configmap.yaml"))
	if err != nil {
		t.Fatalf("reading embedded file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "actual-namespace") {
		t.Errorf("embedded file not substituted: want actual-namespace, got:\n%s", content)
	}
	if !strings.Contains(content, "actual-name") {
		t.Errorf("embedded file not substituted: want actual-name, got:\n%s", content)
	}
}

func TestWriteToDirSubstitutesOverlayFiles(t *testing.T) {
	o := newTestOverlay(t)
	dir := t.TempDir()
	if err := o.WriteToDir(dir); err != nil {
		t.Fatalf("WriteToDir: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "overlays/generated/extra.yaml"))
	if err != nil {
		t.Fatalf("reading overlay resource: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "PLACEHOLDER_NAME") {
		t.Errorf("overlay resource not substituted, got:\n%s", content)
	}
	if !strings.Contains(content, "actual-name-extra") {
		t.Errorf("overlay resource missing substituted value, got:\n%s", content)
	}
}

func TestNoSubstitutionsIsIdentity(t *testing.T) {
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: keep-as-is\n"),
		},
	}
	o := New(configFS)
	o.SetBase("../../base")

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

// newTestOverlay creates an Overlay with a minimal embedded FS and substitutions
// for testing that placeholders are replaced in all output paths.
func newTestOverlay(t *testing.T) *Overlay {
	t.Helper()
	configFS := fstest.MapFS{
		"base/kustomization.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- configmap.yaml\n"),
		},
		"base/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: PLACEHOLDER_NAME\n  namespace: PLACEHOLDER_NS\n"),
		},
	}
	o := New(configFS)
	o.SetBase("../../base")
	o.AddSubstitution("PLACEHOLDER_NS", "actual-namespace")
	o.AddSubstitution("PLACEHOLDER_NAME", "actual-name")
	o.AddResource("extra.yaml", []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: PLACEHOLDER_NAME-extra\n  namespace: PLACEHOLDER_NS\ndata:\n  key: value\n"))
	return o
}

func checkNoPlaceholders(t *testing.T, method, name, namespace string) {
	t.Helper()
	if strings.Contains(name, "PLACEHOLDER") {
		t.Errorf("%s: object name contains placeholder: %s", method, name)
	}
	if strings.Contains(namespace, "PLACEHOLDER") {
		t.Errorf("%s: object namespace contains placeholder: %s", method, namespace)
	}
}

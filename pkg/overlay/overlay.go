// Package overlay builds Kubernetes resources from a kustomize overlay
// on top of embedded config manifests. It supports components, patches,
// inline resources, namespace transforms, and kustomize replacements.
package overlay

import (
	"fmt"
	"io/fs"
	"iter"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/kustomize/api/filters/namespace"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/yaml"
)

// Overlay assembles a kustomize overlay from a base, components, patches,
// and inline resources, then builds the result into Kubernetes objects.
type Overlay struct {
	scheme       *runtime.Scheme
	configFS     fs.FS
	namespace    string
	ownerName    string
	base         string
	components   []string
	patches      map[string][]byte
	resources    map[string]client.Object
	replacements []types.ReplacementField
	overlayDir   string
}

// OverlayDir is the default directory for the generated overlay.
const OverlayDir = "/overlays/generated"

// New creates an Overlay backed by the given embedded filesystem.
func New(scheme *runtime.Scheme, configFS fs.FS) *Overlay {
	return &Overlay{
		scheme:     scheme,
		configFS:   configFS,
		patches:    make(map[string][]byte),
		resources:  make(map[string]client.Object),
		overlayDir: OverlayDir,
	}
}

// SetNamespace sets the default namespace applied to resources that lack one.
func (o *Overlay) SetNamespace(ns string) {
	o.namespace = ns
}

// SetOwnerName sets the owner name used to apply common labels to all built
// resources. When set, Build adds app.kubernetes.io/managed-by,
// app.kubernetes.io/part-of, and app.kubernetes.io/name labels to every
// resource that does not already carry them.
func (o *Overlay) SetOwnerName(name string) {
	o.ownerName = name
}

// SetBase sets the base kustomization directory (relative to the config FS root).
func (o *Overlay) SetBase(relPath string) {
	o.base = o.relativeToOverlayDir(relPath)
}

// SetOverlayDir sets the directory for the generated overlay. Default is [OverlayDir].
func (o *Overlay) SetOverlayDir(overlayDir string) {
	o.overlayDir = overlayDir
}

// TargetName returns a TargetSelector that matches resources by name.
// Non-existent field paths on a matched resource are silently skipped.
func TargetName(name string, fieldPaths ...string) *types.TargetSelector {
	return &types.TargetSelector{
		Select:     &types.Selector{ResId: resid.ResId{Name: name}},
		FieldPaths: fieldPaths,
	}
}

// TargetKindName returns a TargetSelector that matches resources by kind and name.
func TargetKindName(kind, name string, fieldPaths ...string) *types.TargetSelector {
	return &types.TargetSelector{
		Select:     &types.Selector{ResId: resid.ResId{Gvk: resid.Gvk{Kind: kind}, Name: name}},
		FieldPaths: fieldPaths,
	}
}

// ReplaceValue adds a kustomize replacement that sets value on the specified targets.
// Replacements are processed by kustomize during build.
// Order matters: add cross-reference replacements before metadata.name replacements,
// since changing a resource's name affects how subsequent replacements find it.
func (o *Overlay) ReplaceValue(value string, targets ...*types.TargetSelector) {
	o.replacements = append(o.replacements, types.ReplacementField{
		Replacement: types.Replacement{
			SourceValue: &value,
			Targets:     targets,
		},
	})
}

// AddComponent adds a kustomize component directory (relative to the config FS root).
func (o *Overlay) AddComponent(relPath string) {
	o.components = append(o.components, o.relativeToOverlayDir(relPath))
}

// RemoveComponent removes a previously added component by path.
func (o *Overlay) RemoveComponent(relPath string) {
	resolved := o.relativeToOverlayDir(relPath)
	o.components = slices.DeleteFunc(o.components, func(c string) bool { return c == resolved })
}

// relativeToOverlayDir converts a config-FS-root-relative path to a path
// relative to the generated overlay directory.
func (o *Overlay) relativeToOverlayDir(path string) string {
	rel, _ := filepath.Rel(o.overlayDir, filepath.Clean("/"+path))
	return rel
}

// AddPatch adds a strategic-merge patch, marshaling the value to YAML.
func (o *Overlay) AddPatch(path string, patch any) error {
	data, err := yaml.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshaling patch %s: %w", path, err)
	}
	o.patches[path] = data
	return nil
}

// resourceFile returns a filesystem-safe filename for a resource.
// As a side effect, it sets the GVK in obj.
func (o *Overlay) resourceFile(obj client.Object) (string, error) {
	gvk, err := apiutil.GVKForObject(obj, o.scheme)
	if err != nil {
		return "", err
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	parts := []string{strings.ToLower(gvk.Kind)}
	if gvk.Group != "" {
		parts = append(parts, gvk.Group)
	}
	parts = append(parts, obj.GetName())
	if ns := obj.GetNamespace(); ns != "" {
		parts = append(parts, ns)
	}
	return strings.Join(parts, "-") + ".yaml", nil
}

// AddResource adds an inline resource, marshaling the value to YAML.
func (o *Overlay) AddResource(obj client.Object, dir ...string) error {
	name, err := o.resourceFile(obj)
	if err != nil {
		return err
	}
	fullpath := path.Join(append(dir, name)...)
	o.resources[fullpath] = obj
	return nil
}

func (o *Overlay) isEmpty() bool {
	return o.base == "" && len(o.components) == 0 && len(o.resources) == 0
}

func (o *Overlay) buildResMap() (resmap.ResMap, error) {
	if o.isEmpty() {
		return nil, nil
	}
	fSys := filesys.MakeFsInMemory()

	if o.configFS != nil {
		if err := copyFSToKustomizeFS(fSys, o.configFS, "/"); err != nil {
			return nil, fmt.Errorf("loading config manifests: %w", err)
		}
	}

	if err := fSys.MkdirAll(o.overlayDir); err != nil {
		return nil, fmt.Errorf("creating overlay dir: %w", err)
	}
	if err := o.writeOverlayWith(kustomizeFSWriter(fSys, o.overlayDir)); err != nil {
		return nil, fmt.Errorf("writing overlay: %w", err)
	}

	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := k.Run(fSys, o.overlayDir)
	if err != nil {
		return nil, fmt.Errorf("kustomize build: %w", err)
	}

	if o.namespace != "" {
		nsFilter := namespace.Filter{
			Namespace:              o.namespace,
			UnsetOnly:              true,
			SetRoleBindingSubjects: namespace.AllServiceAccountSubjects,
		}
		for _, r := range resMap.Resources() {
			if err := r.ApplyFilter(nsFilter); err != nil {
				return nil, fmt.Errorf("namespace transform: %w", err)
			}
		}
	}

	return resMap, nil
}

// Build runs the kustomize build and returns the resulting Kubernetes objects.
func (o *Overlay) Build() ([]client.Object, error) {
	resMap, err := o.buildResMap()
	if err != nil {
		return nil, err
	}
	if resMap == nil {
		return nil, nil
	}

	var objects []client.Object
	for _, res := range resMap.Resources() {
		jsonBytes, err := res.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshaling resource %s: %w", res.CurId(), err)
		}
		obj := &unstructured.Unstructured{}
		if err := obj.UnmarshalJSON(jsonBytes); err != nil {
			return nil, fmt.Errorf("unmarshaling resource %s: %w", res.CurId(), err)
		}
		if o.ownerName != "" {
			addCommonLabels(obj, o.ownerName)
		}
		objects = append(objects, obj)
	}
	return sortedNamespaceFirst(slices.Values(objects)), nil
}

// BuildYAML runs the kustomize build and returns the result as multi-document YAML.
func (o *Overlay) BuildYAML() ([]byte, error) {
	resMap, err := o.buildResMap()
	if err != nil {
		return nil, err
	}
	if resMap == nil {
		return nil, nil
	}
	return resMap.AsYaml()
}

// WriteToDir writes the full kustomize bundle to configDir.
// The output contains the embedded manifests (observabilityinstaller/, uiplugins/) and
// an overlays/generated/ overlay that can be built with `kustomize build configDir/overlays/generated/`.
func (o *Overlay) WriteToDir(configDir string) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	if o.configFS != nil {
		if err := copyFSToDir(configDir, o.configFS); err != nil {
			return fmt.Errorf("copying config manifests: %w", err)
		}
	}
	return o.writeOverlayWith(dirFileWriter(filepath.Join(configDir, o.overlayDir)))
}

type fileWriter func(path string, content []byte) error

func dirFileWriter(dir string) fileWriter {
	return func(path string, content []byte) error {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		return os.WriteFile(full, content, 0o644)
	}
}

func copyFSToDir(dir string, src fs.FS) error {
	return fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, path)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func kustomizeFSWriter(fSys filesys.FileSystem, dir string) fileWriter {
	return func(path string, content []byte) error {
		full := filepath.Join(dir, path)
		if err := fSys.MkdirAll(filepath.Dir(full)); err != nil {
			return err
		}
		return fSys.WriteFile(full, content)
	}
}

func (o *Overlay) writeOverlayWith(write fileWriter) error {
	kust := types.Kustomization{
		TypeMeta: types.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
	}
	if o.base != "" {
		kust.Resources = append(kust.Resources, o.base)
	}
	kust.Components = o.components
	kust.Replacements = o.replacements
	kust.Resources = append(kust.Resources, slices.Sorted(maps.Keys(o.resources))...)
	for _, path := range slices.Sorted(maps.Keys(o.patches)) {
		kust.Patches = append(kust.Patches, types.Patch{Path: path})
	}

	kustYAML, err := yaml.Marshal(kust)
	if err != nil {
		return fmt.Errorf("marshaling kustomization: %w", err)
	}
	if err := write("kustomization.yaml", kustYAML); err != nil {
		return err
	}

	for _, path := range slices.Sorted(maps.Keys(o.patches)) {
		if err := write(path, o.patches[path]); err != nil {
			return err
		}
	}
	for _, path := range slices.Sorted(maps.Keys(o.resources)) {
		data, err := yaml.Marshal(o.resources[path])
		if err != nil {
			return fmt.Errorf("marshaling resource %s: %w", path, err)
		}
		if err := write(path, data); err != nil {
			return err
		}
	}
	return nil
}

func addCommonLabels(obj client.Object, ownerName string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	want := map[string]string{
		"app.kubernetes.io/managed-by": "observability-operator",
		"app.kubernetes.io/part-of":    ownerName,
		"app.kubernetes.io/name":       obj.GetName(),
	}
	for k, v := range want {
		if _, ok := labels[k]; !ok {
			labels[k] = v
		}
	}
	obj.SetLabels(labels)
}

func copyFSToKustomizeFS(fSys filesys.FileSystem, embedded fs.FS, root string) error {
	return fs.WalkDir(embedded, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(root, path)
		if d.IsDir() {
			return fSys.MkdirAll(target)
		}
		data, err := fs.ReadFile(embedded, path)
		if err != nil {
			return err
		}
		return fSys.WriteFile(target, data)
	})
}

func sortedNamespaceFirst(objs iter.Seq[client.Object]) []client.Object {
	return slices.SortedFunc(objs, cmpNamespaceFirst)
}

func cmpNamespaceFirst(a, b client.Object) int {
	switch {
	case isNamespace(a) && !isNamespace(b):
		return -1
	case isNamespace(b) && !isNamespace(a):
		return 1
	default:
		return 0
	}
}

func isNamespace(o client.Object) bool {
	switch o := o.(type) {
	case *corev1.Namespace:
		return true
	case *unstructured.Unstructured:
		return o.GetKind() == "Namespace"
	default:
		return false
	}
}

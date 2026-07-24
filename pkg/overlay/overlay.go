package overlay

import (
	"cmp"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filters/namespace"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

const (
	overlayDir = "/overlays/generated"
)

type Overlay struct {
	configFS      fs.FS
	namespace     string
	base          string
	components    []string
	patches       map[string][]byte
	resources     map[string][]byte
	substitutions map[string]string
}

func New(configFS fs.FS) *Overlay {
	return &Overlay{
		configFS:      configFS,
		patches:       make(map[string][]byte),
		resources:     make(map[string][]byte),
		substitutions: make(map[string]string),
	}
}

func (o *Overlay) SetNamespace(ns string) {
	o.namespace = ns
}

func (o *Overlay) SetBase(relPath string) {
	o.base = relPath
}

func (o *Overlay) AddSubstitution(from, to string) {
	if to != "" {
		o.substitutions[from] = to
	}
}

func (o *Overlay) AddComponent(relPath string) {
	o.components = append(o.components, relPath)
}

func (o *Overlay) RemoveComponent(relPath string) {
	o.components = slices.DeleteFunc(o.components, func(c string) bool { return c == relPath })
}

func (o *Overlay) AddPatch(path string, content []byte) {
	o.patches[path] = content
}

func (o *Overlay) AddResource(path string, content []byte) {
	o.resources[path] = content
}

func (o *Overlay) AddPatchMap(path string, patch map[string]any) error {
	data, err := yaml.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshaling patch %s: %w", path, err)
	}
	o.patches[path] = data
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

	if err := fSys.MkdirAll(overlayDir); err != nil {
		return nil, fmt.Errorf("creating overlay dir: %w", err)
	}
	if err := o.writeOverlayWith(kustomizeFSWriter(fSys, overlayDir)); err != nil {
		return nil, fmt.Errorf("writing overlay: %w", err)
	}

	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := k.Run(fSys, overlayDir)
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

func (o *Overlay) replacer() *strings.Replacer {
	keys := slices.SortedFunc(maps.Keys(o.substitutions), func(a, b string) int {
		return cmp.Compare(len(b), len(a))
	})
	pairs := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		pairs = append(pairs, k, o.substitutions[k])
	}
	return strings.NewReplacer(pairs...)
}

func (o *Overlay) Build() ([]client.Object, error) {
	resMap, err := o.buildResMap()
	if err != nil {
		return nil, err
	}
	if resMap == nil {
		return nil, nil
	}

	r := o.replacer()
	var objects []client.Object
	for _, res := range resMap.Resources() {
		jsonBytes, err := res.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshaling resource %s: %w", res.CurId(), err)
		}
		obj := &unstructured.Unstructured{}
		if err := obj.UnmarshalJSON([]byte(r.Replace(string(jsonBytes)))); err != nil {
			return nil, fmt.Errorf("unmarshaling resource %s: %w", res.CurId(), err)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func (o *Overlay) BuildYAML() ([]byte, error) {
	resMap, err := o.buildResMap()
	if err != nil {
		return nil, err
	}
	if resMap == nil {
		return nil, nil
	}
	yamlBytes, err := resMap.AsYaml()
	if err != nil {
		return nil, err
	}
	return []byte(o.replacer().Replace(string(yamlBytes))), nil
}

// WriteToDir writes the full kustomize bundle to configDir.
// The output contains the embedded manifests (observabilityinstaller/, uiplugins/) and
// an overlays/generated/ overlay that can be built with `kustomize build configDir/overlays/generated/`.
func (o *Overlay) WriteToDir(configDir string) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	if err := os.CopyFS(configDir, o.configFS); err != nil {
		return fmt.Errorf("copying config manifests: %w", err)
	}
	return o.writeOverlayWith(dirFileWriter(filepath.Join(configDir, overlayDir)))
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
		if err := write(path, o.resources[path]); err != nil {
			return err
		}
	}
	return nil
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

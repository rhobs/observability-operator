package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

// resourceSet collects the output objects of a generation run. Objects are
// deduplicated by the unique YAML filename used when they are written to file.
type resourceSet struct {
	scheme *runtime.Scheme
	byFile map[string]client.Object
}

func newResourceSet(scheme *runtime.Scheme) *resourceSet {
	return &resourceSet{scheme: scheme, byFile: map[string]client.Object{}}
}

// AddResource adds obj to the set, no op if a matching obj (ns, name, gvk) is already in the set.
// If the object has no GroupVersionKind set, the scheme is used to populate it.
func (s *resourceSet) AddResource(obj client.Object) error {
	if obj.GetObjectKind().GroupVersionKind().Kind == "" {
		gvks, _, err := s.scheme.ObjectKinds(obj)
		if err != nil {
			return fmt.Errorf("cannot determine GVK for %T %q: %w", obj, obj.GetName(), err)
		}
		obj.GetObjectKind().SetGroupVersionKind(gvks[0])
	}
	s.byFile[resourceFileName(obj)] = obj
	return nil
}

// Sort by namespace, with non-namespace objects first, then by kind and name.
// Result is sorted by namespace, kind, name
func (s *resourceSet) Build() []client.Object {
	objects := make([]client.Object, 0, len(s.byFile))
	for _, obj := range s.byFile {
		objects = append(objects, obj)
	}
	slices.SortFunc(objects, util.CompareObjects)
	return objects
}

func (s *resourceSet) WriteToDir(dir string) error {
	return writeObjectsToDir(dir, s.Build())
}

func writeObjectsToDir(dir string, objects []client.Object) error {
	for _, obj := range objects {
		data, err := kyaml.Marshal(obj)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, resourceFileName(obj)), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func resourceFileName(obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	kind := strings.ToLower(gvk.Kind)
	if gvk.Group != "" {
		kind = gvk.Group + "_" + kind
	}
	if ns := obj.GetNamespace(); ns != "" {
		return fmt.Sprintf("%s-%s.%s.yaml", kind, obj.GetName(), ns)
	}
	return fmt.Sprintf("%s-%s.yaml", kind, obj.GetName())
}

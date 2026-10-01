package api

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Path is an immutable helper around a filesystem path that makes it convenient
// to build up nested collection directories and write files.
type Path struct {
	string
}

// NewPath creates a Path from the given path segments.
func NewPath(parts ...string) Path {
	return Path{filepath.Join(parts...)}
}

// Add returns a new Path with the given segments appended.
func (p Path) Add(parts ...string) Path {
	parts = append([]string{p.string}, parts...)
	p.string = filepath.Join(parts...)
	return p
}

// ForResource returns a new Path with the group and resource appended, mirroring
// the layout produced by `oc adm inspect`.
func (p Path) ForResource(gvr schema.GroupVersionResource) Path {
	p.string = filepath.Join(p.string, gvr.Group, gvr.Resource)
	return p
}

// WithSuffix returns a new Path with the given suffix appended to the last
// segment (e.g. ".json"). Unlike Add it does not introduce a path separator.
func (p Path) WithSuffix(suffix string) Path {
	p.string += suffix
	return p
}

// String returns the underlying path.
func (p Path) String() string {
	return p.string
}

// MkdirAll creates the directory named by the Path along with any necessary parents.
func (p Path) MkdirAll() error {
	if err := os.MkdirAll(p.string, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", p.string, err)
	}
	return nil
}

// WriteFile writes data to the file named by the Path, creating parent
// directories as needed.
func (p Path) WriteFile(data []byte) error {
	dir := filepath.Dir(p.string)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(p.string, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", p.string, err)
	}
	return nil
}

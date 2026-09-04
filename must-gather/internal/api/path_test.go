package api

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestPathAdd(t *testing.T) {
	p := NewPath("base")
	assert.Equal(t, "base", p.String())

	child := p.Add("a", "b")
	assert.Equal(t, filepath.Join("base", "a", "b"), child.String())

	// Add must not mutate the receiver (Path is value-immutable).
	assert.Equal(t, "base", p.String())
}

func TestPathWithSuffix(t *testing.T) {
	p := NewPath("dir", "object")
	assert.Equal(t, filepath.Join("dir", "object")+".json", p.WithSuffix(".json").String())
	// original unchanged
	assert.Equal(t, filepath.Join("dir", "object"), p.String())
}

func TestPathForResource(t *testing.T) {
	p := NewPath("base")
	gvr := schema.GroupVersionResource{Group: "monitoring.rhobs", Version: "v1alpha1", Resource: "monitoringstacks"}
	assert.Equal(t, filepath.Join("base", "monitoring.rhobs", "monitoringstacks"), p.ForResource(gvr).String())
}

func TestPathMkdirAll(t *testing.T) {
	tmp := t.TempDir()
	p := NewPath(tmp, "a", "b", "c")
	assert.NilError(t, p.MkdirAll())

	info, err := os.Stat(p.String())
	assert.NilError(t, err)
	assert.Assert(t, info.IsDir())
}

func TestPathWriteFileCreatesParents(t *testing.T) {
	tmp := t.TempDir()
	p := NewPath(tmp, "nested", "deeper", "result.json")

	assert.NilError(t, p.WriteFile([]byte("hello")))

	data, err := os.ReadFile(p.String())
	assert.NilError(t, err)
	assert.Equal(t, "hello", string(data))
}

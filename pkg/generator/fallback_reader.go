package generator

import (
	"context"
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FallbackReader implements client.Reader with a set of pre-loaded objects.
// Get checks the pre-loaded set first, then falls back to the underlying Reader.
type FallbackReader struct {
	preloaded map[preloadKey]client.Object
	reader    client.Reader
}

type preloadKey struct {
	key     client.ObjectKey
	objType reflect.Type
}

var _ client.Reader = &FallbackReader{}

func NewFallbackReader(reader client.Reader, preloaded ...client.Object) *FallbackReader {
	r := &FallbackReader{preloaded: make(map[preloadKey]client.Object, len(preloaded)), reader: reader}
	for _, o := range preloaded {
		pk := preloadKey{key: client.ObjectKeyFromObject(o), objType: reflect.TypeOf(o).Elem()}
		r.preloaded[pk] = o
	}
	return r
}

func (r *FallbackReader) Get(ctx context.Context, key client.ObjectKey, into client.Object, opts ...client.GetOption) error {
	pk := preloadKey{key: key, objType: reflect.TypeOf(into).Elem()}
	if obj, ok := r.preloaded[pk]; ok {
		reflect.ValueOf(into).Elem().Set(reflect.ValueOf(obj).Elem())
		return nil
	}
	if r.reader != nil {
		return r.reader.Get(ctx, key, into, opts...)
	}
	return fmt.Errorf("not found: (%T) %v", into, key)
}

func (r *FallbackReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if r.reader != nil {
		return r.reader.List(ctx, list, opts...)
	}
	return nil
}

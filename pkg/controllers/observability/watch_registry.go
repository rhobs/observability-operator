package observability

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type watchRegistry struct {
	mu         sync.Mutex
	registered map[schema.GroupVersionKind]bool
}

func newWatchRegistry() *watchRegistry {
	return &watchRegistry{registered: map[schema.GroupVersionKind]bool{}}
}

// Pending reports whether any object still needs to be registered. It allows
// callers to skip the expensive API discovery needed by RegisterAvailable.
// Objects with an undeterminable kind are reported as pending so that
// RegisterAvailable returns the error.
func (r *watchRegistry) Pending(objects []client.Object, scheme *runtime.Scheme) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, object := range objects {
		gvks, _, err := scheme.ObjectKinds(object)
		if err != nil || len(gvks) == 0 || !r.registered[gvks[0]] {
			return true
		}
	}
	return false
}

// RegisterAvailable registers each object at most once when its kind is served
// by the API server. Failed registrations are deliberately left unmarked for
// retry. Availability must be checked per kind, not per group: capability CRDs
// can share a group with CRDs that are always present.
func (r *watchRegistry) RegisterAvailable(objects []client.Object, kinds map[schema.GroupVersionKind]bool, scheme *runtime.Scheme, register func(client.Object) error) error {
	for _, object := range objects {
		gvks, _, err := scheme.ObjectKinds(object)
		if err != nil {
			return fmt.Errorf("determine watch type for %T: %w", object, err)
		}
		if len(gvks) == 0 {
			return fmt.Errorf("determine watch type for %T: no GVK registered", object)
		}
		gvk := gvks[0]
		if !kinds[gvk] {
			continue
		}

		r.mu.Lock()
		if r.registered[gvk] {
			r.mu.Unlock()
			continue
		}
		if err := register(object); err != nil {
			r.mu.Unlock()
			return fmt.Errorf("register watch for %s: %w", gvk, err)
		}
		r.registered[gvk] = true
		r.mu.Unlock()
	}
	return nil
}

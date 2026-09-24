package storage

import "sigs.k8s.io/controller-runtime/pkg/client"

// Objects returns the materialized resources, omitting absent optional objects.
func (resources *Resources) Objects() []client.Object {
	var objects []client.Object
	if resources.ObjectStorage != nil {
		objects = append(objects, resources.ObjectStorage)
	}
	if resources.ObjectStorageTLS != nil {
		objects = append(objects, resources.ObjectStorageTLS)
	}
	if resources.ObjectStorageCA != nil {
		objects = append(objects, resources.ObjectStorageCA)
	}
	return objects
}

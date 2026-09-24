package v1alpha1

// TracingSpec defines the desired state of the tracing capability.
// +kubebuilder:validation:XValidation:rule="(!has(self.enabled) || !self.enabled) || (has(self.storage) && has(self.storage.objectStorage) && [has(self.storage.objectStorage.s3), has(self.storage.objectStorage.s3STS), has(self.storage.objectStorage.s3CCO), has(self.storage.objectStorage.azure), has(self.storage.objectStorage.azureWIF), has(self.storage.objectStorage.gcs), has(self.storage.objectStorage.gcsWIF)].filter(x, x).size() > 0)",message="Storage configuration is required when tracing is enabled"
type TracingSpec struct {
	CommonCapabilitiesSpec `json:",inline"`

	// Storage defines the storage for the tracing capability
	Storage *TracingStorageSpec `json:"storage,omitempty"`
}

func (t *TracingSpec) GetStorage() *TracingStorageSpec {
	if t != nil {
		return t.Storage
	}
	return nil
}

// TracingStorageSpec defines the storage for tracing capability.
type TracingStorageSpec struct {
	// ObjectStorageSpec defines the object storage configuration for tracing.
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Object storage config"
	ObjectStorageSpec *ObjectStorageSpec `json:"objectStorage,omitempty"`
}

func (s *TracingStorageSpec) GetObjectStorageSpec() *ObjectStorageSpec {
	if s != nil {
		return s.ObjectStorageSpec
	}
	return nil
}

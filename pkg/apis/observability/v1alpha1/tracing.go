package v1alpha1

// TracingSpec defines the desired state of the tracing capability.
// +kubebuilder:validation:XValidation:rule="(!has(self.enabled) || !self.enabled) || [has(self.storage.objectStorage.s3), has(self.storage.objectStorage.s3STS), has(self.storage.objectStorage.s3CCO), has(self.storage.objectStorage.azure), has(self.storage.objectStorage.azureWIF), has(self.storage.objectStorage.gcs), has(self.storage.objectStorage.gcsWIF)].filter(x, x).size() > 0",message="Storage configuration is required when tracing is enabled"
type TracingSpec struct {
	CommonCapabilitiesSpec `json:",inline"`

	// Storage defines the storage for the tracing capability
	Storage TracingStorageSpec `json:"storage,omitempty"`
}

// TracingStorageSpec defines the storage for tracing capability.
type TracingStorageSpec struct {
	// ObjectStorageSpec defines the object storage configuration for tracing.
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Object storage config"
	ObjectStorageSpec TracingObjectStorageSpec `json:"objectStorage,omitempty"`
}

// TracingObjectStorageSpec defines the object storage for the tracing capability.
// +kubebuilder:validation:XValidation:rule="[has(self.s3), has(self.s3STS), has(self.s3CCO), has(self.azure), has(self.azureWIF), has(self.gcs), has(self.gcsWIF)].filter(x, x).size() <= 1",message="Only one or zero storage configurations can be specified"
type TracingObjectStorageSpec struct {
	// S3 defines the S3 object storage configuration.
	S3 *S3Spec `json:"s3,omitempty"`
	// S3STS defines the S3 object storage configuration using short-lived credentials.
	S3STS *S3STSpec `json:"s3STS,omitempty"`
	// S3CCO defines the S3 object storage configuration using CCO.
	S3CCO *S3CCOSpec `json:"s3CCO,omitempty"`

	// Azure defines the Azure Blob Storage configuration.
	Azure *AzureSpec `json:"azure,omitempty"`
	// AzureWIF defines the Azure Blob Storage configuration using a Workload Identity Federation.
	AzureWIF *AzureWIFSpec `json:"azureWIF,omitempty"`

	// GCS defines the Google Cloud Storage configuration.
	GCS *GCSSpec `json:"gcs,omitempty"`
	// GCSSToken defines the Google Cloud Storage configuration using short-lived tokens.
	GCSSTSSpec *GCSWIFSpec `json:"gcsWIF,omitempty"`

	// TLS configuration for reaching the object storage endpoint.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS Config"
	TLS *TLSSpec `json:"tls,omitempty"`
}

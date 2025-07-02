package v1alpha1

// TracingSpec defines the desired state of the tracing capability.
type TracingSpec struct {
	CommonCapabilitiesSpec `json:",inline"`

	// Storage defines the storage for the tracing capability
	Storage TracingStorageSpec `json:"storage,omitempty"`
}

// TracingStorageSpec defines the storage for tracing capability.
type TracingStorageSpec struct {
	// ObjectStorageSpec defines the object storage configuration for tracing.
	ObjectStorageSpec TracingObjectStorageSpec `json:"objectStorage,omitempty"`
}

// TracingObjectStorageSpec defines the object storage for the tracing capability.
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

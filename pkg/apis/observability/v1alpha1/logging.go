package v1alpha1

// LoggingSpec defines the desired state of the logging capability.
// When enabled, logging creates a UIPlugin resource for the OpenShift Console
// logging view. The UIPlugin is shared across ObservabilityInstallers and points
// only to the LokiStack created by the first Installer with logging enabled.
// +kubebuilder:validation:XValidation:rule="(!has(self.enabled) || !self.enabled) || (has(self.lokiStack) && has(self.lokiStack.objectStorage) && [has(self.lokiStack.objectStorage.s3), has(self.lokiStack.objectStorage.s3STS), has(self.lokiStack.objectStorage.s3CCO), has(self.lokiStack.objectStorage.azure), has(self.lokiStack.objectStorage.azureWIF), has(self.lokiStack.objectStorage.gcs), has(self.lokiStack.objectStorage.gcsWIF)].filter(x, x).size() > 0)",message="Object storage configuration is required when logging is enabled"
// +kubebuilder:validation:XValidation:rule="!has(self.lokiStack) || !has(self.lokiStack.objectStorage) || !has(self.lokiStack.objectStorage.azureWIF) || has(self.lokiStack.objectStorage.azureWIF.subscriptionID)",message="Azure workload identity federation requires subscriptionID for logging"
type LoggingSpec struct {
	CommonCapabilitiesSpec `json:",inline"`

	// LokiStack configures a LokiStack store for log data.
	// +optional
	// +kubebuilder:validation:Optional
	LokiStack *LokiStackSpec `json:"lokiStack,omitempty"`
}

func (s *LoggingSpec) GetLokiStack() *LokiStackSpec {
	if s == nil {
		return nil
	}
	return s.LokiStack
}

type LokiStackSpec struct {
	// ObjectStorage configures an object storage bucket.
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Object storage config"
	ObjectStorage *ObjectStorageSpec `json:"objectStorage,omitempty"`

	// Storage class name defines the storage class for ingester/querier PVCs.
	//
	// +required
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:StorageClass",displayName="Storage Class Name"
	StorageClassName string `json:"storageClassName"`

	// Size defines one of the support Loki deployment scale out sizes.
	//
	// +required
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:1x.pico","urn:alm:descriptor:com.tectonic.ui:select:1x.extra-small","urn:alm:descriptor:com.tectonic.ui:select:1x.small","urn:alm:descriptor:com.tectonic.ui:select:1x.medium"},displayName="LokiStack Size"
	Size LokiStackSizeType `json:"size"`

	// Schemas for reading and writing logs.
	//
	// +optional
	// +kubebuilder:validation:Optional
	Schemas []ObjectStorageSchema `json:"schemas,omitempty"`
}

// LokiStackSizeType is the deployment scale-out size understood by LokiStack.
// +kubebuilder:validation:Enum="1x.demo";"1x.pico";"1x.extra-small";"1x.small";"1x.medium"
type LokiStackSizeType string

const (
	SizeOneXDemo       LokiStackSizeType = "1x.demo"
	SizeOneXPico       LokiStackSizeType = "1x.pico"
	SizeOneXExtraSmall LokiStackSizeType = "1x.extra-small"
	SizeOneXSmall      LokiStackSizeType = "1x.small"
	SizeOneXMedium     LokiStackSizeType = "1x.medium"
)

// ObjectStorageSchemaVersion is a Loki object storage schema version.
// +kubebuilder:validation:Enum=v11;v12;v13
type ObjectStorageSchemaVersion string

const (
	ObjectStorageSchemaV11 ObjectStorageSchemaVersion = "v11"
	ObjectStorageSchemaV12 ObjectStorageSchemaVersion = "v12"
	ObjectStorageSchemaV13 ObjectStorageSchemaVersion = "v13"
)

// ObjectStorageSchema defines a schema version and the date when it will become effective.
type ObjectStorageSchema struct {
	// Version for writing and reading logs.
	// +required
	// +kubebuilder:validation:Required
	Version ObjectStorageSchemaVersion `json:"version"`

	// EffectiveDate contains a date in YYYY-MM-DD format which is interpreted in the UTC time zone.
	//
	// The configuration always needs at least one schema that is currently valid. This means that when creating a new
	// LokiStack it is recommended to add a schema with the latest available version and an effective date of "yesterday".
	// New schema versions added to the configuration always needs to be placed "in the future", so that Loki can start
	// using it once the day rolls over.
	// +required
	// +kubebuilder:validation:Required
	EffectiveDate StorageSchemaEffectiveDate `json:"effectiveDate"`
}

// StorageSchemaEffectiveDate is a date in YYYY-MM-DD format.
// +kubebuilder:validation:Pattern:="^([0-9]{4,})([-]([0-9]{2})){2}$"
type StorageSchemaEffectiveDate string

func (s *LokiStackSpec) GetStorage() *ObjectStorageSpec {
	if s == nil {
		return nil
	}
	return s.ObjectStorage
}

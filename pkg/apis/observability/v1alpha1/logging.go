package v1alpha1

import (
	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
)

// LoggingSpec defines the desired state of the logging capability.
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
	Size lokiv1.LokiStackSizeType `json:"size"`

	// Schemas for reading and writing logs.
	//
	// +optional
	// +kubebuilder:validation:Optional
	Schemas []lokiv1.ObjectStorageSchema `json:"schemas,omitempty"`
}

func (s *LokiStackSpec) GetStorage() *ObjectStorageSpec {
	if s == nil {
		return nil
	}
	return s.ObjectStorage
}

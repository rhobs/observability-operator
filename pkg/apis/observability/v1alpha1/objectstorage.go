package v1alpha1

type S3Spec struct {
	// Bucket is the name of the S3 bucket.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
	// Endpoint is the S3 endpoint URL.
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`
	// AccessKeyID is the access key ID for the S3 bucket.
	// +kubebuilder:validation:Required
	AccessKeyID string `json:"accessKeyID"`
	// AccessKeySecret is a reference to a secret containing the access key secret for the S3.
	// +kubebuilder:validation:Required
	AccessKeySecret SecretKeySelector `json:"accessKeySecret,omitempty"`
	// Region is the region where the S3 bucket is located.
	// +kubebuilder:validation:Optional
	Region string `json:"region,omitempty"`
}

type S3STSpec struct {
	// Bucket is the name of the S3 bucket.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
	// RoleARN is the ARN of the IAM role to assume for accessing the S3 bucket.
	// +kubebuilder:validation:Required
	RoleARN string `json:"roleARN"`
	// Region is the region where the S3 bucket is located.
	// +kubebuilder:validation:Optional
	Region string `json:"region,omitempty"`
}

type S3CCOSpec struct {
	// Bucket is the name of the S3 bucket.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
	// Region is the region where the S3 bucket is located.
	// +kubebuilder:validation:Optional
	Region string `json:"region,omitempty"`
}

type AzureSpec struct {
	// Container is the name of the Azure Blob Storage container.
	// +kubebuilder:validation:Required
	Container string `json:"container"`
	// AccountName is the name of the Azure Storage account.
	// +kubebuilder:validation:Required
	AccountName string `json:"accountName"`
	// AccountKey is a reference to a secret containing the account key for the Azure Storage account.
	// +kubebuilder:validation:Required
	AccountKeySecret SecretKeySelector `json:"accountKeySecret"`
}

type AzureWIFSpec struct {
	// Container is the name of the Azure Blob Storage container.
	// +kubebuilder:validation:Required
	Container string `json:"container"`
	// AccountName is the name of the Azure Storage account.
	// +kubebuilder:validation:Required
	AccountName string `json:"accountName"`
	// ClientID is the client ID of the Azure Active Directory application.
	// +kubebuilder:validation:Required
	ClientID string `json:"clientID"`
	// TenantID is the tenant ID of the Azure Active Directory.
	// +kubebuilder:validation:Required
	TenantID string `json:"tenantID"`
	// Audience is the optional audience for the Azure Workload Identity Federation.
	// +kubebuilder:validation:Optional
	Audience string `json:"audience,omitempty"` // Optional audience for the Azure WIF
}

type GCSSpec struct {
	// Bucket is the name of the Google Cloud Storage bucket.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
	// KeyJSON is the key.json file encoded in a secret.
	// +kubebuilder:validation:Required
	KeyJSONSecret SecretKeySelector `json:"keyJSONSecret"`
}

type GCSWIFSpec struct {
	// Bucket is the name of the Google Cloud Storage bucket.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
	// KeyJSON is the key.json file encoded in a secret.
	// +kubebuilder:validation:Required
	KeyJSONSecret SecretKeySelector `json:"keyJSONSecret"`
	// Audience is the optional audience.
	// +kubebuilder:validation:Optional
	Audience string `json:"audience,omitempty"`
}

// SecretKeySelector encodes a reference to a single key in a Secret in the same namespace.
type SecretKeySelector struct {
	// Key contains the name of the key inside the referenced Secret.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Key Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Key string `json:"key"`

	// SecretName contains the name of the Secret containing the referenced value.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Name string `json:"name"`
}

// ConfigMapKeySelector encodes a reference to a single key in a ConfigMap in the same namespace.
type ConfigMapKeySelector struct {
	// Key contains the name of the key inside the referenced Secret.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Key Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Key string `json:"key"`

	// SecretName contains the name of the Secret containing the referenced value.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Name string `json:"name"`
}

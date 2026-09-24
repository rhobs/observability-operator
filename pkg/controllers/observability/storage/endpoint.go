package storage

import (
	"net/url"
	"strings"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

// HasHTTPSEndpoint reports whether static S3 storage uses HTTPS.
func HasHTTPSEndpoint(storageSpec obsv1alpha1.ObjectStorageSpec) bool {
	if storageSpec.S3 == nil {
		return false
	}
	endpoint := strings.TrimSpace(storageSpec.S3.Endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "https")
}

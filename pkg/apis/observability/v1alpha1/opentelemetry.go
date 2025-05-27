package v1alpha1

type OpenTelemetrySpec struct {
	Enabled bool `json:"enabled,omitempty"`

	Exporter OTLPExporter `json:"exporter,omitempty"`
}

type OTLPExporter struct {
	Endpoint string `json:"endpoint,omitempty"`
}

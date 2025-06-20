package v1alpha1

// OpenTelemetrySpec defines the desired state of OpenTelemetry capability.
type OpenTelemetrySpec struct {
	CommonCapabilitiesSpec `json:",inline"`

	// Exporter defines the OpenTelemetry exporter configuration.
	// When defined the collector will export telemetry data to the specified endpoint.
	Exporter *OTLPExporter `json:"exporter,omitempty"`
}

// OTLPExporter defines the OpenTelemetry Protocol (OTLP) exporter configuration.
type OTLPExporter struct {
	// Endpoint is the OTLP endpoint.
	Endpoint string `json:"endpoint,omitempty"`
}

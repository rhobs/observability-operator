# ClusterObservability CRD

This document describes the `ClusterObservability` Custom Resource Definition (CRD).
The goal of this CRD is to provide end-to-end observability capabilities with minimal configuration.
Power users should be able to customize the underlying components via server-side apply.

## Examples

### Logging and tracing

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
metadata:
  name: logging-tracing
spec:
  storage:
    secret:
      name: minio
      type: s3
  capabilities:
    logging:
      enabled: true
    tracing:
      enabled: true
```

Notes:
* installs the Loki, ClusterLogForwarder, Tempo and opentelemetry operators
* creates storage secret for `LokiStack` and `TempoStack` from the secret `minio` which is reconciled by the `ClusterObservability`
* deploys logging stack with `ClusterLogForwarder` and `LokiStack` in the `openshift-logging` namespace
* deploys tracing stack with `OpenTelemetryCollector` and `TempoStack` in the `openshift-distributed-tracing` namespace
* Installs the UI plugins for Loki and Tempo
* The appropriate operators are installed only when given capability is enabled

### OpenTelemetry with tracing and Dynatrace

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
metadata:
  name: logging-tracing
spec:
  storage:
    secret:
      name: minio
      type: s3
  capabilities:
    tracing:
      enabled: true
    opentelemetry:
      enabled: true
      tracesincluster: true 
      exporter:
        endpoint: http://dynatrace:4317
        headers:
          x-dynatrace: "token..."
```

Notes:
* installs the opentelemetry and tempo operators
* deploys tracing stack with `OpenTelemetryCollector` and `TempoStack` in the `openshift-distributed-tracing` namespace
* deploys `OpenTelemetryCollector` in the `openshift-opentelemetry`
* configures OTLP exporter on the collector to send traces to Dynatrace
* configures collector to export trace data to Tempo deployed by the `ClusterObservability` CR

### Install only operators for a given capability

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
metadata:
  name: logging-tracing
spec:
  storage:
    secret:
      name: minio
      type: s3
  capabilities:
    tracing:
      enabled: false
      operators:
        install: true
```

Notes:
* The tracing instance is not deployed, but the operators are installed

### Deploy capability but don't deploy the operators.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
metadata:
  name: logging-tracing
spec:
  storage:
    secret:
      name: minio
      type: s3
  capabilities:
    tracing:
      enabled: true
      operators:
        install: false
```

Notes:
* The tracing instance is deployed, but the operators are not installed via COO.
* In this case, the user is responsible for installing the operators

In this case the COO cannot guarantee that installed operator versions are compatible therefore we could forbit this configuration or show a warning/unmanaged state.

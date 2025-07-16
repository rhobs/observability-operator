# ClusterObservability CRD

This document describes the `ClusterObservability` Custom Resource Definition (CRD).
The goal of this CRD is to provide end-to-end observability capabilities with minimal configuration.
Power users should be able to customize the underlying components via server-side apply.

## Setup

The `ClusterObservability` CRD is not by default enabled. To enable it, you need to perform the following steps:

* Uncomment https://github.com/rhobs/observability-operator/blob/c4564860c698cb8201d368c02de21650a8b7034c/deploy/crds/common/kustomization.yaml#L7
* Enable `--openshift.enabled=true` https://github.com/rhobs/observability-operator/blob/edd13e0a43cab1e74536b01f86ea5cb4ff7fe897/cmd/operator/main.go#L107

```bash

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

## Storage configuration

The storage section of the `ClusterObservability` CRD allows users to configure the storage for all supported observability backends.
At the moment, the only supported backed is Tempo (tracing capability). There are plans to support Loki (logging capability) and Prometheus/Thanos (metrics capability) in the future.
Therefore, the storage configuration has to be flexible and work for all backend types.

Goals:
* Allow users to uniformly configure the storage for all supported observability backends
* Unified storage configuration will abstract away the differences between the storage configuration of different observability backends
* Allow users to use different storage configuration for different observability backends

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
metadata:
  name: example
spec:
  storage:
    pvc:
      storageClassName: "" # Empty defaults to the cluster default storage class.
      storageSize: "" # .
    objectStorage:
      secret:
        name: minio
        type: s3
      tls:
        enabled: false
        ca:
        cert:
        key:
        minimumTLSVersion:
  capabilities:
    tracing:
      enabled: true
    logging:
      enabled: true
```

* In the above example the tracing and logging capabilities will use `minio` secret for connecting to the object storage.
* The controller transforms the `minio` secret into secrets required by the `LokiStack` and `TempoStack` instances. For example, the `TempoStack` requires fields `bucket` and `LokiStack` uses the field `bucketnames`.

The various storage configuration per capability can be achieved by multiple `ClusterObservability` CRs, each with its own storage configuration.

### Object storage types

Each object storage type has its own set of required fields which are configured in the secret.
Object storage types will be added separately, there are plans to support the all the storage types supported by the capabilities.

#### S3 secret

```yaml
kubectl create secret generic s3 \
    --from-literal=bucket="<BUCKET_NAME>" \
    --from-literal=endpoint="<AWS_BUCKET_ENDPOINT>" \
    --from-literal=access_key_id="<AWS_ACCESS_KEY_ID>" \
    --from-literal=access_key_secret="<AWS_ACCESS_KEY_SECRET>" \
    --from-literal=region="<AWS_REGION_YOUR_BUCKET_LIVES_IN>"
```

### References:
* Loki https://loki-operator.dev/docs/object_storage.md/, https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/logging/logging-6-2#log6x-logging-loki-cli-install_installing-logging-6-2
* Tempo https://grafana.com/docs/tempo/latest/setup/operator/object-storage/ and https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/distributed_tracing/distr-tracing-tempo-installing#distr-tracing-tempo-object-storage-setup_distr-tracing-tempo-installing
* Thanos object storage https://github.com/thanos-io/objstore
* `ClusterLogForwarder`'s `ValueReference` and `SecretReference` https://github.com/openshift/cluster-logging-operator/blob/master/api/observability/v1/clusterlogforwarder_types.go#L267

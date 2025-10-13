# ObservabilityInstaller CRD

The `ObservabilityInstaller` Custom Resource Definition (CRD) provides end-to-end observability capabilities with minimal configuration. It simplifies the deployment and management of observability components such as tracing, and the future logging, monitoring g and OpenTelemetry.

## Overview

The ObservabilityInstaller CRD enables:
- **Simple Configuration**: Deploy complete observability stacks with minimal YAML
- **Flexible Capabilities**: Enable/disable tracing, and in the future other capabilities like logging and OpenTelemetry
- **Unified Storage**: Configure storage consistently across all observability backends
- **Operator Management**: Control whether operators are installed automatically
- **Power User Customization**: Advanced users can customize underlying components via server-side apply

The following example demonstrates a basic setup with tracing capability enabled, using MinIO as the object storage backend.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: tracing
  namespace: observability
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
```

## Quick Start Examples

### Install Operators Only

The following CR installs the tracing operators without deploying any tracing instance.
This use-case is useful for users who want to manage the tracing instances themselves, but make sure the 
installed operators are compatible.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: operators-only
  namespace: observability
spec:
  capabilities:
    tracing:
      enabled: false
      operators:
        install: true
      storage:
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
```

### Deploy Without Installing Operators

The following CR deploys the tracing instance but does not install operators. Users are responsible for ensuring compatible operators are installed.

**Note**: When operators are not managed by COO, version compatibility cannot be guaranteed. Consider this configuration carefully.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: no-operators
  namespace: observability
spec:
  capabilities:
    tracing:
      enabled: true
      operators:
        install: false
      storage:
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
```

## Storage configuration

The storage section of the `ObservabilityInstaller` CRD allows users to configure the storage for all supported observability backends.
At the moment, the only supported backed is Tempo (tracing capability). There are plans to support Loki (logging capability) and Prometheus/Thanos (metrics capability) in the future.
Therefore, the storage configuration has to be flexible and work for all backend types.

Goals:
* Allow users to uniformly configure the storage for all supported observability backends
* Unified storage configuration will abstract away the differences between the storage configuration of different observability backends
* Allow users to use different storage configuration for different observability backends

The following CR is for illustration purposes only, not all the fields might be implemented.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: example
  namespace: observability
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        pvc:
          storageClassName: "" # Empty defaults to the cluster default storage class.
          storageSize: "" # .
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
            region: us-east-1
          tls:
            enabled: false
            ca:
            cert:
            key:
            minimumTLSVersion:
```

* In the above example the tracing capability will use S3 as object storage.
* The controller transforms the configuration in `s3` secret into secret required the `TempoStack` instance.

### Object storage types

Each object storage type has its own set of required fields which are configured directly in the CR.
There are plans to support all storage types required by the supported capabilities.

#### Design principles

The `ObservabilityInstaller` object storage configuration is directly held in the CR and secrets are used only to reference sensitive data.

#### Alternative thanos-io/objstore

The https://github.com/thanos-io/objstore client is used by Loki and Thanos to connect to the object storage.
It offers a unified interface for different object storage types with a common configuration file.

At the time of writing this document, the COO does not support Thanos, Loki operator does not directly support the `thanos-io/objstore`
configuration file and the Tempo does not support it at all.

#### Amazon S3 S3 / MinIO

Supported by Tempo and Loki.

```yaml
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3:
            bucket: bucket-name
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
            region: us-east-1
```

```bash
kubectl create secret generic minio-secret \
--from-literal=access_key_secret="supersecret"
```

##### Secret supported by Tempo and Loki operators

```bash
kubectl create secret generic storage-secret \
    --from-literal=bucket="<BUCKET_NAME>" \
    --from-literal=endpoint="<AWS_BUCKET_ENDPOINT>" \
    --from-literal=access_key_id="<AWS_ACCESS_KEY_ID>" \
    --from-literal=access_key_secret="<AWS_ACCESS_KEY_SECRET>" \
    --from-literal=region="<AWS_REGION_YOUR_BUCKET_LIVES_IN>"
```

* `region` - is optional in Tempo and required by Loki.

####  Amazon S3 with Security Token Service (STS) - Short lived

Supported by Tempo and Loki.

```yaml
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3STS:
            bucket: bucket-name
            roleARN: 
            region: us-east-1
```

##### Secret supported by Tempo and Loki operators

```bash
kubectl create secret generic storage-secret \
--from-literal=bucket="<BUCKET_NAME>" \
--from-literal=role_arn="<AWS_ROLE_ARN>" \
--from-literal=region="<AWS_REGION_YOUR_BUCKET_LIVES_IN>"
```

#### Amazon S3 with Cluster Credentials Operator (CCO) - Short lived

Supported by Tempo.

```yaml
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3CCO:
            bucket: bucket-name
            region: us-east-1
```

##### Secret supported by Tempo and Loki operators

```bash
kubectl create secret generic storage-secret \
--from-literal=bucket="<BUCKET_NAME>" \
--from-literal=region="<AWS_REGION_YOUR_BUCKET_LIVES_IN>"
```

#### Microsoft Azure Blob Storage

Supported by Tempo and Loki.

```yaml
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          azure:
            container:
            accountName:
            accountKeySecret:
              name: azure-secret
              key: account_key
```

```bash
kubectl create secret generic azure-secret \
--from-literal=account_key="<ACCOUNT_KEY>"
```

##### Secret supported by Tempo and Loki operators

```bash
kubectl create secret generic storage-secret \
--from-literal=container="<BLOB_STORAGE_CONTAINER_NAME>" \
--from-literal=account_name="<BLOB_STORAGE_ACCOUNT_NAME>" \
--from-literal=account_key="<BLOB_STORAGE_ACCOUNT_KEY>"
```

Loki operator also supports fields:
* `environment`
* `endpoint_suffix` - optional

#### Azure WIF - Short lived

Supported by Tempo.

```yaml
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          azureWIF:
            container:
            accountName:
            audience:
            clientID:
            tenantID:
```

##### Secret supported by Tempo and Loki operators

```bash
kubectl create secret generic storage-secret \
--from-literal=container="<BLOB_STORAGE_CONTAINER_NAME>" \
--from-literal=account_name="<BLOB_STORAGE_ACCOUNT_NAME>" \
--from-literal=audience="<AUDIENCE>" \
--from-literal=client_id="CLIENT_ID>" \
--from-literal=tenant_id="<TENANT_ID>"
```

* `audience` - optional and defaults to `api://AzureADTokenExchange`

#### Google Cloud Storage

Supported by Tempo and Loki.

```yaml
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          gcs:
            bucket: bucket-name
            keyJSONSecret:
              name: gcs-secret
              key: key.json
```

```bash
kubectl -n $NAMESPACE create secret generic gcs-secret \
--from-file=key.json="$GCS_KEY_FILE_PATH"
```

##### Secret supported by Tempo and Loki operators

```yaml
kubectl create secret generic storage-secret \
--from-literal=bucketname="<BUCKET_NAME>" \
--from-literal=key.json="<PATH_TO_JSON_KEY_FILE>"
```

#### Google Cloud Storage WIF - Short lived

Supported by Tempo.

```yaml
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          gcsWIF:
            bucket: bucket-name
            keyJSONSecret:
              name: gcs-secret
              key: key.json
            audience: # optional
```

```bash
kubectl -n $NAMESPACE create secret generic gcs-secret \
--from-file=key.json="$GCS_KEY_FILE_PATH"
```

## Future Work

### Logging capability

This example shows the logging capability and how multiple capabilities can be configured in a single CR using the same object storage backend.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: logging-tracing
  namespace: observability
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
    logging:
      enabled: true
      storage:
        objectStorage:
          s3:
            bucket: loki
            endpoint: http://minio.minio.svc:9000
            accessKeyID: loki
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
```

This configuration:
- Installs Loki, ClusterLogForwarder, Tempo, and OpenTelemetry operators
- Creates storage secrets for `LokiStack` and `TempoStack` from the referenced secrets
- Deploys logging stack with `ClusterLogForwarder` and `LokiStack` in the `observability` namespace
- Deploys tracing stack with `OpenTelemetryCollector` and `TempoStack` in the `observability` namespace
- Installs UI plugins for Loki and Tempo
- Only installs operators for enabled capabilities

### OpenTelemetry capability

This example shows how to deploy OpenTelemetry with an external tracing backend (Dynatrace) while also deploying a local Tempo instance for trace storage.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: otel-dynatrace
  namespace: observability
spec:
  capabilities:
    tracing:
      enabled: true
      storage:
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
    opentelemetry:
      enabled: true
      tracesincluster: true
      exporter:
        endpoint: http://dynatrace:4317
        headers:
          x-dynatrace: "token..."
```

This configuration:
- Installs OpenTelemetry and Tempo operators
- Deploys tracing stack with `OpenTelemetryCollector` and `TempoStack` in the `observability` namespace
- Deploys `OpenTelemetryCollector` in the `openshift-opentelemetry` namespace
- Configures OTLP exporter on the collector to send traces to Dynatrace
- Also exports trace data to the local Tempo instance

## References:
* Loki https://loki-operator.dev/docs/object_storage.md/, https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/logging/logging-6-2#log6x-logging-loki-cli-install_installing-logging-6-2
* Tempo https://grafana.com/docs/tempo/latest/setup/operator/object-storage/ and https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/distributed_tracing/distr-tracing-tempo-installing#distr-tracing-tempo-object-storage-setup_distr-tracing-tempo-installing
* Thanos object storage https://github.com/thanos-io/objstore
* Thanos operator https://prometheus-operator.dev/docs/platform/thanos/
* `ClusterLogForwarder`'s `ValueReference` and `SecretReference` https://github.com/openshift/cluster-logging-operator/blob/master/api/observability/v1/clusterlogforwarder_types.go#L267

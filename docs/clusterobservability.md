# ClusterObservability CRD

This document describes the `ClusterObservability` Custom Resource Definition (CRD).
The goal of this CRD is to provide end-to-end observability capabilities with minimal configuration.
Power users should be able to customize the underlying components via server-side apply.

```bash

## Examples

### Logging and tracing

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
metadata:
  name: logging-tracing
  namespace: observability
spec:
  capabilities:
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

Notes:
* installs the Loki, ClusterLogForwarder, Tempo and opentelemetry operators
* creates storage secret for `LokiStack` and `TempoStack` from the secret `minio` which is reconciled by the `ClusterObservability`
* deploys logging stack with `ClusterLogForwarder` and `LokiStack` in the `observability` namespace
* deploys tracing stack with `OpenTelemetryCollector` and `TempoStack` in the `observability` namespace
* Installs the UI plugins for Loki and Tempo
* The appropriate operators are installed only when given capability is enabled

### OpenTelemetry with tracing and Dynatrace

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
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
* deploys tracing stack with `OpenTelemetryCollector` and `TempoStack` in the `observability` namespace
* deploys `OpenTelemetryCollector` in the `openshift-opentelemetry`
* configures OTLP exporter on the collector to send traces to Dynatrace
* configures collector to export trace data to Tempo deployed by the `ClusterObservability` CR

### Install only operators for a given capability

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
metadata:
  name: logging-tracing
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

Notes:
* The tracing instance is not deployed, but the operators are installed

### Deploy capability but don't deploy the operators.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ClusterObservability
metadata:
  name: logging-tracing
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
            bucket: bucket-name
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
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
    logging:
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

* In the above example the tracing and logging capabilities will use S3 as object storage.
* The controller transforms the configuration in `s3` secret into secrets required by the `LokiStack` and `TempoStack` instances.

The various storage configuration per capability can be achieved by multiple `ClusterObservability` CRs, each with its own storage configuration.

### Object storage types

Each object storage type has its own set of required fields which are configured directly in the CR.
There are plans to support all storage types required by the capabilities.

#### Design principles

The `ClusterObservability` object storage configuration is directly held in the CR and secrets are used only to reference sensitive data.

##### Alternative thanos-io/objstore

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

#### Short lived - Amazon S3 with Security Token Service (STS)

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

#### Short lived - Amazon S3 with Cluster Credentials Operator (CCO)

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

#### Google Cloud Storage with WIF - Short lived

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

### References:
* Loki https://loki-operator.dev/docs/object_storage.md/, https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/logging/logging-6-2#log6x-logging-loki-cli-install_installing-logging-6-2
* Tempo https://grafana.com/docs/tempo/latest/setup/operator/object-storage/ and https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html/distributed_tracing/distr-tracing-tempo-installing#distr-tracing-tempo-object-storage-setup_distr-tracing-tempo-installing
* Thanos object storage https://github.com/thanos-io/objstore
* Thanos operator https://prometheus-operator.dev/docs/platform/thanos/
* `ClusterLogForwarder`'s `ValueReference` and `SecretReference` https://github.com/openshift/cluster-logging-operator/blob/master/api/observability/v1/clusterlogforwarder_types.go#L267

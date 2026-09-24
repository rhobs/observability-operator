# ObservabilityInstaller CRD

`ObservabilityInstaller` is a Technology Preview, namespace-scoped resource for
deploying observability components on OpenShift. It supports:

- **Tracing**: OpenTelemetry Collector, TempoStack, RBAC, storage resources, and
  the distributed-tracing console plugin.
- **Logging**: LokiStack, ClusterLogForwarder for application and infrastructure
  logs, RBAC, storage resources, and the logging console plugin.

The Cluster Observability Operator must run with its OpenShift feature gate
enabled. See the [generated API documentation](api.md) for the complete schema.

## Capability lifecycle

Capabilities are disabled by default. Their common fields are `enabled` and
`operators.install`:

| `enabled` | `operators.install` | Result |
| --- | --- | --- |
| `false` or omitted | `false` or omitted | No operands or COO-managed operators |
| `false` or omitted | `true` | Install operators only |
| `true` | any value | Install operators and reconcile operands |

Enabling a capability always requests its operators, even when
`operators.install` is `false`. COO reuses compatible subscriptions managed by
another component and removes only subscriptions it manages.

Disabling a capability removes its owned operands. Operator subscriptions are
shared across all `ObservabilityInstaller` resources and remain while any
installer needs them. Console UIPlugins are create-only: COO neither overwrites
an existing plugin nor deletes one when a capability is disabled or removed.

## Tracing

Tracing requires one object-storage provider when enabled:

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
        objectStorage:
          s3:
            bucket: tempo
            endpoint: http://minio.minio.svc:9000
            accessKeyID: tempo
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
            region: us-east-1
```

This creates an `OpenTelemetryCollector` and a `TempoStack` named `example` in
the installer namespace. Tempo uses an OpenShift tenant named `application`.

## Logging

Logging requires a `lokiStack` and one object-storage provider.
`storageClassName` and `size` are required; `schemas` follows the Loki Operator
schema:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: example
  namespace: observability
spec:
  capabilities:
    logging:
      enabled: true
      lokiStack:
        storageClassName: gp3-csi
        size: 1x.pico
        schemas:
          - version: v13
            effectiveDate: "2024-10-01"
        objectStorage:
          s3:
            bucket: loki
            endpoint: http://minio.minio.svc:9000
            accessKeyID: loki
            accessKeySecret:
              name: minio-secret
              key: access_key_secret
            region: us-east-1
```

This creates `example-loki` (LokiStack), `example-logging`
(ClusterLogForwarder), and collector RBAC. The generated forwarder collects
application and infrastructure logs, but not audit logs.

Tracing and logging can be enabled in the same resource. Each capability has
independent storage configuration and may use a different provider or bucket.

## Installing operators only

Operators-only mode does not require storage configuration:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: ObservabilityInstaller
metadata:
  name: operators-only
  namespace: observability
spec:
  capabilities:
    tracing:
      operators:
        install: true
    logging:
      operators:
        install: true
```

## Object storage

An enabled capability requires exactly one provider in its `objectStorage`
block. Tracing and logging accept the same provider types, subject to support by
the installed Tempo and Loki Operator versions:

| Field | Authentication | Required values |
| --- | --- | --- |
| `s3` | Static | `bucket`, `endpoint`, `accessKeyID`, `accessKeySecret`; optional `region` |
| `s3STS` | AWS STS | `bucket`, `roleARN`; optional `region` |
| `s3CCO` | OpenShift CCO | `bucket`; optional `region` |
| `azure` | Static | `container`, `accountName`, `accountKeySecret` |
| `azureWIF` | Workload identity | `container`, `accountName`, `clientID`, `tenantID`; optional `audience` |
| `gcs` | Service account | `bucket`, `keyJSONSecret` |
| `gcsWIF` | Workload identity | `bucket`, `keyJSONSecret`; optional `audience` |

Secret selectors identify a Secret and key in the installer namespace. For
example:

```yaml
objectStorage:
  s3:
    bucket: bucket-name
    endpoint: https://s3.example.com
    accessKeyID: access-key-id
    accessKeySecret:
      name: s3-credentials
      key: secret-access-key
```

COO copies referenced values into operand-specific Secrets; credentials are not
stored in the `ObservabilityInstaller` resource. All referenced Secrets and
ConfigMaps must be in the installer namespace.

### TLS

TLS is configured beside the selected provider:

```yaml
objectStorage:
  s3:
    bucket: bucket-name
    endpoint: https://s3.example.com
    accessKeyID: access-key-id
    accessKeySecret:
      name: s3-credentials
      key: secret-access-key
  tls:
    caConfigMap:
      name: storage-ca
      key: service-ca.crt
    certSecret:
      name: storage-client-tls
      key: tls.crt
    keySecret:
      name: storage-client-tls
      key: tls.key
    minVersion: VersionTLS12
```

`certSecret` and `keySecret` must be specified together. Tempo supports all
shown TLS fields. The generated LokiStack currently uses only `caConfigMap` and
otherwise relies on system trust roots. An HTTPS S3 endpoint enables Tempo TLS
even without a `tls` block.

## Status

The following status fields are shown as `kubectl get` columns:

- `status.opentelemetry` and `status.tempo` for tracing;
- `status.lokistack` and `status.logging` for logging.

Tracing reports observed operand versions. Logging is reported when the
ClusterLogForwarder has `Ready=True`. Missing operands trigger a retry; other
status-read failures appear in the top-level `Reconciled` condition.

## Limitations

- Only tracing and logging are supported; there is no separate OpenTelemetry or
  metrics capability.
- The CR requires OpenShift APIs and the OpenShift feature gate.
- Create-only UIPlugins are not updated or garbage-collected by this controller.

## References

- [Generated API documentation](api.md)
- [How to add a capability](add-a-capability.md)
- [Loki Operator object storage](https://loki-operator.dev/docs/object_storage.md/)
- [Tempo object storage](https://grafana.com/docs/tempo/latest/setup/operator/object-storage/)


# Table of contents
1. [Understanding Server Side Apply](#understanding-ssa)
2. [Using SSA to customize `Prometheus` resource generated from MonitoringStack](#customize-prometheus)
    1. [Setup](#setup)
    2. [Modifying a field not controlled by MonitoringStack](#not-controlled-by-MS)
    3. [Modifying a field managed by MonitoringStack](#controlled-by-MS)
3. [Caveats](#caveats)

# Understanding Server Side Apply <a name="understanding-ssa"></a>
Server Side Apply [\[1\]](#ref-ssa-k8s) allows declarative configuration management by updating a resource's state without needing to delete and recreate it, and Field Management allows users to specify which fields of a resource they want to update, without affecting the other fields. 

[1] [Server Side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/)<a name="ref-ssa-k8s"></a>

#  Using SSA to customize `Prometheus` resource generated from MonitoringStack <a name="customize-prometheus"></a>


## Setup <a name="setup"></a>
 - Setup a cluster with following MonitoringStack spec

```yaml
apiVersion: monitoring.rhobs/v1alpha1
kind: MonitoringStack
metadata:
  labels:
    obo: example
  name: sample-monitoring-stack
  namespace: obo-demo
spec:
  logLevel: debug
  retention: 1d
  resourceSelector:
    matchLabels:
      app: demo
```

This will generate a Prometheus resource named `sample-monitoring-stack` in `obo-demo` namespace.

Get the managed fields of the generated Prometheus resource using command 
```sh
oc -n obo-demo get Prometheus.monitoring.rhobs -oyaml --show-managed-fields
```

```yaml
managedFields:
- apiVersion: monitoring.rhobs/v1
  fieldsType: FieldsV1
  fieldsV1:
    f:metadata:
      f:labels:
        f:app.kubernetes.io/managed-by: {}
        f:app.kubernetes.io/name: {}
        f:app.kubernetes.io/part-of: {}
      f:ownerReferences:
        k:{"uid":"81da0d9a-61aa-4df3-affc-71015bcbde5a"}: {}
    f:spec:
      f:additionalScrapeConfigs: {}
      f:affinity:
        f:podAntiAffinity:
          f:requiredDuringSchedulingIgnoredDuringExecution: {}
      f:alerting:
        f:alertmanagers: {}
      f:arbitraryFSAccessThroughSMs: {}
      f:logLevel: {}
      f:podMetadata:
        f:labels:
          f:app.kubernetes.io/component: {}
          f:app.kubernetes.io/part-of: {}
      f:podMonitorSelector: {}
      f:replicas: {}
      f:resources:
        f:limits:
          f:cpu: {}
          f:memory: {}
        f:requests:
          f:cpu: {}
          f:memory: {}
      f:retention: {}
      f:ruleSelector: {}
      f:rules:
        f:alert: {}
      f:securityContext:
        f:fsGroup: {}
        f:runAsNonRoot: {}
        f:runAsUser: {}
      f:serviceAccountName: {}
      f:serviceMonitorSelector: {}
      f:thanos:
        f:baseImage: {}
        f:resources: {}
        f:version: {}
      f:tsdb: {}
  manager: observability-operator
  operation: Apply
- apiVersion: monitoring.rhobs/v1
  fieldsType: FieldsV1
  fieldsV1:
    f:status:
      .: {}
      f:availableReplicas: {}
      f:conditions:
        .: {}
        k:{"type":"Available"}:
          .: {}
          f:lastTransitionTime: {}
          f:observedGeneration: {}
          f:status: {}
          f:type: {}
        k:{"type":"Reconciled"}:
          .: {}
          f:lastTransitionTime: {}
          f:observedGeneration: {}
          f:status: {}
          f:type: {}
      f:paused: {}
      f:replicas: {}
      f:shardStatuses:
        .: {}
        k:{"shardID":"0"}:
          .: {}
          f:availableReplicas: {}
          f:replicas: {}
          f:shardID: {}
          f:unavailableReplicas: {}
          f:updatedReplicas: {}
      f:unavailableReplicas: {}
      f:updatedReplicas: {}
  manager: PrometheusOperator
  operation: Update
  subresource: status
```

Check the `metadata.managedFields` values, and observe that some fields in `metadata` and `spec` are managed by MonitoringStack.

## Modifying a field not controlled by MonitoringStack <a name="not-controlled-by-MS"></a>

 - Change `spec.enforcedSampleLimit`, a field not set by MonitoringStack using the following yaml and command

```yaml
apiVersion: monitoring.rhobs/v1
kind: Prometheus
metadata:
  name: sample-monitoring-stack
  namespace: obo-demo
spec:
  enforcedSampleLimit: 1000
```

```sh
$ oc apply -f ./prom-spec-edited.yaml --server-side 
prometheus.monitoring.rhobs/sample-monitoring-stack serverside-applied
```

**Note**: Must use the `--server-side` flag.

Get the changed Prometheus object with managedFields, and note that object contains the added field, and there is one more section in `managedFields` which has `spec.enforcedSampleLimit`

```yaml
managedFields:
- apiVersion: monitoring.rhobs/v1
  fieldsType: FieldsV1
  fieldsV1:
    f:metadata:
      f:labels:
        f:app.kubernetes.io/managed-by: {}
        f:app.kubernetes.io/name: {}
        f:app.kubernetes.io/part-of: {}
    f:spec:
      f:enforcedSampleLimit: {}
  manager: kubectl
  operation: Apply
```

## Modifying a field managed by MonitoringStack <a name="controlled-by-MS"></a>

 - Change `spec.LogLevel`, a field managed by MonitoringStack using the following yaml and command

```yaml
# changing the logLevel from debug to info
apiVersion: monitoring.rhobs/v1
kind: Prometheus
metadata:
  name: sample-monitoring-stack
  namespace: obo-demo
spec:
  logLevel: info
```


```sh
$ oc apply -f ./prom-spec-edited.yaml --server-side 
error: Apply failed with 1 conflict: conflict with "observability-operator": .spec.logLevel
Please review the fields above--they currently have other managers. Here
are the ways you can resolve this warning:
* If you intend to manage all of these fields, please re-run the apply
  command with the `--force-conflicts` flag.
* If you do not intend to manage all of the fields, please edit your
  manifest to remove references to the fields that should keep their
  current managers.
* You may co-own fields by updating your manifest to match the existing
  value; in this case, you'll become the manager if the other manager(s)
  stop managing the field (remove it from their configuration).
See https://kubernetes.io/docs/reference/using-api/server-side-apply/#conflicts
```

 Notice that the field `spec.logLevel` cannot be changed using server side apply, because it is already managed by `observability-operator`.

 Use `--force-conflicts` flag to force the change.

```sh
$ oc apply -f ./prom-spec-edited.yaml --server-side --force-conflicts 
prometheus.monitoring.rhobs/sample-monitoring-stack serverside-applied
```

 With `--force-conflicts` flag, the field can be forced to change, but since the same field is also managed by MonitoringStack, the Observability operator will detect the change, and revert it back to the value set by MonitoringStack.

 Note: Some Prometheus fields generated by MonitoringStack are influenced by fields in MonitoringStack spec itself, e.g. `logLevel`. These can be changed by changing the MonitoringStack spec.

 For example, to change the logLevel in Prometheus object, apply the following yaml.

```yaml
apiVersion: monitoring.rhobs/v1alpha1
kind: MonitoringStack
metadata:
  name: sample-monitoring-stack
  labels:
    obo: example
spec:
  logLevel: info
```

Verify by:

```sh
$ oc -n obo-demo get Prometheus.monitoring.rhobs -o=jsonpath='{.items[0].spec.logLevel}' 
info
```

# Caveats <a name="caveats"></a>
## New version of Operator generates field controlled by user
Consider a scenario in which user is managing a particular field which is not generated by MonitoringStack, say `enforcedSampleLimit`. Now if the Observability operator is upgraded, and the new version of operator generates a value for `enforcedSampleLimit`, the value set by user will be overwritten.

## Fields with default value
The `Prometheus` object generated by MonitoringStack may contain some fields which are not explicitly set by MonitoringStack. These fields appear because they have a default value. e.g. `scrapeInterval`. These fields though appear in the `Prometheus` object but can be changed by user.


# API Reference

Packages:

- [monitoring.rhobs/v1alpha1](#monitoringrhobsv1alpha1)

# monitoring.rhobs/v1alpha1

Resource Types:

- [MonitoringStack](#monitoringstack)

- [ThanosQuerier](#thanosquerier)




## MonitoringStack
<sup><sup>[↩ Parent](#monitoringrhobsv1alpha1 )</sup></sup>






MonitoringStack is the Schema for the monitoringstacks API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>monitoring.rhobs/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>MonitoringStack</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspec">spec</a></b></td>
        <td>object</td>
        <td>
          MonitoringStackSpec is the specification for desired Monitoring Stack<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>object</td>
        <td>
          MonitoringStackStatus defines the observed state of MonitoringStack. It should always be reconstructable from the state of the cluster and/or outside world.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec
<sup><sup>[↩ Parent](#monitoringstack)</sup></sup>



MonitoringStackSpec is the specification for desired Monitoring Stack

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>logLevel</b></td>
        <td>enum</td>
        <td>
          Loglevel set log levels of configured components<br/>
          <br/>
            <i>Enum</i>: debug, info, warning<br/>
            <i>Default</i>: info<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfig">prometheusConfig</a></b></td>
        <td>object</td>
        <td>
          Define prometheus config<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecresourceselector">resourceSelector</a></b></td>
        <td>object</td>
        <td>
          Label selector for Monitoring Stack Resources.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecresources">resources</a></b></td>
        <td>object</td>
        <td>
          Define resources requests and limits for Monitoring Stack Pods.<br/>
          <br/>
            <i>Default</i>: map[limits:map[cpu:500m memory:512M] requests:map[cpu:100m memory:256M]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>retention</b></td>
        <td>string</td>
        <td>
          Time duration to retain data for. Default is '120h', and must match the regular expression `[0-9]+(ms|s|m|h|d|w|y)` (milliseconds seconds minutes hours days weeks years).<br/>
          <br/>
            <i>Default</i>: 120h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig
<sup><sup>[↩ Parent](#monitoringstackspec)</sup></sup>



Define prometheus config

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaim">persistentVolumeClaim</a></b></td>
        <td>object</td>
        <td>
          Define persistent volume claim for prometheus<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfig)</sup></sup>



Define persistent volume claim for prometheus

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>
          AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>
          This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source. If the AnyVolumeDataSource feature gate is enabled, this field will always have the same contents as the DataSourceRef field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimdatasourceref">dataSourceRef</a></b></td>
        <td>object</td>
        <td>
          Specifies the object from which to populate the volume with data, if a non-empty volume is desired. This may be any local object from a non-empty API group (non core object) or a PersistentVolumeClaim object. When this field is specified, volume binding will only succeed if the type of the specified object matches some installed volume populator or dynamic provisioner. This field will replace the functionality of the DataSource field and as such if both fields are non-empty, they must have the same value. For backwards compatibility, both fields (DataSource and DataSourceRef) will be set to the same value automatically if one of them is empty and the other is non-empty. There are two important differences between DataSource and DataSourceRef: * While DataSource only allows two specific types of objects, DataSourceRef   allows any non-core object, as well as PersistentVolumeClaim objects. * While DataSource ignores disallowed values (dropping them), DataSourceRef   preserves all values, and generates an error if a disallowed value is   specified. (Alpha) Using this field requires the AnyVolumeDataSource feature gate to be enabled.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources represents the minimum resources the volume should have. If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements that are lower than previous value but must still be higher than capacity recorded in the status field of the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimselector">selector</a></b></td>
        <td>object</td>
        <td>
          A label query over volumes to consider for binding.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>
          Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>
          volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          VolumeName is the binding reference to the PersistentVolume backing this claim.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.dataSource
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaim)</sup></sup>



This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source. If the AnyVolumeDataSource feature gate is enabled, this field will always have the same contents as the DataSourceRef field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is the type of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>
          APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.dataSourceRef
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaim)</sup></sup>



Specifies the object from which to populate the volume with data, if a non-empty volume is desired. This may be any local object from a non-empty API group (non core object) or a PersistentVolumeClaim object. When this field is specified, volume binding will only succeed if the type of the specified object matches some installed volume populator or dynamic provisioner. This field will replace the functionality of the DataSource field and as such if both fields are non-empty, they must have the same value. For backwards compatibility, both fields (DataSource and DataSourceRef) will be set to the same value automatically if one of them is empty and the other is non-empty. There are two important differences between DataSource and DataSourceRef: * While DataSource only allows two specific types of objects, DataSourceRef   allows any non-core object, as well as PersistentVolumeClaim objects. * While DataSource ignores disallowed values (dropping them), DataSourceRef   preserves all values, and generates an error if a disallowed value is   specified. (Alpha) Using this field requires the AnyVolumeDataSource feature gate to be enabled.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is the type of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>
          APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.resources
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaim)</sup></sup>



Resources represents the minimum resources the volume should have. If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements that are lower than previous value but must still be higher than capacity recorded in the status field of the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.selector
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaim)</sup></sup>



A label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaimselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.resourceSelector
<sup><sup>[↩ Parent](#monitoringstackspec)</sup></sup>



Label selector for Monitoring Stack Resources.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#monitoringstackspecresourceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.resourceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#monitoringstackspecresourceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.resources
<sup><sup>[↩ Parent](#monitoringstackspec)</sup></sup>



Define resources requests and limits for Monitoring Stack Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## ThanosQuerier
<sup><sup>[↩ Parent](#monitoringrhobsv1alpha1 )</sup></sup>






ThanosQuerier outlines the Thanos querier components, managed by this stack

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>monitoring.rhobs/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ThanosQuerier</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#thanosquerierspec">spec</a></b></td>
        <td>object</td>
        <td>
          ThanosQuerierSpec defines a single Thanos Querier instance. This means a label selector by which Monitoring Stack instances to query are selected, and an optional namespace selector and a list of replica labels by which to deduplicate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>object</td>
        <td>
          ThanosQuerierStatus defines the observed state of ThanosQuerier. It should always be reconstructable from the state of the cluster and/or outside world.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ThanosQuerier.spec
<sup><sup>[↩ Parent](#thanosquerier)</sup></sup>



ThanosQuerierSpec defines a single Thanos Querier instance. This means a label selector by which Monitoring Stack instances to query are selected, and an optional namespace selector and a list of replica labels by which to deduplicate.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#thanosquerierspecselector">selector</a></b></td>
        <td>object</td>
        <td>
          Selector to select Monitoring stacks to unify<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#thanosquerierspecnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          Selector to select which namespaces the Monitoring Stack objects are discovered from.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicaLabels</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ThanosQuerier.spec.selector
<sup><sup>[↩ Parent](#thanosquerierspec)</sup></sup>



Selector to select Monitoring stacks to unify

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#thanosquerierspecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ThanosQuerier.spec.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#thanosquerierspecselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ThanosQuerier.spec.namespaceSelector
<sup><sup>[↩ Parent](#thanosquerierspec)</sup></sup>



Selector to select which namespaces the Monitoring Stack objects are discovered from.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>any</b></td>
        <td>boolean</td>
        <td>
          Boolean describing whether all namespaces are selected in contrast to a list restricting them.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchNames</b></td>
        <td>[]string</td>
        <td>
          List of namespace names.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
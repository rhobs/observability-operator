# API Reference

Packages:

- [monitoring.rhobs/v1alpha1](#monitoringrhobsv1alpha1)
- [observability.openshift.io/v1alpha1](#observabilityopenshiftiov1alpha1)

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
        <td><b><a href="#monitoringstackstatus">status</a></b></td>
        <td>object</td>
        <td>
          MonitoringStackStatus defines the observed state of MonitoringStack.
It should always be reconstructable from the state of the cluster and/or outside world.<br/>
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
        <td><b><a href="#monitoringstackspecalertmanagerconfig">alertmanagerConfig</a></b></td>
        <td>object</td>
        <td>
          Define Alertmanager config<br/>
          <br/>
            <i>Default</i>: map[disabled:false]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logLevel</b></td>
        <td>enum</td>
        <td>
          Loglevel set log levels of configured components<br/>
          <br/>
            <i>Enum</i>: debug, info, warn, error<br/>
            <i>Default</i>: info<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          Namespace selector for Monitoring Stack Resources.
To monitor everything, set to empty map selector. E.g. namespaceSelector: {}.
To monitor resources in the namespace where Monitoring Stack was created in, set to null. E.g. namespaceSelector:.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfig">prometheusConfig</a></b></td>
        <td>object</td>
        <td>
          Define prometheus config<br/>
          <br/>
            <i>Default</i>: map[replicas:2]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecresourceselector">resourceSelector</a></b></td>
        <td>object</td>
        <td>
          Label selector for Monitoring Stack Resources.
To monitor everything, set to empty map selector. E.g. resourceSelector: {}.
To disable service discovery, set to null. E.g. resourceSelector:.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecresources">resources</a></b></td>
        <td>object</td>
        <td>
          Define resources requests and limits for Monitoring Stack Pods.<br/>
          <br/>
            <i>Default</i>: map[limits:map[cpu:500m memory:512Mi] requests:map[cpu:100m memory:256Mi]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>retention</b></td>
        <td>string</td>
        <td>
          Time duration to retain data for. Default is '120h',
and must match the regular expression `[0-9]+(ms|s|m|h|d|w|y)` (milliseconds seconds minutes hours days weeks years).<br/>
          <br/>
            <i>Default</i>: 120h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.alertmanagerConfig
<sup><sup>[↩ Parent](#monitoringstackspec)</sup></sup>



Define Alertmanager config

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
        <td><b>disabled</b></td>
        <td>boolean</td>
        <td>
          Disables the deployment of Alertmanager.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.namespaceSelector
<sup><sup>[↩ Parent](#monitoringstackspec)</sup></sup>



Namespace selector for Monitoring Stack Resources.
To monitor everything, set to empty map selector. E.g. namespaceSelector: {}.
To monitor resources in the namespace where Monitoring Stack was created in, set to null. E.g. namespaceSelector:.

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
        <td><b><a href="#monitoringstackspecnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#monitoringstackspecnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

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
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
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
        <td><b>enableOtlpHttpReceiver</b></td>
        <td>boolean</td>
        <td>
          Enable Prometheus to accept OpenTelemetry Metrics via the otlp/http protocol.
Defaults to the value of `false`.
The resulting endpoint is /api/v1/otlp/v1/metrics.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableRemoteWriteReceiver</b></td>
        <td>boolean</td>
        <td>
          Enable Prometheus to be used as a receiver for the Prometheus remote write protocol. Defaults to the value of `false`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalLabels</b></td>
        <td>map[string]string</td>
        <td>
          Define ExternalLabels for prometheus<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaim">persistentVolumeClaim</a></b></td>
        <td>object</td>
        <td>
          Define persistent volume claim for prometheus<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindex">remoteWrite</a></b></td>
        <td>[]object</td>
        <td>
          Define remote write for prometheus<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Number of replicas/pods to deploy for a Prometheus deployment.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 2<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scrapeInterval</b></td>
        <td>string</td>
        <td>
          Default interval between scrapes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigwebtlsconfig">webTLSConfig</a></b></td>
        <td>object</td>
        <td>
          Configure TLS options for the web endpoint<br/>
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
          accessModes contains the desired access modes the volume should have.
More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>
          dataSource field can be used to specify either:
* An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot)
* An existing PVC (PersistentVolumeClaim)
If the provisioner or an external controller can support the specified data source,
it will create a new volume based on the contents of the specified data source.
When the AnyVolumeDataSource feature gate is enabled, dataSource contents will be copied to dataSourceRef,
and dataSourceRef contents will be copied to dataSource when dataSourceRef.namespace is not specified.
If the namespace is specified, then dataSourceRef will not be copied to dataSource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimdatasourceref">dataSourceRef</a></b></td>
        <td>object</td>
        <td>
          dataSourceRef specifies the object from which to populate the volume with data, if a non-empty
volume is desired. This may be any object from a non-empty API group (non
core object) or a PersistentVolumeClaim object.
When this field is specified, volume binding will only succeed if the type of
the specified object matches some installed volume populator or dynamic
provisioner.
This field will replace the functionality of the dataSource field and as such
if both fields are non-empty, they must have the same value. For backwards
compatibility, when namespace isn't specified in dataSourceRef,
both fields (dataSource and dataSourceRef) will be set to the same
value automatically if one of them is empty and the other is non-empty.
When namespace is specified in dataSourceRef,
dataSource isn't set to the same value and must be empty.
There are three important differences between dataSource and dataSourceRef:
* While dataSource only allows two specific types of objects, dataSourceRef
  allows any non-core object, as well as PersistentVolumeClaim objects.
* While dataSource ignores disallowed values (dropping them), dataSourceRef
  preserves all values, and generates an error if a disallowed value is
  specified.
* While dataSource only allows local objects, dataSourceRef allows objects
  in any namespaces.
(Beta) Using this field requires the AnyVolumeDataSource feature gate to be enabled.
(Alpha) Using the namespace field of dataSourceRef requires the CrossNamespaceVolumeDataSource feature gate to be enabled.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimresources">resources</a></b></td>
        <td>object</td>
        <td>
          resources represents the minimum resources the volume should have.
If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements
that are lower than previous value but must still be higher than capacity recorded in the
status field of the claim.
More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigpersistentvolumeclaimselector">selector</a></b></td>
        <td>object</td>
        <td>
          selector is a label query over volumes to consider for binding.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>
          storageClassName is the name of the StorageClass required by the claim.
More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeAttributesClassName</b></td>
        <td>string</td>
        <td>
          volumeAttributesClassName may be used to set the VolumeAttributesClass used by this claim.
If specified, the CSI driver will create or update the volume with the attributes defined
in the corresponding VolumeAttributesClass. This has a different purpose than storageClassName,
it can be changed after the claim is created. An empty string value means that no VolumeAttributesClass
will be applied to the claim but it's not allowed to reset this field to empty string once it is set.
If unspecified and the PersistentVolumeClaim is unbound, the default VolumeAttributesClass
will be set by the persistentvolume controller if it exists.
If the resource referred to by volumeAttributesClass does not exist, this PersistentVolumeClaim will be
set to a Pending state, as reflected by the modifyVolumeStatus field, until such as a resource
exists.
More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#volumeattributesclass
(Alpha) Using this field requires the VolumeAttributesClass feature gate to be enabled.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>
          volumeMode defines what type of volume is required by the claim.
Value of Filesystem is implied when not included in claim spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          volumeName is the binding reference to the PersistentVolume backing this claim.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.dataSource
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaim)</sup></sup>



dataSource field can be used to specify either:
* An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot)
* An existing PVC (PersistentVolumeClaim)
If the provisioner or an external controller can support the specified data source,
it will create a new volume based on the contents of the specified data source.
When the AnyVolumeDataSource feature gate is enabled, dataSource contents will be copied to dataSourceRef,
and dataSourceRef contents will be copied to dataSource when dataSourceRef.namespace is not specified.
If the namespace is specified, then dataSourceRef will not be copied to dataSource.

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
          APIGroup is the group for the resource being referenced.
If APIGroup is not specified, the specified Kind must be in the core API group.
For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.dataSourceRef
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaim)</sup></sup>



dataSourceRef specifies the object from which to populate the volume with data, if a non-empty
volume is desired. This may be any object from a non-empty API group (non
core object) or a PersistentVolumeClaim object.
When this field is specified, volume binding will only succeed if the type of
the specified object matches some installed volume populator or dynamic
provisioner.
This field will replace the functionality of the dataSource field and as such
if both fields are non-empty, they must have the same value. For backwards
compatibility, when namespace isn't specified in dataSourceRef,
both fields (dataSource and dataSourceRef) will be set to the same
value automatically if one of them is empty and the other is non-empty.
When namespace is specified in dataSourceRef,
dataSource isn't set to the same value and must be empty.
There are three important differences between dataSource and dataSourceRef:
* While dataSource only allows two specific types of objects, dataSourceRef
  allows any non-core object, as well as PersistentVolumeClaim objects.
* While dataSource ignores disallowed values (dropping them), dataSourceRef
  preserves all values, and generates an error if a disallowed value is
  specified.
* While dataSource only allows local objects, dataSourceRef allows objects
  in any namespaces.
(Beta) Using this field requires the AnyVolumeDataSource feature gate to be enabled.
(Alpha) Using the namespace field of dataSourceRef requires the CrossNamespaceVolumeDataSource feature gate to be enabled.

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
          APIGroup is the group for the resource being referenced.
If APIGroup is not specified, the specified Kind must be in the core API group.
For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace is the namespace of resource being referenced
Note that when a namespace is specified, a gateway.networking.k8s.io/ReferenceGrant object is required in the referent namespace to allow that namespace's owner to accept the reference. See the ReferenceGrant documentation for details.
(Alpha) This field requires the CrossNamespaceVolumeDataSource feature gate to be enabled.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.resources
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaim)</sup></sup>



resources represents the minimum resources the volume should have.
If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements
that are lower than previous value but must still be higher than capacity recorded in the
status field of the claim.
More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

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
          Limits describes the maximum amount of compute resources allowed.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.
If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
otherwise to an implementation-defined value. Requests cannot exceed Limits.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.selector
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaim)</sup></sup>



selector is a label query over volumes to consider for binding.

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
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.persistentVolumeClaim.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigpersistentvolumeclaimselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

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
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index]
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfig)</sup></sup>



RemoteWriteSpec defines the configuration to write samples from Prometheus
to a remote endpoint.

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
        <td><b>url</b></td>
        <td>string</td>
        <td>
          The URL of the endpoint to send samples to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexauthorization">authorization</a></b></td>
        <td>object</td>
        <td>
          Authorization section for the URL.


It requires Prometheus >= v2.26.0.


Cannot be set at the same time as `sigv4`, `basicAuth`, `oauth2`, or `azureAd`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexazuread">azureAd</a></b></td>
        <td>object</td>
        <td>
          AzureAD for the URL.


It requires Prometheus >= v2.45.0.


Cannot be set at the same time as `authorization`, `basicAuth`, `oauth2`, or `sigv4`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexbasicauth">basicAuth</a></b></td>
        <td>object</td>
        <td>
          BasicAuth configuration for the URL.


Cannot be set at the same time as `sigv4`, `authorization`, `oauth2`, or `azureAd`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>bearerToken</b></td>
        <td>string</td>
        <td>
          *Warning: this field shouldn't be used because the token value appears
in clear-text. Prefer using `authorization`.*


Deprecated: this will be removed in a future release.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>bearerTokenFile</b></td>
        <td>string</td>
        <td>
          File from which to read bearer token for the URL.


Deprecated: this will be removed in a future release. Prefer using `authorization`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableHTTP2</b></td>
        <td>boolean</td>
        <td>
          Whether to enable HTTP2.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>headers</b></td>
        <td>map[string]string</td>
        <td>
          Custom HTTP headers to be sent along with each remote write request.
Be aware that headers that are set by Prometheus itself can't be overwritten.


It requires Prometheus >= v2.25.0.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexmetadataconfig">metadataConfig</a></b></td>
        <td>object</td>
        <td>
          MetadataConfig configures the sending of series metadata to the remote storage.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The name of the remote write queue, it must be unique if specified. The
name is used in metrics and logging in order to differentiate queues.


It requires Prometheus >= v2.15.0.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexoauth2">oauth2</a></b></td>
        <td>object</td>
        <td>
          OAuth2 configuration for the URL.


It requires Prometheus >= v2.27.0.


Cannot be set at the same time as `sigv4`, `authorization`, `basicAuth`, or `azureAd`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>proxyUrl</b></td>
        <td>string</td>
        <td>
          Optional ProxyURL.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexqueueconfig">queueConfig</a></b></td>
        <td>object</td>
        <td>
          QueueConfig allows tuning of the remote write queue parameters.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>remoteTimeout</b></td>
        <td>string</td>
        <td>
          Timeout for requests to the remote write endpoint.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sendExemplars</b></td>
        <td>boolean</td>
        <td>
          Enables sending of exemplars over remote write. Note that
exemplar-storage itself must be enabled using the `spec.enableFeature`
option for exemplars to be scraped in the first place.


It requires Prometheus >= v2.27.0.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sendNativeHistograms</b></td>
        <td>boolean</td>
        <td>
          Enables sending of native histograms, also known as sparse histograms
over remote write.


It requires Prometheus >= v2.40.0.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexsigv4">sigv4</a></b></td>
        <td>object</td>
        <td>
          Sigv4 allows to configures AWS's Signature Verification 4 for the URL.


It requires Prometheus >= v2.26.0.


Cannot be set at the same time as `authorization`, `basicAuth`, `oauth2`, or `azureAd`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindextlsconfig">tlsConfig</a></b></td>
        <td>object</td>
        <td>
          TLS Config to use for the URL.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexwriterelabelconfigsindex">writeRelabelConfigs</a></b></td>
        <td>[]object</td>
        <td>
          The list of remote write relabel configurations.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].authorization
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



Authorization section for the URL.


It requires Prometheus >= v2.26.0.


Cannot be set at the same time as `sigv4`, `basicAuth`, `oauth2`, or `azureAd`.

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
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexauthorizationcredentials">credentials</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a Secret in the namespace that contains the credentials for authentication.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>credentialsFile</b></td>
        <td>string</td>
        <td>
          File to read a secret from, mutually exclusive with `credentials`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Defines the authentication type. The value is case-insensitive.


"Basic" is not a supported value.


Default: "Bearer"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].authorization.credentials
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexauthorization)</sup></sup>



Selects a key of a Secret in the namespace that contains the credentials for authentication.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].azureAd
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



AzureAD for the URL.


It requires Prometheus >= v2.45.0.


Cannot be set at the same time as `authorization`, `basicAuth`, `oauth2`, or `sigv4`.

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
        <td><b>cloud</b></td>
        <td>enum</td>
        <td>
          The Azure Cloud. Options are 'AzurePublic', 'AzureChina', or 'AzureGovernment'.<br/>
          <br/>
            <i>Enum</i>: AzureChina, AzureGovernment, AzurePublic<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexazureadmanagedidentity">managedIdentity</a></b></td>
        <td>object</td>
        <td>
          ManagedIdentity defines the Azure User-assigned Managed identity.
Cannot be set at the same time as `oauth`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexazureadoauth">oauth</a></b></td>
        <td>object</td>
        <td>
          OAuth defines the oauth config that is being used to authenticate.
Cannot be set at the same time as `managedIdentity`.


It requires Prometheus >= v2.48.0.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].azureAd.managedIdentity
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexazuread)</sup></sup>



ManagedIdentity defines the Azure User-assigned Managed identity.
Cannot be set at the same time as `oauth`.

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
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          The client id<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].azureAd.oauth
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexazuread)</sup></sup>



OAuth defines the oauth config that is being used to authenticate.
Cannot be set at the same time as `managedIdentity`.


It requires Prometheus >= v2.48.0.

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
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          `clientID` is the clientId of the Azure Active Directory application that is being used to authenticate.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexazureadoauthclientsecret">clientSecret</a></b></td>
        <td>object</td>
        <td>
          `clientSecret` specifies a key of a Secret containing the client secret of the Azure Active Directory application that is being used to authenticate.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>tenantId</b></td>
        <td>string</td>
        <td>
          `tenantID` is the tenant ID of the Azure Active Directory application that is being used to authenticate.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].azureAd.oauth.clientSecret
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexazureadoauth)</sup></sup>



`clientSecret` specifies a key of a Secret containing the client secret of the Azure Active Directory application that is being used to authenticate.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].basicAuth
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



BasicAuth configuration for the URL.


Cannot be set at the same time as `sigv4`, `authorization`, `oauth2`, or `azureAd`.

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
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexbasicauthpassword">password</a></b></td>
        <td>object</td>
        <td>
          `password` specifies a key of a Secret containing the password for
authentication.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexbasicauthusername">username</a></b></td>
        <td>object</td>
        <td>
          `username` specifies a key of a Secret containing the username for
authentication.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].basicAuth.password
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexbasicauth)</sup></sup>



`password` specifies a key of a Secret containing the password for
authentication.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].basicAuth.username
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexbasicauth)</sup></sup>



`username` specifies a key of a Secret containing the username for
authentication.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].metadataConfig
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



MetadataConfig configures the sending of series metadata to the remote storage.

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
        <td><b>send</b></td>
        <td>boolean</td>
        <td>
          Defines whether metric metadata is sent to the remote storage or not.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sendInterval</b></td>
        <td>string</td>
        <td>
          Defines how frequently metric metadata is sent to the remote storage.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].oauth2
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



OAuth2 configuration for the URL.


It requires Prometheus >= v2.27.0.


Cannot be set at the same time as `sigv4`, `authorization`, `basicAuth`, or `azureAd`.

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
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexoauth2clientid">clientId</a></b></td>
        <td>object</td>
        <td>
          `clientId` specifies a key of a Secret or ConfigMap containing the
OAuth2 client's ID.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexoauth2clientsecret">clientSecret</a></b></td>
        <td>object</td>
        <td>
          `clientSecret` specifies a key of a Secret containing the OAuth2
client's secret.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>tokenUrl</b></td>
        <td>string</td>
        <td>
          `tokenURL` configures the URL to fetch the token from.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>endpointParams</b></td>
        <td>map[string]string</td>
        <td>
          `endpointParams` configures the HTTP parameters to append to the token
URL.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scopes</b></td>
        <td>[]string</td>
        <td>
          `scopes` defines the OAuth2 scopes used for the token request.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].oauth2.clientId
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexoauth2)</sup></sup>



`clientId` specifies a key of a Secret or ConfigMap containing the
OAuth2 client's ID.

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
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexoauth2clientidconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>
          ConfigMap containing data to use for the targets.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexoauth2clientidsecret">secret</a></b></td>
        <td>object</td>
        <td>
          Secret containing data to use for the targets.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].oauth2.clientId.configMap
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexoauth2clientid)</sup></sup>



ConfigMap containing data to use for the targets.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].oauth2.clientId.secret
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexoauth2clientid)</sup></sup>



Secret containing data to use for the targets.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].oauth2.clientSecret
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexoauth2)</sup></sup>



`clientSecret` specifies a key of a Secret containing the OAuth2
client's secret.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].queueConfig
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



QueueConfig allows tuning of the remote write queue parameters.

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
        <td><b>batchSendDeadline</b></td>
        <td>string</td>
        <td>
          BatchSendDeadline is the maximum time a sample will wait in buffer.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>capacity</b></td>
        <td>integer</td>
        <td>
          Capacity is the number of samples to buffer per shard before we start
dropping them.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxBackoff</b></td>
        <td>string</td>
        <td>
          MaxBackoff is the maximum retry delay.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxRetries</b></td>
        <td>integer</td>
        <td>
          MaxRetries is the maximum number of times to retry a batch on recoverable errors.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxSamplesPerSend</b></td>
        <td>integer</td>
        <td>
          MaxSamplesPerSend is the maximum number of samples per send.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxShards</b></td>
        <td>integer</td>
        <td>
          MaxShards is the maximum number of shards, i.e. amount of concurrency.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minBackoff</b></td>
        <td>string</td>
        <td>
          MinBackoff is the initial retry delay. Gets doubled for every retry.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minShards</b></td>
        <td>integer</td>
        <td>
          MinShards is the minimum number of shards, i.e. amount of concurrency.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>retryOnRateLimit</b></td>
        <td>boolean</td>
        <td>
          Retry upon receiving a 429 status code from the remote-write storage.
This is experimental feature and might change in the future.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].sigv4
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



Sigv4 allows to configures AWS's Signature Verification 4 for the URL.


It requires Prometheus >= v2.26.0.


Cannot be set at the same time as `authorization`, `basicAuth`, `oauth2`, or `azureAd`.

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
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexsigv4accesskey">accessKey</a></b></td>
        <td>object</td>
        <td>
          AccessKey is the AWS API key. If not specified, the environment variable
`AWS_ACCESS_KEY_ID` is used.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>profile</b></td>
        <td>string</td>
        <td>
          Profile is the named AWS profile used to authenticate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>region</b></td>
        <td>string</td>
        <td>
          Region is the AWS region. If blank, the region from the default credentials chain used.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>roleArn</b></td>
        <td>string</td>
        <td>
          RoleArn is the named AWS profile used to authenticate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindexsigv4secretkey">secretKey</a></b></td>
        <td>object</td>
        <td>
          SecretKey is the AWS API secret. If not specified, the environment
variable `AWS_SECRET_ACCESS_KEY` is used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].sigv4.accessKey
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexsigv4)</sup></sup>



AccessKey is the AWS API key. If not specified, the environment variable
`AWS_ACCESS_KEY_ID` is used.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].sigv4.secretKey
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindexsigv4)</sup></sup>



SecretKey is the AWS API secret. If not specified, the environment
variable `AWS_SECRET_ACCESS_KEY` is used.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].tlsConfig
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



TLS Config to use for the URL.

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
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindextlsconfigca">ca</a></b></td>
        <td>object</td>
        <td>
          Certificate authority used when verifying server certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>caFile</b></td>
        <td>string</td>
        <td>
          Path to the CA cert in the Prometheus container to use for the targets.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindextlsconfigcert">cert</a></b></td>
        <td>object</td>
        <td>
          Client certificate to present when doing client-authentication.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          Path to the client cert file in the Prometheus container for the targets.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          Disable target certificate validation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keyFile</b></td>
        <td>string</td>
        <td>
          Path to the client key file in the Prometheus container for the targets.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindextlsconfigkeysecret">keySecret</a></b></td>
        <td>object</td>
        <td>
          Secret containing the client key file for the targets.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>serverName</b></td>
        <td>string</td>
        <td>
          Used to verify the hostname for the targets.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].tlsConfig.ca
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindextlsconfig)</sup></sup>



Certificate authority used when verifying server certificates.

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
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindextlsconfigcaconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>
          ConfigMap containing data to use for the targets.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindextlsconfigcasecret">secret</a></b></td>
        <td>object</td>
        <td>
          Secret containing data to use for the targets.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].tlsConfig.ca.configMap
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindextlsconfigca)</sup></sup>



ConfigMap containing data to use for the targets.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].tlsConfig.ca.secret
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindextlsconfigca)</sup></sup>



Secret containing data to use for the targets.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].tlsConfig.cert
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindextlsconfig)</sup></sup>



Client certificate to present when doing client-authentication.

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
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindextlsconfigcertconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>
          ConfigMap containing data to use for the targets.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigremotewriteindextlsconfigcertsecret">secret</a></b></td>
        <td>object</td>
        <td>
          Secret containing data to use for the targets.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].tlsConfig.cert.configMap
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindextlsconfigcert)</sup></sup>



ConfigMap containing data to use for the targets.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].tlsConfig.cert.secret
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindextlsconfigcert)</sup></sup>



Secret containing data to use for the targets.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].tlsConfig.keySecret
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindextlsconfig)</sup></sup>



Secret containing the client key file for the targets.

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.remoteWrite[index].writeRelabelConfigs[index]
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigremotewriteindex)</sup></sup>



RelabelConfig allows dynamic rewriting of the label set for targets, alerts,
scraped samples and remote write samples.


More info: https://prometheus.io/docs/prometheus/latest/configuration/configuration/#relabel_config

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
        <td><b>action</b></td>
        <td>enum</td>
        <td>
          Action to perform based on the regex matching.


`Uppercase` and `Lowercase` actions require Prometheus >= v2.36.0.
`DropEqual` and `KeepEqual` actions require Prometheus >= v2.41.0.


Default: "Replace"<br/>
          <br/>
            <i>Enum</i>: replace, Replace, keep, Keep, drop, Drop, hashmod, HashMod, labelmap, LabelMap, labeldrop, LabelDrop, labelkeep, LabelKeep, lowercase, Lowercase, uppercase, Uppercase, keepequal, KeepEqual, dropequal, DropEqual<br/>
            <i>Default</i>: replace<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>modulus</b></td>
        <td>integer</td>
        <td>
          Modulus to take of the hash of the source label values.


Only applicable when the action is `HashMod`.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>regex</b></td>
        <td>string</td>
        <td>
          Regular expression against which the extracted value is matched.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replacement</b></td>
        <td>string</td>
        <td>
          Replacement value against which a Replace action is performed if the
regular expression matches.


Regex capture groups are available.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>separator</b></td>
        <td>string</td>
        <td>
          Separator is the string between concatenated SourceLabels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sourceLabels</b></td>
        <td>[]string</td>
        <td>
          The source labels select values from existing labels. Their content is
concatenated using the configured Separator and matched against the
configured regular expression.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetLabel</b></td>
        <td>string</td>
        <td>
          Label to which the resulting string is written in a replacement.


It is mandatory for `Replace`, `HashMod`, `Lowercase`, `Uppercase`,
`KeepEqual` and `DropEqual` actions.


Regex capture groups are available.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.webTLSConfig
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfig)</sup></sup>



Configure TLS options for the web endpoint

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
        <td><b><a href="#monitoringstackspecprometheusconfigwebtlsconfigca">ca</a></b></td>
        <td>object</td>
        <td>
          Secret containing the CA certificate to verify the server certificates<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigwebtlsconfigcert">cert</a></b></td>
        <td>object</td>
        <td>
          Secret containing the TLS certificate for the server<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#monitoringstackspecprometheusconfigwebtlsconfigkey">key</a></b></td>
        <td>object</td>
        <td>
          Secret containing the TLS key for the server<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.webTLSConfig.ca
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigwebtlsconfig)</sup></sup>



Secret containing the CA certificate to verify the server certificates

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The name of the secret in the object's namespace to select from.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.webTLSConfig.cert
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigwebtlsconfig)</sup></sup>



Secret containing the TLS certificate for the server

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The name of the secret in the object's namespace to select from.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.prometheusConfig.webTLSConfig.key
<sup><sup>[↩ Parent](#monitoringstackspecprometheusconfigwebtlsconfig)</sup></sup>



Secret containing the TLS key for the server

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
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The name of the secret in the object's namespace to select from.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.resourceSelector
<sup><sup>[↩ Parent](#monitoringstackspec)</sup></sup>



Label selector for Monitoring Stack Resources.
To monitor everything, set to empty map selector. E.g. resourceSelector: {}.
To disable service discovery, set to null. E.g. resourceSelector:.

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
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.resourceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#monitoringstackspecresourceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

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
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
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
        <td><b><a href="#monitoringstackspecresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims,
that are used by this container.


This is an alpha field and requires enabling the
DynamicResourceAllocation feature gate.


This field is immutable. It can only be set for containers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.
If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
otherwise to an implementation-defined value. Requests cannot exceed Limits.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### MonitoringStack.spec.resources.claims[index]
<sup><sup>[↩ Parent](#monitoringstackspecresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

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
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of
the Pod where this field is used. It makes that resource available
inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### MonitoringStack.status
<sup><sup>[↩ Parent](#monitoringstack)</sup></sup>



MonitoringStackStatus defines the observed state of MonitoringStack.
It should always be reconstructable from the state of the cluster and/or outside world.

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
        <td><b><a href="#monitoringstackstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          Conditions provide status information about the MonitoringStack<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### MonitoringStack.status.conditions[index]
<sup><sup>[↩ Parent](#monitoringstackstatus)</sup></sup>





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
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown, Degraded<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.
The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt)<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
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
          ThanosQuerierSpec defines a single Thanos Querier instance. This means a
label selector by which Monitoring Stack instances to query are selected, and
an optional namespace selector and a list of replica labels by which to
deduplicate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>object</td>
        <td>
          ThanosQuerierStatus defines the observed state of ThanosQuerier.
It should always be reconstructable from the state of the cluster and/or outside world.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ThanosQuerier.spec
<sup><sup>[↩ Parent](#thanosquerier)</sup></sup>



ThanosQuerierSpec defines a single Thanos Querier instance. This means a
label selector by which Monitoring Stack instances to query are selected, and
an optional namespace selector and a list of replica labels by which to
deduplicate.

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
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ThanosQuerier.spec.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#thanosquerierspecselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

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
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
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
          Boolean describing whether all namespaces are selected in contrast to a
list restricting them.<br/>
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

# observability.openshift.io/v1alpha1

Resource Types:

- [UIPlugin](#uiplugin)




## UIPlugin
<sup><sup>[↩ Parent](#observabilityopenshiftiov1alpha1 )</sup></sup>






UIPlugin defines an observability console plugin.

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
      <td>observability.openshift.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>UIPlugin</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#uipluginspec">spec</a></b></td>
        <td>object</td>
        <td>
          UIPluginSpec is the specification for desired state of UIPlugin.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#uipluginstatus">status</a></b></td>
        <td>object</td>
        <td>
          UIPluginStatus defines the observed state of UIPlugin.
It should always be reconstructable from the state of the cluster and/or outside world.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### UIPlugin.spec
<sup><sup>[↩ Parent](#uiplugin)</sup></sup>



UIPluginSpec is the specification for desired state of UIPlugin.

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
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type defines the UI plugin.<br/>
          <br/>
            <i>Enum</i>: Dashboards<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### UIPlugin.status
<sup><sup>[↩ Parent](#uiplugin)</sup></sup>



UIPluginStatus defines the observed state of UIPlugin.
It should always be reconstructable from the state of the cluster and/or outside world.

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
        <td><b><a href="#uipluginstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          Conditions provide status information about the plugin.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### UIPlugin.status.conditions[index]
<sup><sup>[↩ Parent](#uipluginstatus)</sup></sup>





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
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown, Degraded<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.
The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt)<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
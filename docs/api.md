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
        <td><b>additionalNamespaces</b></td>
        <td>[]string</td>
        <td>
          Namespaces to monitor in addition to the MonitoringStack namespace<br/>
        </td>
        <td>false</td>
      </tr><tr>
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
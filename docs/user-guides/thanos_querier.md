# Using ThanosQuerier to federate MonitoringStacks

`ThanosQuerier` can be used to query data from a set of `MonitoringStack` resources.

## Architecture

A `ThanosQuerier` resource selects a set of `MonitoringStack` resources using
label and namespace selectors.

Under the hood, the observability operator creates a Kubernetes Deployment
which is configured to connect to the Thanos sidecars running in the Prometheus
pods.

## Tutorial

### Pre-requisites
* Observability operator installed and running.
* Cluster admin permissions.

### Installation

We are going to create
* 1 `MonitoringStack` running in the `project-a` namespace.
* 1 `MonitoringStack` running in the `project-b` namespace.
* 1 `ThanosQuerier` running in the `project-c` namespace.

To install the example, run:

```shell
kubectl apply -f docs/user-guides/thanos_querier/install
```

To verify the installation, run:

```shell
kubectl wait --for=condition=Available -A --timeout=10s -l app=example monitoringstacks
kubectl wait --for=condition=Available -A --timeout=10s -l app.kubernetes.io/managed-by=observability-operator deployments
```

To access the Thanos Query UI, run:

```shell
kubectl port-forward -n project-c svc/thanos-querier-example 10902:localhost:10902
```

Then open `http://localhost:10902` in your browser.

### Configuring a dashboard

To install the example dashboard (+datasource), run:

```shell
kubectl apply -f docs/user-guides/thanos_querier/dashboard
```

To verify the installation, run:

```
kubectl wait --for=condition=Available uiplugins dashboards
```

You should now be able to access the custom dashboard under `Observe > Dashboards`.

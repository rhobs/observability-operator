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
* 1 `MonitoringStack` running in the `project-a` namespace which monitors API services running in `project-a` and `project-b`.
* 1 `MonitoringStack` running in the `project-c` namespace which monitors the backend service running in `project-c`.
* 1 `ThanosQuerier` running in the `project-d` namespace which federates data from the 2 `MonitoringStack`s.
* Deployments running in `project-a`, `project-b` and `project-c` namespaces which represent a multi-service application.
* Load generation Deployment running in the `project-d` namespace.

To install the environment, run:

```shell
kubectl apply -f docs/user-guides/thanos_querier/install
```

To verify the installation, run:

```shell
kubectl wait --for=condition=Available -A --timeout=10s -l app.kubernetes.io/part-of monitoringstacks
kubectl wait --for=condition=Available -A --timeout=10s -l app.kubernetes.io/managed-by=observability-operator deployments
kubectl wait --for=condition=Available -A --timeout=10s -l app.kubernetes.io/part-of=myapp deployments
```

To access the Thanos Query UI, run:

```shell
kubectl port-forward -n project-c svc/thanos-querier-example 10902:localhost:10902
```

Then open `http://localhost:10902` in your browser. You can check that all Prometheus instances are present in the Stores page and that metrics are showing up.

### Configuring a Perses dashboard

To install the example Perses dashboard (+datasource), run:

```shell
kubectl apply -f docs/user-guides/thanos_querier/console
```

To verify the installation, run:

```
kubectl wait --for=condition=Available uiplugins monitoring
```

You should now be able to access the custom dashboard under `Observe > Dashboards (Perses)` in the `project-d` namespace.


# Procedure to assess resources used by Observability Operator

1. Provision an OpenShift cluster

2. Run `oc apply -f hack/olm/catalog-src.yaml` to install the Observability Operator (OO) catalogue.

3. Using the UI install OO

4. Scale down the following deployments, so we can remove the currently set limits on OO:

```bash
# Scale down the cluster version operator
oc -n openshift-cluster-version scale deployment.apps/cluster-version-operator --replicas=0
# Scale down the OLM operator
oc -n openshift-operator-lifecycle-manager scale deployment.apps/olm-operator --replicas=0
```

5. Edit the OO and Prometheus Operator deployment to remove it's limits with:

```bash
oc -n openshift-operators patch deployment.apps/observability-operator --type='json' -p='[{"op": "remove", "path": "/spec/template/spec/containers/0/resources/limits"}]'
oc -n openshift-operators patch deployment.apps/observability-operator-prometheus-operator --type='json' -p='[{"op": "remove", "path": "/spec/template/spec/containers/0/resources/limits"}]'
```

6. Run the load tests with `./hack/loadtest/test.sh`

7. Using the OpenShift UI in the Developer tab, navigate to Observe and input the following querries.
    1. For memory we should look at `container_memory_rss` as that is the metric used by kubelet to OOM kill the container
    2. For CPU we should look at `container_cpu_usage_seconds_total` as that is the metric used by kubelet

```bash
# PromQL for memory
container_memory_rss{container!~"|POD", namespace="openshift-operators"}
# PromQL for CPU
sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace='openshift-operators'}) by (pod)
```

8. Take for both OO and Prometheus Operator measurements of their preformance
    1. Establish a baseline for both CPU and memory (minimum they consume), those will be our `requests`
    2. Multiply that value by 3 and validate that it fits the intervals of values observed, those will be our `limits`
    3. Give some extra head room to `limits` to anticipate feature growth
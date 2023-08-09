# observability-operator

The Observability Operator is a Kubernetes operator which enables the
management of Monitoring/Alerting stacks through Kubernetes CRDs. Eventually it
might also cover Logging and Tracing.

The project relies heavily on the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library.

## Trying out the Operator

Observability Operator requires Operator Lifecycle Manager (OLM) to be running
in cluster. The easiest way is to use OpenShift where OLM is preinstalled.

### OpenShift

Add the Observability Operator Catalog as shown below.

```
kubectl apply -f hack/olm/catalog-src.yaml
```
This adds a new Catalog to the list of catalogs. Now, you should be able to use
OLM Web interface to install/uninstall Observability Operator like any other
operator.

If you prefer CLI, applying the subscription as shown below will install the
operator.

```
kubectl create -f ./hack/olm/subscription.yaml
```


####  Uninstall

It is easier to use the web console to remove the installed operator.
Instructions below removes all traces of what was setup in the previous step
including removing the catalog.
```
oc delete -n operators csv \
    -l operators.coreos.com/observability-operator.operators=

oc delete -n openshift-operators  \
    installplan,subscriptions \
    -l operators.coreos.com/observability-operator.operators=

oc delete -f hack/olm

oc delete crds "$(oc api-resources --api-group=monitoring.rhobs -o name)"

```

### Kubernetes

As mentioned above, Observability Operator requires Operator Lifecycle Manager
(OLM) to be running in cluster, so installing OLM is the first step to getting
the Observability Operator running on k8s.


```
operator-sdk olm install
kubectl create -f ./hack/olm/k8s/catalog-src.yaml
kubectl create -f ./hack/olm/k8s/subscription.yaml

```
**NOTE:** To install  `operator-sdk`, you can make use of  `make tools` which
installs `operator-sdk` (along with other tools needed for development)
to `tmp/bin`

For more information, about running Observability Operator (ObO) on Kind,
please refer to the [Developer Docs](./docs/developer.md).

####  Uninstalling
```
kubectl delete -n operators csv \
    -l operators.coreos.com/observability-operator.operators=

kubectl delete -n operators  \
    installplan,subscriptions \
    -l operators.coreos.com/observability-operator.operators=

kubectl delete -f hack/olm/k8s

kubectl delete crds "$(kubectl api-resources --api-group=monitoring.rhobs -o name)"

```
## Development

Please refer to [Developer Docs](./docs/developer.md)

## Meetings
___
- Weekly meeting: [Thursday at 08:00 CET (Central European Time)](https://meet.google.com/gwy-vssi-hfr)
  - [Meeting notes and Agenda](https://docs.google.com/document/d/1Iy3CRIEzsHUhtMuzCVRX-8fbmsivcu2iju1J2vN2knQ/edit?usp=meetingnotes&showmeetingnotespromo=true).

## Contact
___
- Red Hat Slack #observability-operator-users and ping @obo-support-team.
- [Mailing list](mso-users@redhat.com)
- Github Team: @rhobs/observability-operator-maintainers

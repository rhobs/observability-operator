# Authenticated access to Thanos Querier using OAuth Proxy

`ThanosQuerier` can be configured with an oauth-proxy sidecar to enforce authenticated access to metrics.

## Architecture

This example deploys three isolated namespaces — `project-a`, `project-b`, and `project-c` — each containing a `MonitoringStack` (Prometheus + Alertmanager) and a `ThanosQuerier` protected by an oauth-proxy sidecar. `NetworkPolicy` objects enforce isolation between namespaces, as is common in multi-tenant clusters.

The oauth-proxy sidecar delegates authentication to the OpenShift API server via subject access reviews. Users must authenticate with their OpenShift credentials before they can reach the Thanos Query UI.

Access is controlled per namespace:

* `user1` can query metrics in `project-a` only.
* `user2` can query metrics in `project-b` only.
* `user3` can query metrics from both `project-b` and `project-c` via the `ThanosQuerier` in `project-c`.

The `ThanosQuerier` in `project-c` uses a `namespaceSelector` to federate metrics from both `project-b` and `project-c`. A `NetworkPolicy` in `project-b` permits the Thanos sidecar in `project-c` to scrape it.

In `project-a`, bearer token authentication is also enabled, allowing a `robot-user` ServiceAccount to query metrics programmatically without browser-based login.

## Tutorial

### Pre-requisites

* Observability Operator installed and running.
* Cluster admin permissions.
* OpenShift Container Platform 4.16+.

### Create test users and configure the identity provider

Create an htpasswd file with the three test users:

```sh
htpasswd -c -B -b redhatusers.htpasswd user1 RedHat!
htpasswd -B -b redhatusers.htpasswd user2 RedHat!
htpasswd -B -b redhatusers.htpasswd user3 RedHat!
```

Create the htpasswd secret in the `openshift-config` namespace:

```sh
kubectl create secret generic htpass-secret \
  --from-file=htpasswd=redhatusers.htpasswd \
  -n openshift-config
```

Configure the cluster identity provider to use this secret:

```yaml
apiVersion: config.openshift.io/v1
kind: OAuth
metadata:
  name: cluster
spec:
  identityProviders:
  - name: my_htpasswd_provider
    mappingMethod: claim
    type: HTPasswd
    htpasswd:
      fileData:
        name: htpass-secret # 👈 secret created in the preceding step
```

Or run

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/00-identity-provider.yaml
```

**NOTE:** Applying the OAuth resource triggers a restart of the cluster authentication pods. Wait for them to become ready before proceeding.

### Setting up project-a

#### Create the namespace, NetworkPolicies, MonitoringStack, and ThanosQuerier

```yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: project-a
  labels:
    monitoring.rhobs/stack: project-a # 👈 matched by MonitoringStack namespaceSelector below
---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: deny-by-default
  namespace: project-a
spec:
  podSelector: {}
  ingress: []
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-openshift-ingress
  namespace: project-a
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          policy-group.network.openshift.io/ingress: ""
  podSelector: {}
  policyTypes:
  - Ingress
---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: allow-same-namespace
  namespace: project-a
spec:
  podSelector: {}
  ingress:
  - from:
    - podSelector: {}
---
apiVersion: monitoring.rhobs/v1alpha1
kind: MonitoringStack
metadata:
  name: example-coo-monitoring-stack
  namespace: project-a
  labels:
    discover: project-a # 👈 matched by ThanosQuerier selector below
spec:
  logLevel: debug
  retention: 1d
  resourceSelector:
    matchLabels:
      k8s-app: prometheus-coo-example-monitor
  namespaceSelector:
    matchLabels:
      monitoring.rhobs/stack: project-a
---
apiVersion: monitoring.rhobs/v1alpha1
kind: ThanosQuerier
metadata:
  name: example-coo-thanos
  namespace: project-a
spec:
  selector:
    matchLabels:
      discover: project-a
```

Or run

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/00-project-a.yaml
```

#### Deploy a test application

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: prometheus-coo-example-app
  name: prometheus-coo-example-app
  namespace: project-a
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus-coo-example-app
  template:
    metadata:
      labels:
        app: prometheus-coo-example-app
    spec:
      containers:
      - image: ghcr.io/rhobs/prometheus-example-app:0.4.2
        imagePullPolicy: IfNotPresent
        name: prometheus-coo-example-app
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: prometheus-coo-example-app
  name: prometheus-coo-example-app
  namespace: project-a
spec:
  ports:
  - port: 8080
    protocol: TCP
    targetPort: 8080
    name: web
  selector:
    app: prometheus-coo-example-app
  type: ClusterIP
---
apiVersion: monitoring.rhobs/v1
kind: ServiceMonitor
metadata:
  labels:
    k8s-app: prometheus-coo-example-monitor # 👈 matched by MonitoringStack resourceSelector
  name: prometheus-coo-example-monitor
  namespace: project-a
spec:
  endpoints:
  - interval: 30s
    port: web
    scheme: http
  selector:
    matchLabels:
      app: prometheus-coo-example-app
---
apiVersion: monitoring.rhobs/v1
kind: PrometheusRule
metadata:
  name: example-alert
  namespace: project-a
  labels:
    k8s-app: prometheus-coo-example-monitor # 👈 matched by MonitoringStack resourceSelector
spec:
  groups:
  - name: example
    rules:
    - alert: VersionAlert
      for: 1m
      expr: version{job="prometheus-coo-example-app"} > 0
      labels:
        severity: warning
```

**NOTE:** The test application is included in `00-project-a.yaml` applied in the previous step. Equivalent applications for `project-b` and `project-c` are included in `01-project-b.yaml` and `02-project-c.yaml` respectively.

#### Enable bearer token authentication

To allow ServiceAccounts to authenticate with bearer tokens, grant the `thanos-querier` ServiceAccount permission to create `TokenReview` resources:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: oauth-create-tokenreviews
  # 👇 Without this binding the oauth-proxy will log:
  #    Failed to make webhook authenticator request: tokenreviews.authentication.k8s.io is forbidden
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator # 👈 grants permission to create TokenReviews and SubjectAccessReviews
subjects:
- kind: ServiceAccount
  name: thanos-querier
  namespace: project-a
```

Or run

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/03-tokenreview.yaml
```

**NOTE:** This step is only required for `project-a`. Projects `project-b` and `project-c` use OpenShift user authentication only.

#### Configure OAuth Proxy

Create the session secret used by the oauth-proxy to encrypt login cookies:

```sh
kubectl -n project-a create secret generic thanos-proxy \
  --from-literal=session_secret=$(head -c 32 /dev/urandom | base64)
```

Create the `thanos-querier` ServiceAccount and annotate it to enable the OAuth redirect flow:

```sh
kubectl -n project-a create serviceaccount thanos-querier

kubectl -n project-a annotate serviceaccount thanos-querier \
  serviceaccounts.openshift.io/oauth-redirectreference.thanos-querier='{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"thanos-querier-authenticated"}}'
```

Annotate the Thanos service to trigger automatic TLS certificate generation by the OpenShift serving-cert operator:

```sh
kubectl -n project-a annotate service thanos-querier-example-coo-thanos \
  service.alpha.openshift.io/serving-cert-secret-name="thanos-tls"
```

Create a `robot-user` ServiceAccount for bearer token access and grant it view permissions:

```sh
kubectl -n project-a create serviceaccount robot-user
kubectl -n project-a create rolebinding view-robot-user \
  --clusterrole=view \
  --serviceaccount=project-a:robot-user
```

#### Inject the OAuth Proxy sidecar

**NOTE:** The `thanos-querier` ServiceAccount, `thanos-proxy` secret, and `thanos-tls` serving-cert secret created in the preceding step must all exist before applying this patch, otherwise the new pod will fail to start.

The `ThanosQuerier` Deployment is managed by the Observability Operator. Patch it using server-side apply to inject the oauth-proxy container and update the Service to route traffic through the proxy:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: thanos-querier-example-coo-thanos
  namespace: project-a
spec:
  ports:
  - name: proxy
    port: 8888
    protocol: TCP
    targetPort: oauth-proxy # 👈 route external traffic to the oauth-proxy container port
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/instance: thanos-querier-example-coo-thanos
    app.kubernetes.io/managed-by: observability-operator
    app.kubernetes.io/part-of: ThanosQuerier
  name: thanos-querier-example-coo-thanos
  namespace: project-a
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: thanos-querier-example-coo-thanos
  template:
    metadata:
      labels:
        app.kubernetes.io/instance: thanos-querier-example-coo-thanos
        app.kubernetes.io/managed-by: observability-operator
        app.kubernetes.io/part-of: ThanosQuerier
    spec:
      containers:
      - args:
        - -provider=openshift
        - -https-address=:8888
        - -http-address=
        - -email-domain=*
        - -upstream=http://localhost:10902 # 👈 Thanos Query UI listens on this port
        - -tls-cert=/etc/tls/private/tls.crt
        - -tls-key=/etc/tls/private/tls.key
        - -cookie-secret-file=/etc/proxy/secrets/session_secret
        - -openshift-service-account=thanos-querier
        - -openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        - -skip-auth-regex=^/metrics
        - -openshift-sar={"resource":"namespaces","resourceName":"thanos-querier","namespace":"project-a","verb":"get"} # 👈 user must have get access on this namespace to authenticate
        - -openshift-delegate-urls={"/":{"resource":"pods","namespace":"project-a","verb":"get"}} # 👈 enables bearer token delegation for ServiceAccounts
        image: quay.io/openshift/origin-oauth-proxy:4.19
        name: oauth-proxy
        ports:
        - containerPort: 8888
          name: oauth-proxy
          protocol: TCP
        volumeMounts:
        - mountPath: /etc/tls/private
          name: secret-thanos-tls
        - mountPath: /etc/proxy/secrets
          name: secret-thanos-proxy
      serviceAccount: thanos-querier
      serviceAccountName: thanos-querier
      volumes:
      - name: secret-thanos-tls
        secret:
          secretName: thanos-tls # 👈 created automatically by the serving-cert annotation above
      - name: secret-thanos-proxy
        secret:
          secretName: thanos-proxy # 👈 the session secret created above
```

Or run

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/04-oauth-project-a.yaml --server-side
```

**NOTE:** The `--server-side` flag is required because the Deployment is managed by the Observability Operator. For more details see [Using Server-Side Apply to customize Prometheus resources](server-side-apply.md).

#### Create the authenticated route

```yaml
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: thanos-querier-authenticated
  namespace: project-a
spec:
  port:
    targetPort: proxy
  tls:
    insecureEdgeTerminationPolicy: Redirect
    termination: reencrypt
  to:
    kind: Service
    name: thanos-querier-example-coo-thanos
    weight: 100
```

Or run

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/07-route-project-a.yaml
```

#### Grant user1 access

```sh
kubectl -n project-a create rolebinding view-user1 \
  --clusterrole=view \
  --user=user1
```

### Setting up project-b

`project-b` follows the same pattern as `project-a` with two differences:

* Bearer token authentication is not enabled.
* The `MonitoringStack` uses a shared `discover: multi-ns` label so that the `ThanosQuerier` in `project-c` can also discover it.

The `project-b` namespace includes an additional `allow-thanos` NetworkPolicy that permits the Thanos sidecar in `project-c` to connect to the Prometheus instance here. For more information on cross-namespace Thanos discovery see [Using ThanosQuerier to federate MonitoringStacks](thanos_querier.md).

#### Create the namespace, NetworkPolicies, MonitoringStack, and ThanosQuerier

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/01-project-b.yaml
```

#### Configure OAuth Proxy

```sh
kubectl -n project-b create secret generic thanos-proxy \
  --from-literal=session_secret=$(head -c 32 /dev/urandom | base64)

kubectl -n project-b create serviceaccount thanos-querier

kubectl -n project-b annotate serviceaccount thanos-querier \
  serviceaccounts.openshift.io/oauth-redirectreference.thanos-querier='{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"thanos-querier-authenticated"}}'

kubectl -n project-b annotate service thanos-querier-example-coo-thanos \
  service.alpha.openshift.io/serving-cert-secret-name="thanos-tls"
```

#### Inject the OAuth Proxy sidecar

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/05-oauth-project-b.yaml --server-side
```

#### Create the authenticated route

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/08-route-project-b.yaml
```

#### Grant user2 access

```sh
kubectl -n project-b create rolebinding view-user2 \
  --clusterrole=view \
  --user=user2
```

### Setting up project-c

`project-c` follows the same pattern as `project-b`. The key difference is that the `ThanosQuerier` in `project-c` uses a `namespaceSelector` to federate metrics from both `project-b` and `project-c`, giving `user3` a unified view of both namespaces.

#### Create the namespace, NetworkPolicies, MonitoringStack, and ThanosQuerier

The `project-c` namespace is labeled `project: project-c` so that the `allow-thanos` NetworkPolicy in `project-b` can permit its Thanos sidecar to connect.

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/02-project-c.yaml
```

#### Configure OAuth Proxy

```sh
kubectl -n project-c create secret generic thanos-proxy \
  --from-literal=session_secret=$(head -c 32 /dev/urandom | base64)

kubectl -n project-c create serviceaccount thanos-querier

kubectl -n project-c annotate serviceaccount thanos-querier \
  serviceaccounts.openshift.io/oauth-redirectreference.thanos-querier='{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"thanos-querier-authenticated"}}'

kubectl -n project-c annotate service thanos-querier-example-coo-thanos \
  service.alpha.openshift.io/serving-cert-secret-name="thanos-tls"
```

#### Inject the OAuth Proxy sidecar

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/06-oauth-project-c.yaml --server-side
```

#### Create the authenticated route

```sh
kubectl apply -f docs/user-guides/oauth-proxy/manifests/09-route-project-c.yaml
```

#### Grant user3 access

```sh
kubectl -n project-c create rolebinding view-user3 \
  --clusterrole=view \
  --user=user3
```

## Validation

### Browser-based access

Retrieve the route hostname for each namespace:

```sh
kubectl get route -n project-a thanos-querier-authenticated -o jsonpath="{.spec.host}"
kubectl get route -n project-b thanos-querier-authenticated -o jsonpath="{.spec.host}"
kubectl get route -n project-c thanos-querier-authenticated -o jsonpath="{.spec.host}"
```

Browse to each URL. You will be prompted to authenticate with your OpenShift credentials.

* `user1` can authenticate to the `project-a` route only.
* `user2` can authenticate to the `project-b` route only.
* `user3` can authenticate to the `project-c` route and will see metrics from both `project-b` and `project-c`.

### Bearer token access (project-a only)

Generate a short-lived token for the `robot-user` ServiceAccount and query the Thanos API directly:

```sh
TOKEN=$(kubectl -n project-a create token robot-user)
ROUTE=$(kubectl get route -n project-a thanos-querier-authenticated -o jsonpath='{.spec.host}')
curl -H "Authorization: Bearer ${TOKEN}" "https://${ROUTE}/api/v1/query?query=up"
```

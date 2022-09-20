# observability-operator

The Observability Operator (previously known as Monitoring Stack Operator) is a
Kubernetes operator which enables the management of Monitoring, Logging and
Tracing stacks through Kubernetes CRDs.

The project is based on the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library.

## Trying out the Operator

### OLM

The easiest way to try out the operator is to use OLM (that comes shipped with
OpenShift). Assuming you are using OpenShift, add the Observability Operator Catalog
as shown below.

```
kubectl apply -f hack/olm/catalog-src.yaml
```
This adds a new Catalog to the list of Catalogs. Now, you should be able to use
OLM Web interface to install Observability Operator like any other operator.

### Kubernetes

Refer `Running the Operator in Kind `below to run the operator in Kubernetes.

## Development

In order to contribute to this project, make sure you have go 1.17 on your local machine.

### Dependencies

The build system assumes the following binaries are in the `PATH`:

```
make git go npm
```

Please make sure to install the relevant packages.

### Commit message convention

Commit messages need to comply to the [Conventional Commits specification](https://www.conventionalcommits.org/en/v1.0.0/) and should be structured as follows:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

The type and description are used to generate a changelog and determine the next release version.
Most commonly used types are:

* `fix:` a commit of the type fix patches a bug in your codebase (this correlates with PATCH in Semantic Versioning).

* `feat:` a commit of the type feat introduces a new feature to the codebase (this correlates with MINOR in Semantic Versioning).

* `BREAKING CHANGE:` a commit that has a footer BREAKING CHANGE:, or appends a
 `!` after the type/scope, introduces a breaking API change (correlating with
 MAJOR in Semantic Versioning). A BREAKING CHANGE can be part of commits of any type.

Other than `fix:` and `feat:`, the following type can also be used: `build:`, `chore:`, `ci:`, `docs:`, `style:`, `refactor:`, `perf:` and `test:`.

### Manifest generation

The Kubernetes CRDs and the ClusterRole needed for their management are generated from go types in `pkg/apis`.
Run `make generate` to regenerate the Kubernetes manifests when changing these files.

This project uses the [controller-gen](https://github.com/kubernetes-sigs/controller-tools/tree/master/cmd/controller-gen) for code generation.
For detailed information on the available code generation markers, please refer
to the controller-gen CLI page in the [kubebuilder documentation](https://book.kubebuilder.io/reference/markers.html)

### Running the operator in Kind cluster

Run `./hack/kind/setup.sh` to create a kind cluster named `obs-operator` and
sets up all the required dependencies (OLM, infra-node, registry).

To cleanup the cluster or to retry the setup script after a failed attempt run `kind delete cluster --name obs-operator`

#### Install CRDs and Prometheus Operator

* Apply the CRDs by running
  ```sh
  kubectl create -k deploy/crds/kubernetes
  ```

* Apply the dependencies; i.e. - Prometheus Operator by running

  ```sh
  kubectl create ns operators
  kubectl apply -k deploy/dependencies
  ```

* Set the `KUBECONFIG` environment variable to your local cluster and run the
  operator with `go run ./cmd/operator/...`.

* Alternatively, you can also set the `kubeconfig` on the command line:
  `go run ./cmd/operator/... --kubeconfig <path-to-kubeconfig>`

**TIP**: Using `fd` and `entr` to automatically build and run operator can come handy.
E.g. the following re-runs operator when any of the `.go` files change.

 ```sh
  fd .go$ -- cmd pkg |
    entr -rs 'go run ./cmd/operator/... --zap-devel --zap-log-level=10' 2>&1 |
    tee ./tmp/operator.log
 ```

The above can be tweaked to automatically run `make generate` when the `pkg/apis/` change.

### Running E2E Tests

E2E tests are run against a deployment, so it requires operator to be running
in cluster or outside it (`go run ./cmd/operator/...`). To run e2e tests locally,

* Follow the section above to run the operator

* In a new terminal, run `go test -v ./test/e2e/...`

Note: when running the operator outside of the cluster the post e2e tests are
expected to fail since it requires operator to be deployed in the cluster.
Refer to the "Running Operator Bundle" section below.

## Running Operator Bundle

OperatorSDK provides `run bundle` command allowing us to run the Operator in
`kind` cluster as if it is installed via OLM. To run the operator follow the
steps below

1. Build and push operator images to the `local-registry`
```
make operator-image bundle bundle-image \
  operator-push bundle-push \
  IMAGE_BASE=local-registry:30000/observability-operator \
  VERSION=0.0.0-ci  \
  FIRST_OLM_RELEASE=true
```

5. Run operator-sdk run bundle to subscribe to the operator
```
./tmp/bin/operator-sdk run bundle \
    local-registry:30000/observability-operator-bundle:0.0.0-ci \
    --install-mode AllNamespaces  \
    --skip-tls \
    --namespace operators \
    --verbose | tee tmp/olm.log
```
6. Run `e2e` tests
```
go test -v ./test/e2e/...
```
A good way to monitor if the tests are running correctly is to watch the
pods in the `e2e-tests` namespace. E.g.
```
kubectl get pods -w --output-watch-events -n e2e-tests
```
## Managing releases

This project follows [SemVer 2.0.0](https://semver.org/)

```
Given a version number MAJOR.MINOR.PATCH, increment the:

MAJOR version when you make incompatible API changes,
MINOR version when you add functionality in a backwards compatible manner, and
PATCH version when you make backwards compatible bug fixes.
Additional labels for pre-release and build metadata are available as extensions to the MAJOR.MINOR.PATCH format.
```

Creating new releases is fully automated and does not require any human interaction.
The changelog, release notes and release version are generated by the CI based on the commits added since the latest release.

### Triggering new releases

In order to trigger a new release, create a new branch from the latest main and run `make initiate-release`.
This will create a new commit with a message which the CI uses to initiate the release process.
The commit will contain a change to the CHANGELOG.md and VERSION inferred based on the commits added since the previous release.

Create a PR against the `main` branch and merge it once it is approved. Monitor the release process and ensure that:

* The correct tag has been created for the newly created release.

* A pre-release is created in Github Releases for the newly created tag.

* A new OLM bundle has been generated and added to the candidate channel.

### Forcing a release version

A release version can be forced by running

```sh
make initiate-release-as RELEASE_VERSION=<version>
```

For example,

```sh
make initiate-release-as RELEASE_VERSION=1.4.0
```

## Meetings
___
- Regular East friendly meeting: [Thursday at 08:00 CET (Central European Time)](https://meet.google.com/gwy-vssi-hfr)
  - [Meeting notes and Agenda](https://docs.google.com/document/d/1Iy3CRIEzsHUhtMuzCVRX-8fbmsivcu2iju1J2vN2knQ/edit?usp=meetingnotes&showmeetingnotespromo=true).

- Regular West friendly meeting: [Thursday at 16:30 CET (Central European Time)](https://meet.google.com/gwy-vssi-hfr)
  - [Meeting notes and Agenda](https://docs.google.com/document/d/1Iy3CRIEzsHUhtMuzCVRX-8fbmsivcu2iju1J2vN2knQ/edit?usp=meetingnotes&showmeetingnotespromo=true).

## Contact
___
- CoreOS Slack #observability-users and ping @obo-support.
- [Mailing list](mso-users@redhat.com)
- Github Team: @rhobs/observability-operator-maintainers

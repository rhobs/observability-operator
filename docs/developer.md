# Developer Docs

In order to contribute to this project, make sure you have go 1.20 installed
on your local machine.

## Dependencies

The build system assumes the following binaries are in the `PATH`:

```
make git go npm kind podman (or docker)
```
Once these tools are installed, run `make tools` to install all required
project dependencies to ``tmp/bin``
```
make tools
```
## Development Environment Setup

To setup your development, it is recommended to run the helper script
`hack/kind/setup.sh`. This script does the following
    * sets up a Kind cluster
    * installs OLM
    * sets up a local registry to push locally built operator and bundle images

```sh
./hack/kind/setup.sh
```

Once done, the cluster can be deleted by running

```
kind delete cluster --name obs-operator
```

## Running End to End tests

Running E2E locally against the kind cluster that was setup following the
instructions above is as easy as running `./test/run-e2e.sh`.

```sh
 ./test/run-e2e.sh
```
**NOTE:** `./test/run-e2e.sh --help` shows options that are useful when
rerunning tests.


## Running Operator In Kind Cluster

Observability Operator relies heavily on (forked) Promethues Operator to do
most of the heavy lifting of creation of Prometheus and Alertmanager.
The easiest way to use deploy prometheus operator is to run the `observability-operator`
bundle which installs `obo` and `prometheus-operator`,  and then scale the
`observabilty-operator` deployment to 0, so that the operator can be  run
out of cluster using `go run`

### Create Operator Bundle - 0.0.0-dev

The command below builds the operator and olm bundle, pushes it the
local-registry running in Kind cluster.
```sh
make operator-image bundle-image operator-push bundle-push  \
    IMAGE_BASE="local-registry:30000/observability-operator" \
    VERSION=0.0.0-dev  \
    PUSH_OPTIONS=--tls-verify=false
```


### Run Operator Bundle - 0.0.0-dev

Use operator-sdk to run the operator bundle

```
./tmp/bin/operator-sdk run bundle \
    local-registry:30000/observability-operator-bundle:0.0.0-dev \
    --install-mode AllNamespaces \
    --namespace operators --skip-tls

```
Running the above should deploy operator and show

```
INFO[0044] OLM has successfully installed "observability-operator.v0.0.0-dev"

```

### Running Operator Out of cluster


Scale down the operator installed in cluster

```
kubectl scale --replicas=0 -n operators deploy/observability-operator
```

Run the operator out of cluster
* Set the `KUBECONFIG` environment variable to your local cluster and run the
  operator with `go run ./cmd/operator/...`.

* Alternatively, you can also set the `kubeconfig` on the command line:
  `go run ./cmd/operator/... --kubeconfig <path-to-kubeconfig>`

```
go run ./cmd/operator/... --zap-devel  --zap-log-level=100 2>&1 |
  tee tmp/operator.log
```

# Manifest generation

The Kubernetes CRDs and the ClusterRole needed for their management are
generated from go types in `pkg/apis`. Run `make generate` to regenerate the
Kubernetes manifests when changing these files.

This project uses  [controller-gen](https://github.com/kubernetes-sigs/controller-tools/tree/master/cmd/controller-gen)
for code generation. For detailed information on the available code generation
markers, please refer to the controller-gen CLI page in
the [kubebuilder documentation](https://book.kubebuilder.io/reference/markers.html)


# Contributions

## Commit message convention

Commit messages need to comply to the [Conventional Commits specification](https://www.conventionalcommits.org/en/v1.0.0/)
and should be structured as follows:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

The type and description are used to generate a changelog and determine the
next release version.
Most commonly used types are:

* `fix:` a commit of the type fix patches a bug in your codebase. This
  correlates with PATCH in Semantic Versioning.

* `feat:` a commit of the type feat introduces a new feature to the codebase.
  This correlates with MINOR in Semantic Versioning.

* `BREAKING CHANGE:` a commit that has a footer BREAKING CHANGE:, or appends a
 `!` after the type/scope, introduces a breaking API change (correlating with
 MAJOR in Semantic Versioning).
 A BREAKING CHANGE can be part of commits of any type.

Other than `fix:` and `feat:`, the following type can also be used: `build:`,
`chore:`, `ci:`, `docs:`, `style:`, `refactor:`, `perf:` and `test:`.

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

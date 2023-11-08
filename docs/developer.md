* [Developer documentation](#developer-documentation)
* [Contribution guidelines](#contribution-guidelines)
* [Release management](#release-management)

# Developer Documentation

## Development Tools

The build system assumes the following binaries are in the `PATH`:

```
make
git
go
npm
kind
podman (or docker)
```

Make sure you have installed on your local machine the Go version mentioned in
the root `go.mod` file

Once these tools are installed, run `make tools` to install all required
project dependencies to ``tmp/bin``

```sh
make tools
```

## Environment Setup

To setup the environment, it is recommended to run the helper script
`hack/kind/setup.sh`. The script does the following:
* Sets up a local Kind cluster.
* Installs the Operator Lifecycle Manager (OLM) in the cluster.
* Sets up a local registry to push the local operator and bundle images.

```sh
./hack/kind/setup.sh
```

Once done, the cluster can be deleted by running

```
kind delete cluster --name obs-operator
```

## Running End to End tests

To run the E2E tests locally against the kind cluster that was setup following
the instructions above:

```sh
./test/run-e2e.sh
```

**NOTE:** `./test/run-e2e.sh --help` shows options that are useful when
rerunning tests.

## Running the Operator locally

Observability Operator relies heavily on the (forked) Prometheus Operator to do
most of the heavy lifting of creation of Prometheus and Alertmanager.  The
easiest way to use deploy prometheus operator is to run the
`observability-operator` bundle which installs both `observability-operator`
and `prometheus-operator`,  and then scale the `observability-operator`
deployment to 0, so that the operator can be  run out of cluster using `go run`

### Create the development Operator Bundle

The command below builds the operator + OLM bundle and pushes them to the
local-registry running in Kind cluster:

```sh
make operator-image bundle-image operator-push bundle-push  \
    IMAGE_BASE="local-registry:30000/observability-operator" \
    VERSION=0.0.0-dev  \
    PUSH_OPTIONS=--tls-verify=false
```

### Deploy the development Operator Bundle

Use `operator-sdk` to deploy the operator bundle:

```sh
./tmp/bin/operator-sdk run bundle \
    local-registry:30000/observability-operator-bundle:0.0.0-dev \
    --install-mode AllNamespaces \
    --namespace operators --skip-tls

```
Running the above should deploy operator and show

```
INFO[0044] OLM has successfully installed "observability-operator.v0.0.0-dev"

```

### Run the Operator from your local machine

Scale down the operator currently deployed in cluster:

```sh
kubectl scale --replicas=0 -n operators deployment/observability-operator
```

Start the operator locally:

```sh
# replace ~/.kube/config with your own KUBECONFIG path if different.
go run ./cmd/operator/... --zap-devel  --zap-log-level=100 --kubeconfig ~/.kube/config 2>&1 |
  tee tmp/operator.log
```

# Contribution guidelines

## Manifests and code generation

The Kubernetes CRDs and the ClusterRole needed for their management are
generated from the Go types in `pkg/apis`. Run `make generate` to regenerate the
Kubernetes manifests when changing these files.

The project uses [controller-gen](https://github.com/kubernetes-sigs/controller-tools/tree/master/cmd/controller-gen)
for code generation. For detailed information on the available code generation
markers, please refer to the controller-gen CLI page in
the [kubebuilder documentation](https://book.kubebuilder.io/reference/markers.html)

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

# Release management

The project follows [SemVer 2.0.0](https://semver.org/)

```
Given a version number MAJOR.MINOR.PATCH, increment the:

MAJOR version when you make incompatible API changes,
MINOR version when you add functionality in a backwards compatible manner, and
PATCH version when you make backwards compatible bug fixes.
Additional labels for pre-release and build metadata are available as extensions to the MAJOR.MINOR.PATCH format.
```

Creating new releases is fully automated and requires minimal human
interaction. The changelog, release notes and release version are generated by
the CI based on the commits added since the latest release.

## How to create a new release

```sh
git checkout main
git pull
git checkout -b cut-new-release
make initiate-release
```

This will create a new commit with a message which the CI uses to initiate the
release process. The commit will contain changes to the `CHANGELOG.md` and
`VERSION` files based on the commits added since the previous release.

After reviewing the commit, push the local branch to your repository fork and
create a pull request against the `main` branch.

Once the pull request is approved and merged, monitor the [release
automation](./design/release.md) and ensure that:

* The correct tag has been created for the newly created release.

* A **pre-release** is created in Github Releases for the newly created tag.

* A new OLM bundle has been generated and added to the candidate channel.

The final **manual** step is to uncheck the `Set as a pre-release` checkbox in
the GitHub release page to mark this release as production-ready. This will
trigger GitHub actions that publish the operator and catalog images to the
stable channel.

### How to force a release version

A release version can be forced like this:

```sh
RELEASE_VERSION=1.4.0
make initiate-release-as RELEASE_VERSION=$RELEASE_VERSION
```

## How to publish a new release to the Community Catalog

After a new release has been published on the GitHub repository, it's time to update the operator version listed in the [OpenShift community catalog](https://github.com/redhat-openshift-ecosystem/community-operators-prod).

The gist of the steps involved are as follows:
1. Find the commit in the `olm-catalog` branch corresponding to
	 the release.
1. Copy the bundle directory to the community operators repository fork.
1. Submit a pull request to the Community catalog.

In this example below, the release used is `0.0.25`.

### Find the commit which updates the olm-catalog

1. Go to [`stable workflow`](https://github.com/rhobs/observability-operator/actions/workflows/olm-stable.yaml)
2. Find the `release` job corresponding to `0.0.25` version
  * e.g. https://github.com/rhobs/observability-operator/actions/runs/5795055095/job/15705870971

3. Expand the `Publish` step to find the commit
4. Navigate to the last line of the step and [find the commit](https://github.com/rhobs/observability-operator/actions/runs/5795055095/job/15705870971#step:4:901)
   that was pushed to the `olm-catalog` branch

```
Writing manifest to image destination
Storing signatures
To https://github.com/rhobs/observability-operator
   e8d7666..4d3769f  HEAD -> olm-catalog
             ☝️ is the commit that got pushed
```

### Copy the bundle directory from the `olm-catalog` branch to community-catalog

Assumptions:

* You have already forked and cloned `https://github.com/redhat-openshift-ecosystem/community-operators-prod` to your machine.
* The `origin` remote refers to the upstream repository and the `fork` remote to the forked repository.

1. Copy the bundle directory from the checkout to your
   community-catalog fork:

```sh
VERSION=0.0.25
# From the observability-operator local directory.
git checkout -b publish-$VERSION 4d3769f

cd ../../redhat-openshift-ecosystem/community-operators-prod
git checkout main
git fetch && git reset --hard origin/main

git checkout -b observability-operator-$VERSION
mkdir -p operators/observability-operator/$VERSION
cp -r ../../rhobs/observability-operator/bundle operators/observability-operator/$VERSION
```

2. validate the bundle, note this should have been already done in CI, however
   it is good to validate before submission.

```sh
operator-sdk bundle validate operators/observability-operator/$VERSION \
	--select-optional name=operatorhub \
	--optional-values=k8s-version=1.21 \
	--select-optional suite=operatorframework
```

3. Commit (signed) and push for review

NOTE: The commit message follows a convention (see `git log`) and must be signed

```sh
git add operators/observability-operator/$VERSION
git commit -sS  -m "operator observability-operator ($VERSION)"
git push -u fork HEAD
```

4. submit the pull request, e.g: https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/3084

5. There may be some changes that are required to fix the bundle. Make those
	 changes and ensure the fixes are ported back to Observability Operator repo.
   E.g.: https://github.com/rhobs/observability-operator/pull/333

## How to update the forked prometheus-operator

The observability operator uses a forked (downstream) version of the upstream
Prometheus operator to ensure that it can be installed alongside the upstream
operator without conflict. The forked operator is maintained at
(https://github.com/rhobs/obo-prometheus-operator/) which contains the
instructions to synchronize from upstream.

When a new downstream version is available (e.g. `v0.69.0-rhobs1`), you need to
update these 2 filesand replace the old version by the new one:

* `go.mod`
* `deploy/dependencies/kustomization.yaml`

Then regenerate all the manifests:

```sh
make generate
```

Finally submit a pull request with all the changes.

Example: (https://github.com/rhobs/observability-operator/pull/380)

# monitoring-stack-operator
The monitoring stack operator is a Kubernetes operator which enables the management of independent and self-contained, Prometheus-based monitoring stacks through Kubernetes CRDs.

The project is based on the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library. 

## Development

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
* `BREAKING CHANGE:` a commit that has a footer BREAKING CHANGE:, or appends a ! after the type/scope, introduces a breaking API change (correlating with MAJOR in Semantic Versioning). A BREAKING CHANGE can be part of commits of any type.

Other than `fix:` and `feat:`, the following type can also be used: `build:`, `chore:`, `ci:`, `docs:`, `style:`, `refactor:`, `perf:` and `test:`.

### Manifest generation
The Kubernetes CRDs and the ClusterRole needed for their management are generated from go types in `pkg/apis`.   
Run `make generate` to regenerate the Kubernetes manifests when changing these files.

This project uses the [controller-gen](https://github.com/kubernetes-sigs/controller-tools/tree/master/cmd/controller-gen) for code generation.
For detailed information on the available code generation markers, please refer to the controller-gen CLI page in the Kubebuilder documentation: https://book.kubebuilder.io/reference/markers.html

### Running the operator in kind
* Install [kind](https://github.com/kubernetes-sigs/kind)
* Create a local kubernetes cluster with `kind create cluster`. 
* Build the operator image with `IMAGE_TAG_BASE=monitoring-stack-operator make operator-image`.
* Load the image into your cluster with `kind load docker-image monitoring-stack-operator:$(cat VERSION)`
* Apply the CRDs by running `kubectl apply -k deploy/crds/`
* Set the `KUBECONFIG` environment variable to your local cluster and run the with `go run cmd/operator/main.go`. 
  * Alternatively, you can also set the kubeconfig on the command line: `go run cmd/operator/main.go --kubeconfig <path-to-kubeconfig>`


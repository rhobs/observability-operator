# monitoring-stack-operator
The monitoring stack operator is a Kubernetes operator which enables the management of independent and self-contained, Prometheus-based monitoring stacks through Kubernetes CRDs.

The project is based on the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library. 

## Development

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


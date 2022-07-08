#!/usr/bin/env bash
set -e -u -o pipefail

declare -r SCRIPT_PATH=$(readlink -f "$0")
declare -r SCRIPT_DIR=$(cd $(dirname "$SCRIPT_PATH") && pwd)
declare -r PROJECT_ROOT_DIR="$SCRIPT_DIR/../../"
declare -r OP_NAME="obs-operator"

### create a separate config file for kind clusters
export KUBECONFIG=~/.kube/kind/obs-operator

err() {
  echo -e "ERROR: $@" >&2
}

info() {
  echo -e "INFO: $@"
}

die() {
  echo -e "FATAL: $@" >&2
  exit 1
}

# turn the control-plane into to infra to validate if the operator pods
# get deployed on infra nodes
label_infra_node() {
  kubectl label nodes  \
    -l kubernetes.io/hostname=="${OP_NAME}-control-plane"\
    node-role.kubernetes.io/infra="" \

  [[ $( kubectl get nodes -l node-role.kubernetes.io/infra=="" -o name | wc -l ) -ne 0 ]] || {
    die "No infra nodes were found"
  }
}

setup_olm() {
  [[ -x $PROJECT_ROOT_DIR/tmp/bin/operator-sdk  ]] || {
    die "operator-sdk not found - did you run 'make tools'?"
  }
  $PROJECT_ROOT_DIR/tmp/bin/operator-sdk olm install
  echo -e "      ---------------------------------- \n"
}

run_registry() {
  info "Deploying Registry ..."

  grep -q local-registry /etc/hosts || {
    err "No local-registry entry in hosts; try:"
    info "echo \"127.0.0.1\tlocal-registry\" | sudo tee -a /etc/hosts"
    die "/etc/hosts does not contain local-registry entry"
  }

  kubectl apply -f ./hack/kind/registry.yaml -n operators
  kubectl rollout status deployment local-registry -n operators

  kubectl wait --for=condition=Available deploy local-registry -n operators --timeout=300s

   curl --connect-timeout 5 --max-time 10 \
    --retry 5 --retry-delay 0  --retry-max-time 40 \
    http://local-registry:30000 || {
    die "Failed to reach local-registry running"
  }
  echo -e "      ---------------------------------- \n"

}

setup_cluster() {
  kind create cluster \
    -v 10 \
    --name $OP_NAME \
    --config "$SCRIPT_DIR/config.yaml" \
    --kubeconfig $KUBECONFIG \
    2>&1 | tee tmp/logs/kind.log

  info "Wait for all pods to be ready ..."
  kubectl wait --for=condition=Ready pods --all --all-namespaces --timeout=300s
}

install_kubectl() {
    if ! command -v kubectl &> /dev/null; then
    info "kubectl not found, attempting to install"
    mkdir -p tmp/bin
    curl -o tmp/bin/kubectl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x tmp/bin/kubectl
    export PATH="$(pwd)/tmp/bin/:$PATH"
fi

}

main() {
  ## NOTE: all paths are relative to the root of the project
  cd "$PROJECT_ROOT_DIR"
  mkdir -p tmp/logs "$(dirname $KUBECONFIG)"

  install_kubectl
  setup_cluster
  label_infra_node
  setup_olm
  run_registry

  info "Waiting for cluster boot to complete ..."
  echo -e "      ---------------------------------- \n"
  kubectl wait --for=condition=Ready pods --all --all-namespaces --timeout=300s
  echo -e "      ---------------------------------- \n"

  info "NOTE: To delete the cluster, run:
    kind delete cluster --name obs-operator \n"

  info "NOTE: export KUBECONFIG=$KUBECONFIG \n"

  info "NEXT: deploy Prometheus Operator and required CRDs by:
    kubectl --kubeconfig=$KUBECONFIG create -k deploy/crds/kubernetes
    kubectl --kubeconfig=$KUBECONFIG create -k deploy/dependencies"

  return $?
}

main "$@"

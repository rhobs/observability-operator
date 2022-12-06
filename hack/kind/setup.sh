#!/usr/bin/env bash
set -e -u -o pipefail

declare -r SCRIPT_PATH=$(readlink -f "$0")
declare -r SCRIPT_DIR=$(cd $(dirname "$SCRIPT_PATH") && pwd)
declare -r PROJECT_ROOT_DIR="$SCRIPT_DIR/../../"
declare -r OP_NAME="obs-operator"

# use tools installed to tmp/bin
export PATH="$PROJECT_ROOT_DIR/tmp/bin/:$PATH"

### create a separate config file for kind clusters
export KUBECONFIG=~/.kube/kind/obs-operator

header(){
  local title="🔆🔆🔆  $*  🔆🔆🔆 "

  local len=40
  if [[ ${#title} -gt $len ]]; then
    len=${#title}
  fi

  echo -e "\n\n  \033[1m${title}\033[0m"
  echo -n "━━━━━"
  printf '━%.0s' $(seq "$len")
  echo "━━━━━"
}

info(){
  echo " 🔔 $*"
}

ok(){
  echo " ✅ $*"
}

err(){
  echo " 🛑 $*"
}

die(){
  echo -e "\n ✋ $* "
  echo -e "──────────────────── ⛔️⛔️⛔️ ────────────────────────\n"
  exit 1
}

line(){
  local len="$1"; shift

  echo -n "────"
  printf '─%.0s' $(seq "$len")
  echo "────────"
}

validate_prerequisites(){
  header "Validating Prerequisites"

  local fail=0

  [[ -x $PROJECT_ROOT_DIR/tmp/bin/operator-sdk  ]] || {
    err "operator-sdk not found - did you run 'make tools'?"
    fail=1
  }

  grep -q local-registry /etc/hosts || {
    err "/etc/hosts does not contain local-registry entry"

    info "No local-registry entry in hosts; run:"
    echo -e "    ❯ echo \"127.0.0.1\tlocal-registry\" | sudo tee -a /etc/hosts"
    fail=1
  }

  return $fail
}

# turn the control-plane into to infra to validate if the operator pods
# get deployed on infra nodes
label_infra_node() {
  header "Labeling Infra Nodes"

  kubectl label nodes  \
    -l kubernetes.io/hostname=="${OP_NAME}-control-plane"\
    node-role.kubernetes.io/infra="" \

  [[ $( kubectl get nodes -l node-role.kubernetes.io/infra=="" -o name | wc -l ) -ne 0 ]] || {
    die "No infra nodes were found"
  }
  line 50
}


setup_olm() {
  header "Install OLM"
  $PROJECT_ROOT_DIR/tmp/bin/operator-sdk olm install
  line 50
}

run_registry() {
  header "Deploying Registry"

  kubectl apply -f ./hack/kind/registry.yaml -n operators
  kubectl rollout status deployment local-registry -n operators

  kubectl wait --for=condition=Available deploy local-registry -n operators --timeout=300s

   curl --connect-timeout 5 --max-time 10 \
    --retry 5 --retry-delay 0  --retry-max-time 40 \
    http://local-registry:30000 || {
    die "Failed to reach local-registry running"
  }
  line 50

}

setup_cluster() {
  header "Setting up Cluster"

  mkdir -p "$(dirname $KUBECONFIG)"

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
  header "Installing kubectl"

  if ! command -v kubectl &> /dev/null; then
    info "kubectl not found, attempting to install"
    mkdir -p tmp/bin
    curl -o tmp/bin/kubectl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x tmp/bin/kubectl
  fi

}

create_platform_mon_crds() {
  header "Installing Monitoring CRDs"

  kubectl create -k deploy/crds/kubernetes
  kubectl wait --for=condition=Established crds --all --timeout=120s
}


main() {
  ## NOTE: all paths are relative to the root of the project
  cd "$PROJECT_ROOT_DIR"
  validate_prerequisites || die "fix errors above and rerun the script"

  mkdir -p tmp/logs

  install_kubectl
  setup_cluster
  label_infra_node
  setup_olm
  run_registry
  create_platform_mon_crds

  header "Waiting for cluster boot to complete ..."
  kubectl wait --for=condition=Ready pods --all --all-namespaces --timeout=300s
  line 50

  header "Setup Complete"
  info "NOTE: To delete the cluster, run:
    kind delete cluster --name obs-operator \n"

  info "NOTE: export KUBECONFIG=$KUBECONFIG \n"

  info "NEXT: deploy Prometheus Operator and required CRDs by:
    ❯ kubectl --kubeconfig=$KUBECONFIG create -k deploy/crds/kubernetes
    ❯ kubectl --kubeconfig=$KUBECONFIG create -k deploy/dependencies"
  line 50

  return $?
}

main "$@"

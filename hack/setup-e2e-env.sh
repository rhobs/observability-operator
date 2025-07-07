#!/usr/bin/env bash
set -e -u -o pipefail

SCRIPT_PATH=$(readlink -f "$0")
declare -r SCRIPT_PATH
SCRIPT_DIR=$(cd "$(dirname "$SCRIPT_PATH")" && pwd)
declare -r SCRIPT_DIR
declare -r PROJECT_ROOT_DIR="$SCRIPT_DIR/../"

# Default configuration
declare INSTALL_KIND=true
declare INSTALL_KUBECTL=true

declare SETUP_CLUSTER=true
declare SETUP_OLM=true
declare SETUP_REGISTRY=true
declare VALIDATE_ONLY=false
declare CLUSTER_NAME="obs-operator"
declare KIND_VERSION="v0.29.0"
declare KIND_IMAGE="kindest/node:v1.33.1"
declare KUBECONFIG_PATH="$HOME/.kube/kind/obs-operator"
declare SHOW_USAGE=false
declare -a EXTRA_PACKAGES=()
declare SKIP_HOST_CHECK=false

# shellcheck source=/dev/null
source "$PROJECT_ROOT_DIR/test/lib/utils.bash"

# set PATH to include tools
export PATH="$PROJECT_ROOT_DIR/tmp/bin/:$PATH"

print_usage() {
    local scr
    scr="$(basename "$0")"

    read -r -d '' help <<-EOF_HELP || true
		Usage:
		  $scr [OPTIONS] [PACKAGES...]

		This script sets up the e2e test environment with kind cluster.
		It can be used both locally and in CI environments.
		Run 'make tools' first to install project-specific tools.

		Options:
		  -h|--help                    show this help
		  --validate-only             only validate prerequisites, don't install anything
		  --no-kind                   skip kind installation
		  --no-kubectl                skip kubectl installation
		  --no-cluster                skip cluster setup
		  --no-olm                    skip OLM installation
		  --no-registry               skip local registry setup
		  --skip-host-check           skip /etc/hosts validation (useful for CI)
		  --cluster-name NAME         kind cluster name (default: $CLUSTER_NAME)
		  --kind-version VERSION      kind version to install (default: $KIND_VERSION)
		  --kind-image IMAGE          kind node image (default: $KIND_IMAGE)
		  --kubeconfig PATH           kubeconfig file path (default: $KUBECONFIG_PATH)

		Examples:
		  $scr                        # Full setup with default options
		  $scr --validate-only        # Only check prerequisites
		  $scr --no-cluster           # Install tools but don't create cluster
		  $scr curl jq tree htop       # Install additional system packages
		  $scr --skip-host-check --no-olm  # CI-friendly setup without OLM

	EOF_HELP

    echo -e "$help"
    return 0
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
        -h | --help)
            SHOW_USAGE=true
            return 0
            ;;
        --validate-only)
            VALIDATE_ONLY=true
            shift
            ;;
        --no-kind)
            INSTALL_KIND=false
            shift
            ;;
        --no-kubectl)
            INSTALL_KUBECTL=false
            shift
            ;;
        --no-cluster)
            SETUP_CLUSTER=false
            shift
            ;;
        --no-olm)
            SETUP_OLM=false
            shift
            ;;
        --no-registry)
            SETUP_REGISTRY=false
            shift
            ;;
        --skip-host-check)
            SKIP_HOST_CHECK=true
            shift
            ;;
        --cluster-name)
            shift
            CLUSTER_NAME="$1"
            shift
            ;;
        --kind-version)
            shift
            KIND_VERSION="$1"
            shift
            ;;
        --kind-image)
            shift
            KIND_IMAGE="$1"
            shift
            ;;
        --kubeconfig)
            shift
            KUBECONFIG_PATH="$1"
            shift
            ;;

        --*)
            die "Unknown option: $1"
            ;;
        *)
            EXTRA_PACKAGES+=("$1")
            shift
            ;;
        esac
    done
    return 0
}

check_command() {
    local cmd="$1"
    if command -v "$cmd" &> /dev/null; then
        ok "$cmd is available"
        return 0
    else
        warn "$cmd is not available"
        return 1
    fi
}

validate_prerequisites() {
    header "Validating Prerequisites"

    local fail=0

    # Check for required system tools
    check_command "go" || fail=1
    check_command "git" || fail=1
    check_command "curl" || fail=1

    # Check for project tools that should be installed via 'make tools'
    local tools_missing=0
    local required_tools=("operator-sdk" "oc" "controller-gen" "kustomize")

    for tool in "${required_tools[@]}"; do
        if [[ -x "$PROJECT_ROOT_DIR/tmp/bin/$tool" ]]; then
            ok "$tool is available in tmp/bin"
        else
            warn "$tool not found in tmp/bin"
            tools_missing=1
        fi
    done

    if [[ $tools_missing -eq 1 ]]; then
        info "Install missing tools by running:"
        echo "    ❯ make tools"
        # Don't fail here as some operations might still work
    fi

    # Check /etc/hosts for local-registry entry (skip in CI)
    if ! $SKIP_HOST_CHECK; then
        if grep -q "local-registry" /etc/hosts 2>/dev/null; then
            ok "/etc/hosts contains local-registry entry"
        else
            warn "/etc/hosts does not contain local-registry entry"
            info "Add local-registry entry:"
            printf "    ❯ echo \"127.0.0.1\tlocal-registry\" | sudo tee -a /etc/hosts"
            fail=1
        fi
    else
        ok "Skipping /etc/hosts check (CI mode)"
    fi

    return $fail
}

install_kind() {
    if ! $INSTALL_KIND; then
        info "Skipping kind installation"
        return 0
    fi

    header "Installing kind"

    if command -v kind &> /dev/null; then
        local current_version
        current_version=$(kind version | grep -o 'v[0-9.]*' | head -1)
        if [[ "$current_version" == "$KIND_VERSION" ]]; then
            ok "kind $KIND_VERSION is already installed"
            return 0
        else
            info "kind $current_version is installed, but we need $KIND_VERSION"
        fi
    fi

    local os arch bin_path
    os=$(go env GOOS)
    arch=$(go env GOARCH)
    bin_path="$PROJECT_ROOT_DIR/tmp/bin"

    mkdir -p "$bin_path"

    info "Downloading kind $KIND_VERSION for $os/$arch"
    curl -sSLo "$bin_path/kind" "https://kind.sigs.k8s.io/dl/$KIND_VERSION/kind-$os-$arch"
    chmod +x "$bin_path/kind"

    ok "kind $KIND_VERSION installed to $bin_path/kind"
}

install_kubectl() {
    if ! $INSTALL_KUBECTL; then
        info "Skipping kubectl installation"
        return 0
    fi

    header "Installing kubectl"

    if command -v kubectl &> /dev/null; then
        ok "kubectl is already available"
        return 0
    fi

    local bin_path="$PROJECT_ROOT_DIR/tmp/bin"
    mkdir -p "$bin_path"

    info "Downloading latest kubectl"
    curl -sSLo "$bin_path/kubectl" "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x "$bin_path/kubectl"

    ok "kubectl installed to $bin_path/kubectl"
}



install_extra_packages() {
    if [[ ${#EXTRA_PACKAGES[@]} -eq 0 ]]; then
        return 0
    fi

    header "Installing Extra Packages"

    for package in "${EXTRA_PACKAGES[@]}"; do
        if command -v "$package" &> /dev/null; then
            ok "$package is already available"
            continue
        fi

        info "Installing $package"
        local install_success=false

        # Try different package managers
        if command -v apt-get &> /dev/null; then
            info "Using apt-get to install $package"
            if sudo apt-get update && sudo apt-get install -y "$package"; then
                install_success=true
            fi
        elif command -v dnf &> /dev/null; then
            info "Using dnf to install $package"
            if sudo dnf install -y "$package"; then
                install_success=true
            fi
        elif command -v yum &> /dev/null; then
            info "Using yum to install $package"
            if sudo yum install -y "$package"; then
                install_success=true
            fi
        elif command -v zypper &> /dev/null; then
            info "Using zypper to install $package"
            if sudo zypper install -y "$package"; then
                install_success=true
            fi
        elif command -v pacman &> /dev/null; then
            info "Using pacman to install $package"
            if sudo pacman -S --noconfirm "$package"; then
                install_success=true
            fi
        elif command -v brew &> /dev/null; then
            info "Using brew to install $package"
            if brew install "$package"; then
                install_success=true
            fi
        elif command -v apk &> /dev/null; then
            info "Using apk to install $package"
            if sudo apk add "$package"; then
                install_success=true
            fi
        else
            warn "No supported package manager found"
            info "Supported package managers: apt-get, dnf, yum, zypper, pacman, brew, apk"
            info "Please install $package manually"
            continue
        fi

        if ! $install_success; then
            warn "Failed to install $package"
            info "Please install $package manually or check if the package name is correct"
        fi
    done
}

setup_cluster() {
    if ! $SETUP_CLUSTER; then
        info "Skipping cluster setup"
        return 0
    fi

    header "Setting up Kind Cluster"

    # Set up kubeconfig path
    mkdir -p "$(dirname "$KUBECONFIG_PATH")"
    export KUBECONFIG="$KUBECONFIG_PATH"

    # Check if cluster already exists
    if kind get clusters | grep -q "^$CLUSTER_NAME$"; then
        info "Kind cluster '$CLUSTER_NAME' already exists"
        ok "Using existing cluster"
    else
        info "Creating kind cluster '$CLUSTER_NAME' with image '$KIND_IMAGE'"
        mkdir -p "$PROJECT_ROOT_DIR/tmp/logs"

        kind create cluster \
            --name "$CLUSTER_NAME" \
            --image "$KIND_IMAGE" \
            --config "$SCRIPT_DIR/kind/config.yaml" \
            --kubeconfig "$KUBECONFIG_PATH" \
            2>&1 | tee "$PROJECT_ROOT_DIR/tmp/logs/kind.log"
    fi

    info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s
    kubectl wait --for=condition=Ready pods --all --all-namespaces --timeout=300s

    # export $KUBECONFIG so its available for other steps
    if [[ "$GITHUB_ACTIONS" ]]; then
        echo "KUBECONFIG=$KUBECONFIG" >> "$GITHUB_ENV"
    fi
    ok "Kind cluster is ready"
}

label_infra_nodes() {
    if ! $SETUP_CLUSTER; then
        return 0
    fi

    header "Labeling Infra Nodes"

    kubectl label nodes \
        -l "kubernetes.io/hostname=${CLUSTER_NAME}-control-plane" \
        node-role.kubernetes.io/infra="" \
        --overwrite

    local infra_nodes
    infra_nodes=$(kubectl get nodes -l node-role.kubernetes.io/infra=="" -o name | wc -l)

    if [[ "$infra_nodes" -eq 0 ]]; then
        die "No infra nodes were found"
    fi

    ok "Labeled $infra_nodes infra node(s)"
}

setup_olm() {
    if ! $SETUP_OLM; then
        info "Skipping OLM installation"
        return 0
    fi

    header "Installing OLM"

    # Use operator-sdk from project tools
    local operator_sdk="$PROJECT_ROOT_DIR/tmp/bin/operator-sdk"
    if [[ ! -x "$operator_sdk" ]]; then
        die "operator-sdk not found in tmp/bin - run 'make tools' first"
    fi

    # Pin to OLM v0.28.0 because v0.29.0 fails on Kind
    "$operator_sdk" olm install --version 0.28.0

    ok "OLM installed successfully"
}

setup_registry() {
    if ! $SETUP_REGISTRY; then
        info "Skipping local registry setup"
        return 0
    fi

    header "Setting up Local Registry"

    kubectl apply -f "$SCRIPT_DIR/kind/registry.yaml" -n operators
    kubectl rollout status deployment local-registry -n operators --timeout=300s
    kubectl wait --for=condition=Available deploy local-registry -n operators --timeout=300s
    # add local-registry to /etc/hosts in case we run in a github action
    if [[ "$GITHUB_ACTIONS" ]]; then
        echo "Detected github actions run, adding \"127.0.0.1 local-registry\" to \"/etc/hosts\""
        echo "127.0.0.1 local-registry" | sudo tee -a /etc/hosts
    fi

    # Test registry connectivity
    local max_attempts=10
    local attempt=0
    while [[ $attempt -lt $max_attempts ]]; do
        if curl --connect-timeout 5 --max-time 10 http://local-registry:30000 &>/dev/null; then
            break
        fi
        attempt=$((attempt + 1))
        info "Waiting for registry to be ready ($attempt/$max_attempts)..."
        sleep 5
    done

    if [[ $attempt -eq $max_attempts ]]; then
        die "Failed to reach local registry"
    fi

    ok "Local registry is ready"
}

create_monitoring_crds() {
    if ! $SETUP_CLUSTER; then
        return 0
    fi

    header "Installing Monitoring CRDs"

    kubectl create -k "$PROJECT_ROOT_DIR/deploy/crds/kubernetes" || \
        kubectl replace -k "$PROJECT_ROOT_DIR/deploy/crds/kubernetes"

    kubectl wait --for=condition=Established crds --all --timeout=120s

    ok "Monitoring CRDs installed"
}

print_config() {
    header "Configuration"
    cat <<-EOF
		  Cluster Name:     $CLUSTER_NAME
		  Kind Version:     $KIND_VERSION
		  Kind Image:       $KIND_IMAGE
		  Kubeconfig:       $KUBECONFIG_PATH
		  Install Kind:     $INSTALL_KIND
		  Install kubectl:  $INSTALL_KUBECTL
		  Setup Cluster:    $SETUP_CLUSTER
		  Setup OLM:        $SETUP_OLM
		  Setup Registry:   $SETUP_REGISTRY
		  Extra Packages:   ${EXTRA_PACKAGES[*]:-none}
		  Skip Host Check:  $SKIP_HOST_CHECK

	EOF
    line 50
}

main() {
    parse_args "$@" || die "Failed to parse args"

    if $SHOW_USAGE; then
        print_usage
        exit 0
    fi

    cd "$PROJECT_ROOT_DIR"

    print_config

    # Install tools first
    install_kind
    install_kubectl
    install_extra_packages

    # Validate prerequisites after package installation
    validate_prerequisites || die "Fix prerequisite errors above and rerun"

    if $VALIDATE_ONLY; then
        ok "Validation completed successfully"
        exit 0
    fi

    # Set up cluster and components
    setup_cluster
    if $SETUP_CLUSTER; then
        label_infra_nodes
        setup_olm
        setup_registry
        create_monitoring_crds

        header "Waiting for cluster to stabilize..."
        kubectl wait --for=condition=Ready pods --all --all-namespaces --timeout=300s
        line 50
    fi

    header "Setup Complete"

    if $SETUP_CLUSTER; then
        info "To use this cluster, run:"
        echo "    ❯ export KUBECONFIG=$KUBECONFIG_PATH"
        echo ""
        info "To delete the cluster, run:"
        echo "    ❯ kind delete cluster --name $CLUSTER_NAME"
        echo ""
        info "To run e2e tests, run:"
        echo "    ❯ export KUBECONFIG=$KUBECONFIG_PATH"
        echo "    ❯ ./test/run-e2e.sh"
    fi

    line 50
    ok "All done! 🎉"

    return 0
}

main "$@"

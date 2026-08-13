#!/bin/bash
#
# Uninstall observability-operator and all its managed resources from a cluster.
# Works with both OpenShift (oc) and plain Kubernetes (kubectl).
#
# Usage: hack/uninstall.sh [--force]
#   --force  Skip confirmation prompt

set -euo pipefail

FORCE=false
[[ "${1:-}" == "--force" ]] && FORCE=true

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RESET='\033[0m'

if command -v oc &>/dev/null; then
    CLI=oc
else
    CLI=kubectl
fi

log()  { echo -e "${GREEN}[uninstall]${RESET} $*"; }
warn() { echo -e "${YELLOW}[uninstall]${RESET} $*"; }
err()  { echo -e "${RED}[uninstall]${RESET} $*" >&2; }

if ! command -v jq &>/dev/null; then
    err "jq is required but not found in PATH"
    exit 1
fi

delete_if_exists() {
    local resource="$1"
    shift
    if $CLI get "$resource" "$@" &>/dev/null 2>&1; then
        log "Deleting $resource $*"
        $CLI delete "$resource" "$@" --wait=false 2>/dev/null || true
    fi
}

delete_all_in_namespace() {
    local resource="$1"
    local ns="${2:-}"
    local ns_flag=()
    [[ -n "$ns" ]] && ns_flag=(-n "$ns")
    if "$CLI" get "$resource" ${ns_flag[@]+"${ns_flag[@]}"} &>/dev/null 2>&1; then
        local items
        items=$("$CLI" get "$resource" ${ns_flag[@]+"${ns_flag[@]}"} -o name 2>/dev/null) || true
        if [[ -n "$items" ]]; then
            log "Deleting all $resource ${ns:+in $ns}"
            echo "$items" | xargs -r "$CLI" delete ${ns_flag[@]+"${ns_flag[@]}"} --wait=false 2>/dev/null || true
        fi
    fi
}

remove_finalizers() {
    local resource="$1"
    shift
    log "Removing finalizers from $resource $*"
    $CLI patch "$resource" "$@" --type=merge -p '{"metadata":{"finalizers":null}}' 2>/dev/null || true
}

# -------------------------------------------------------------------
# Confirmation
# -------------------------------------------------------------------
if [[ "$FORCE" != "true" ]]; then
    warn "This will delete ALL observability-operator resources from the cluster:"
    echo "  - ObservabilityInstallers, MonitoringStacks, ThanosQueriers, UIPlugins"
    echo "  - OpenTelemetryCollectors, TempoStacks (installed by ObservabilityInstaller)"
    echo "  - OLM Subscriptions, CSVs, CatalogSources for the operator and its dependencies"
    echo "  - Operator deployments, CRDs, RBAC, and namespace"
    echo ""
    read -rp "Continue? [y/N] " answer
    [[ "$answer" =~ ^[Yy]$ ]] || { log "Aborted."; exit 0; }
fi

# -------------------------------------------------------------------
# Phase 1: Delete custom resources (operands first, then operator CRs)
# -------------------------------------------------------------------
log "=== Phase 1: Delete custom resources ==="

# ObservabilityInstaller-managed operands
for cr in opentelemetrycollectors.opentelemetry.io tempostacks.tempo.grafana.com; do
    if $CLI api-resources --api-group="${cr#*.}" &>/dev/null 2>&1; then
        delete_all_in_namespace "$cr" ""
        # Also get across all namespaces
        items=$($CLI get "$cr" --all-namespaces -o json 2>/dev/null | \
            jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
        for item in $items; do
            ns="${item%%/*}"
            name="${item#*/}"
            log "Deleting $cr $name in $ns"
            $CLI delete "$cr" "$name" -n "$ns" --wait=false 2>/dev/null || true
        done
    fi
done

# Perses operands
for cr in persesdashboards.perses.dev persesdatasources.perses.dev persesglobaldatasources.perses.dev perses.perses.dev; do
    if $CLI get crd "$cr" &>/dev/null 2>&1; then
        items=$($CLI get "$cr" --all-namespaces -o json 2>/dev/null | \
            jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
        for item in $items; do
            ns="${item%%/*}"
            name="${item#*/}"
            $CLI delete "$cr" "$name" -n "$ns" --wait=false 2>/dev/null || true
        done
    fi
done

# Prometheus-operator operands created by MonitoringStack (monitoring.rhobs group)
for cr in prometheuses.monitoring.rhobs alertmanagers.monitoring.rhobs servicemonitors.monitoring.rhobs \
          prometheusrules.monitoring.rhobs podmonitors.monitoring.rhobs probes.monitoring.rhobs \
          alertmanagerconfigs.monitoring.rhobs scrapeconfigs.monitoring.rhobs prometheusagents.monitoring.rhobs; do
    if $CLI get crd "$cr" &>/dev/null 2>&1; then
        items=$($CLI get "$cr" --all-namespaces -o json 2>/dev/null | \
            jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
        for item in $items; do
            ns="${item%%/*}"
            name="${item#*/}"
            $CLI delete "$cr" "$name" -n "$ns" --wait=false 2>/dev/null || true
        done
    fi
done

# Operator's own CRs
for cr in observabilityinstallers.observability.openshift.io uiplugins.observability.openshift.io \
          monitoringstacks.monitoring.rhobs thanosqueriers.monitoring.rhobs; do
    if $CLI get crd "$cr" &>/dev/null 2>&1; then
        items=$($CLI get "$cr" --all-namespaces -o json 2>/dev/null | \
            jq -r '.items[] | "\(.metadata.namespace // "")/\(.metadata.name)"' 2>/dev/null) || true
        for item in $items; do
            ns="${item%%/*}"
            name="${item#*/}"
            if [[ -n "$ns" ]]; then
                log "Deleting $cr $name in $ns"
                $CLI delete "$cr" "$name" -n "$ns" --wait=false 2>/dev/null || true
            else
                log "Deleting $cr $name"
                $CLI delete "$cr" "$name" --wait=false 2>/dev/null || true
            fi
        done
    fi
done

# Wait a bit for controllers to process deletions
log "Waiting for CR deletions to propagate..."
sleep 5

# -------------------------------------------------------------------
# Phase 2: Remove stuck finalizers on CRs that won't delete
# -------------------------------------------------------------------
log "=== Phase 2: Clear stuck finalizers ==="

for cr in observabilityinstallers.observability.openshift.io uiplugins.observability.openshift.io \
          monitoringstacks.monitoring.rhobs thanosqueriers.monitoring.rhobs \
          prometheuses.monitoring.rhobs alertmanagers.monitoring.rhobs \
          perses.perses.dev persesdashboards.perses.dev; do
    if $CLI get crd "$cr" &>/dev/null 2>&1; then
        stuck=$($CLI get "$cr" --all-namespaces -o json 2>/dev/null | \
            jq -r '.items[] | select(.metadata.deletionTimestamp != null) | "\(.metadata.namespace // "")/\(.metadata.name)"' 2>/dev/null) || true
        for item in $stuck; do
            ns="${item%%/*}"
            name="${item#*/}"
            if [[ -n "$ns" ]]; then
                remove_finalizers "$cr" "$name" -n "$ns"
            else
                remove_finalizers "$cr" "$name"
            fi
        done
    fi
done

# -------------------------------------------------------------------
# Phase 3: Delete OLM resources (Subscriptions, CSVs, CatalogSources)
# -------------------------------------------------------------------
log "=== Phase 3: Delete OLM resources ==="

# Subscriptions (search all namespaces)
for pattern in '^observability-operator$' '^opentelemetry-product$' '^tempo-product$'; do
    items=$($CLI get subscriptions.operators.coreos.com --all-namespaces -o json 2>/dev/null | \
        jq -r --arg pat "$pattern" '.items[] | select(.metadata.name | test($pat)) | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
    for item in $items; do
        ns="${item%%/*}"
        name="${item#*/}"
        log "Deleting subscription $name in $ns"
        $CLI delete subscriptions.operators.coreos.com "$name" -n "$ns" 2>/dev/null || true
    done
done

# ClusterServiceVersions (search all namespaces)
items=$($CLI get clusterserviceversion --all-namespaces -o json 2>/dev/null | \
    jq -r '.items[] | select(.metadata.name | test("^observability-operator\\.")) | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
for item in $items; do
    ns="${item%%/*}"
    name="${item#*/}"
    log "Deleting CSV $name in $ns"
    $CLI delete clusterserviceversion "$name" -n "$ns" 2>/dev/null || true
done

# Operators (cluster-scoped, operators.coreos.com)
for pattern in observability-operator opentelemetry-product tempo-product; do
    items=$($CLI get operators.operators.coreos.com -o name 2>/dev/null | grep "$pattern" || true)
    for item in $items; do
        log "Deleting $item"
        $CLI delete "$item" 2>/dev/null || true
    done
done

# CatalogSources (search all namespaces)
items=$($CLI get catalogsource --all-namespaces -o json 2>/dev/null | \
    jq -r '.items[] | select(.metadata.name | test("^observability-operator")) | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
for item in $items; do
    ns="${item%%/*}"
    name="${item#*/}"
    log "Deleting CatalogSource $name in $ns"
    $CLI delete catalogsource "$name" -n "$ns" 2>/dev/null || true
done

# InstallPlans referencing the operator (search all namespaces)
items=$($CLI get installplan --all-namespaces -o json 2>/dev/null | \
    jq -r '.items[] | select(.spec.clusterServiceVersionNames[]? | test("observability-operator")) | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
for item in $items; do
    ns="${item%%/*}"
    name="${item#*/}"
    log "Deleting InstallPlan $name in $ns"
    $CLI delete installplan "$name" -n "$ns" 2>/dev/null || true
done

# -------------------------------------------------------------------
# Phase 4: Delete operator deployments and supporting resources
# -------------------------------------------------------------------
log "=== Phase 4: Delete operator deployments ==="

for dep in observability-operator obo-prometheus-operator perses-operator; do
    items=$($CLI get deployment --all-namespaces -o json 2>/dev/null | \
        jq -r --arg dep "$dep" '.items[] | select(.metadata.name == $dep) | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
    for item in $items; do
        ns="${item%%/*}"
        name="${item#*/}"
        log "Deleting deployment $name in $ns"
        $CLI delete deployment "$name" -n "$ns" 2>/dev/null || true
    done
done

# ConsolePlugins (cluster-scoped, OpenShift only)
if $CLI api-resources | grep -q consoleplugins 2>/dev/null; then
    for p in console-dashboards-plugin troubleshooting-panel-console-plugin distributed-tracing-console-plugin logging-view-plugin monitoring-console-plugin; do
        if $CLI get consoleplugins "$p" &>/dev/null 2>&1; then
            log "Deleting ConsolePlugin $p"
            $CLI delete consoleplugins "$p" 2>/dev/null || true
        fi
    done
fi

# -------------------------------------------------------------------
# Phase 5: Delete CRDs
# -------------------------------------------------------------------
log "=== Phase 5: Delete CRDs ==="

for crd in \
    monitoringstacks.monitoring.rhobs \
    thanosqueriers.monitoring.rhobs \
    uiplugins.observability.openshift.io \
    observabilityinstallers.observability.openshift.io \
    alertmanagerconfigs.monitoring.rhobs \
    alertmanagers.monitoring.rhobs \
    podmonitors.monitoring.rhobs \
    probes.monitoring.rhobs \
    prometheusagents.monitoring.rhobs \
    prometheuses.monitoring.rhobs \
    prometheusrules.monitoring.rhobs \
    scrapeconfigs.monitoring.rhobs \
    servicemonitors.monitoring.rhobs \
    thanosrulers.monitoring.rhobs \
    perses.perses.dev \
    persesdashboards.perses.dev \
    persesdatasources.perses.dev \
    persesglobaldatasources.perses.dev; do
    delete_if_exists crd "$crd"
done

# -------------------------------------------------------------------
# Phase 6: Clean up RBAC and other cluster-scoped resources
# -------------------------------------------------------------------
log "=== Phase 6: Clean up RBAC ==="

for kind in clusterrole clusterrolebinding; do
    items=$($CLI get "$kind" -o name 2>/dev/null | grep -E 'observability-operator|obo-prometheus|perses-operator|monitoring-stack|thanos-querier' || true)
    for item in $items; do
        log "Deleting $item"
        $CLI delete "$item" 2>/dev/null || true
    done
done

# Webhooks
for kind in validatingwebhookconfigurations mutatingwebhookconfigurations; do
    items=$($CLI get "$kind" -o name 2>/dev/null | grep -E 'observability-operator|monitoring\.rhobs' || true)
    for item in $items; do
        log "Deleting $item"
        $CLI delete "$item" 2>/dev/null || true
    done
done

# -------------------------------------------------------------------
# Phase 7: Clean up namespaces (optional)
# -------------------------------------------------------------------
log "=== Phase 7: Clean up namespaces ==="

$CLI delete namespace observability-operator openshift-cluster-observability-operator --ignore-not-found --wait=false 2>/dev/null || true

log "=== Uninstall complete ==="
log "Run '$CLI get crd | grep -E \"rhobs|observability|perses\"' to verify all CRDs are gone."

#!/bin/bash
#
# Uninstall observability-operator and all its managed resources from a cluster.
# Works with both OpenShift (oc) and plain Kubernetes (kubectl).
#
# Usage: hack/uninstall.sh

set -euo pipefail

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

delete_all() {
    log "Deleting all $1"
    $CLI delete "$1" --all --all-namespaces --wait=false --ignore-not-found 2>/dev/null || true
}

remove_finalizers() {
    log "Removing finalizers from $*"
    $CLI patch "$@" --type=merge -p '{"metadata":{"finalizers":null}}' 2>/dev/null || true
}

# -------------------------------------------------------------------
# Phase 1: Delete custom resources (operands first, then operator CRs)
# -------------------------------------------------------------------
log "=== Phase 1: Delete custom resources ==="

# ObservabilityInstaller-managed operands
for cr in opentelemetrycollectors.opentelemetry.io tempostacks.tempo.grafana.com; do
     delete_all "$cr"
done

# Perses operands
for cr in persesdashboards.perses.dev persesdatasources.perses.dev persesglobaldatasources.perses.dev perses.perses.dev; do
    delete_all "$cr"
done

# Prometheus-operator operands created by MonitoringStack (monitoring.rhobs group)
for cr in prometheuses.monitoring.rhobs alertmanagers.monitoring.rhobs servicemonitors.monitoring.rhobs \
          prometheusrules.monitoring.rhobs podmonitors.monitoring.rhobs probes.monitoring.rhobs \
          alertmanagerconfigs.monitoring.rhobs scrapeconfigs.monitoring.rhobs prometheusagents.monitoring.rhobs; do
    delete_all "$cr"
done

# Operator's own CRs
for cr in observabilityinstallers.observability.openshift.io uiplugins.observability.openshift.io \
          monitoringstacks.monitoring.rhobs thanosqueriers.monitoring.rhobs; do
    delete_all "$cr"
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
        $CLI delete subscriptions.operators.coreos.com "$name" -n "$ns" --ignore-not-found 2>/dev/null || true
    done
done

# ClusterServiceVersions (search all namespaces)
items=$($CLI get clusterserviceversion --all-namespaces -o json 2>/dev/null | \
    jq -r '.items[] | select(.metadata.name | test("^observability-operator\\.")) | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
for item in $items; do
    ns="${item%%/*}"
    name="${item#*/}"
    log "Deleting CSV $name in $ns"
    $CLI delete clusterserviceversion "$name" -n "$ns" --ignore-not-found 2>/dev/null || true
done

# Operators (cluster-scoped, operators.coreos.com)
for pattern in observability-operator opentelemetry-product tempo-product; do
    items=$($CLI get operators.operators.coreos.com -o name 2>/dev/null | grep "$pattern" || true)
    for item in $items; do
        log "Deleting $item"
        $CLI delete "$item" --ignore-not-found 2>/dev/null || true
    done
done

# CatalogSources (search all namespaces)
items=$($CLI get catalogsource --all-namespaces -o json 2>/dev/null | \
    jq -r '.items[] | select(.metadata.name | test("^observability-operator")) | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
for item in $items; do
    ns="${item%%/*}"
    name="${item#*/}"
    log "Deleting CatalogSource $name in $ns"
    $CLI delete catalogsource "$name" -n "$ns" --ignore-not-found 2>/dev/null || true
done

# InstallPlans referencing the operator (search all namespaces)
items=$($CLI get installplan --all-namespaces -o json 2>/dev/null | \
    jq -r '.items[] | select(.spec.clusterServiceVersionNames[]? | test("observability-operator")) | "\(.metadata.namespace)/\(.metadata.name)"' 2>/dev/null) || true
for item in $items; do
    ns="${item%%/*}"
    name="${item#*/}"
    log "Deleting InstallPlan $name in $ns"
    $CLI delete installplan "$name" -n "$ns" --ignore-not-found 2>/dev/null || true
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
        $CLI delete deployment "$name" -n "$ns" --ignore-not-found 2>/dev/null || true
    done
done

# ConsolePlugins (cluster-scoped, OpenShift only)
for p in console-dashboards-plugin troubleshooting-panel-console-plugin distributed-tracing-console-plugin logging-view-plugin monitoring-console-plugin; do
    $CLI delete consoleplugins "$p" --ignore-not-found 2>/dev/null || true
done

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
    log "Deleting CRD $crd"
    $CLI delete crd "$crd" --ignore-not-found 2>/dev/null || true
done

# -------------------------------------------------------------------
# Phase 6: Clean up RBAC and other cluster-scoped resources
# -------------------------------------------------------------------
log "=== Phase 6: Clean up RBAC ==="

for kind in clusterrole clusterrolebinding; do
    items=$($CLI get "$kind" -o name 2>/dev/null | grep -E 'observability-operator|obo-prometheus|perses-operator|monitoring-stack|thanos-querier' || true)
    for item in $items; do
        log "Deleting $item"
        $CLI delete "$item" --ignore-not-found 2>/dev/null || true
    done
done

# Webhooks
for kind in validatingwebhookconfigurations mutatingwebhookconfigurations; do
    items=$($CLI get "$kind" -o name 2>/dev/null | grep -E 'observability-operator|monitoring\.rhobs' || true)
    for item in $items; do
        log "Deleting $item"
        $CLI delete "$item" --ignore-not-found 2>/dev/null || true
    done
done

# -------------------------------------------------------------------
# Phase 7: Clean up namespaces (optional)
# -------------------------------------------------------------------
log "=== Phase 7: Clean up namespaces ==="

$CLI delete namespace observability-operator openshift-cluster-observability-operator --ignore-not-found --wait=false 2>/dev/null || true

log "=== Uninstall complete ==="
log "Run '$CLI get crd | grep -E \"rhobs|observability|perses\"' to verify all CRDs are gone."

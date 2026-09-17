#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT

declare -r OPERATORS_NS="${OPERATORS_NS:-coo}"
declare -r TEST_NAMESPACE="${MUST_GATHER_TEST_NAMESPACE:-must-gather-e2e}"
declare -r STACK_NAME="${MUST_GATHER_STACK_NAME:-must-gather}"
declare -r ARTIFACT_DIR="${ARTIFACT_DIR:-$PROJECT_ROOT/tmp/must-gather-e2e}"
declare -r OUTPUT_DIR="$ARTIFACT_DIR/output"

cleanup() {
	oc delete namespace "$TEST_NAMESPACE" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup EXIT

rm -rf "$OUTPUT_DIR"
mkdir -p "$ARTIFACT_DIR"

oc delete namespace "$TEST_NAMESPACE" --ignore-not-found --wait=true
oc create namespace "$TEST_NAMESPACE"

oc apply -f - <<EOF
apiVersion: monitoring.rhobs/v1alpha1
kind: MonitoringStack
metadata:
  name: $STACK_NAME
  namespace: $TEST_NAMESPACE
spec:
  resourceSelector: {}
  prometheusConfig:
    replicas: 1
  alertmanagerConfig:
    replicas: 1
EOF

oc -n "$TEST_NAMESPACE" wait --for=create "statefulset/prometheus-$STACK_NAME" --timeout=2m
oc -n "$TEST_NAMESPACE" wait --for=create "statefulset/alertmanager-$STACK_NAME" --timeout=2m
oc -n "$TEST_NAMESPACE" rollout status "statefulset/prometheus-$STACK_NAME" --timeout=5m
oc -n "$TEST_NAMESPACE" rollout status "statefulset/alertmanager-$STACK_NAME" --timeout=5m

if [[ -z "${MUST_GATHER_IMAGE:-}" ]]; then
	MUST_GATHER_IMAGE=$(oc -n "$OPERATORS_NS" get deployment observability-operator -o jsonpath='{.spec.template.spec.containers[0].image}')
fi
if [[ -z "$MUST_GATHER_IMAGE" ]]; then
	echo "MUST_GATHER_IMAGE is empty" >&2
	exit 1
fi

oc adm must-gather \
	--dest-dir="$OUTPUT_DIR" \
	--image="$MUST_GATHER_IMAGE" \
	-- /usr/bin/gather 2>&1 | tee "$ARTIFACT_DIR/must-gather-command.log"

prometheus_file=$(find "$OUTPUT_DIR" -type f \
	-path "*/monitoring/observability-operator/$TEST_NAMESPACE/$STACK_NAME/prometheus/status/flags.json" \
	-print -quit)
alertmanager_file=$(find "$OUTPUT_DIR" -type f \
	-path "*/monitoring/observability-operator/$TEST_NAMESPACE/$STACK_NAME/alertmanager/status.json" \
	-print -quit)
operator_file=$(find "$OUTPUT_DIR" -type f \
	-path "*/monitoring/observability-operator/operator.yaml" \
	-print -quit)

for file in "$prometheus_file" "$alertmanager_file" "$operator_file"; do
	if [[ -z "$file" || ! -s "$file" ]]; then
		echo "expected non-empty must-gather artifact was not found" >&2
		exit 1
	fi
done

if find "$OUTPUT_DIR" -type f -name '*.stderr' -print -quit | grep -q .; then
	echo "must-gather produced stderr artifacts" >&2
	exit 1
fi

jq -e '.status == "success" and (.data | type == "object")' "$prometheus_file" >/dev/null
jq -e '.cluster.status == "ready" and (.versionInfo | type == "object")' "$alertmanager_file" >/dev/null
grep -q '^apiVersion: v1$' "$operator_file"
grep -q '^kind: List$' "$operator_file"

debug_log=$(find "$OUTPUT_DIR" -type f -name gather-debug.log -print -quit)
if [[ -z "$debug_log" || ! -s "$debug_log" ]] || grep -Eq 'WARN:|FAILED:' "$debug_log"; then
	echo "must-gather debug log is missing or contains failures" >&2
	exit 1
fi

echo "must-gather end-to-end test passed"

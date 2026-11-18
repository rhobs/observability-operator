#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/../../.." && pwd)
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
export MOCK_DIR="$tmpdir"

yq eval -r '.spec.steps[0].script' \
  "$root/konflux/integration-tests/tasks/upgrade-fixture.yaml" > "$tmpdir/fixture.sh"

# Exercise the actual Tekton script without a cluster. The rollout mock
# deliberately succeeds even when a StatefulSet has no replicas.
cat > "$tmpdir/oc" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ ${1-} == -n ]]; then shift 2; fi
command=${1-}
shift || true
case "$command" in
  create)
    if [[ ${1-} == namespace ]]; then exit 0; fi
    cat > "$MOCK_DIR/created"
    case $(yq eval -r '.kind' "$MOCK_DIR/created") in
      MonitoringStack) cp "$MOCK_DIR/created" "$MOCK_DIR/stack.yaml" ;;
      ConfigMap) cp "$MOCK_DIR/created" "$MOCK_DIR/baseline.json" ;;
      *) echo 'unexpected created resource' >&2; exit 1 ;;
    esac
    ;;
  get)
    case ${1-} in
      configmap) cat "$MOCK_DIR/baseline.json" ;;
      monitoringstacks.monitoring.rhobs)
        jq -nc '{metadata:{generation:2},status:{conditions:[{type:"Reconciled",status:"True",observedGeneration:2}]}}' ;;
      prometheuses.monitoring.rhobs)
        jq -nc '{spec:{externalLabels:{coo_upgrade_verified:"true"}}}' ;;
      statefulset/alertmanager-upgrade-probe)
        if [[ ${2-} == -o ]]; then
          jq -nc --argjson replicas "${MOCK_ALERT_REPLICAS:-1}" --argjson ready "${MOCK_ALERT_READY:-1}" \
            '{spec:{replicas:$replicas},status:{readyReplicas:$ready}}'
        fi ;;
      statefulset/prometheus-upgrade-probe)
        jq -nc --argjson replicas "${MOCK_PROM_REPLICAS:-1}" --argjson ready "${MOCK_PROM_READY:-1}" \
          '{spec:{replicas:$replicas},status:{readyReplicas:$ready}}' ;;
      pod)
        jq -nc --arg scenario "${MOCK_POD_SCENARIO:-normal}" '
          {
            metadata:{ownerReferences:[{kind:"StatefulSet",name:"prometheus-upgrade-probe",controller:true}]},
            status:{conditions:[{type:"Ready",status:"True"}],containerStatuses:[{name:"prometheus",ready:true}]},
            spec:{
              containers:[{name:"prometheus",volumeMounts:[{name:"data",mountPath:"/prometheus"}]}],
              volumes:[{name:"data",persistentVolumeClaim:{claimName:"prometheus-data"}}]
            }
          } |
          if $scenario == "emptydir" then
            .spec.volumes[0] = {name:"data",emptyDir:{}}
          elif $scenario == "unmounted" then
            .spec.containers[0].volumeMounts[0].mountPath = "/elsewhere"
          elif $scenario == "readonly" then
            .spec.containers[0].volumeMounts[0].readOnly = true
          elif $scenario == "sidecar" then
            .spec.containers[0].volumeMounts = [] |
            .spec.containers += [{name:"sidecar",volumeMounts:[{name:"data",mountPath:"/prometheus"}]}]
          elif $scenario == "changed-claim" then
            .spec.volumes[0].persistentVolumeClaim.claimName = "other-data"
          elif $scenario == "not-ready" then
            .status.conditions[0].status = "False"
          elif $scenario == "container-not-ready" then
            .status.containerStatuses[0].ready = false
          elif $scenario == "wrong-owner" then
            .metadata.ownerReferences[0].name = "other-statefulset"
          else . end' ;;
      pvc)
        jq -nc --arg name "$2" --arg uid "${MOCK_PVC_UID:-original-uid}" \
          --arg volume "${MOCK_PVC_VOLUME:-original-pv}" --arg phase "${MOCK_PVC_PHASE:-Bound}" \
          '{metadata:{name:$name,uid:$uid},spec:{volumeName:$volume},status:{phase:$phase}}' ;;
      monitoringstacks.monitoring.rhobs,*) exit 0 ;;
      *) echo "unexpected get: $*" >&2; exit 1 ;;
    esac ;;
  patch|wait|rollout) exit 0 ;;
  *) echo "unexpected command: $command $*" >&2; exit 1 ;;
esac
EOF
chmod +x "$tmpdir/oc"
export PATH="$tmpdir:$PATH"

OPERATION=prepare bash "$tmpdir/fixture.sh" > "$tmpdir/prepare.log"
jq -e '.data.pvc | fromjson | .name == "prometheus-data" and .uid == "original-uid" and .volume == "original-pv"' \
  "$tmpdir/baseline.json" > /dev/null
OPERATION=verify bash "$tmpdir/fixture.sh" > "$tmpdir/verify.log"

expect_failure() {
  local label=$1
  local message=$2
  shift 2
  if env OPERATION=verify "$@" bash "$tmpdir/fixture.sh" > "$tmpdir/failure.log" 2>&1; then
    echo "Expected failure: $label" >&2
    exit 1
  fi
  if ! grep -Fq "$message" "$tmpdir/failure.log"; then
    echo "Unexpected failure: $label" >&2
    cat "$tmpdir/failure.log" >&2
    exit 1
  fi
}

expect_failure 'Alertmanager scaled to zero' 'Expected one ready alertmanager replica' \
  MOCK_ALERT_REPLICAS=0 MOCK_ALERT_READY=0
expect_failure 'Alertmanager scaled down with stale status' 'Expected one ready alertmanager replica' \
  MOCK_ALERT_REPLICAS=0 MOCK_ALERT_READY=1
expect_failure 'Alertmanager not ready' 'Expected one ready alertmanager replica' MOCK_ALERT_READY=0
expect_failure 'Alertmanager missing readiness' 'Expected one ready alertmanager replica' MOCK_ALERT_READY=null
expect_failure 'Prometheus scaled to zero' 'Expected one ready prometheus replica' \
  MOCK_PROM_REPLICAS=0 MOCK_PROM_READY=0
for scenario in emptydir unmounted readonly sidecar not-ready container-not-ready wrong-owner; do
  expect_failure "$scenario" 'The ready Prometheus pod must mount a PVC at /prometheus' \
    MOCK_POD_SCENARIO="$scenario"
done
expect_failure 'different mounted claim' 'The Prometheus data PVC or its persistent volume changed' \
  MOCK_POD_SCENARIO=changed-claim
expect_failure 'PVC replaced' 'The Prometheus data PVC or its persistent volume changed' \
  MOCK_PVC_UID=replacement-uid
expect_failure 'PV changed' 'The Prometheus data PVC or its persistent volume changed' \
  MOCK_PVC_VOLUME=replacement-pv
expect_failure 'PVC unbound' 'must be bound' MOCK_PVC_PHASE=Pending

if env OPERATION=prepare MOCK_POD_SCENARIO=emptydir bash "$tmpdir/fixture.sh" > "$tmpdir/failure.log" 2>&1; then
  echo 'Expected prepare to reject an emptyDir data volume' >&2
  exit 1
fi
grep -Fq 'The ready Prometheus pod must mount a PVC at /prometheus' "$tmpdir/failure.log"

echo 'Upgrade fixture mock tests passed'

#!/bin/bash

# safeguards
set -o nounset
set -o errexit
set -o pipefail

get_first_ready_prom_pod() {
  local ns="$1"; shift
  local name="$1"; shift
  readarray -t READY_PROM_PODS < <(
    oc get pods -n "$ns" -l app.kubernetes.io/part-of="$name",app.kubernetes.io/component=prometheus --field-selector=status.phase==Running \
      --no-headers -o custom-columns=":metadata.name"
  )
  echo "${READY_PROM_PODS[0]}"
}

get_first_ready_alertmanager_pod() {
  local ns="$1"; shift
  local name="$1"; shift
  readarray -t READY_AM_PODS < <(
    oc get pods -n "$ns" -l app.kubernetes.io/part-of="$name",app.kubernetes.io/component=alertmanager --field-selector=status.phase==Running \
      --no-headers -o custom-columns=":metadata.name"
  )
  echo "${READY_AM_PODS[0]}"
}

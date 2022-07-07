#!/usr/bin/env bash

set -e -u -o pipefail
trap cleanup INT

# Functions that given a number it creates a namespace
# and in that namespace it creates a monitoring stack
create_monitoring_stack() {

  local stack_number=$1; shift
  local ms_name=stack-$stack_number
  local namespace=loadtest-$stack_number

    monitoring_stack=$(cat <<- EOF
apiVersion: monitoring.rhobs/v1alpha1
kind: MonitoringStack
metadata:
  name: ${ms_name}
  namespace: ${namespace}
  labels:
    load-test: test
spec:
  logLevel: debug
  retention: 15d
  resourceSelector:
    matchLabels:
      load-test-instance: ${ms_name}
EOF
)

  kubectl create namespace "$namespace"
  echo "$monitoring_stack" | kubectl -n "$namespace" apply -f -
}

cleanup() {
  echo "INFO: cleaning up all namespaces"
  kubectl delete ns loadtest-{1..10}
}

main() {
  # Goal: create 10 monitoring stack CRs, wait for OO to
  # reconcile and then clean-up

  echo "INFO: Running load test"
  for ((i=1; i<=10; i++)); do
    create_monitoring_stack "$i"
  done

  # Give some time for OO to reconcile all the MS
  # and create the necessary resources
  local timeout=180
  echo "INFO: sleeping for $timeout"
  sleep "$timeout"

  cleanup
}

 main "$@"

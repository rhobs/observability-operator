#!/usr/bin/env bash

set -e -u -o pipefail

# Functions that given a number it creates a namespace
# and in that namespace it creates a monitoring stack
create_monitoring_stack() {
    stack_number=$1
    MS_NAME=stack-$stack_number
    NAMESPACE=loadtest-$stack_number

    monitoring_stack=$(cat <<- EOF
apiVersion: monitoring.rhobs/v1alpha1
kind: MonitoringStack
metadata:
  name: ${MS_NAME}
  namespace: ${NAMESPACE}
  labels:
    load-test: test
spec:
  logLevel: debug
  retention: 15d
  resourceSelector:
    matchLabels:
      load-test-instance: ${MS_NAME}
EOF
)

    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    echo "$monitoring_stack" | kubectl -n $NAMESPACE apply -f -

}

cleanup() { 
    kubectl delete ns loadtest-{1..10}
}

main() {
  # Goal: create 10 monitoring stack CRs, wait for OO to
  # reconcile and then clean-up
  for ((i=1; i<=10; i++)); do
    create_monitoring_stack $i
  done
  # Give some time for OO to reconcile all the MS
  # and create the necessary resources
  sleep 180
  
  cleanup
}
 main "$@"
#!/bin/bash

cd ..

# Define ANSI color codes
GREEN='\033[0;32m'
ENDCOLOR='\033[0m' 

# Get the current date and time in 'monDD-HHMM' lowercase format
# For example: sep12-1216
TIMESTAMP=$(date +'%b%d-%H%M' | tr '[:upper:]' '[:lower:]')

# Replace IMG_BASE with the image your image registry
IMG_BASE="${IMG_BASE:-"quay.io/jezhu/observability-operator"}"
VERSION="${VERSION:-1.0.0-dev-${TIMESTAMP}}"

print_title() {
  echo -e "\n${GREEN} =============================================== ${ENDCOLOR}\n"
  echo -e "${GREEN} $1 ${ENDCOLOR}"
  echo -e "\n${GREEN} =============================================== ${ENDCOLOR}\n"
}

# Enabled UIPlugins >> openshift.enabled=true
perl -pi -e 's/(flag\.BoolVar\(&openShiftEnabled,\s*"openshift\.enabled",\s*)false/$1true/' ./cmd/operator/main.go

# Build Bundle
print_title "Build Bundle: make operator-image bundle-image operator-push bundle-push"
make operator-image bundle-image operator-push bundle-push \
  IMG_BASE="${IMG_BASE}" \
  VERSION="${VERSION}"

# Delete Previous CatalogSource, Subscription, and ClusterServiceVersion
print_title "Delete Previous ClusterServiceVersion and Subscription"
# oc project openshift-operators
CAT_NAME=$(oc get catalogsource | grep 'observability-operator' | awk '{print $1}') && oc delete catalogsource "${CAT_NAME}"
SUB_NAME=$(oc get subscriptions | grep 'observability-operator' | awk '{print $1}') && oc delete subscriptions "${SUB_NAME}"
CSV_NAME=$(oc get clusterserviceversion | grep 'observability-operator' | awk '{print $1}') && oc delete clusterserviceversion "${CSV_NAME}"

# OR Delete the whole operator 
operator-sdk cleanup observability-operator -n openshift-operators

# Run the bundle using the fully qualified image tag.
print_title "Run Bundle: operator-sdk run bundle" 
operator-sdk run bundle \
  "${IMG_BASE}-bundle:${VERSION}" \
  --install-mode AllNamespaces \
  --namespace openshift-operators \
  --security-context-config restricted

# Revert to Original State and Disable UIPlugins >> openshift.enabled=false
perl -pi -e 's/(flag\.BoolVar\(&openShiftEnabled,\s*"openshift\.enabled",\s*)true/$1false/' ./cmd/operator/main.go

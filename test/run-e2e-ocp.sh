#!/usr/bin/env bash
set -e -u -o pipefail

trap cleanup EXIT

# NOTE: install Observability Operator and run e2e against the installation

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT

# shellcheck source=/dev/null
source "$PROJECT_ROOT/test/lib/utils.bash"

# NOTE: openshift-operators is the namespace used in subscription.yaml to install
# obo, so this is harded coded for the test as well.
declare -r OPERATORS_NS="openshift-operators"

### Configuration
declare NO_INSTALL=false
declare NO_UNINSTALL=false
declare SHOW_USAGE=false
declare POSTPONE_RESTORATION=""

cleanup() {
	# skip cleanup if user requested help
	$SHOW_USAGE && return 0

	delete_obo || true
	return 0
}

install_obo() {
	header "Install Observability Operator"

	$NO_INSTALL && {
		skip "installation of Observability Operator"
		return 0
	}

	# NOTE: catalog-src is added to "openshift-marketplace" namespace
	oc apply -f ./hack/olm/catalog-src.yaml

	# NOTE: obo gets installed to "openshift-operators" namespace
	oc apply -f ./hack/olm/subscription.yaml

	oc -n "$OPERATORS_NS" wait --for=condition=CatalogSourcesUnhealthy=False \
		subscription.operators.coreos.com observability-operator --timeout=60s

	ok "Observability Operator subscription is ready"
	wait_for_operators_ready "$OPERATORS_NS"

	enable_ocp
}

enable_ocp() {
  CSV_NAME=$(oc -n "$OPERATORS_NS" get sub observability-operator -o jsonpath='{.status.installedCSV}')

  local deployment_index
  deployment_index=$(oc -n "$OPERATORS_NS" get csv "${CSV_NAME}" -o json | \
    jq '[.spec.install.spec.deployments[].name] | index("observability-operator")')
  if [[ -z "$deployment_index" || "$deployment_index" == "null" ]]; then
    err "Could not find observability-operator deployment in CSV ${CSV_NAME}"
    exit 1
  fi

  # Retry logic
  max_retries=3
  retry_count=0
  while [ "$retry_count" -lt "$max_retries" ]; do
    if oc -n "$OPERATORS_NS" patch csv "${CSV_NAME}" --type=json \
      -p "[{\"op\": \"add\", \"path\": \"/spec/install/spec/deployments/${deployment_index}/spec/template/spec/containers/0/args/-\", \"value\": \"--openshift.enabled=true\"}]"; then
      ok "Successfully updated CSV ${CSV_NAME}"
      break
    else
      echo "oc patch failed (attempt $((retry_count+1))/$max_retries), retrying..."
    fi
    sleep 10
    ((retry_count++))
    if [ "$retry_count" -eq "$max_retries" ]; then
      err "Failed to update CSV ${CSV_NAME} after $max_retries attempts"
      exit 1
    fi
  done

  # Enable platform monitoring.
  oc label ns "$OPERATORS_NS" openshift.io/cluster-monitoring=true

  oc wait --for=condition=Established crd/uiplugins.observability.openshift.io --timeout=60s
  ok "Enable OCP mode successfully"
}

delete_obo() {
  header "Deleting Observability Operator subscription"

  $NO_UNINSTALL && {
   skip "uninstallation of Observability Operator"
   return 0
  }

  oc delete -n "$OPERATORS_NS" csv \
   -l operators.coreos.com/observability-operator."$OPERATORS_NS"= || true

  oc delete -n "$OPERATORS_NS" installplan,subscriptions \
   -l operators.coreos.com/observability-operator."$OPERATORS_NS"= || true

  oc delete -f hack/olm/subscription.yaml || true
  oc delete -f hack/olm/catalog-src.yaml || true
  oc delete crds "$(oc api-resources --api-group=monitoring.rhobs -o name)"
  ok "Observability operator uninstalled"
}

parse_args() {
  ### while there are args parse them
  while [[ -n "${1+xxx}" ]]; do
		case $1 in
		-h | --help)
			SHOW_USAGE=true
			break
			;; # exit the loop
		--no-install)
			NO_INSTALL=true
			shift
			;;
		--no-uninstall)
			NO_UNINSTALL=true
			shift
			;;
		--postpone-restoration)
			shift
			POSTPONE_RESTORATION=$1
			shift
			;;
		*) return 1 ;; # show usage on everything else
		esac
	done
	return 0
}

print_usage() {
	local scr
	scr="$(basename "$0")"

	read -r -d '' help <<-EOF_HELP || true
		Usage:
		  $scr
		  $scr  --no-install
		  $scr  --no-uninstall
		  $scr  -h|--help

		Options:
		  -h|--help          show this help
		  --no-install       do not install Observability Operator, useful for rerunning tests
		  --no-uninstall     do not uninstall Observability Operator after test
		  --postpone-restoration DURATION
		                     delay operator Subscription restoration after uninstall
		                     tests (e.g. 10m) to allow manual cluster inspection
	EOF_HELP

	echo -e "$help"
	return 0
}

main() {
	parse_args "$@" || die "parse args failed"
	$SHOW_USAGE && {
		print_usage
		exit 0
	}

	cd "$PROJECT_ROOT"
	install_obo

	local -i ret=0
	local -a extra_args=()
	[[ -n "$POSTPONE_RESTORATION" ]] && extra_args+=(--postpone-restoration "$POSTPONE_RESTORATION")
	# Increase test timeout when running on OpenShift because more tests are
	# executed.
	export TEST_TIMEOUT="${TEST_TIMEOUT:-30m}"
	./test/run-e2e.sh --no-deploy --ns "$OPERATORS_NS" --ci "${extra_args[@]}" || ret=$?

	# NOTE: delete_obo will be automatically called when script exits
	return $ret
}

main "$@"

#!/usr/bin/env bash
set -e -u -o pipefail

trap cleanup EXIT

# NOTE: install ObO and run e2e against the installation

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

cleanup() {
	# skip cleanup if user requested help
	$SHOW_USAGE && return 0

	delete_obo || true
	return 0
}

install_obo() {
	header "Install ObO"

	$NO_INSTALL && {
		skip "installation of obo "
		return 0
	}

	# NOTE: catalog-src is added to "openshift-marketplace" namespace
	oc apply -f ./hack/olm/catalog-src.yaml

	# NOTE: obo gets installed to "openshift-operators" namespace
	oc apply -f ./hack/olm/subscription.yaml

	oc -n "$OPERATORS_NS" wait --for=condition=CatalogSourcesUnhealthy=False \
		subscription.operators.coreos.com observability-operator --timeout=60s

	ok "ObO subscription is ready"
	wait_for_operators_ready "$OPERATORS_NS"

	enable_ocp
}

enable_ocp() {
  # Get ObO CSV json file
	CSV_NAME=$(oc -n "$OPERATORS_NS" get sub observability-operator -o jsonpath='{.status.installedCSV}')
	CSV_JSON_FILE=$(mktemp /tmp/"$CSV_NAME"XXXXXX.json)
	if [ -e "$CSV_JSON_FILE" ]; then
		rm -f "$CSV_JSON_FILE"
	fi
	oc -n "$OPERATORS_NS" get csv "${CSV_NAME}" -o json > "$CSV_JSON_FILE"
	# Update CSV json file to enable OCP mode
	ARGS_JSON=$(printf '%s\n' "--openshift.enabled=true" | jq -R . | jq -s .)
	jq --arg container_name operator --argjson args "$ARGS_JSON" '
      (.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name == $container_name) | .args) += $args
    ' "$CSV_JSON_FILE" > /tmp/tmp.$$.json && mv /tmp/tmp.$$.json "$CSV_JSON_FILE"
  ok "Added arguments to container operator in '$CSV_JSON_FILE'."

	oc apply -f "$CSV_JSON_FILE"
	rm -f "$CSV_JSON_FILE"
	oc wait --for=condition=Established crd/uiplugins.observability.openshift.io --timeout=60s
	ok "Enable OCP mode successfully"
}

delete_obo() {
	header "Deleting ObO subscription"

	$NO_UNINSTALL && {
		skip "uninstallation of obo"
		return 0
	}

	oc delete -n "$OPERATORS_NS" csv \
		-l operators.coreos.com/observability-operator."$OPERATORS_NS"= || true

	oc delete -n "$OPERATORS_NS" installplan,subscriptions \
		-l operators.coreos.com/observability-operator."$OPERATORS_NS"= || true

	oc delete -f hack/olm/subscription.yaml || true
	oc delete -f hack/olm/catalog-src.yaml || true
	oc delete crds "$(oc api-resources --api-group=monitoring.rhobs -o name)"
	ok "uninstalled ObO"
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
		  --no-install       do not install OBO, useful for rerunning tests
		  --no-uninstall     do not uninstall OBO after test
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
	./test/run-e2e.sh --no-deploy --ns "$OPERATORS_NS" --ci || ret=$?

	# NOTE: delete_obo will be automatically called when script exits
	return $ret
}

main "$@"

#!/usr/bin/env bash
set -e -u -o pipefail

trap cleanup INT

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT

# shellcheck source=/dev/null
source "$PROJECT_ROOT/test/lib/utils.bash"

declare -r OBO_VERSION="0.0.0-e2e"
declare -r OBO_DEPLOYMENT_YAML="deploy/operator/observability-operator-deployment.yaml"

declare OBO_IMG_REPO="${OBO_IMG_REPO:-local-registry:30000/observability-operator}"
declare BUNDLE_IMG="$OBO_IMG_REPO-bundle:$OBO_VERSION"

declare CI_MODE=false
declare NO_DEPLOY=false
declare NO_BUILDS=false
declare SHOW_USAGE=false
declare LOGS_DIR="tmp/e2e"
declare OPERATORS_NS="operators"
declare TEST_TIMEOUT="15m"

cleanup() {
	info "Cleaning up ..."
	# shell check  ignore word splitting when using jobs -p
	# shellcheck disable=SC2046
	[[ -z "$(jobs -p)" ]] || kill $(jobs -p) || true
}

delete_olm_subscription() {
	header "Delete Old Deployments"

	$CI_MODE && {
		ok "skipping deletion of old deployment in CI mode"
		return 0
	}

	kubectl delete -n "$OPERATORS_NS" csv --all || true
	kubectl delete -n "$OPERATORS_NS" installplan,subscriptions,catalogsource \
		-l operators.coreos.com/observability-operator.operators= || true
	kubectl delete -n "$OPERATORS_NS" installplan,subscriptions,catalogsource \
		-l operators.coreos.com/observability-operator.openshift-operators= || true
	kubectl delete -n "$OPERATORS_NS" catalogsource observability-operator-catalog || true
}

build_bundle() {
	header "Build Operator Bundle"

	$NO_BUILDS && {
		info "skipping building of images"
		return 0
	}

	make operator-image bundle bundle-image \
		IMAGE_BASE="$OBO_IMG_REPO" VERSION="$OBO_VERSION"
}

push_bundle() {
	header "Push Operator Bundle Images"
	$NO_BUILDS && {
		info "skipping pushing images"
		return 0
	}

	make operator-push bundle-push \
		IMAGE_BASE="$OBO_IMG_REPO" VERSION="$OBO_VERSION" \
		PUSH_OPTIONS=--tls-verify=false

}

expose_metrics() {
	header "Expose Operator Metrics"

	kubectl apply -n "$OPERATORS_NS" -f hack/kind/operator-metrics-service.yaml
	sleep 2

	# wait for the service to create an endpoint
	local i=0
	while ! kubectl get -n "$OPERATORS_NS" ep operator-metrics-service && [[ "$i" -le 5 ]]; do
		((i++))
		echo " - $i ... repeat "
		sleep 2
	done
}

assert_no_reconciliation_errors() {
	local stage="$1"
	shift
	local log_file="$LOGS_DIR/operator-$stage.log"

	header "Ensure No Reconciliation Errors [ $stage ]"

	sleep 3
	./tmp/bin/promq -t http://localhost:30001/metrics \
		-q 'controller_runtime_reconcile_errors_total' -o yaml

	local reconcile_errors
	reconcile_errors=$(./tmp/bin/promq \
		-t http://localhost:30001/metrics -o yaml \
		-q 'sum(controller_runtime_reconcile_errors_total)' |
		tr -d ' ' | grep ^v: | cut -f2 -d:)

	info "Reconcile errors [$stage]: $reconcile_errors"

	kubectl logs -n "$OPERATORS_NS" deploy/observability-operator >"$log_file"

	if [[ "$reconcile_errors" -eq 0 ]]; then
		ok "0 reconciliation errors 🥳"
		return 0
	fi

	info "Reconciliation Errors 😱😞"
	line 50
	grep error "$log_file"
	line 50
	warn "Expected 0 reconcile errors but found $reconcile_errors 😱😞"
	return 1

}

run_bundle() {
	header "Running ObO Bundle"

	./tmp/bin/operator-sdk run bundle "$BUNDLE_IMG" \
		--install-mode AllNamespaces --namespace "$OPERATORS_NS" --skip-tls
}

log_events() {
	local ns="$1"
	shift
	kubectl get events -w \
		-o custom-columns=FirstSeen:.firstTimestamp,LastSeen:.lastTimestamp,Count:.count,From:.source.component,Type:.type,Reason:.reason,Message:.message \
		-n "$ns" | tee "$LOGS_DIR/$ns-events.log"
}

watch_obo_errors() {
	local err_log="$1"
	shift

	kubectl logs -f -n "$OPERATORS_NS" deploy/observability-operator | grep error | tee "$err_log"
}

run_e2e() {
	header "Running e2e tests"

	local obo_error_log="$LOGS_DIR/operator-errors.log"

	log_events "$OPERATORS_NS" &
	log_events "e2e-tests" &
	watch_obo_errors "$obo_error_log" &

	local ret=0
	go test -v -failfast -timeout $TEST_TIMEOUT ./test/e2e/... --retain=true | tee "$LOGS_DIR/e2e.log" || ret=1

	# terminte both log_events
	{ jobs -p | xargs -I {} -- pkill -TERM -P {}; } || true

	if [[ "$ret" -ne 0 ]]; then
		# logging of errors may not be immediate, so it is better to read logs again
		# than dumping the $obo_error_log file
		sleep 2
		info "ObO Error Logs"
		line 50
		kubectl logs -n "$OPERATORS_NS" deploy/observability-operator | grep error | tee "$obo_error_log"
		line 50
	fi

	return $ret
}

parse_args() {
	### while there are args parse them
	while [[ -n "${1+xxx}" ]]; do
		case $1 in
		-h | --help)
			SHOW_USAGE=true
			break
			;; # exit the loop
		--no-deploy)
			NO_DEPLOY=true
			shift
			;;
		--no-builds)
			NO_BUILDS=true
			shift
			;;
		--ci)
			CI_MODE=true
			shift
			;;
		--image-repo)
			shift
			OBO_IMG_REPO="$1"
			BUNDLE_IMG="$OBO_IMG_REPO-bundle:$OBO_VERSION"
			shift
			;;
		--ns)
			shift
			OPERATORS_NS=$1
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
		  $scr  --no-deploy
		  $scr  -h|--help


		Options:
		  -h|--help        show this help
		  --ci             run in CI mode
		  --no-deploy      do not build and deploy 0b0, useful for rerunning tests
		  --no-builds      skip building operator images, useful when operator image is already
		                   built and pushed
		  --ns NAMESPACE   namespace to deploy operators (default: $OPERATORS_NS)
		                   For running against openshift use --ns openshift-operators


	EOF_HELP

	echo -e "$help"
	return 0
}

init_logs_dir() {
	rm -rf "$LOGS_DIR-prev"
	mv "$LOGS_DIR" "$LOGS_DIR-prev" || true
	mkdir -p "$LOGS_DIR"
}

restart_obo() {
	header "Restart ObO deployment"

	ensure_obo_deploy_img_is_always_pulled || return 1

	info "scale down ObO"
	kubectl scale -n "$OPERATORS_NS" --replicas=0 deploy/observability-operator
	kubectl wait -n "$OPERATORS_NS" --for=delete pods -l app.kubernetes.io/component=operator --timeout=60s

	info "scale up ObO"
	kubectl scale -n "$OPERATORS_NS" --replicas=1 deploy/observability-operator
	wait_for_operators_ready "$OPERATORS_NS"

	ok "ObO deployment restarted"

}

update_cluster_mon_crds() {
	# try replacing any installed crds; failure is often because the
	# CRDs are absent and in that case, try creating and fail if that fails

	kubectl replace -k deploy/crds/kubernetes ||
		kubectl create -k deploy/crds/kubernetes || return 1

	kubectl wait --for=condition=Established crds --all --timeout=120s

	return 0
}

deploy_obo() {
	header "Build and Deploy Obo"

	delete_olm_subscription || true
	ensure_obo_imgpullpolicy_always_in_yaml
	update_cluster_mon_crds
	build_bundle
	push_bundle
	run_bundle
	wait_for_operators_ready "$OPERATORS_NS"
	expose_metrics
}

ensure_obo_imgpullpolicy_always_in_yaml() {
	$CI_MODE && {
		ok "skipping check of imagePullPolicy in deployment yaml"
		return 0
	}

	local pull_policy
	pull_policy=$(grep '\s\+imagePullPolicy:' "$OBO_DEPLOYMENT_YAML" | tr -d ' ' | cut -f2 -d:)

	[[ "$pull_policy" != "Always" ]] && {
		info "Modify $OBO_DEPLOYMENT_YAML imagePullPolicy -> Always"
		info "  ❯ sed -e 's|imagePullPolicy: .*|imagePullPolicy: Always|g' -i $OBO_DEPLOYMENT_YAML"
		warn "Deployment's imagePullPolicy must be Always instead of $pull_policy"
		return 1
	}

	ok "ObO deployment yaml imagePullPolicy is Always"
}

ensure_obo_deploy_img_is_always_pulled() {
	$CI_MODE && {
		ok "skipping check of imagePullPolicy of ObO deployment"
		return 0
	}

	local pull_policy
	pull_policy=$(kubectl get deploy observability-operator \
		-n "$OPERATORS_NS" \
		-ojsonpath='{.spec.template.spec.containers[].imagePullPolicy}')

	if [[ "$pull_policy" != "Always" ]]; then
		info "Edit $OBO_DEPLOYMENT_YAML imagePullPolicy and redeploy"
		info "  ❯ sed -e 's|imagePullPolicy: .*|imagePullPolicy: Always|g' -i $OBO_DEPLOYMENT_YAML"
		warn "Deployment's imagePullPolicy must be Always instead of $pull_policy"
		return 1
	fi
	ok "ObO deployment imagePullPolicy is Always"
}

reset_env() {
	kubectl delete --wait ns e2e-tests || true
}

print_config() {
	header "Test Configuration"
	cat <<-EOF
		  image repo:  $OBO_IMG_REPO
		  bundle:      $BUNDLE_IMG
		  CI Mode:     $CI_MODE
		  Skip Builds: $NO_BUILDS
		  Skip Deploy: $NO_DEPLOY
		  Operator namespace: $OPERATORS_NS
		  Logs directory: $LOGS_DIR

	EOF
	line 50
}

main() {
	parse_args "$@" || die "parse args failed"
	$SHOW_USAGE && {
		print_usage
		exit 0
	}

	cd "$PROJECT_ROOT"

	# delete the e2e-tests but contine deploying obo
	reset_env & # note must wait before runnng tests
	init_logs_dir
	print_config

	if ! $NO_DEPLOY; then
		deploy_obo
	else
		restart_obo || die "restarting ObO failed 🤕"
	fi

	# wait for the deletion to complete before running tests
	wait

	assert_no_reconciliation_errors pre-e2e ||
		die "ObO has reconciliation errors before running test"

	local ret=0
	run_e2e || ret=1
	assert_no_reconciliation_errors post-e2e || {
		# see: https://github.com/rhobs/observability-operator/issues/200
		skip "post-e2e reconciliation test until #200 is fixed"
		# ret=1
	}

	info "e2e test - exit code: $ret"
	line 50

	return $ret
}

main "$@"
